package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// AgentPublicChecker checks if an agent is marked as public.
type AgentPublicChecker interface {
	IsAgentPublic(ctx context.Context, name string) (exists bool, public bool, err error)
}

// AgentVisibilityMiddleware restricts access to agents based on API key scopes.
// If the requester has ScopeAdmin or ScopeChat, all agents are accessible.
// If the requester only has ScopeChatPublic, only agents with public=true are allowed.
type AgentVisibilityMiddleware struct {
	checker AgentPublicChecker
}

// NewAgentVisibilityMiddleware creates a new AgentVisibilityMiddleware.
func NewAgentVisibilityMiddleware(checker AgentPublicChecker) *AgentVisibilityMiddleware {
	return &AgentVisibilityMiddleware{checker: checker}
}

// Middleware returns an http.Handler middleware that enforces agent visibility.
func (m *AgentVisibilityMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scopes, _ := r.Context().Value(ContextKeyScopes).(int)

		// Admin and full chat scopes can access all agents.
		if scopes&ScopeAdmin != 0 || scopes&ScopeChat != 0 {
			next.ServeHTTP(w, r)
			return
		}

		// ScopeChatPublic: only public agents.
		if scopes&ScopeChatPublic == 0 {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}

		agentName := chi.URLParam(r, "name")
		if agentName == "" {
			next.ServeHTTP(w, r)
			return
		}

		exists, public, err := m.checker.IsAgentPublic(r.Context(), agentName)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "check agent visibility: "+err.Error())
			return
		}
		if !exists {
			writeJSONError(w, http.StatusNotFound, "agent not found: "+agentName)
			return
		}
		if !public {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "agent is not public"})
			return
		}

		next.ServeHTTP(w, r)
	})
}

// hasFullChatAccess returns true if the requester has admin or full chat scope.
func hasFullChatAccess(scopes int) bool {
	return scopes&ScopeAdmin != 0 || scopes&ScopeChat != 0
}
