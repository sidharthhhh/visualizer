package containerscope

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type Option func(*Client)

func WithAPIKey(key string) Option {
	return func(c *Client) {
		c.apiKey = key
	}
}

func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.httpClient = client
	}
}

func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type Container struct {
	ID           string            `json:"id"`
	ConnectionID string            `json:"connection_id"`
	RuntimeID    string            `json:"runtime_id"`
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	State        string            `json:"state"`
	Labels       map[string]string `json:"labels,omitempty"`
	Ports        []Port            `json:"ports,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
}

type Port struct {
	HostPort      string `json:"host_port"`
	ContainerPort string `json:"container_port"`
	Protocol      string `json:"protocol"`
}

type Network struct {
	ID           string    `json:"id"`
	ConnectionID string    `json:"connection_id"`
	Name         string    `json:"name"`
	Driver       string    `json:"driver"`
	Scope        string    `json:"scope"`
	Subnet       string    `json:"subnet"`
	CreatedAt    time.Time `json:"created_at"`
}

type Edge struct {
	ID             string    `json:"id"`
	ConnectionID   string    `json:"connection_id"`
	SrcContainerID string    `json:"src_container_id"`
	DstContainerID string    `json:"dst_container_id,omitempty"`
	DstIP          string    `json:"dst_ip"`
	DstPort        int       `json:"dst_port"`
	Protocol       string    `json:"protocol"`
	FirstSeen      time.Time `json:"first_seen"`
	LastSeen       time.Time `json:"last_seen"`
}

type Topology struct {
	Containers []Container `json:"containers"`
	Networks   []Network   `json:"networks"`
	Edges      []Edge      `json:"edges"`
}

func (c *Client) GetTopology(orgID, connectionID string) (*Topology, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/connections/%s/topology", orgID, connectionID)
	var topology Topology
	if err := c.get(path, &topology); err != nil {
		return nil, err
	}
	return &topology, nil
}

func (c *Client) ListContainers(orgID, connectionID string) ([]Container, error) {
	topology, err := c.GetTopology(orgID, connectionID)
	if err != nil {
		return nil, err
	}
	return topology.Containers, nil
}

func (c *Client) ListConnections(orgID string) ([]Connection, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/connections", orgID)
	var connections []Connection
	if err := c.get(path, &connections); err != nil {
		return nil, err
	}
	return connections, nil
}

type Connection struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type MetricSample struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

func (c *Client) GetMetrics(orgID, connectionID, runtimeID, metric string) ([]MetricSample, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/connections/%s/metrics?runtime_id=%s&metric=%s",
		orgID, connectionID, runtimeID, metric)
	var response struct {
		Metric  string         `json:"metric"`
		Results []MetricSample `json:"results"`
	}
	if err := c.get(path, &response); err != nil {
		return nil, err
	}
	return response.Results, nil
}

type Alert struct {
	ID         string            `json:"id"`
	RuleID     string            `json:"rule_id"`
	RuleName   string            `json:"rule_name"`
	Severity   string            `json:"severity"`
	Status     string            `json:"status"`
	Labels     map[string]string `json:"labels,omitempty"`
	StartsAt   time.Time         `json:"starts_at"`
	EndsAt     *time.Time        `json:"ends_at,omitempty"`
	FiredCount int               `json:"fired_count"`
}

func (c *Client) ListAlerts(orgID, connectionID string) ([]Alert, error) {
	path := fmt.Sprintf("/api/v1/orgs/%s/connections/%s/alerts", orgID, connectionID)
	var alerts []Alert
	if err := c.get(path, &alerts); err != nil {
		return nil, err
	}
	return alerts, nil
}

func (c *Client) get(path string, result interface{}) error {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return err
	}

	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) post(path string, body interface{}, result interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}
	return nil
}
