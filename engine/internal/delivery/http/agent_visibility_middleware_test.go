package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAgentPublicChecker struct {
	agents map[string]bool // name -> public
	err    error
}

func (m *mockAgentPublicChecker) IsAgentPublic(_ context.Context, name string) (bool, bool, error) {
	if m.err != nil {
		return false, false, m.err
	}
	public, exists := m.agents[name]
	return exists, public, nil
}

func newVisibilityRouter(checker AgentPublicChecker) *chi.Mux {
	mw := NewAgentVisibilityMiddleware(checker)
	r := chi.NewRouter()
	r.With(mw.Middleware).Post("/api/v1/agents/{name}/chat", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return r
}

func requestWithScopes(method, url string, scopes int) *http.Request {
	req := httptest.NewRequest(method, url, nil)
	ctx := context.WithValue(req.Context(), ContextKeyScopes, scopes)
	return req.WithContext(ctx)
}

func TestAgentVisibilityMiddleware(t *testing.T) {
	checker := &mockAgentPublicChecker{
		agents: map[string]bool{
			"public-agent":  true,
			"private-agent": false,
		},
	}
	router := newVisibilityRouter(checker)

	tests := []struct {
		name       string
		agent      string
		scopes     int
		wantStatus int
	}{
		{
			name:       "admin scope bypasses visibility check",
			agent:      "private-agent",
			scopes:     ScopeAdmin,
			wantStatus: http.StatusOK,
		},
		{
			name:       "full chat scope bypasses visibility check",
			agent:      "private-agent",
			scopes:     ScopeChat,
			wantStatus: http.StatusOK,
		},
		{
			name:       "chat public scope allows public agent",
			agent:      "public-agent",
			scopes:     ScopeChatPublic,
			wantStatus: http.StatusOK,
		},
		{
			name:       "chat public scope blocks private agent",
			agent:      "private-agent",
			scopes:     ScopeChatPublic,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "chat public scope returns 404 for unknown agent",
			agent:      "nonexistent",
			scopes:     ScopeChatPublic,
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "no relevant scope returns forbidden",
			agent:      "public-agent",
			scopes:     ScopeAgentsRead,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "combined admin and chat public passes",
			agent:      "private-agent",
			scopes:     ScopeAdmin | ScopeChatPublic,
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := requestWithScopes(http.MethodPost, "/api/v1/agents/"+tt.agent+"/chat", tt.scopes)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, tt.wantStatus, rec.Code)
		})
	}
}

func TestAgentVisibilityMiddleware_CheckerError(t *testing.T) {
	checker := &mockAgentPublicChecker{err: fmt.Errorf("db connection failed")}
	router := newVisibilityRouter(checker)

	req := requestWithScopes(http.MethodPost, "/api/v1/agents/any-agent/chat", ScopeChatPublic)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestAgentHandler_List_PublicFiltering(t *testing.T) {
	agents := []AgentInfo{
		{Name: "public-bot", Description: "Public", ToolsCount: 5, Kit: "crm", Public: true},
		{Name: "private-bot", Description: "Private", ToolsCount: 3, HasKnowledge: true, Public: false},
		{Name: "also-public", Description: "Also public", Public: true},
	}
	handler := NewAgentHandler(&mockAgentLister{agents: agents})
	router := newAgentRouter(handler)

	t.Run("admin sees all agents with all fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/agents", nil)
		ctx := context.WithValue(req.Context(), ContextKeyScopes, ScopeAdmin)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var result []AgentInfo
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
		assert.Len(t, result, 3)
	})

	t.Run("full chat scope sees all agents", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/agents", nil)
		ctx := context.WithValue(req.Context(), ContextKeyScopes, ScopeChat)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var result []AgentInfo
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
		assert.Len(t, result, 3)
	})

	t.Run("chat public scope sees only public agents with limited fields", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/agents", nil)
		ctx := context.WithValue(req.Context(), ContextKeyScopes, ScopeChatPublic)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var result []AgentInfo
		require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
		require.Len(t, result, 2)

		// Only public agents
		assert.Equal(t, "public-bot", result[0].Name)
		assert.Equal(t, "Public", result[0].Description)
		assert.True(t, result[0].Public)
		// Limited fields: no tools_count, kit, has_knowledge
		assert.Equal(t, 0, result[0].ToolsCount)
		assert.Empty(t, result[0].Kit)
		assert.False(t, result[0].HasKnowledge)

		assert.Equal(t, "also-public", result[1].Name)
	})
}

func TestHasFullChatAccess(t *testing.T) {
	tests := []struct {
		name   string
		scopes int
		want   bool
	}{
		{"admin", ScopeAdmin, true},
		{"chat", ScopeChat, true},
		{"chat_public_only", ScopeChatPublic, false},
		{"agents_read", ScopeAgentsRead, false},
		{"admin_and_chat_public", ScopeAdmin | ScopeChatPublic, true},
		{"no_scopes", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasFullChatAccess(tt.scopes))
		})
	}
}
