package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAuditLogger struct {
	entries []AuditEntry
	err     error
}

func (m *mockAuditLogger) Log(_ context.Context, entry AuditEntry) error {
	m.entries = append(m.entries, entry)
	return m.err
}

func TestAuditMiddleware_LogsAPICall(t *testing.T) {
	logger := &mockAuditLogger{}
	handler := AuditMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agents", nil)
	ctx := context.WithValue(req.Context(), ContextKeyActorType, "admin")
	ctx = context.WithValue(ctx, ContextKeyActorID, "user-1")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Len(t, logger.entries, 1)
	entry := logger.entries[0]
	assert.Equal(t, "admin", entry.ActorType)
	assert.Equal(t, "user-1", entry.ActorID)
	assert.Equal(t, "api_call", entry.Action)
	assert.Equal(t, "GET /api/v1/agents", entry.Resource)
	assert.Equal(t, http.MethodGet, entry.Details["method"])
	assert.Equal(t, "/api/v1/agents", entry.Details["path"])
	assert.Equal(t, http.StatusOK, entry.Details["status_code"])
}

func TestAuditMiddleware_CapturesStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &mockAuditLogger{}
			handler := AuditMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			require.Len(t, logger.entries, 1)
			assert.Equal(t, tt.statusCode, logger.entries[0].Details["status_code"])
		})
	}
}

func TestAuditMiddleware_DefaultStatusOK(t *testing.T) {
	logger := &mockAuditLogger{}
	handler := AuditMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No explicit WriteHeader — defaults to 200
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Len(t, logger.entries, 1)
	assert.Equal(t, http.StatusOK, logger.entries[0].Details["status_code"])
}

func TestAuditMiddleware_NoActorContext(t *testing.T) {
	logger := &mockAuditLogger{}
	handler := AuditMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/reload", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Len(t, logger.entries, 1)
	assert.Equal(t, "", logger.entries[0].ActorType)
	assert.Equal(t, "", logger.entries[0].ActorID)
}

func TestAuditMiddleware_LogsMethodAndPath(t *testing.T) {
	logger := &mockAuditLogger{}
	handler := AuditMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	require.Len(t, logger.entries, 1)
	assert.Equal(t, "POST /api/v1/tasks", logger.entries[0].Resource)
}

func TestResolveAuditAction_SemanticActions(t *testing.T) {
	cases := []struct {
		name, method, path string
		status             int
		want               string
	}{
		{"create agent", "POST", "/api/v1/agents", 201, "agent.create"},
		{"update agent", "PATCH", "/api/v1/agents/ag-a", 200, "agent.update"},
		{"delete agent", "DELETE", "/api/v1/agents/ag-a", 204, "agent.delete"},
		{"create schema", "POST", "/api/v1/schemas", 201, "schema.create"},
		{"delete schema", "DELETE", "/api/v1/schemas/s-1", 204, "schema.delete"},
		{"chat message", "POST", "/api/v1/schemas/s-1/chat", 200, "chat.message"},
		{"agent relation", "POST", "/api/v1/schemas/s-1/agent-relations", 201, "agent_relation.create"},
		{"create model", "POST", "/api/v1/models", 201, "model.create"},
		{"delete mcp", "DELETE", "/api/v1/mcp-servers/m-1", 204, "mcp.delete"},
		{"token create", "POST", "/api/v1/auth/tokens", 201, "token.create"},
		{"token revoke", "DELETE", "/api/v1/auth/tokens/t-1", 204, "token.revoke"},
		{"local session success", "POST", "/api/v1/auth/local-session", 200, "auth.success"},
		{"local session fail", "POST", "/api/v1/auth/local-session", 401, "auth.fail"},
		{"GET agents list fallback", "GET", "/api/v1/agents", 200, "api_call"},
		{"unknown path", "POST", "/api/v1/unknown", 200, "api_call"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, resolveAuditAction(c.method, c.path, c.status))
		})
	}
}

func TestStatusWriter_PreventDoubleWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

	sw.WriteHeader(http.StatusCreated)
	sw.WriteHeader(http.StatusNotFound) // should be ignored

	assert.Equal(t, http.StatusCreated, sw.status)
}

func TestStatusWriter_Unwrap(t *testing.T) {
	inner := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: inner}

	got := sw.Unwrap()
	assert.Equal(t, inner, got, "Unwrap must return the underlying ResponseWriter")
}

func TestStatusWriter_Flush(t *testing.T) {
	// httptest.ResponseRecorder implements http.Flusher
	inner := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: inner}

	assert.True(t, inner.Flushed == false)
	sw.Flush()
	assert.True(t, inner.Flushed, "Flush must delegate to underlying Flusher")
}

func TestFindFlusher_ThroughStatusWriter(t *testing.T) {
	// Simulate the middleware chain: statusWriter wraps httptest.ResponseRecorder.
	// findFlusher should unwrap statusWriter and find the Flusher on the recorder.
	inner := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: inner}

	flush := findFlusher(sw)

	// Should NOT be a no-op — it should actually flush.
	flush()
	assert.True(t, inner.Flushed, "findFlusher must traverse statusWriter via Unwrap() to find Flusher")
}

func TestFindFlusher_DoubleWrapped(t *testing.T) {
	// Two layers of statusWriter wrapping.
	inner := httptest.NewRecorder()
	sw1 := &statusWriter{ResponseWriter: inner}
	sw2 := &statusWriter{ResponseWriter: sw1}

	flush := findFlusher(sw2)
	flush()
	assert.True(t, inner.Flushed, "findFlusher must traverse multiple wrapper layers")
}
