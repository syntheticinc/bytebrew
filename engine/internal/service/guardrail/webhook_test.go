package guardrail

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockHTTPClient struct {
	statusCode int
	response   WebhookResponse
	err        error
	lastReq    *http.Request
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	m.lastReq = req
	if m.err != nil {
		return nil, m.err
	}
	body, _ := json.Marshal(m.response)
	return &http.Response{
		StatusCode: m.statusCode,
		Body:       io.NopCloser(strings.NewReader(string(body))),
	}, nil
}

func TestWebhookChecker_Pass(t *testing.T) {
	client := &mockHTTPClient{
		statusCode: 200,
		response:   WebhookResponse{Pass: true, Reason: "looks good"},
	}
	checker := NewWebhookChecker(WebhookConfig{
		URL:    "https://example.com/guardrail",
		Client: client,
		Agent:  "test-agent",
	})

	result, err := checker.Check(context.Background(), "agent output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass")
	}
	if result.Reason != "looks good" {
		t.Errorf("expected reason %q, got %q", "looks good", result.Reason)
	}
}

func TestWebhookChecker_Fail(t *testing.T) {
	client := &mockHTTPClient{
		statusCode: 200,
		response:   WebhookResponse{Pass: false, Reason: "inappropriate content"},
	}
	checker := NewWebhookChecker(WebhookConfig{
		URL:    "https://example.com/guardrail",
		Client: client,
	})

	result, err := checker.Check(context.Background(), "bad output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail")
	}
}

func TestWebhookChecker_AuthHeader(t *testing.T) {
	client := &mockHTTPClient{
		statusCode: 200,
		response:   WebhookResponse{Pass: true},
	}
	checker := NewWebhookChecker(WebhookConfig{
		URL:       "https://example.com/guardrail",
		AuthType:  "api_key",
		AuthToken: "my-secret-token",
		Client:    client,
	})

	_, err := checker.Check(context.Background(), "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	authHeader := client.lastReq.Header.Get("Authorization")
	if authHeader != "Bearer my-secret-token" {
		t.Errorf("expected auth header %q, got %q", "Bearer my-secret-token", authHeader)
	}
}

func TestWebhookChecker_RequestPayload(t *testing.T) {
	client := &mockHTTPClient{
		statusCode: 200,
		response:   WebhookResponse{Pass: true},
	}
	checker := NewWebhookChecker(WebhookConfig{
		URL:       "https://example.com/guardrail",
		Agent:     "support-agent",
		SessionID: "sess-123",
		Client:    client,
	})

	_, err := checker.Check(context.Background(), "the response")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body, _ := io.ReadAll(client.lastReq.Body)
	var req WebhookRequest
	json.Unmarshal(body, &req)

	if req.Event != "guardrail_check" {
		t.Errorf("expected event %q, got %q", "guardrail_check", req.Event)
	}
	if req.Agent != "support-agent" {
		t.Errorf("expected agent %q, got %q", "support-agent", req.Agent)
	}
	if req.Response != "the response" {
		t.Errorf("expected response %q, got %q", "the response", req.Response)
	}
}

func TestWebhookChecker_HTTPError(t *testing.T) {
	client := &mockHTTPClient{err: fmt.Errorf("connection refused")}
	checker := NewWebhookChecker(WebhookConfig{
		URL:    "https://example.com/guardrail",
		Client: client,
	})

	_, err := checker.Check(context.Background(), "output")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestWebhookChecker_Non200Status(t *testing.T) {
	client := &mockHTTPClient{statusCode: 500}
	checker := NewWebhookChecker(WebhookConfig{
		URL:    "https://example.com/guardrail",
		Client: client,
	})

	_, err := checker.Check(context.Background(), "output")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
}
