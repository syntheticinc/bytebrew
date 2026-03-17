package task

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// EscalationDetector checks agent responses for escalation trigger keywords.
type EscalationDetector struct{}

// NewEscalationDetector creates a new EscalationDetector.
func NewEscalationDetector() *EscalationDetector {
	return &EscalationDetector{}
}

// Check returns the first matched trigger keyword (case-insensitive), or "" if none.
func (d *EscalationDetector) Check(response string, triggers []string) string {
	lower := strings.ToLower(response)
	for _, trigger := range triggers {
		if strings.Contains(lower, strings.ToLower(trigger)) {
			return trigger
		}
	}
	return ""
}

// EscalationWebhookPayload is sent to the escalation webhook URL.
type EscalationWebhookPayload struct {
	SessionID           string           `json:"session_id"`
	TaskID              uint             `json:"task_id"`
	Reason              string           `json:"reason"`
	AgentName           string           `json:"agent_name"`
	ConversationSummary string           `json:"conversation_summary"`
	LastMessages        []MessageSummary `json:"last_messages"`
}

// MessageSummary is a compact representation of a message for escalation payloads.
type MessageSummary struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// EscalationWebhookSender sends escalation webhooks with exponential backoff retry.
type EscalationWebhookSender struct {
	client     *http.Client
	maxRetries int
	baseDelay  time.Duration
}

// NewEscalationWebhookSender creates a sender with sensible defaults.
func NewEscalationWebhookSender() *EscalationWebhookSender {
	return &EscalationWebhookSender{
		client:     &http.Client{Timeout: 10 * time.Second},
		maxRetries: 3,
		baseDelay:  time.Second,
	}
}

// Send dispatches the escalation payload to webhookURL with retry on failure.
func (s *EscalationWebhookSender) Send(ctx context.Context, webhookURL string, payload EscalationWebhookPayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		if attempt > 0 {
			delay := s.baseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(data))
		if reqErr != nil {
			return fmt.Errorf("create request: %w", reqErr)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, doErr := s.client.Do(req)
		if doErr != nil {
			lastErr = doErr
			slog.WarnContext(ctx, "escalation webhook failed", "attempt", attempt+1, "error", doErr)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("webhook returned status %d", resp.StatusCode)
		slog.WarnContext(ctx, "escalation webhook non-2xx", "attempt", attempt+1, "status", resp.StatusCode)
	}
	return fmt.Errorf("escalation webhook failed after %d retries: %w", s.maxRetries+1, lastErr)
}
