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

func TestStatusWriter_PreventDoubleWriteHeader(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

	sw.WriteHeader(http.StatusCreated)
	sw.WriteHeader(http.StatusNotFound) // should be ignored

	assert.Equal(t, http.StatusCreated, sw.status)
}
