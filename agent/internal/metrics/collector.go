package metrics

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type MetricType string

const (
	MetricCPU   MetricType = "cpu"
	MetricMem   MetricType = "mem"
	MetricNetRx MetricType = "net_rx"
	MetricNetTx MetricType = "net_tx"
	MetricDiskR MetricType = "disk_r"
	MetricDiskW MetricType = "disk_w"
)

type Sample struct {
	ContainerID string
	RuntimeID   string
	Metric      MetricType
	Value       float64
	Timestamp   time.Time
}

type Collector struct {
	client   *client.Client
	logger   *slog.Logger
	samples  chan Sample
	interval time.Duration
}

func NewCollector(logger *slog.Logger, interval time.Duration) (*Collector, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}

	return &Collector{
		client:   cli,
		logger:   logger,
		samples:  make(chan Sample, 1000),
		interval: interval,
	}, nil
}

func (c *Collector) Close() error {
	close(c.samples)
	return c.client.Close()
}

func (c *Collector) Samples() <-chan Sample {
	return c.samples
}

func (c *Collector) Start(ctx context.Context, containerIDs []string) {
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				for _, id := range containerIDs {
					if err := c.collectContainer(ctx, id); err != nil {
						c.logger.Error("collecting metrics", "container", id, "error", err)
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *Collector) CollectOnce(ctx context.Context, containerID string) ([]Sample, error) {
	var samples []Sample
	now := time.Now()

	statsResponse, err := c.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("getting stats: %w", err)
	}
	defer statsResponse.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(statsResponse.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("decoding stats: %w", err)
	}

	var cpuPercent float64
	if stats.CPUStats.CPUUsage.TotalUsage > 0 && stats.PreCPUStats.CPUUsage.TotalUsage > 0 {
		cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
		sysDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
		if sysDelta > 0 {
			cpuPercent = (cpuDelta / sysDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
		}
	}

	memUsage := float64(stats.MemoryStats.Usage)

	var netRx, netTx float64
	for _, v := range stats.Networks {
		netRx += float64(v.RxBytes)
		netTx += float64(v.TxBytes)
	}

	var diskRead, diskWrite float64
	for _, v := range stats.BlkioStats.IoServiceBytesRecursive {
		switch strings.ToLower(v.Op) {
		case "read":
			diskRead += float64(v.Value)
		case "write":
			diskWrite += float64(v.Value)
		}
	}

	samples = append(samples,
		Sample{RuntimeID: containerID, Metric: MetricCPU, Value: cpuPercent, Timestamp: now},
		Sample{RuntimeID: containerID, Metric: MetricMem, Value: memUsage, Timestamp: now},
		Sample{RuntimeID: containerID, Metric: MetricNetRx, Value: netRx, Timestamp: now},
		Sample{RuntimeID: containerID, Metric: MetricNetTx, Value: netTx, Timestamp: now},
		Sample{RuntimeID: containerID, Metric: MetricDiskR, Value: diskRead, Timestamp: now},
		Sample{RuntimeID: containerID, Metric: MetricDiskW, Value: diskWrite, Timestamp: now},
	)

	return samples, nil
}

func (c *Collector) collectContainer(ctx context.Context, containerID string) error {
	samples, err := c.CollectOnce(ctx, containerID)
	if err != nil {
		return err
	}

	for _, s := range samples {
		select {
		case c.samples <- s:
		default:
			c.logger.Warn("sample channel full, dropping sample")
		}
	}

	return nil
}

func CollectHostMetrics() (map[string]float64, error) {
	metrics := make(map[string]float64)

	file, err := os.Open("/proc/stat")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				user, _ := strconv.ParseFloat(fields[1], 64)
				system, _ := strconv.ParseFloat(fields[3], 64)
				idle, _ := strconv.ParseFloat(fields[4], 64)
				total := user + system + idle
				if total > 0 {
					metrics["host_cpu_percent"] = ((user + system) / total) * 100
				}
			}
			break
		}
	}

	memInfo, err := os.ReadFile("/proc/meminfo")
	if err == nil {
		lines := strings.Split(string(memInfo), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					val, _ := strconv.ParseFloat(fields[1], 64)
					metrics["host_mem_total_kb"] = val
				}
			} else if strings.HasPrefix(line, "MemAvailable:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					val, _ := strconv.ParseFloat(fields[1], 64)
					metrics["host_mem_available_kb"] = val
				}
			}
		}
	}

	return metrics, nil
}
