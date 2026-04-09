package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockCircuitBreakerQuerier struct {
	snapshots []CircuitBreakerState
	resetOK   bool
	resetLog  []string
}

func (m *mockCircuitBreakerQuerier) Snapshots() []CircuitBreakerState { return m.snapshots }
func (m *mockCircuitBreakerQuerier) Reset(name string) bool {
	m.resetLog = append(m.resetLog, name)
	return m.resetOK
}

type mockDeadLetterQuerier struct {
	entries []DeadLetterEntry
}

func (m *mockDeadLetterQuerier) DeadLetters() []DeadLetterEntry { return m.entries }

type mockHeartbeatQuerier struct {
	entries []HeartbeatEntry
}

func (m *mockHeartbeatQuerier) Snapshots() []HeartbeatEntry { return m.entries }

// --- tests ---

func TestResilienceHandler_ListCircuitBreakers(t *testing.T) {
	tests := []struct {
		name       string
		querier    CircuitBreakerQuerier
		wantCount  int
		wantStatus int
	}{
		{
			name:       "nil querier returns empty list",
			querier:    nil,
			wantCount:  0,
			wantStatus: http.StatusOK,
		},
		{
			name: "returns all breakers",
			querier: &mockCircuitBreakerQuerier{
				snapshots: []CircuitBreakerState{
					{Name: "openai", State: "closed"},
					{Name: "mcp-git", State: "open", FailureCount: 3},
				},
			},
			wantCount:  2,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty registry",
			querier:    &mockCircuitBreakerQuerier{snapshots: []CircuitBreakerState{}},
			wantCount:  0,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewResilienceHandler(tt.querier, nil, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/resilience/circuit-breakers", nil)
			w := httptest.NewRecorder()

			h.ListCircuitBreakers(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var result []CircuitBreakerState
			require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestResilienceHandler_ResetCircuitBreaker(t *testing.T) {
	tests := []struct {
		name       string
		querier    CircuitBreakerQuerier
		cbName     string
		wantStatus int
	}{
		{
			name:       "nil querier returns 404",
			querier:    nil,
			cbName:     "openai",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "empty name returns 400",
			querier:    &mockCircuitBreakerQuerier{resetOK: true},
			cbName:     "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found returns 404",
			querier:    &mockCircuitBreakerQuerier{resetOK: false},
			cbName:     "unknown",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "successful reset",
			querier:    &mockCircuitBreakerQuerier{resetOK: true},
			cbName:     "openai",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewResilienceHandler(tt.querier, nil, nil)

			r := chi.NewRouter()
			r.Post("/api/v1/admin/resilience/circuit-breakers/{name}/reset", h.ResetCircuitBreaker)

			path := "/api/v1/admin/resilience/circuit-breakers/" + tt.cbName + "/reset"
			req := httptest.NewRequest(http.MethodPost, path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestResilienceHandler_ListDeadLetters(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		querier    DeadLetterQuerier
		wantCount  int
		wantStatus int
	}{
		{
			name:       "nil querier returns empty list",
			querier:    nil,
			wantCount:  0,
			wantStatus: http.StatusOK,
		},
		{
			name: "returns dead letters",
			querier: &mockDeadLetterQuerier{
				entries: []DeadLetterEntry{
					{TaskID: "t1", AgentID: "a1", StartedAt: now, Status: "timeout"},
					{TaskID: "t2", AgentID: "a2", StartedAt: now, Status: "timeout"},
				},
			},
			wantCount:  2,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewResilienceHandler(nil, tt.querier, nil)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/resilience/dead-letters", nil)
			w := httptest.NewRecorder()

			h.ListDeadLetters(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var result []DeadLetterEntry
			require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
			assert.Len(t, result, tt.wantCount)
		})
	}
}

func TestResilienceHandler_ListHeartbeats(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		querier    HeartbeatQuerier
		wantCount  int
		wantStatus int
	}{
		{
			name:       "nil querier returns empty list",
			querier:    nil,
			wantCount:  0,
			wantStatus: http.StatusOK,
		},
		{
			name: "returns heartbeats",
			querier: &mockHeartbeatQuerier{
				entries: []HeartbeatEntry{
					{AgentID: "a1", AgentType: "spawn", LastHeartbeat: now, CurrentStep: "step-1"},
				},
			},
			wantCount:  1,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewResilienceHandler(nil, nil, tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/resilience/heartbeats", nil)
			w := httptest.NewRecorder()

			h.ListHeartbeats(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var result []HeartbeatEntry
			require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
			assert.Len(t, result, tt.wantCount)
		})
	}
}
