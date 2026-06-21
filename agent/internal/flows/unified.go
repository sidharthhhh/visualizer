package flows

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	ebpfcollector "github.com/containerscope/agent/internal/ebpf"
)

type FlowEvent struct {
	SrcIP       string
	DstIP       string
	SrcPort     uint16
	DstPort     uint16
	Protocol    string
	Bytes       uint64
	Packets     uint64
	LatencyMs   float64
	ContainerID string
	Timestamp   time.Time
}

type CollectorMode string

const (
	ModeEBPF CollectorMode = "ebpf"
	ModeProc CollectorMode = "proc"
)

type UnifiedCollector struct {
	logger        *slog.Logger
	ebpfCollector *ebpfcollector.Collector
	procCollector *ProcCollector
	mode          CollectorMode
	events        chan FlowEvent
}

func NewUnifiedCollector(logger *slog.Logger) (*UnifiedCollector, error) {
	ebpfCol, err := ebpfcollector.NewCollector(logger)
	if err != nil {
		return nil, fmt.Errorf("creating ebpf collector: %w", err)
	}

	procCol := NewProcCollector(logger)

	mode := ModeProc
	if ebpfCol.IsAvailable() {
		mode = ModeEBPF
	}

	logger.Info("flow collector initialized", "mode", mode, "ebpf_available", ebpfCol.IsAvailable(), "fallback", ebpfCol.IsFallback())

	return &UnifiedCollector{
		logger:        logger,
		ebpfCollector: ebpfCol,
		procCollector: procCol,
		mode:          mode,
		events:        make(chan FlowEvent, 1000),
	}, nil
}

func (c *UnifiedCollector) Mode() CollectorMode {
	return c.mode
}

func (c *UnifiedCollector) Events() <-chan FlowEvent {
	return c.events
}

func (c *UnifiedCollector) Start(ctx context.Context) error {
	if c.mode == ModeEBPF {
		return c.ebpfCollector.Start(ctx)
	}

	c.procCollector.Start(ctx)

	go func() {
		for event := range c.procCollector.Events() {
			flowEvent := FlowEvent{
				SrcIP:       event.SrcIP,
				DstIP:       event.DstIP,
				SrcPort:     event.SrcPort,
				DstPort:     event.DstPort,
				Protocol:    event.Protocol,
				Bytes:       event.Bytes,
				Packets:     event.Packets,
				LatencyMs:   event.LatencyMs,
				ContainerID: event.ContainerID,
				Timestamp:   event.Timestamp,
			}
			select {
			case c.events <- flowEvent:
			default:
				c.logger.Warn("flow event channel full, dropping event")
			}
		}
	}()

	return nil
}

func (c *UnifiedCollector) Close() error {
	close(c.events)
	if c.mode == ModeEBPF {
		return c.ebpfCollector.Close()
	}
	return c.procCollector.Close()
}

type ProcCollector struct {
	logger *slog.Logger
	events chan FlowEvent
}

func NewProcCollector(logger *slog.Logger) *ProcCollector {
	return &ProcCollector{
		logger: logger,
		events: make(chan FlowEvent, 1000),
	}
}

func (c *ProcCollector) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.collect(); err != nil {
					c.logger.Error("collecting /proc flows", "error", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *ProcCollector) Events() <-chan FlowEvent {
	return c.events
}

func (c *ProcCollector) Close() error {
	close(c.events)
	return nil
}

func (c *ProcCollector) collect() error {
	conns, err := ParseProcNet(ProtocolTCP)
	if err != nil {
		return fmt.Errorf("parsing /proc/net/tcp: %w", err)
	}

	conns6, err := ParseProcNet(ProtocolTCP6)
	if err != nil {
		return fmt.Errorf("parsing /proc/net/tcp6: %w", err)
	}

	conns = append(conns, conns6...)

	pidToContainer, err := GetContainerPIDs()
	if err != nil {
		return fmt.Errorf("getting container PIDs: %w", err)
	}

	conns = MapSocketsToContainers(conns, pidToContainer)

	for _, conn := range conns {
		if conn.State != StateEstablished {
			continue
		}

		if conn.ContainerID == "" {
			continue
		}

		event := FlowEvent{
			SrcIP:       conn.LocalIP,
			DstIP:       conn.RemoteIP,
			SrcPort:     conn.LocalPort,
			DstPort:     conn.RemotePort,
			Protocol:    string(conn.Protocol),
			ContainerID: conn.ContainerID,
			Timestamp:   time.Now(),
		}

		select {
		case c.events <- event:
		default:
			c.logger.Warn("flow event channel full, dropping event")
		}
	}

	return nil
}

func GetCollectorInfo() map[string]interface{} {
	return map[string]interface{}{
		"os":       "linux",
		"arch":     "amd64",
		"ebpf_req": "kernel 5.4+, CAP_BPF or CAP_SYS_ADMIN",
	}
}
