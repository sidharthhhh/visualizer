package handlers

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/containerscope/backend/internal/middleware"
	"github.com/containerscope/backend/internal/store"
)

type TopologyHandler struct {
	store  *store.Store
	logger *slog.Logger
}

func NewTopologyHandler(store *store.Store, logger *slog.Logger) *TopologyHandler {
	return &TopologyHandler{store: store, logger: logger}
}

type TopologyResponse struct {
	Containers []store.Container `json:"containers"`
	Networks   []store.Network   `json:"networks"`
	Edges      []store.Edge      `json:"edges"`
}

func (h *TopologyHandler) GetTopology(w http.ResponseWriter, r *http.Request) {
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

	conn, err := h.store.GetConnectionByID(r.Context(), connID)
	if err != nil {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	if conn.OrgID != orgID {
		writeError(w, http.StatusForbidden, "connection does not belong to this org")
		return
	}

	containers, err := h.store.ListContainers(r.Context(), connID)
	if err != nil {
		h.logger.Error("listing containers", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	networks, err := h.store.ListNetworks(r.Context(), connID)
	if err != nil {
		h.logger.Error("listing networks", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	edges, err := h.store.ListEdges(r.Context(), connID)
	if err != nil {
		h.logger.Error("listing edges", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, TopologyResponse{
		Containers: containers,
		Networks:   networks,
		Edges:      edges,
	})
}
