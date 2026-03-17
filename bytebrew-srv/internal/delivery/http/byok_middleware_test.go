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
	req.Header.Set("X-Model-Provider", "openai")
	req.Header.Set("X-Model-API-Key", "sk-123")
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

	var capturedProvider, capturedKey, capturedModel string
	handler := mw.InjectBYOK(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedProvider, _ = r.Context().Value(ContextKeyBYOKProvider).(string)
		capturedKey, _ = r.Context().Value(ContextKeyBYOKAPIKey).(string)
		capturedModel, _ = r.Context().Value(ContextKeyBYOKModel).(string)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Model-Provider", "OpenAI")
	req.Header.Set("X-Model-API-Key", "sk-test-key")
	req.Header.Set("X-Model-Name", "gpt-4o")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "openai", capturedProvider) // lowercased
	assert.Equal(t, "sk-test-key", capturedKey)
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
	req.Header.Set("X-Model-Provider", "anthropic")
	req.Header.Set("X-Model-API-Key", "sk-123")
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
	req.Header.Set("X-Model-Provider", "openai")
	// No X-Model-API-Key
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "X-Model-Provider and X-Model-API-Key are required")
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
	req.Header.Set("X-Model-Provider", "openai")
	req.Header.Set("X-Model-API-Key", "sk-123")
	// No X-Model-Name
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
	req.Header.Set("X-Model-Provider", "anything")
	req.Header.Set("X-Model-API-Key", "sk-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "anything", capturedProvider)
}
