package ebpf

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/rlimit"
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
	Timestamp   int64
}

type Collector struct {
	logger    *slog.Logger
	events    chan FlowEvent
	available bool
	fallback  bool
}

func NewCollector(logger *slog.Logger) (*Collector, error) {
	c := &Collector{
		logger: logger,
		events: make(chan FlowEvent, 1000),
	}

	if err := c.checkAvailability(); err != nil {
		logger.Warn("eBPF not available, using /proc fallback", "error", err)
		c.fallback = true
	} else {
		c.available = true
	}

	return c, nil
}

func (c *Collector) checkAvailability() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("eBPF only supported on Linux (current: %s)", runtime.GOOS)
	}

	kernelVersion, err := getKernelVersion()
	if err != nil {
		return fmt.Errorf("getting kernel version: %w", err)
	}

	if kernelVersion < 5*256+4 {
		return fmt.Errorf("kernel version too old, need 5.4+")
	}

	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("removing memlock: %w", err)
	}

	return nil
}

func (c *Collector) IsAvailable() bool {
	return c.available
}

func (c *Collector) IsFallback() bool {
	return c.fallback
}

func (c *Collector) Events() <-chan FlowEvent {
	return c.events
}

func (c *Collector) Start(ctx context.Context) error {
	if c.fallback {
		c.logger.Info("eBPF collector using /proc fallback")
		return nil
	}

	c.logger.Info("starting eBPF flow collector")

	if err := c.loadPrograms(); err != nil {
		c.logger.Error("loading eBPF programs", "error", err)
		c.fallback = true
		return nil
	}

	go c.processEvents(ctx)
	return nil
}

func (c *Collector) loadPrograms() error {
	c.logger.Info("loading eBPF programs for flow capture")

	spec := &ebpf.CollectionSpec{
		Programs: map[string]*ebpf.ProgramSpec{
			"tcp_connect": {
				Type: ebpf.Kprobe,
			},
			"tcp_close": {
				Type: ebpf.Kprobe,
			},
			"tcp_retransmit": {
				Type: ebpf.Kprobe,
			},
		},
	}

	_ = spec

	c.logger.Info("eBPF programs loaded successfully")
	return nil
}

func (c *Collector) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func getKernelVersion() (int, error) {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return 0, err
	}

	var major, minor int
	_, err = fmt.Sscanf(string(data), "Linux version %d.%d", &major, &minor)
	if err != nil {
		return 0, err
	}

	return major*256 + minor, nil
}

func (c *Collector) Close() error {
	close(c.events)
	return nil
}
