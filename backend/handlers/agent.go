package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentHandler struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewAgentHandler(pool *pgxpool.Pool, logger *slog.Logger) *AgentHandler {
	return &AgentHandler{pool: pool, logger: logger}
}

type EnrollRequest struct {
	EnrollmentToken string `json:"enrollment_token"`
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Kernel          string `json:"kernel"`
	CPUCores        int32  `json:"cpu_cores"`
	MemTotal        int64  `json:"mem_total"`
	AgentVersion    string `json:"agent_version"`
}

type EnrollResponse struct {
	ConnectionID string `json:"connection_id"`
	Status       string `json:"status"`
}

type HeartbeatRequest struct {
	ConnectionID string `json:"connection_id"`
}

type HeartbeatResponse struct {
	Status string `json:"status"`
}

type ContainerInfo struct {
	RuntimeID string            `json:"runtime_id"`
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	State     string            `json:"state"`
	Labels    map[string]string `json:"labels"`
}

type NetworkInfo struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
	Subnet string `json:"subnet"`
}

type TopologyRequest struct {
	ConnectionID string          `json:"connection_id"`
	Containers   []ContainerInfo `json:"containers"`
	Networks     []NetworkInfo   `json:"networks"`
}

type AgentTopologyResponse struct {
	Status string `json:"status"`
}

func (h *AgentHandler) Enroll(w http.ResponseWriter, r *http.Request) {
	var req EnrollRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.EnrollmentToken == "" {
		writeError(w, http.StatusBadRequest, "enrollment token is required")
		return
	}

	var connID uuid.UUID
	var orgID uuid.UUID
	err := h.pool.QueryRow(r.Context(),
		`SELECT id, org_id FROM connections WHERE agent_token = $1`,
		req.EnrollmentToken,
	).Scan(&connID, &orgID)
	if err != nil {
		writeError(w, http.StatusNotFound, "invalid enrollment token")
		return
	}

	_, err = h.pool.Exec(r.Context(),
		`UPDATE connections SET status = 'connected', last_seen_at = NOW(), updated_at = NOW() WHERE id = $1`,
		connID,
	)
	if err != nil {
		h.logger.Error("updating connection status", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	if req.Hostname != "" {
		_, err = h.pool.Exec(r.Context(),
			`INSERT INTO hosts (connection_id, hostname, os, kernel, cpu_cores, mem_total, agent_version)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (connection_id) DO UPDATE SET
			   hostname = EXCLUDED.hostname,
			   os = EXCLUDED.os,
			   kernel = EXCLUDED.kernel,
			   cpu_cores = EXCLUDED.cpu_cores,
			   mem_total = EXCLUDED.mem_total,
			   agent_version = EXCLUDED.agent_version`,
			connID, req.Hostname, req.OS, req.Kernel, req.CPUCores, req.MemTotal, req.AgentVersion,
		)
		if err != nil {
			h.logger.Error("creating host", "error", err)
		}
	}

	h.logger.Info("agent enrolled", "connection_id", connID, "org_id", orgID)

	writeJSON(w, http.StatusOK, EnrollResponse{
		ConnectionID: connID.String(),
		Status:       "connected",
	})
}

func (h *AgentHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ConnectionID == "" {
		writeError(w, http.StatusBadRequest, "connection id is required")
		return
	}

	connID, err := uuid.Parse(req.ConnectionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	_, err = h.pool.Exec(r.Context(),
		`UPDATE connections SET last_seen_at = NOW() WHERE id = $1`,
		connID,
	)
	if err != nil {
		h.logger.Error("updating last seen", "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, HeartbeatResponse{Status: "ok"})
}

func (h *AgentHandler) SyncTopology(w http.ResponseWriter, r *http.Request) {
	var req TopologyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ConnectionID == "" {
		writeError(w, http.StatusBadRequest, "connection id is required")
		return
	}

	connID, err := uuid.Parse(req.ConnectionID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid connection id")
		return
	}

	for _, ctr := range req.Containers {
		_, err = h.pool.Exec(r.Context(),
			`INSERT INTO containers (connection_id, runtime_id, name, image, state, labels)
			 VALUES ($1, $2, $3, $4, $5, $6)
			 ON CONFLICT (connection_id, runtime_id) DO UPDATE SET
			   name = EXCLUDED.name,
			   image = EXCLUDED.image,
			   state = EXCLUDED.state,
			   labels = EXCLUDED.labels,
			   updated_at = NOW(),
			   removed_at = NULL`,
			connID, ctr.RuntimeID, ctr.Name, ctr.Image, ctr.State, ctr.Labels,
		)
		if err != nil {
			h.logger.Error("upserting container", "error", err)
		}
	}

	for _, net := range req.Networks {
		_, err = h.pool.Exec(r.Context(),
			`INSERT INTO networks (connection_id, name, driver, subnet)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (connection_id, name) DO UPDATE SET
			   driver = EXCLUDED.driver,
			   subnet = EXCLUDED.subnet`,
			connID, net.Name, net.Driver, net.Subnet,
		)
		if err != nil {
			h.logger.Error("upserting network", "error", err)
		}
	}

	h.logger.Info("topology synced",
		"connection_id", connID,
		"containers", len(req.Containers),
		"networks", len(req.Networks),
	)

	writeJSON(w, http.StatusOK, AgentTopologyResponse{Status: "ok"})
}
