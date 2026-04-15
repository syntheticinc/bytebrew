package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBYOKMiddleware_NoHeaders(t *testing.T) {
	mw := NewBYOKMiddleware(BYOKConfig{Enabled: true, AllowedProviders: []string{"openai"}})

	var called bool
	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		// No BYOK context should be set
		provider, _ := r.Context().Value(ContextKeyBYOKProvider).(string)
		assert.Empty(t, provider)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestBYOKMiddleware_Disabled(t *testing.T) {
	mw := NewBYOKMiddleware(BYOKConfig{Enabled: false})

	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-BYOK-Provider", "openai")
	req.Header.Set("X-BYOK-API-Key", "sk-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "BYOK is disabled")
}

func TestBYOKMiddleware_ValidHeaders(t *testing.T) {
	mw := NewBYOKMiddleware(BYOKConfig{
		Enabled:          true,
		AllowedProviders: []string{"openai", "anthropic"},
	})

	var capturedProvider, capturedKey, capturedModel, capturedBaseURL string
	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedProvider, _ = r.Context().Value(ContextKeyBYOKProvider).(string)
		capturedKey, _ = r.Context().Value(ContextKeyBYOKAPIKey).(string)
		capturedModel, _ = r.Context().Value(ContextKeyBYOKModel).(string)
		capturedBaseURL, _ = r.Context().Value(ContextKeyBYOKBaseURL).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-BYOK-Provider", "OpenAI")
	req.Header.Set("X-BYOK-API-Key", "sk-test-key")
	req.Header.Set("X-BYOK-Model", "gpt-4o")
	req.Header.Set("X-BYOK-Base-URL", "https://example.com/v1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "openai", capturedProvider) // lowercased
	assert.Equal(t, "sk-test-key", capturedKey)
	assert.Equal(t, "gpt-4o", capturedModel)
	assert.Equal(t, "https://example.com/v1", capturedBaseURL)
}

// LegacyHeaders verifies that pre-V2 X-Model-* headers still work as a
// transitional fallback for clients that have not migrated to the
// canonical X-BYOK-* names yet.
func TestBYOKMiddleware_LegacyHeaders(t *testing.T) {
	mw := NewBYOKMiddleware(BYOKConfig{
		Enabled:          true,
		AllowedProviders: []string{"openai"},
	})

	var capturedProvider, capturedKey, capturedModel string
	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedProvider, _ = r.Context().Value(ContextKeyBYOKProvider).(string)
		capturedKey, _ = r.Context().Value(ContextKeyBYOKAPIKey).(string)
		capturedModel, _ = r.Context().Value(ContextKeyBYOKModel).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Model-Provider", "openai")
	req.Header.Set("X-Model-API-Key", "sk-legacy")
	req.Header.Set("X-Model-Name", "gpt-4o")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "openai", capturedProvider)
	assert.Equal(t, "sk-legacy", capturedKey)
	assert.Equal(t, "gpt-4o", capturedModel)
}

func TestBYOKMiddleware_ProviderNotAllowed(t *testing.T) {
	mw := NewBYOKMiddleware(BYOKConfig{
		Enabled:          true,
		AllowedProviders: []string{"openai"},
	})

	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-BYOK-Provider", "anthropic")
	req.Header.Set("X-BYOK-API-Key", "sk-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "provider not allowed")
}

func TestBYOKMiddleware_MissingAPIKey(t *testing.T) {
	mw := NewBYOKMiddleware(BYOKConfig{
		Enabled:          true,
		AllowedProviders: []string{"openai"},
	})

	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-BYOK-Provider", "openai")
	// No X-BYOK-API-Key
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "X-BYOK-Provider and X-BYOK-API-Key are required")
}

func TestBYOKMiddleware_NoModelName(t *testing.T) {
	mw := NewBYOKMiddleware(BYOKConfig{
		Enabled:          true,
		AllowedProviders: []string{"openai"},
	})

	var capturedModel string
	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedModel, _ = r.Context().Value(ContextKeyBYOKModel).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-BYOK-Provider", "openai")
	req.Header.Set("X-BYOK-API-Key", "sk-123")
	// No X-BYOK-Model
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, capturedModel)
}

func TestBYOKMiddleware_AllProvidersAllowed(t *testing.T) {
	// Empty AllowedProviders = all providers allowed
	mw := NewBYOKMiddleware(BYOKConfig{
		Enabled:          true,
		AllowedProviders: nil,
	})

	var capturedProvider string
	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedProvider, _ = r.Context().Value(ContextKeyBYOKProvider).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-BYOK-Provider", "anything")
	req.Header.Set("X-BYOK-API-Key", "sk-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "anything", capturedProvider)
}

// SetConfig hot-swaps the active BYOK config so admin UI changes take
// effect without a restart. Verifies that a request flipped between
// enabled/disabled hits the new branch immediately.
func TestBYOKMiddleware_SetConfig(t *testing.T) {
	mw := NewBYOKMiddleware(BYOKConfig{Enabled: true, AllowedProviders: []string{"openai"}})

	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	makeReq := func() *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-BYOK-Provider", "openai")
		req.Header.Set("X-BYOK-API-Key", "sk-1")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	assert.Equal(t, http.StatusOK, makeReq().Code)

	// Flip to disabled — next request must be rejected.
	mw.SetConfig(BYOKConfig{Enabled: false})
	rec := makeReq()
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "BYOK is disabled")

	// Flip back to enabled with a stricter allowlist — passes again.
	mw.SetConfig(BYOKConfig{Enabled: true, AllowedProviders: []string{"openai"}})
	assert.Equal(t, http.StatusOK, makeReq().Code)

	// Flip allowlist to anthropic only — openai key must now be rejected.
	mw.SetConfig(BYOKConfig{Enabled: true, AllowedProviders: []string{"anthropic"}})
	rec = makeReq()
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "provider not allowed")
}
