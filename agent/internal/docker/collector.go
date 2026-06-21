package docker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type Container struct {
	RuntimeID string
	Name      string
	Image     string
	State     string
	Labels    map[string]string
	Ports     []Port
	CreatedAt time.Time
}

type Port struct {
	HostPort      string
	ContainerPort string
	Protocol      string
}

type Network struct {
	Name   string
	Driver string
	Scope  string
	Subnet string
}

type EventType string

const (
	EventCreate  EventType = "create"
	EventStart   EventType = "start"
	EventStop    EventType = "stop"
	EventDie     EventType = "die"
	EventDestroy EventType = "destroy"
)

type TopologyEvent struct {
	Type      EventType
	Container *Container
	Timestamp time.Time
}

type Collector struct {
	client   *client.Client
	logger   *slog.Logger
	eventsCh chan TopologyEvent
}

func NewCollector(logger *slog.Logger) (*Collector, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}

	return &Collector{
		client:   cli,
		logger:   logger,
		eventsCh: make(chan TopologyEvent, 100),
	}, nil
}

func (c *Collector) Close() error {
	close(c.eventsCh)
	return c.client.Close()
}

func (c *Collector) ListContainers(ctx context.Context) ([]Container, error) {
	containers, err := c.client.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	result := make([]Container, 0, len(containers))
	for _, ctr := range containers {
		name := ""
		if len(ctr.Names) > 0 {
			name = ctr.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}

		ports := make([]Port, 0, len(ctr.Ports))
		for _, p := range ctr.Ports {
			ports = append(ports, Port{
				HostPort:      fmt.Sprintf("%d", p.PublicPort),
				ContainerPort: fmt.Sprintf("%d", p.PrivatePort),
				Protocol:      p.Type,
			})
		}

		result = append(result, Container{
			RuntimeID: ctr.ID,
			Name:      name,
			Image:     ctr.Image,
			State:     ctr.State,
			Labels:    ctr.Labels,
			Ports:     ports,
			CreatedAt: time.Unix(ctr.Created, 0),
		})
	}

	return result, nil
}

func (c *Collector) ListNetworks(ctx context.Context) ([]Network, error) {
	networks, err := c.client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}

	result := make([]Network, 0, len(networks))
	for _, net := range networks {
		subnet := ""
		if len(net.IPAM.Config) > 0 {
			subnet = net.IPAM.Config[0].Subnet
		}

		result = append(result, Network{
			Name:   net.Name,
			Driver: net.Driver,
			Scope:  net.Scope,
			Subnet: subnet,
		})
	}

	return result, nil
}

func (c *Collector) Subscribe(ctx context.Context) {
	go func() {
		filter := filters.NewArgs()
		filter.Add("type", "container")
		filter.Add("event", "create")
		filter.Add("event", "start")
		filter.Add("event", "stop")
		filter.Add("event", "die")
		filter.Add("event", "destroy")

		eventsCh, errCh := c.client.Events(ctx, events.ListOptions{
			Filters: filter,
		})

		for {
			select {
			case event := <-eventsCh:
				evt := TopologyEvent{
					Type:      EventType(event.Action),
					Timestamp: time.Unix(event.Time, 0),
				}

				ctr, err := c.getContainerByID(ctx, event.Actor.ID)
				if err != nil {
					c.logger.Error("getting container", "id", event.Actor.ID, "error", err)
					continue
				}
				evt.Container = ctr

				select {
				case c.eventsCh <- evt:
				default:
					c.logger.Warn("event channel full, dropping event")
				}
			case err := <-errCh:
				if err != nil {
					c.logger.Error("docker events error", "error", err)
				}
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *Collector) Events() <-chan TopologyEvent {
	return c.eventsCh
}

func (c *Collector) getContainerByID(ctx context.Context, id string) (*Container, error) {
	inspect, err := c.client.ContainerInspect(ctx, id)
	if err != nil {
		return nil, err
	}

	name := inspect.Name
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	ports := make([]Port, 0)
	for portProto, bindings := range inspect.NetworkSettings.Ports {
		for _, binding := range bindings {
			ports = append(ports, Port{
				HostPort:      binding.HostPort,
				ContainerPort: portProto.Port(),
				Protocol:      string(portProto.Proto()),
			})
		}
	}

	state := "unknown"
	if inspect.State != nil {
		state = inspect.State.Status
	}

	return &Container{
		RuntimeID: inspect.ID,
		Name:      name,
		Image:     inspect.Config.Image,
		State:     state,
		Labels:    inspect.Config.Labels,
		Ports:     ports,
		CreatedAt: func() time.Time {
			t, _ := time.Parse(time.RFC3339Nano, inspect.Created)
			return t
		}(),
	}, nil
}
