package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/llm"
)

// fakeChatService is a minimal ChatService that captures the context it
// was called with so the test can assert that BYOK credentials were
// propagated end-to-end.
type fakeChatService struct {
	gotCreds *llm.BYOKCredentials
	gotAgent string
}

func (f *fakeChatService) Chat(ctx context.Context, agentName, _, _, _ string) (<-chan SSEEvent, error) {
	f.gotAgent = agentName
	if c := llm.BYOKCredentialsFrom(ctx); c != nil {
		// Copy so the test sees a stable value even if the caller mutates.
		f.gotCreds = &llm.BYOKCredentials{
			Provider: c.Provider,
			APIKey:   c.APIKey,
			Model:    c.Model,
			BaseURL:  c.BaseURL,
		}
	}
	// Return immediately closed channel — handler must not hang.
	ch := make(chan SSEEvent)
	close(ch)
	return ch, nil
}

// fakeForwardHeaders returns no header forwarding — focus is BYOK only.
func fakeForwardHeaders() []string { return nil }

// TestChatHandler_BYOKHeaders_ReachServiceContext asserts the V2 §5.8
// integration contract: a chat request carrying X-BYOK-* headers passes
// through BYOKMiddleware into the chat handler, which lifts the values
// into llm.BYOKCredentials so the downstream factory can build an
// ad-hoc per-end-user ChatModel.
func TestChatHandler_BYOKHeaders_ReachServiceContext(t *testing.T) {
	svc := &fakeChatService{}
	handler := NewChatHandler(svc, nil, fakeForwardHeaders)

	mw := NewBYOKMiddleware(BYOKConfig{
		Enabled:          true,
		AllowedProviders: []string{"openai", "anthropic", "openai_compatible"},
	})

	// Wire the route the same way server.go does — BYOK middleware
	// AFTER auth (auth is a no-op in this test) and BEFORE the handler.
	r := chi.NewRouter()
	r.Use(mw.InjectBYOK)
	r.Post("/api/v1/agents/{name}/chat", handler.Chat)

	body := strings.NewReader(`{"message":"hi","stream":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/my-agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BYOK-Provider", "openai_compatible")
	req.Header.Set("X-BYOK-API-Key", "sk-byok-secret")
	req.Header.Set("X-BYOK-Model", "gpt-4o-mini")
	req.Header.Set("X-BYOK-Base-URL", "https://example.com/v1")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())
	assert.Equal(t, "my-agent", svc.gotAgent)
	require.NotNil(t, svc.gotCreds, "BYOK credentials must be attached to chat service ctx")
	assert.Equal(t, "openai_compatible", svc.gotCreds.Provider)
	assert.Equal(t, "sk-byok-secret", svc.gotCreds.APIKey)
	assert.Equal(t, "gpt-4o-mini", svc.gotCreds.Model)
	assert.Equal(t, "https://example.com/v1", svc.gotCreds.BaseURL)
}

// TestChatHandler_DisallowedProvider_Returns403 covers the negative path:
// allowed_providers=["openai"], request comes with provider=anthropic →
// the middleware short-circuits before the chat handler ever runs.
func TestChatHandler_DisallowedProvider_Returns403(t *testing.T) {
	svc := &fakeChatService{}
	handler := NewChatHandler(svc, nil, fakeForwardHeaders)
	mw := NewBYOKMiddleware(BYOKConfig{
		Enabled:          true,
		AllowedProviders: []string{"openai"},
	})

	r := chi.NewRouter()
	r.Use(mw.InjectBYOK)
	r.Post("/api/v1/agents/{name}/chat", handler.Chat)

	body := strings.NewReader(`{"message":"hi"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/my-agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BYOK-Provider", "anthropic")
	req.Header.Set("X-BYOK-API-Key", "sk-x")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "provider not allowed")
	assert.Nil(t, svc.gotCreds, "service must not be reached when middleware rejects")

	// Sanity: the body is JSON.
	var parsed map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &parsed))
	assert.Contains(t, parsed["error"], "anthropic")
}

// TestChatHandler_MissingKey_Returns400 covers the negative path: BYOK
// enabled, provider header set, but no API key — must reject with 400
// (V2 §5.8 "missing key when required").
func TestChatHandler_MissingKey_Returns400(t *testing.T) {
	svc := &fakeChatService{}
	handler := NewChatHandler(svc, nil, fakeForwardHeaders)
	mw := NewBYOKMiddleware(BYOKConfig{
		Enabled:          true,
		AllowedProviders: []string{"openai"},
	})

	r := chi.NewRouter()
	r.Use(mw.InjectBYOK)
	r.Post("/api/v1/agents/{name}/chat", handler.Chat)

	body := strings.NewReader(`{"message":"hi"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/my-agent/chat", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BYOK-Provider", "openai")
	// No X-BYOK-API-Key

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "X-BYOK-API-Key")
	assert.Nil(t, svc.gotCreds, "service must not be reached when middleware rejects")
}

// TestChatHandler_NoBYOK_TenantConfigPath asserts that requests without
// any BYOK headers fall through to the chat service with no credentials
// attached — the tenant-configured model must remain in effect.
func TestChatHandler_NoBYOK_TenantConfigPath(t *testing.T) {
	svc := &fakeChatService{}
	handler := NewChatHandler(svc, nil, fakeForwardHeaders)
	mw := NewBYOKMiddleware(BYOKConfig{Enabled: true, AllowedProviders: []string{"openai"}})

	r := chi.NewRouter()
	r.Use(mw.InjectBYOK)
	r.Post("/api/v1/agents/{name}/chat", handler.Chat)

	body := strings.NewReader(`{"message":"hi","stream":false}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/agents/my-agent/chat", body)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "my-agent", svc.gotAgent)
	assert.Nil(t, svc.gotCreds, "no BYOK headers ⇒ no creds attached ⇒ tenant model used")
}
