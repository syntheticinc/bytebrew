package escalation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// CapabilityReader reads enabled capabilities for an agent.
type CapabilityReader interface {
	GetEscalationConfig(ctx context.Context, agentName string) (*Config, error)
}

// Config holds the escalation configuration extracted from agent capabilities.
type Config struct {
	Action     string `json:"action"`      // transfer_to_user, notify_webhook
	WebhookURL string `json:"webhook_url"` // URL for notify_webhook action
}

// Handler implements the EscalationHandler interface for the escalate tool.
type Handler struct {
	reader     CapabilityReader
	httpClient *http.Client
}

// NewHandler creates a new escalation handler.
func NewHandler(reader CapabilityReader) *Handler {
	return &Handler{
		reader: reader,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Escalate reads the agent's escalation config and performs the configured action.
func (h *Handler) Escalate(ctx context.Context, sessionID, agentName, reason string) (string, error) {
	cfg, err := h.reader.GetEscalationConfig(ctx, agentName)
	if err != nil {
		return "", fmt.Errorf("read escalation config: %w", err)
	}

	action := "transfer_to_user"
	if cfg != nil && cfg.Action != "" {
		action = cfg.Action
	}

	slog.InfoContext(ctx, "[Escalation] triggered",
		"session_id", sessionID, "agent", agentName, "reason", reason, "action", action)

	switch action {
	case "notify_webhook":
		if cfg == nil || cfg.WebhookURL == "" {
			return "Escalation triggered: notify_webhook (no webhook URL configured — skipped)", nil
		}
		if err := h.callWebhook(ctx, cfg.WebhookURL, sessionID, agentName, reason); err != nil {
			slog.ErrorContext(ctx, "[Escalation] webhook call failed",
				"url", cfg.WebhookURL, "error", err)
			return fmt.Sprintf("Escalation triggered: notify_webhook (webhook failed: %v)", err), nil
		}
		return "Escalation triggered: notify_webhook (webhook notified successfully)", nil

	default: // transfer_to_user
		return "Escalation triggered: transfer_to_user. The conversation has been flagged for human review. " +
			"Reason: " + reason, nil
	}
}

// callWebhook sends an escalation notification to the configured webhook URL.
func (h *Handler) callWebhook(ctx context.Context, url, sessionID, agentName, reason string) error {
	payload := map[string]string{
		"event":      "escalation",
		"agent":      agentName,
		"session_id": sessionID,
		"reason":     reason,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("webhook POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
