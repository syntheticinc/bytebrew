package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockTriggerService implements TriggerService for testing.
type mockTriggerService struct {
	createErr    error
	createResult *TriggerResponse
}

func (m *mockTriggerService) ListTriggers(_ context.Context) ([]TriggerResponse, error) {
	return nil, nil
}

func (m *mockTriggerService) ListTriggersBySchema(_ context.Context, _ string) ([]TriggerResponse, error) {
	return nil, nil
}

func (m *mockTriggerService) CreateTrigger(_ context.Context, _ CreateTriggerRequest) (*TriggerResponse, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.createResult, nil
}

func (m *mockTriggerService) UpdateTrigger(_ context.Context, _ string, _ CreateTriggerRequest) (*TriggerResponse, error) {
	return nil, nil
}

func (m *mockTriggerService) DeleteTrigger(_ context.Context, _ string) error {
	return nil
}

func (m *mockTriggerService) SetTriggerTarget(_ context.Context, _ string, _ string) (*TriggerResponse, error) {
	return nil, nil
}

func (m *mockTriggerService) ClearTriggerTarget(_ context.Context, _ string) error {
	return nil
}

// TestTriggerHandler_Create_RejectsNonEntryAgent verifies that creating a trigger
// for a non-entry agent returns an error containing "entry agent".
func TestTriggerHandler_Create_RejectsNonEntryAgent(t *testing.T) {
	service := &mockTriggerService{
		createErr: fmt.Errorf("agent %q is not an entry agent: it has incoming agent relations", "worker"),
	}
	handler := NewTriggerHandler(service)

	body, _ := json.Marshal(CreateTriggerRequest{
		Type:    "schedule",
		Title:   "daily sync",
		AgentID: "agent-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	errMsg, ok := resp["error"]
	if !ok {
		t.Fatal("expected error field in response")
	}
	if !contains(errMsg, "entry agent") {
		t.Errorf("expected error to contain %q, got %q", "entry agent", errMsg)
	}
}

// TestTriggerHandler_Create_AcceptsEntryAgent verifies that creating a trigger
// for a valid entry agent returns 201.
func TestTriggerHandler_Create_AcceptsEntryAgent(t *testing.T) {
	service := &mockTriggerService{
		createResult: &TriggerResponse{
			ID:        "trigger-1",
			Type:      "schedule",
			Title:     "daily sync",
			AgentID:   "agent-1",
			Enabled:   true,
			CreatedAt: "2026-04-08T00:00:00Z",
		},
	}
	handler := NewTriggerHandler(service)

	body, _ := json.Marshal(CreateTriggerRequest{
		Type:    "schedule",
		Title:   "daily sync",
		AgentID: "agent-1",
	})
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

// contains checks if s contains substr (avoids importing strings in test).
func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
