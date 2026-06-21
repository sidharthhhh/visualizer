package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	pool   *pgxpool.Pool
	vmURL  string
	logger *slog.Logger
}

func NewHealthHandler(pool *pgxpool.Pool, vmURL string, logger *slog.Logger) *HealthHandler {
	return &HealthHandler{pool: pool, vmURL: vmURL, logger: logger}
}

type ServiceHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Latency int64  `json:"latency_ms"`
	Message string `json:"message,omitempty"`
}

type HealthResponse struct {
	Status   string          `json:"status"`
	Services []ServiceHealth `json:"services"`
}

func (h *HealthHandler) GetServicesHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	services := make([]ServiceHealth, 0)

	// Check Postgres
	start := time.Now()
	if err := h.pool.Ping(ctx); err != nil {
		services = append(services, ServiceHealth{
			Name:    "PostgreSQL",
			Status:  "critical",
			Latency: time.Since(start).Milliseconds(),
			Message: err.Error(),
		})
	} else {
		services = append(services, ServiceHealth{
			Name:    "PostgreSQL",
			Status:  "healthy",
			Latency: time.Since(start).Milliseconds(),
		})
	}

	// Check VictoriaMetrics
	start = time.Now()
	vmHealthy := false
	if h.vmURL != "" {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get(h.vmURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == 200 {
				vmHealthy = true
			}
		}
	}
	if vmHealthy {
		services = append(services, ServiceHealth{
			Name:    "VictoriaMetrics",
			Status:  "healthy",
			Latency: time.Since(start).Milliseconds(),
		})
	} else {
		services = append(services, ServiceHealth{
			Name:    "VictoriaMetrics",
			Status:  "warning",
			Latency: time.Since(start).Milliseconds(),
			Message: "not reachable",
		})
	}

	// Check ClickHouse
	start = time.Now()
	chHealthy := false
	if h.vmURL != "" {
		parsedURL, _ := url.Parse(h.vmURL)
		if parsedURL != nil {
			chURL := "http://" + parsedURL.Hostname() + ":8123/ping"
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(chURL)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == 200 {
					chHealthy = true
				}
			}
		}
	}
	if chHealthy {
		services = append(services, ServiceHealth{
			Name:    "ClickHouse",
			Status:  "healthy",
			Latency: time.Since(start).Milliseconds(),
		})
	} else {
		services = append(services, ServiceHealth{
			Name:    "ClickHouse",
			Status:  "warning",
			Latency: time.Since(start).Milliseconds(),
			Message: "not reachable",
		})
	}

	// Overall status
	overallStatus := "healthy"
	for _, s := range services {
		if s.Status == "critical" {
			overallStatus = "critical"
			break
		}
		if s.Status == "warning" {
			overallStatus = "degraded"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(HealthResponse{
		Status:   overallStatus,
		Services: services,
	})
}
