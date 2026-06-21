package compliance

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

type Finding struct {
	ID          string
	Title       string
	Description string
	Severity    Severity
	Resource    string
	Category    string
	Remediation string
}

type CheckFunc func(ctx context.Context, container ContainerInfo) []Finding

type ContainerInfo struct {
	ID            string
	Name          string
	Image         string
	User          string
	Privileged    bool
	NetworkMode   string
	PIDMode       string
	ReadOnly      bool
	CapAdd        []string
	CapDrop       []string
	MemoryLimit   int64
	CPUShares     int64
	HealthCheck   bool
	RestartPolicy string
	SecurityOpt   []string
}

type Checker struct {
	logger *slog.Logger
	client *client.Client
	checks []CheckFunc
}

func NewChecker(logger *slog.Logger) (*Checker, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("creating docker client: %w", err)
	}

	c := &Checker{
		logger: logger,
		client: cli,
	}

	c.checks = []CheckFunc{
		c.checkPrivileged,
		c.checkRootUser,
		c.checkHostNetwork,
		c.checkHostPID,
		c.checkReadOnlyRoot,
		c.checkDangerousCapabilities,
		c.checkHealthCheck,
		c.checkMemoryLimit,
		c.checkCPUShares,
		c.checkRestartPolicy,
	}

	return c, nil
}

func (c *Checker) Close() error {
	return c.client.Close()
}

func (c *Checker) CheckContainer(ctx context.Context, containerID string) ([]Finding, error) {
	inspect, err := c.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspecting container: %w", err)
	}

	info := ContainerInfo{
		ID:            inspect.ID,
		Name:          inspect.Name,
		Image:         inspect.Config.Image,
		User:          inspect.Config.User,
		Privileged:    inspect.HostConfig.Privileged,
		NetworkMode:   string(inspect.HostConfig.NetworkMode),
		PIDMode:       string(inspect.HostConfig.PidMode),
		ReadOnly:      inspect.HostConfig.ReadonlyRootfs,
		CapAdd:        inspect.HostConfig.CapAdd,
		CapDrop:       inspect.HostConfig.CapDrop,
		MemoryLimit:   inspect.HostConfig.Memory,
		CPUShares:     inspect.HostConfig.CPUShares,
		RestartPolicy: string(inspect.HostConfig.RestartPolicy.Name),
		SecurityOpt:   inspect.HostConfig.SecurityOpt,
	}

	info.HealthCheck = inspect.Config.Healthcheck != nil

	var findings []Finding
	for _, check := range c.checks {
		findings = append(findings, check(ctx, info)...)
	}

	return findings, nil
}

func (c *Checker) CheckAllContainers(ctx context.Context) (map[string][]Finding, error) {
	containers, err := c.client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing containers: %w", err)
	}

	result := make(map[string][]Finding)
	for _, ctr := range containers {
		findings, err := c.CheckContainer(ctx, ctr.ID)
		if err != nil {
			c.logger.Error("checking container", "id", ctr.ID, "error", err)
			continue
		}
		if len(findings) > 0 {
			result[ctr.ID] = findings
		}
	}

	return result, nil
}

func (c *Checker) checkPrivileged(ctx context.Context, ctr ContainerInfo) []Finding {
	if ctr.Privileged {
		return []Finding{{
			ID:          "CIS-001",
			Title:       "Privileged container detected",
			Description: fmt.Sprintf("Container '%s' is running in privileged mode", ctr.Name),
			Severity:    SeverityCritical,
			Resource:    ctr.ID,
			Category:    "security",
			Remediation: "Remove --privileged flag and use specific capabilities instead",
		}}
	}
	return nil
}

func (c *Checker) checkRootUser(ctx context.Context, ctr ContainerInfo) []Finding {
	if ctr.User == "" || ctr.User == "root" || ctr.User == "0" {
		return []Finding{{
			ID:          "CIS-002",
			Title:       "Container running as root",
			Description: fmt.Sprintf("Container '%s' is running as root user", ctr.Name),
			Severity:    SeverityHigh,
			Resource:    ctr.ID,
			Category:    "security",
			Remediation: "Use USER directive in Dockerfile or --user flag",
		}}
	}
	return nil
}

