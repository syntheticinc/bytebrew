package http

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// resolveAuditAction maps an HTTP method + path to a semantic audit action
// token ("agent.create", "auth.fail", etc.). Unknown combinations fall back
// to "api_call" so the audit log is never silently empty. Compliance auditors
// query by action, so this mapping is part of the audit contract.
func resolveAuditAction(method, path string, status int) string {
	switch {
	// Auth endpoints — status-dependent semantics.
	case method == "POST" && strings.HasPrefix(path, "/api/v1/auth/local-session"):
		if status >= 400 {
			return "auth.fail"
		}
		return "auth.success"
	case method == "POST" && path == "/api/v1/auth/tokens":
		return "token.create"
	case method == "DELETE" && strings.HasPrefix(path, "/api/v1/auth/tokens/"):
		return "token.revoke"

	// Agent CRUD.
	case method == "POST" && path == "/api/v1/agents":
		return "agent.create"
	case (method == "PUT" || method == "PATCH") && strings.HasPrefix(path, "/api/v1/agents/"):
		return "agent.update"
	case method == "DELETE" && strings.HasPrefix(path, "/api/v1/agents/"):
		return "agent.delete"

	// Schema CRUD. Note: /schemas/{id}/chat and /schemas/{id}/agent-relations
	// are handled as their own actions below.
	case method == "POST" && strings.Contains(path, "/agent-relations"):
		return "agent_relation.create"
	case method == "DELETE" && strings.Contains(path, "/agent-relations/"):
		return "agent_relation.delete"
	case method == "POST" && strings.HasSuffix(path, "/chat") && strings.HasPrefix(path, "/api/v1/schemas/"):
		return "chat.message"
	case method == "POST" && path == "/api/v1/schemas":
		return "schema.create"
	case (method == "PUT" || method == "PATCH") && strings.HasPrefix(path, "/api/v1/schemas/"):
		return "schema.update"
	case method == "DELETE" && strings.HasPrefix(path, "/api/v1/schemas/"):
		return "schema.delete"

	// Model / MCP / KB / Settings CRUD.
	case method == "POST" && path == "/api/v1/models":
		return "model.create"
	case (method == "PUT" || method == "PATCH") && strings.HasPrefix(path, "/api/v1/models/"):
		return "model.update"
	case method == "DELETE" && strings.HasPrefix(path, "/api/v1/models/"):
		return "model.delete"
	case method == "POST" && path == "/api/v1/mcp-servers":
		return "mcp.create"
	case method == "DELETE" && strings.HasPrefix(path, "/api/v1/mcp-servers/"):
		return "mcp.delete"
	case method == "POST" && path == "/api/v1/knowledge-bases":
		return "kb.create"
	case method == "DELETE" && strings.HasPrefix(path, "/api/v1/knowledge-bases/"):
		return "kb.delete"
	case (method == "PUT" || method == "PATCH") && strings.HasPrefix(path, "/api/v1/settings/"):
		return "setting.update"

	// Session CRUD.
	case method == "DELETE" && strings.HasPrefix(path, "/api/v1/sessions/"):
		return "session.delete"
	}
	return "api_call"
}

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
				Action:    resolveAuditAction(r.Method, r.URL.Path, sw.status),
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
