package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type AlertStatus string

const (
	StatusFiring   AlertStatus = "firing"
	StatusResolved AlertStatus = "resolved"
	StatusSilenced AlertStatus = "silenced"
)

type AlertSeverity string

const (
	AlertCritical AlertSeverity = "critical"
	AlertHigh     AlertSeverity = "high"
	AlertMedium   AlertSeverity = "medium"
	AlertLow      AlertSeverity = "low"
)

type Rule struct {
	ID          string
	Name        string
	Description string
	Severity    AlertSeverity
	Condition   Condition
	Channels    []string
	Labels      map[string]string
	Enabled     bool
}

type Condition struct {
	Type      string
	Metric    string
	Threshold float64
	Operator  string
	Duration  time.Duration
}

type Alert struct {
	ID          string
	RuleID      string
	RuleName    string
	Severity    AlertSeverity
	Status      AlertStatus
	Labels      map[string]string
	Annotations map[string]string
	StartsAt    time.Time
	EndsAt      *time.Time
	FiredCount  int
	LastSentAt  *time.Time
}

type NotificationChannel struct {
	ID      string
	Name    string
	Type    string
	Config  ChannelConfig
	Enabled bool
}

type ChannelConfig struct {
	WebhookURL string
	SlackURL   string
	SMTPHost   string
	SMTPPort   int
	SMTPFrom   string
	SMTPTo     []string
}

type Engine struct {
	logger   *slog.Logger
	rules    map[string]*Rule
	alerts   map[string]*Alert
	channels map[string]*NotificationChannel
	mu       sync.RWMutex
}

func NewEngine(logger *slog.Logger) *Engine {
	return &Engine{
		logger:   logger,
		rules:    make(map[string]*Rule),
		alerts:   make(map[string]*Alert),
		channels: make(map[string]*NotificationChannel),
	}
}

func (e *Engine) AddRule(rule *Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[rule.ID] = rule
	e.logger.Info("alert rule added", "id", rule.ID, "name", rule.Name)
}

func (e *Engine) RemoveRule(ruleID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.rules, ruleID)
}

func (e *Engine) GetRules() []*Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	rules := make([]*Rule, 0, len(e.rules))
	for _, r := range e.rules {
		rules = append(rules, r)
	}
	return rules
}

func (e *Engine) AddChannel(channel *NotificationChannel) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.channels[channel.ID] = channel
	e.logger.Info("notification channel added", "id", channel.ID, "name", channel.Name)
}

func (e *Engine) GetChannels() []*NotificationChannel {
	e.mu.RLock()
	defer e.mu.RUnlock()
	channels := make([]*NotificationChannel, 0, len(e.channels))
	for _, c := range e.channels {
		channels = append(channels, c)
	}
	return channels
}

func (e *Engine) Evaluate(ctx context.Context, metrics map[string]float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		alertID := fmt.Sprintf("%s-%d", rule.ID, time.Now().Unix())
		existingAlert := e.findAlertByRule(rule.ID)

		triggered := e.evaluateCondition(rule.Condition, metrics)

		if triggered && existingAlert == nil {
			alert := &Alert{
				ID:       alertID,
				RuleID:   rule.ID,
				RuleName: rule.Name,
				Severity: rule.Severity,
				Status:   StatusFiring,
				Labels:   rule.Labels,
				Annotations: map[string]string{
					"description": rule.Description,
				},
				StartsAt:   time.Now(),
				FiredCount: 1,
			}
			e.alerts[alertID] = alert
			e.logger.Warn("alert fired", "rule", rule.Name, "severity", rule.Severity)
			go e.notify(ctx, alert, rule.Channels)
		} else if triggered && existingAlert != nil {
			existingAlert.FiredCount++
		} else if !triggered && existingAlert != nil {
			now := time.Now()
			existingAlert.EndsAt = &now
			existingAlert.Status = StatusResolved
			e.logger.Info("alert resolved", "rule", rule.Name)
			go e.notify(ctx, existingAlert, rule.Channels)
		}
	}
}

func (e *Engine) evaluateCondition(cond Condition, metrics map[string]float64) bool {
	value, ok := metrics[cond.Metric]
	if !ok {
		return false
	}

	switch cond.Operator {
	case ">":
		return value > cond.Threshold
	case ">=":
		return value >= cond.Threshold
	case "<":
		return value < cond.Threshold
	case "<=":
		return value <= cond.Threshold
	case "==":
		return value == cond.Threshold
	case "!=":
		return value != cond.Threshold
	default:
		return false
	}
}

func (e *Engine) findAlertByRule(ruleID string) *Alert {
	for _, alert := range e.alerts {
		if alert.RuleID == ruleID && alert.Status == StatusFiring {
			return alert
		}
	}
	return nil
}

func (e *Engine) GetAlerts() []*Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()
	alerts := make([]*Alert, 0, len(e.alerts))
	for _, a := range e.alerts {
		alerts = append(alerts, a)
	}
	return alerts
}

func (e *Engine) GetFiringAlerts() []*Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var alerts []*Alert
	for _, a := range e.alerts {
		if a.Status == StatusFiring {
			alerts = append(alerts, a)
		}
	}
	return alerts
}

func (e *Engine) SilenceAlert(alertID string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if alert, ok := e.alerts[alertID]; ok {
		alert.Status = StatusSilenced
		e.logger.Info("alert silenced", "id", alertID)
	}
}

func (e *Engine) notify(ctx context.Context, alert *Alert, channelIDs []string) {
	for _, channelID := range channelIDs {
		channel, ok := e.channels[channelID]
		if !ok || !channel.Enabled {
			continue
		}

		var err error
		switch channel.Type {
		case "webhook":
			err = e.sendWebhook(ctx, channel.Config.WebhookURL, alert)
		case "slack":
			err = e.sendSlack(ctx, channel.Config.SlackURL, alert)
		}

		if err != nil {
			e.logger.Error("sending notification", "channel", channel.Name, "error", err)
		} else {
			now := time.Now()
			alert.LastSentAt = &now
		}
	}
}

func (e *Engine) sendWebhook(ctx context.Context, url string, alert *Alert) error {
	payload := map[string]interface{}{
		"alert_id":    alert.ID,
		"rule_id":     alert.RuleID,
		"rule_name":   alert.RuleName,
		"severity":    alert.Severity,
		"status":      alert.Status,
		"labels":      alert.Labels,
		"starts_at":   alert.StartsAt,
		"ends_at":     alert.EndsAt,
		"fired_count": alert.FiredCount,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (e *Engine) sendSlack(ctx context.Context, url string, alert *Alert) error {
	color := "#ff4444"
	if alert.Status == StatusResolved {
		color = "#00ff88"
	}

	severityEmoji := map[AlertSeverity]string{
		AlertCritical: "🔴",
		AlertHigh:     "🟠",
		AlertMedium:   "🟡",
		AlertLow:      "🟢",
	}

	payload := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"color": color,
				"blocks": []map[string]interface{}{
					{
						"type": "section",
						"text": map[string]interface{}{
							"type": "mrkdwn",
							"text": fmt.Sprintf("%s *%s*\nStatus: %s\nSeverity: %s",
								severityEmoji[alert.Severity],
								alert.RuleName,
								alert.Status,
								alert.Severity,
							),
						},
					},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}
