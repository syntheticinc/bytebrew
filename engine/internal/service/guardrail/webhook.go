package guardrail

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	webhookTimeout     = 10 * time.Second
	webhookMaxRetries  = 1
)

// WebhookRequest is the payload sent to the guardrail webhook.
type WebhookRequest struct {
	Event     string                 `json:"event"`
	Agent     string                 `json:"agent"`
	SessionID string                 `json:"session_id"`
	Response  string                 `json:"response"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// WebhookResponse is the expected response from the guardrail webhook.
type WebhookResponse struct {
	Pass   bool   `json:"pass"`
	Reason string `json:"reason,omitempty"`
}

// WebhookChecker validates agent output by calling an external webhook.
type WebhookChecker struct {
	url       string
	authType  string // "none", "api_key", "forward_headers", "oauth2"
	authToken string
	agent     string
	sessionID string
	client    HTTPClient
}

// HTTPClient is a consumer-side interface for making HTTP requests.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// WebhookConfig holds configuration for creating a WebhookChecker.
type WebhookConfig struct {
	URL       string
	AuthType  string
	AuthToken string
	Agent     string
	SessionID string
	Client    HTTPClient // optional, uses default if nil
}

// NewWebhookChecker creates a new WebhookChecker.
func NewWebhookChecker(cfg WebhookConfig) *WebhookChecker {
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: webhookTimeout}
	}
	return &WebhookChecker{
		url:       cfg.URL,
		authType:  cfg.AuthType,
		authToken: cfg.AuthToken,
		agent:     cfg.Agent,
		sessionID: cfg.SessionID,
		client:    client,
	}
}

// Check sends the output to the webhook and returns the result.
func (w *WebhookChecker) Check(ctx context.Context, output string) (*CheckResult, error) {
	payload := WebhookRequest{
		Event:     "guardrail_check",
		Agent:     w.agent,
		SessionID: w.sessionID,
		Response:  output,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal webhook payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= webhookMaxRetries; attempt++ {
		result, err := w.doRequest(ctx, body)
		if err != nil {
			lastErr = err
			continue
		}
		return result, nil
	}

	return nil, fmt.Errorf("webhook failed after %d attempts: %w", webhookMaxRetries+1, lastErr)
}

func (w *WebhookChecker) doRequest(ctx context.Context, body []byte) (*CheckResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Apply auth
	switch w.authType {
	case "api_key":
		if w.authToken != "" {
			req.Header.Set("Authorization", "Bearer "+w.authToken)
		}
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read webhook response: %w", err)
	}

	var webhookResp WebhookResponse
	if err := json.Unmarshal(respBody, &webhookResp); err != nil {
		return nil, fmt.Errorf("parse webhook response: %w", err)
	}

	return &CheckResult{
		Passed: webhookResp.Pass,
		Reason: webhookResp.Reason,
	}, nil
}