func (c *Checker) checkHostNetwork(ctx context.Context, ctr ContainerInfo) []Finding {
	if ctr.NetworkMode == "host" {
		return []Finding{{
			ID:          "CIS-003",
			Title:       "Host network mode",
			Description: fmt.Sprintf("Container '%s' is using host network mode", ctr.Name),
			Severity:    SeverityHigh,
			Resource:    ctr.ID,
			Category:    "network",
			Remediation: "Use bridge or custom network mode instead of host",
		}}
	}
	return nil
}

func (c *Checker) checkHostPID(ctx context.Context, ctr ContainerInfo) []Finding {
	if ctr.PIDMode == "host" {
		return []Finding{{
			ID:          "CIS-004",
			Title:       "Host PID namespace",
			Description: fmt.Sprintf("Container '%s' is sharing host PID namespace", ctr.Name),
			Severity:    SeverityHigh,
			Resource:    ctr.ID,
			Category:    "isolation",
			Remediation: "Remove --pid=host flag",
		}}
	}
	return nil
}

func (c *Checker) checkReadOnlyRoot(ctx context.Context, ctr ContainerInfo) []Finding {
	if !ctr.ReadOnly {
		return []Finding{{
			ID:          "CIS-005",
			Title:       "Writable root filesystem",
			Description: fmt.Sprintf("Container '%s' has a writable root filesystem", ctr.Name),
			Severity:    SeverityMedium,
			Resource:    ctr.ID,
			Category:    "security",
			Remediation: "Use --read-only flag and mount tmpfs for writable directories",
		}}
	}
	return nil
}

func (c *Checker) checkDangerousCapabilities(ctx context.Context, ctr ContainerInfo) []Finding {
	dangerousCaps := []string{"SYS_ADMIN", "NET_ADMIN", "ALL"}
	var findings []Finding

	for _, cap := range ctr.CapAdd {
		for _, dangerous := range dangerousCaps {
			if strings.ToUpper(cap) == dangerous {
				findings = append(findings, Finding{
					ID:          "CIS-006",
					Title:       "Dangerous capability added",
					Description: fmt.Sprintf("Container '%s' has dangerous capability: %s", ctr.Name, cap),
					Severity:    SeverityHigh,
					Resource:    ctr.ID,
					Category:    "security",
					Remediation: fmt.Sprintf("Remove capability %s if not required", cap),
				})
			}
		}
	}

	return findings
}

func (c *Checker) checkHealthCheck(ctx context.Context, ctr ContainerInfo) []Finding {
	if !ctr.HealthCheck {
		return []Finding{{
			ID:          "CIS-007",
			Title:       "No health check defined",
			Description: fmt.Sprintf("Container '%s' has no health check", ctr.Name),
			Severity:    SeverityLow,
			Resource:    ctr.ID,
			Category:    "reliability",
			Remediation: "Add HEALTHCHECK instruction to Dockerfile",
		}}
	}
	return nil
}

func (c *Checker) checkMemoryLimit(ctx context.Context, ctr ContainerInfo) []Finding {
	if ctr.MemoryLimit == 0 {
		return []Finding{{
			ID:          "CIS-008",
			Title:       "No memory limit",
			Description: fmt.Sprintf("Container '%s' has no memory limit", ctr.Name),
			Severity:    SeverityMedium,
			Resource:    ctr.ID,
			Category:    "resources",
			Remediation: "Set memory limit with --memory flag",
		}}
	}
	return nil
}

func (c *Checker) checkCPUShares(ctx context.Context, ctr ContainerInfo) []Finding {
	if ctr.CPUShares == 0 {
		return []Finding{{
			ID:          "CIS-009",
			Title:       "No CPU limit",
			Description: fmt.Sprintf("Container '%s' has no CPU shares set", ctr.Name),
			Severity:    SeverityLow,
			Resource:    ctr.ID,
			Category:    "resources",
			Remediation: "Set CPU shares with --cpu-shares flag",
		}}
	}
	return nil
}

func (c *Checker) checkRestartPolicy(ctx context.Context, ctr ContainerInfo) []Finding {
	if ctr.RestartPolicy == "always" {
		return []Finding{{
			ID:          "CIS-010",
			Title:       "Always restart policy",
			Description: fmt.Sprintf("Container '%s' has 'always' restart policy", ctr.Name),
			Severity:    SeverityInfo,
			Resource:    ctr.ID,
			Category:    "reliability",
			Remediation: "Consider using 'unless-stopped' or 'on-failure' instead",
		}}
	}
	return nil
}
