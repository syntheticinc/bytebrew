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
	stuck   []StuckAgentEntry
}

func (m *mockHeartbeatQuerier) Snapshots() []HeartbeatEntry        { return m.entries }
func (m *mockHeartbeatQuerier) StuckSnapshots() []StuckAgentEntry  { return m.stuck }

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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/resilience/circuit-breakers", nil)
			w := httptest.NewRecorder()

			h.ListCircuitBreakers(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var result struct {
				Breakers []CircuitBreakerState `json:"breakers"`
			}
			require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
			assert.Len(t, result.Breakers, tt.wantCount)
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
			r.Post("/api/v1/resilience/circuit-breakers/{name}/reset", h.ResetCircuitBreaker)

			path := "/api/v1/resilience/circuit-breakers/" + tt.cbName + "/reset"
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
			name: "returns dead letters with extended fields",
			querier: &mockDeadLetterQuerier{
				entries: []DeadLetterEntry{
					{TaskID: "t1", AgentID: "a1", AgentName: "support-agent", StartedAt: now, MovedAt: now.Add(5 * time.Minute), Status: "timeout", Reason: "task_timeout", ElapsedMs: 300000, LastError: "context deadline exceeded"},
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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/resilience/dead-letter", nil)
			w := httptest.NewRecorder()

			h.ListDeadLetters(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var result struct {
				Tasks []DeadLetterEntry `json:"tasks"`
			}
			require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
			assert.Len(t, result.Tasks, tt.wantCount)

			// When we have entries, verify the extended fields round-trip.
			if tt.wantCount > 0 {
				assert.Equal(t, "support-agent", result.Tasks[0].AgentName)
				assert.Equal(t, "task_timeout", result.Tasks[0].Reason)
				assert.Equal(t, int64(300000), result.Tasks[0].ElapsedMs)
				assert.Equal(t, "context deadline exceeded", result.Tasks[0].LastError)
				assert.False(t, result.Tasks[0].MovedAt.IsZero(), "moved_at should be non-zero for timed-out task")
			}
		})
	}
}

func TestResilienceHandler_ListStuckAgents(t *testing.T) {
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
			name: "returns only stuck agents",
			querier: &mockHeartbeatQuerier{
				stuck: []StuckAgentEntry{
					{AgentID: "a1", AgentType: "spawn", LastHeartbeat: now.Add(-time.Minute), ElapsedMs: 60000, Status: "stuck"},
				},
			},
			wantCount:  1,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewResilienceHandler(nil, nil, tt.querier)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/resilience/stuck-agents", nil)
			w := httptest.NewRecorder()

			h.ListStuckAgents(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var result struct {
				Agents []StuckAgentEntry `json:"agents"`
			}
			require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
			assert.Len(t, result.Agents, tt.wantCount)
			if tt.wantCount > 0 {
				assert.Equal(t, "stuck", result.Agents[0].Status)
				assert.Equal(t, int64(60000), result.Agents[0].ElapsedMs)
			}
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
			req := httptest.NewRequest(http.MethodGet, "/api/v1/resilience/heartbeats", nil)
			w := httptest.NewRecorder()

			h.ListHeartbeats(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var result []HeartbeatEntry
			require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
			assert.Len(t, result, tt.wantCount)
		})
	}
}
