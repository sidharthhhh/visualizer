package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/containerscope/backend/internal/metrics"
	"github.com/containerscope/backend/internal/middleware"
	"github.com/containerscope/backend/internal/store"
)

type MetricsHandler struct {
	store         *store.Store
	metricsClient *metrics.Client
	logger        *slog.Logger
}

func NewMetricsHandler(store *store.Store, metricsClient *metrics.Client, logger *slog.Logger) *MetricsHandler {
	return &MetricsHandler{
		store:         store,
		metricsClient: metricsClient,
		logger:        logger,
	}
}

func (h *MetricsHandler) GetContainerMetrics(w http.ResponseWriter, r *http.Request) {
	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	runtimeID := r.URL.Query().Get("runtime_id")
	if runtimeID == "" {
		writeError(w, http.StatusBadRequest, "runtime_id is required")
		return
	}

	metricType := r.URL.Query().Get("metric")
	if metricType == "" {
		metricType = "cpu"
	}

	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	_ = orgID

	query := buildMetricQuery(connID, runtimeID, metricType)

	end := time.Now()
	start := end.Add(-1 * time.Hour)
	step := 15 * time.Second

	startStr := r.URL.Query().Get("start")
	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			start = t
		}
	}

	endStr := r.URL.Query().Get("end")
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			end = t
		}
	}

	results, err := h.metricsClient.QueryRange(r.Context(), query, start, end, step)
	if err != nil {
		h.logger.Error("querying metrics", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to query metrics")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"metric":  metricType,
		"results": results,
		"start":   start,
		"end":     end,
		"step":    step,
	})
}

func (h *MetricsHandler) GetContainerMetricsInstant(w http.ResponseWriter, r *http.Request) {
	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	runtimeID := r.URL.Query().Get("runtime_id")
	if runtimeID == "" {
		writeError(w, http.StatusBadRequest, "runtime_id is required")
		return
	}

	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	_ = orgID

	metrics := []string{"cpu", "mem", "net_rx", "net_tx", "disk_r", "disk_w"}
	results := make(map[string]interface{})

	for _, m := range metrics {
		query := buildMetricQuery(connID, runtimeID, m)
		queryResults, err := h.metricsClient.QueryInstant(r.Context(), query)
		if err != nil {
			h.logger.Error("querying metric", "metric", m, "error", err)
			continue
		}
		if len(queryResults) > 0 {
			results[m] = queryResults[0].Value
		}
	}

	writeJSON(w, http.StatusOK, results)
}

func buildMetricQuery(connID uuid.UUID, runtimeID, metricType string) string {
	return metricType + `{connection_id="` + connID.String() + `",runtime_id="` + runtimeID + `"}`
}
