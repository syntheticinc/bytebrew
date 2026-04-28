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
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

type mockDispatchQueryer struct {
	tasks map[string]*domain.TaskPacket
}

func (m *mockDispatchQueryer) GetTask(taskID string) (*domain.TaskPacket, bool) {
	tp, ok := m.tasks[taskID]
	return tp, ok
}

func (m *mockDispatchQueryer) ListTasksBySession(sessionID string) []*domain.TaskPacket {
	var result []*domain.TaskPacket
	for _, tp := range m.tasks {
		if tp.SessionID == sessionID {
			result = append(result, tp)
		}
	}
	return result
}

func newTestDispatchHandler() (*DispatchHandler, *mockDispatchQueryer) {
	now := time.Date(2026, 4, 9, 12, 0, 0, 0, time.UTC)
	mock := &mockDispatchQueryer{
		tasks: map[string]*domain.TaskPacket{
			"task-1": {
				ID:          "task-1",
				ParentAgent: "supervisor",
				ChildAgent:  "coder",
				SessionID:   "session-abc",
				Input:       "implement feature X",
				Status:      domain.TaskPacketCompleted,
				Result:      "done",
				CreatedAt:   now,
				StartedAt:   now.Add(1 * time.Second),
				FinishedAt:  now.Add(10 * time.Second),
			},
			"task-2": {
				ID:          "task-2",
				ParentAgent: "supervisor",
				ChildAgent:  "tester",
				SessionID:   "session-abc",
				Input:       "run tests",
				Status:      domain.TaskPacketRunning,
				CreatedAt:   now,
				StartedAt:   now.Add(2 * time.Second),
			},
			"task-3": {
				ID:          "task-3",
				ParentAgent: "supervisor",
				ChildAgent:  "coder",
				SessionID:   "session-other",
				Input:       "fix bug",
				Status:      domain.TaskPacketPending,
				CreatedAt:   now,
			},
		},
	}
	return NewDispatchHandler(mock), mock
}

func TestDispatchHandler_Get(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		wantStatus int
		wantState  string
	}{
		{"existing task", "task-1", http.StatusOK, "completed"},
		{"not found", "nonexistent", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _ := newTestDispatchHandler()

			r := chi.NewRouter()
			r.Get("/api/v1/dispatch/tasks/{taskId}", handler.Get)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/dispatch/tasks/"+tt.taskID, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			if tt.wantStatus == http.StatusOK {
				var resp TaskPacketResponse
				require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
				assert.Equal(t, tt.taskID, resp.ID)
				assert.Equal(t, tt.wantState, resp.State)
				assert.Equal(t, "session-abc", resp.SessionID)
				assert.NotEmpty(t, resp.CreatedAt)
				assert.NotEmpty(t, resp.UpdatedAt)
			}
		})
	}
}

func TestDispatchHandler_Get_ResponseFields(t *testing.T) {
	handler, _ := newTestDispatchHandler()

	r := chi.NewRouter()
	r.Get("/api/v1/dispatch/tasks/{taskId}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dispatch/tasks/task-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp TaskPacketResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	assert.Equal(t, "task-1", resp.ID)
	assert.Equal(t, "coder", resp.AgentName)
	assert.Equal(t, "implement feature X", resp.Task)
	assert.Equal(t, "session-abc", resp.SessionID)
	assert.Equal(t, "completed", resp.State)
	assert.Equal(t, "done", resp.Result)
}

func TestDispatchHandler_ListBySession(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		wantCount  int
		wantStatus int
	}{
		{"session with tasks", "session-abc", 2, http.StatusOK},
		{"session with one task", "session-other", 1, http.StatusOK},
		{"empty session", "no-such-session", 0, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, _ := newTestDispatchHandler()

			r := chi.NewRouter()
			r.Get("/api/v1/sessions/{sessionId}/dispatch-tasks", handler.ListBySession)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions/"+tt.sessionID+"/dispatch-tasks", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)

			var resp []TaskPacketResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			assert.Equal(t, tt.wantCount, len(resp))
		})
	}
}

func TestDispatchHandler_UpdatedAt_Heuristic(t *testing.T) {
	handler, _ := newTestDispatchHandler()

	r := chi.NewRouter()
	r.Get("/api/v1/dispatch/tasks/{taskId}", handler.Get)

	// task-2 is running: updatedAt should be startedAt
	req := httptest.NewRequest(http.MethodGet, "/api/v1/dispatch/tasks/task-2", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp TaskPacketResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "running", resp.State)
	// updatedAt should reflect startedAt, not createdAt
	assert.NotEqual(t, resp.CreatedAt, resp.UpdatedAt)

	// task-3 is pending: updatedAt should equal createdAt
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/dispatch/tasks/task-3", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	var resp2 TaskPacketResponse
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp2))
	assert.Equal(t, "pending", resp2.State)
	assert.Equal(t, resp2.CreatedAt, resp2.UpdatedAt)
}
