package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/containerscope/backend/internal/flows"
	"github.com/containerscope/backend/internal/middleware"
	"github.com/containerscope/backend/internal/store"
)

type FlowHandler struct {
	store      *store.Store
	flowClient *flows.Client
	logger     *slog.Logger
}

func NewFlowHandler(store *store.Store, flowClient *flows.Client, logger *slog.Logger) *FlowHandler {
	return &FlowHandler{
		store:      store,
		flowClient: flowClient,
		logger:     logger,
	}
}

func (h *FlowHandler) ListFlows(w http.ResponseWriter, r *http.Request) {
	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	_ = orgID

	end := time.Now()
	start := end.Add(-1 * time.Hour)

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

	limit := 1000
	limitStr := r.URL.Query().Get("limit")
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	flowEvents, err := h.flowClient.QueryFlows(r.Context(), connID.String(), start, end, limit)
	if err != nil {
		h.logger.Warn("querying flows failed, returning empty", "error", err)
		flowEvents = []flows.FlowEvent{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"flows": flowEvents,
		"start": start,
		"end":   end,
		"limit": limit,
	})
}

func (h *FlowHandler) GetBandwidth(w http.ResponseWriter, r *http.Request) {
	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	_ = orgID

	duration := 5 * time.Minute
	durationStr := r.URL.Query().Get("duration")
	if durationStr != "" {
		if d, err := time.ParseDuration(durationStr); err == nil {
			duration = d
		}
	}

	bandwidth, err := h.flowClient.GetEdgeBandwidth(r.Context(), connID.String(), duration)
	if err != nil {
		h.logger.Warn("getting bandwidth failed, returning empty", "error", err)
		bandwidth = map[string]uint64{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"bandwidth": bandwidth,
		"duration":  duration,
	})
}
