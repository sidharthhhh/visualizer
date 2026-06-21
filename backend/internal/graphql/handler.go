package graphql

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/containerscope/backend/internal/store"
)

type Handler struct {
	store  *store.Store
	logger *slog.Logger
}

func NewHandler(store *store.Store, logger *slog.Logger) *Handler {
	return &Handler{store: store, logger: logger}
}

type Request struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type Response struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []Error     `json:"errors,omitempty"`
}

type Error struct {
	Message string `json:"message"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Errors: []Error{{Message: "invalid request body"}},
		})
		return
	}

	h.logger.Debug("graphql query", "query", req.Query)

	result := h.executeQuery(req.Query, req.Variables)

	writeJSON(w, http.StatusOK, Response{
		Data: result,
	})
}

func (h *Handler) executeQuery(query string, variables map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	if contains(query, "containers") {
		result["containers"] = []interface{}{}
	}
	if contains(query, "topology") {
		result["topology"] = map[string]interface{}{
			"containers": []interface{}{},
			"networks":   []interface{}{},
			"edges":      []interface{}{},
		}
	}
	if contains(query, "alerts") {
		result["alerts"] = []interface{}{}
	}

	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0:1] == substr[0:1] && contains(s[1:], substr[1:])))
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
