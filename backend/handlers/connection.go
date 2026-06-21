package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/containerscope/backend/internal/middleware"
	"github.com/containerscope/backend/internal/store"
)

type ConnectionHandler struct {
	store  *store.Store
	logger *slog.Logger
}

func NewConnectionHandler(store *store.Store, logger *slog.Logger) *ConnectionHandler {
	return &ConnectionHandler{store: store, logger: logger}
}

type CreateConnectionRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

func (h *ConnectionHandler) Create(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	var req CreateConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type == "" || req.Name == "" {
		writeError(w, http.StatusBadRequest, "type and name are required")
		return
	}

	validTypes := map[string]bool{"docker": true, "k8s": true}
	if !validTypes[req.Type] {
		writeError(w, http.StatusBadRequest, "invalid type: must be docker or k8s")
		return
	}

	conn, err := h.store.CreateConnection(r.Context(), orgID, req.Type, req.Name)
	if err != nil {
		h.logger.Error("creating connection", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	claims, _ := middleware.GetClaims(r.Context())
	if claims != nil {
		_ = h.store.CreateAuditLog(r.Context(), orgID, claims.UserID, "connection.create", conn.ID.String(), nil)
	}

	writeJSON(w, http.StatusCreated, conn)
}

func (h *ConnectionHandler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrgID(r.Context())
	if !ok {
		writeError(w, http.StatusBadRequest, "missing org id")
		return
	}

	conns, err := h.store.ListConnections(r.Context(), orgID)
	if err != nil {
		h.logger.Error("listing connections", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, conns)
}

func (h *ConnectionHandler) Get(w http.ResponseWriter, r *http.Request) {
	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	conn, err := h.store.GetConnectionByID(r.Context(), connID)
	if err != nil {
		writeError(w, http.StatusNotFound, "connection not found")
		return
	}

	writeJSON(w, http.StatusOK, conn)
}

func (h *ConnectionHandler) ListHosts(w http.ResponseWriter, r *http.Request) {
	connIDStr := chi.URLParam(r, "connectionID")
	connID, err := uuid.Parse(connIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	hosts, err := h.store.ListHosts(r.Context(), connID)
	if err != nil {
		h.logger.Error("listing hosts", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, hosts)
}
