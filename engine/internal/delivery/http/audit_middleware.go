package http

import (
	"context"
	"net/http"
	"time"
)

// AuditLogger is used by the audit middleware to record API calls.
type AuditLogger interface {
	Log(ctx context.Context, entry AuditEntry) error
}

// AuditEntry represents a single audit log entry for the middleware.
type AuditEntry struct {
	Timestamp time.Time
	ActorType string
	ActorID   string
	Action    string
	Resource  string
	Details   map[string]interface{}
	SessionID string
}

// AuditMiddleware returns middleware that logs all API calls to the audit log.
func AuditMiddleware(logger AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actorType, _ := r.Context().Value(ContextKeyActorType).(string)
			actorID, _ := r.Context().Value(ContextKeyActorID).(string)

			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)

			_ = logger.Log(r.Context(), AuditEntry{
				Timestamp: time.Now(),
				ActorType: actorType,
				ActorID:   actorID,
				Action:    "api_call",
				Resource:  r.Method + " " + r.URL.Path,
				Details: map[string]interface{}{
					"method":      r.Method,
					"path":        r.URL.Path,
					"status_code": sw.status,
				},
			})
		})
	}
}

// statusWriter wraps http.ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.status = code
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

// Unwrap returns the underlying ResponseWriter, allowing middleware traversal
// (e.g. findFlusher in chat_handler.go can reach http.Flusher through the chain).
func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Flush delegates to the underlying ResponseWriter if it implements http.Flusher.
// This is critical for SSE streaming — without it, events buffer in Go's internal
// writer and browsers receive them in ~4KB TCP batches instead of per-token.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
