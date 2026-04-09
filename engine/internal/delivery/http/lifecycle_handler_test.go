package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLifecycleProvider struct {
	status *LifecycleStatus
	err    error
}

func (m *mockLifecycleProvider) GetLifecycleStatus(_ context.Context, _, _ string) (*LifecycleStatus, error) {
	return m.status, m.err
}

func TestLifecycleHandler_Status_OK(t *testing.T) {
	provider := &mockLifecycleProvider{
		status: &LifecycleStatus{
			Mode:          "persistent",
			State:         "ready",
			TasksHandled:  5,
			ContextTokens: 1200,
			MaxContext:    16000,
		},
	}
	handler := NewLifecycleHandler(provider)

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{name}/lifecycle", handler.Status)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/my-agent/lifecycle?session_id=sess-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var status LifecycleStatus
	err := json.Unmarshal(w.Body.Bytes(), &status)
	require.NoError(t, err)
	assert.Equal(t, "persistent", status.Mode)
	assert.Equal(t, "ready", status.State)
	assert.Equal(t, 5, status.TasksHandled)
	assert.Equal(t, 1200, status.ContextTokens)
	assert.Equal(t, 16000, status.MaxContext)
}

func TestLifecycleHandler_Status_NotFound(t *testing.T) {
	provider := &mockLifecycleProvider{status: nil}
	handler := NewLifecycleHandler(provider)

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{name}/lifecycle", handler.Status)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/no-agent/lifecycle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestLifecycleHandler_Status_NoSessionID(t *testing.T) {
	provider := &mockLifecycleProvider{
		status: &LifecycleStatus{
			Mode:       "spawn",
			State:      "no_session",
			MaxContext: 16000,
		},
	}
	handler := NewLifecycleHandler(provider)

	r := chi.NewRouter()
	r.Get("/api/v1/agents/{name}/lifecycle", handler.Status)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents/my-agent/lifecycle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var status LifecycleStatus
	err := json.Unmarshal(w.Body.Bytes(), &status)
	require.NoError(t, err)
	assert.Equal(t, "spawn", status.Mode)
	assert.Equal(t, "no_session", status.State)
}
