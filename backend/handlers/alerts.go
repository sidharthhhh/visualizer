package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/containerscope/backend/internal/alerts"
)

type AlertHandler struct {
	engine *alerts.Engine
	logger *slog.Logger
}

func NewAlertHandler(engine *alerts.Engine, logger *slog.Logger) *AlertHandler {
	return &AlertHandler{engine: engine, logger: logger}
}

type CreateRuleRequest struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Condition   alerts.Condition  `json:"condition"`
	Channels    []string          `json:"channels"`
	Labels      map[string]string `json:"labels"`
	Enabled     bool              `json:"enabled"`
}

type CreateChannelRequest struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Type    string               `json:"type"`
	Config  alerts.ChannelConfig `json:"config"`
	Enabled bool                 `json:"enabled"`
}

func (h *AlertHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules := h.engine.GetRules()
	writeJSON(w, http.StatusOK, rules)
}

func (h *AlertHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	rule := &alerts.Rule{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Severity:    alerts.AlertSeverity(req.Severity),
		Condition:   req.Condition,
		Channels:    req.Channels,
		Labels:      req.Labels,
		Enabled:     req.Enabled,
	}

	h.engine.AddRule(rule)
	writeJSON(w, http.StatusCreated, rule)
}

func (h *AlertHandler) ListChannels(w http.ResponseWriter, r *http.Request) {
	channels := h.engine.GetChannels()
	writeJSON(w, http.StatusOK, channels)
}

func (h *AlertHandler) CreateChannel(w http.ResponseWriter, r *http.Request) {
	var req CreateChannelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	channel := &alerts.NotificationChannel{
		ID:      req.ID,
		Name:    req.Name,
		Type:    req.Type,
		Config:  req.Config,
		Enabled: req.Enabled,
	}

	h.engine.AddChannel(channel)
	writeJSON(w, http.StatusCreated, channel)
}

func (h *AlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := h.engine.GetAlerts()
	writeJSON(w, http.StatusOK, alerts)
}

func (h *AlertHandler) ListFiringAlerts(w http.ResponseWriter, r *http.Request) {
	alerts := h.engine.GetFiringAlerts()
	writeJSON(w, http.StatusOK, alerts)
}

func (h *AlertHandler) SilenceAlert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AlertID string `json:"alert_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	h.engine.SilenceAlert(req.AlertID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "silenced"})
}
