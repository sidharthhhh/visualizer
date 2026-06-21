package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/containerscope/agent/internal/docker"
	"github.com/containerscope/agent/internal/host"
)

const version = "0.1.0"

type EnrollRequest struct {
	EnrollmentToken string `json:"enrollment_token"`
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Kernel          string `json:"kernel"`
	CPUCores        int32  `json:"cpu_cores"`
	MemTotal        int64  `json:"mem_total"`
	AgentVersion    string `json:"agent_version"`
}

type EnrollResponse struct {
	ConnectionID string `json:"connection_id"`
	Status       string `json:"status"`
}

type HeartbeatRequest struct {
	ConnectionID string `json:"connection_id"`
}

type ContainerInfo struct {
	RuntimeID string            `json:"runtime_id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	State     string            `json:"state"`
	Labels    map[string]string `json:"labels"`
}

type NetworkInfo struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Subnet string `json:"subnet"`
}

type TopologyRequest struct {
	ConnectionID string          `json:"connection_id"`
	Containers   []ContainerInfo `json:"containers"`
	Networks     []NetworkInfo   `json:"networks"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "localhost:8080"
	}

	token := os.Getenv("ENROLLMENT_TOKEN")
	if token == "" {
		logger.Error("ENROLLMENT_TOKEN is required")
		os.Exit(1)
	}

	logger.Info("starting agent", "version", version, "backend", backendURL)

	info, err := host.Collect(version)
	if err != nil {
		logger.Error("collecting host info", "error", err)
		os.Exit(1)
	}

	baseURL := fmt.Sprintf("http://%s", backendURL)

	enrollResp, err := enroll(baseURL, token, info)
	if err != nil {
		logger.Error("enrollment failed", "error", err)
		os.Exit(1)
	}

	logger.Info("enrolled successfully",
		"connection_id", enrollResp.ConnectionID,
		"status", enrollResp.Status,
	)

	collector, err := docker.NewCollector(logger)
	if err != nil {
		logger.Error("creating docker collector", "error", err)
		os.Exit(1)
	}
	defer collector.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	collector.Subscribe(ctx)

	if err := syncTopology(baseURL, enrollResp.ConnectionID, collector, ctx); err != nil {
		logger.Error("initial topology sync failed", "error", err)
	}

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	topologyTicker := time.NewTicker(60 * time.Second)
	defer topologyTicker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-heartbeatTicker.C:
			if err := heartbeat(baseURL, enrollResp.ConnectionID); err != nil {
				logger.Error("heartbeat failed", "error", err)
			}

		case <-topologyTicker.C:
			if err := syncTopology(baseURL, enrollResp.ConnectionID, collector, ctx); err != nil {
				logger.Error("topology sync failed", "error", err)
			}

		case event := <-collector.Events():
			logger.Info("docker event",
				"type", event.Type,
				"container", event.Container.Name,
			)

		case <-quit:
			logger.Info("shutting down agent")
			return
		}
	}
}

func enroll(baseURL, token string, info *host.Info) (*EnrollResponse, error) {
	req := EnrollRequest{
		EnrollmentToken: token,
		Hostname:        info.Hostname,
		OS:              info.OS,
		Kernel:          info.Kernel,
		CPUCores:        info.CPUCores,
		MemTotal:        info.MemTotal,
		AgentVersion:    info.AgentVersion,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(baseURL+"/api/v1/agent/enroll", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("enrollment failed with status: %d", resp.StatusCode)
	}

	var enrollResp EnrollResponse
	if err := json.NewDecoder(resp.Body).Decode(&enrollResp); err != nil {
		return nil, err
	}

	return &enrollResp, nil
}

func heartbeat(baseURL, connectionID string) error {
	req := HeartbeatRequest{ConnectionID: connectionID}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(baseURL+"/api/v1/agent/heartbeat", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	return nil
}

func syncTopology(baseURL, connectionID string, collector *docker.Collector, ctx context.Context) error {
	containers, err := collector.ListContainers(ctx)
	if err != nil {
		return err
	}

	networks, err := collector.ListNetworks(ctx)
	if err != nil {
		return err
	}

	ctrInfos := make([]ContainerInfo, 0, len(containers))
	for _, ctr := range containers {
		ctrInfos = append(ctrInfos, ContainerInfo{
			RuntimeID: ctr.RuntimeID,
			Name:      ctr.Name,
			Image:     ctr.Image,
			State:     ctr.State,
			Labels:    ctr.Labels,
		})
	}

	netInfos := make([]NetworkInfo, 0, len(networks))
	for _, net := range networks {
		netInfos = append(netInfos, NetworkInfo{
			Name:   net.Name,
			Driver: net.Driver,
			Subnet: net.Subnet,
		})
	}

	req := TopologyRequest{
		ConnectionID: connectionID,
		Containers:   ctrInfos,
		Networks:     netInfos,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := http.Post(baseURL+"/api/v1/agent/topology", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("topology sync failed with status: %d", resp.StatusCode)
	}

	return nil
}
