package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/containerscope/agent/internal/docker"
	"github.com/containerscope/agent/internal/host"
)

type Client struct {
	conn         *grpc.ClientConn
	logger       *slog.Logger
	connectionID string
}

type EnrollRequest struct {
	EnrollmentToken string
	HostInfo        *host.Info
}

type EnrollResponse struct {
	ConnectionId string
	Status       string
}

type HeartbeatRequest struct {
	ConnectionId string
	Timestamp    int64
}

type HeartbeatResponse struct {
	Status string
}

type ContainerInfo struct {
	RuntimeId string
	Name      string
	Image     string
	State     string
	Labels    map[string]string
	Ports     []*PortMapping
	CreatedAt int64
}

type PortMapping struct {
	HostPort      string
	ContainerPort string
	Protocol      string
}

type NetworkInfo struct {
	Name   string
	Driver string
	Scope  string
	Subnet string
}

type TopologySnapshot struct {
	ConnectionId string
	Containers   []*ContainerInfo
	Networks     []*NetworkInfo
	Timestamp    int64
}

type TopologyResponse struct {
	Status string
}

func NewClient(addr string, logger *slog.Logger) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, logger: logger}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) Enroll(token string, info *host.Info) (*EnrollResponse, error) {
	client := NewAgentServiceClient(c.conn)
	resp, err := client.Enroll(context.Background(), &EnrollRequest{
		EnrollmentToken: token,
		HostInfo:        info,
	})
	if err != nil {
		return nil, err
	}
	c.connectionID = resp.ConnectionId
	return &EnrollResponse{
		ConnectionId: resp.ConnectionId,
		Status:       resp.Status,
	}, nil
}

func (c *Client) Heartbeat() (*HeartbeatResponse, error) {
	if c.connectionID == "" {
		return nil, nil
	}
	client := NewAgentServiceClient(c.conn)
	resp, err := client.Heartbeat(context.Background(), &HeartbeatRequest{
		ConnectionId: c.connectionID,
		Timestamp:    time.Now().Unix(),
	})
	if err != nil {
		return nil, err
	}
	return &HeartbeatResponse{
		Status: resp.Status,
	}, nil
}

func (c *Client) SyncTopology(containers []*ContainerInfo, networks []*NetworkInfo) (*TopologyResponse, error) {
	if c.connectionID == "" {
		return nil, fmt.Errorf("not enrolled")
	}
	client := NewAgentServiceClient(c.conn)
	return client.SyncTopology(context.Background(), &TopologySnapshot{
		ConnectionId: c.connectionID,
		Containers:   containers,
		Networks:     networks,
		Timestamp:    time.Now().Unix(),
	})
}

func (c *Client) ConnectionID() string {
	return c.connectionID
}

type DockerCollector struct {
	collector *docker.Collector
	client    *Client
	logger    *slog.Logger
}

func NewDockerCollector(logger *slog.Logger) (*DockerCollector, error) {
	collector, err := docker.NewCollector(logger)
	if err != nil {
		return nil, err
	}
	return &DockerCollector{
		collector: collector,
		logger:    logger,
	}, nil
}

func (dc *DockerCollector) Close() error {
	return dc.collector.Close()
}

func (dc *DockerCollector) Subscribe(ctx context.Context) {
	dc.collector.Subscribe(ctx)
}

func (dc *DockerCollector) Events() <-chan docker.TopologyEvent {
	return dc.collector.Events()
}

func (dc *DockerCollector) SyncNow(ctx context.Context, connectionID string) error {
	containers, err := dc.collector.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	networks, err := dc.collector.ListNetworks(ctx)
	if err != nil {
		return fmt.Errorf("listing networks: %w", err)
	}

	grpcContainers := make([]*ContainerInfo, 0, len(containers))
	for _, ctr := range containers {
		ports := make([]*PortMapping, 0, len(ctr.Ports))
		for _, p := range ctr.Ports {
			ports = append(ports, &PortMapping{
				HostPort:      p.HostPort,
				ContainerPort: p.ContainerPort,
				Protocol:      p.Protocol,
			})
		}

		grpcContainers = append(grpcContainers, &ContainerInfo{
			RuntimeId: ctr.RuntimeID,
			Name:      ctr.Name,
			Image:     ctr.Image,
			State:     ctr.State,
			Labels:    ctr.Labels,
			Ports:     ports,
			CreatedAt: ctr.CreatedAt.Unix(),
		})
	}

	grpcNetworks := make([]*NetworkInfo, 0, len(networks))
	for _, net := range networks {
		grpcNetworks = append(grpcNetworks, &NetworkInfo{
			Name:   net.Name,
			Driver: net.Driver,
			Scope:  net.Scope,
			Subnet: net.Subnet,
		})
	}

	if dc.client == nil {
		return fmt.Errorf("client not connected")
	}

	_, err = dc.client.SyncTopology(grpcContainers, grpcNetworks)
	return err
}

type AgentServiceClient interface {
	Enroll(ctx context.Context, req *EnrollRequest) (*EnrollResponse, error)
	Heartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error)
	SyncTopology(ctx context.Context, req *TopologySnapshot) (*TopologyResponse, error)
}

func NewAgentServiceClient(cc *grpc.ClientConn) AgentServiceClient {
	return &agentServiceClient{cc}
}

type agentServiceClient struct {
	cc *grpc.ClientConn
}

func (c *agentServiceClient) Enroll(ctx context.Context, req *EnrollRequest) (*EnrollResponse, error) {
	out := new(EnrollResponse)
	err := c.cc.Invoke(ctx, "/containerscope.agent.AgentService/Enroll", req, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentServiceClient) Heartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error) {
	out := new(HeartbeatResponse)
	err := c.cc.Invoke(ctx, "/containerscope.agent.AgentService/Heartbeat", req, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *agentServiceClient) SyncTopology(ctx context.Context, req *TopologySnapshot) (*TopologyResponse, error) {
	out := new(TopologyResponse)
	err := c.cc.Invoke(ctx, "/containerscope.agent.AgentService/SyncTopology", req, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}
