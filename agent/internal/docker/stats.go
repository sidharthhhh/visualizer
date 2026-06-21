package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ContainerStats struct {
	ContainerID    string    `json:"container_id"`
	CPUPercent     float64   `json:"cpu_percent"`
	MemoryUsageMB  float64   `json:"memory_usage_mb"`
	MemoryLimitMB  float64   `json:"memory_limit_mb"`
	MemoryPercent  float64   `json:"memory_percent"`
	NetworkRxBytes int64     `json:"network_rx_bytes"`
	NetworkTxBytes int64     `json:"network_tx_bytes"`
	DiskReadBytes  int64     `json:"disk_read_bytes"`
	DiskWriteBytes int64     `json:"disk_write_bytes"`
	Pids           int       `json:"pids"`
	Timestamp      time.Time `json:"timestamp"`
}

type ContainerLog struct {
	ContainerID string    `json:"container_id"`
	Timestamp   time.Time `json:"timestamp"`
	Message     string    `json:"message"`
}

type StatsCollector struct {
	client *client.Client
	logger *slog.Logger
}

func NewStatsCollector(logger *slog.Logger) (*StatsCollector, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}

	return &StatsCollector{
		client: cli,
		logger: logger,
	}, nil
}

func (sc *StatsCollector) Close() error {
	return sc.client.Close()
}

func (sc *StatsCollector) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	statsResp, err := sc.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("getting stats: %w", err)
	}
	defer statsResp.Body.Close()

	var stats types.StatsJSON
	if err := json.NewDecoder(statsResp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decoding stats: %w", err)
	}

	cpuPercent := calculateCPUPercent(stats)
	memUsageMB := float64(stats.MemoryStats.Usage) / 1024 / 1024
	memLimitMB := float64(stats.MemoryStats.Limit) / 1024 / 1024
	memPercent := (float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit)) * 100

	var netRx, netTx int64
	for _, v := range stats.Networks {
		netRx += int64(v.RxBytes)
		netTx += int64(v.TxBytes)
	}

	var diskRead, diskWrite int64
	for _, v := range stats.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(v.Op) {
		case "read":
			diskRead += int64(v.Value)
		case "write":
			diskWrite += int64(v.Value)
		}
	}

	return &ContainerStats{
		ContainerID:    containerID,
		CPUPercent:     cpuPercent,
		MemoryUsageMB:  memUsageMB,
		MemoryLimitMB:  memLimitMB,
		MemoryPercent:  memPercent,
		NetworkRxBytes: netRx,
		NetworkTxBytes: netTx,
		DiskReadBytes:  diskRead,
		DiskWriteBytes: diskWrite,
		Pids:           stats.PidsStats.Current,
		Timestamp:      time.Now(),
	}, nil
}

func calculateCPUPercent(stats types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if sysDelta > 0 && cpuDelta > 0 {
		return (cpuDelta / sysDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	}
	return 0
}

func (sc *StatsCollector) GetContainerLogs(ctx context.Context, containerID string, tail int) ([]ContainerLog, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
		Timestamps: true,
	}

	reader, err := sc.client.ContainerLogs(ctx, containerID, options)
	if err != nil {
		return nil, fmt.Errorf("getting logs: %w", err)
	}
	defer reader.Close()

	var logs []ContainerLog
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()

		// Docker multiplexed stream has 8-byte header
		if len(line) > 8 {
			line = line[8:]
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}

		ts, err := time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			ts = time.Now()
		}

		logs = append(logs, ContainerLog{
			ContainerID: containerID,
			Timestamp:   ts,
			Message:     strings.TrimSpace(parts[1]),
		})
	}

	return logs, nil
}

func (sc *StatsCollector) GetAllContainerStats(ctx context.Context) (map[string]*ContainerStats, error) {
	containers, err := sc.client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	result := make(map[string]*ContainerStats)
	for _, ctr := range containers {
		stats, err := sc.GetContainerStats(ctx, ctr.ID)
		if err != nil {
			sc.logger.Error("getting stats", "container", ctr.ID[:12], "error", err)
			continue
		}
		result[ctr.ID] = stats
	}

	return result, nil
}
