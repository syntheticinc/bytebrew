package http

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"
)

const (
	// ContextKeyBYOKProvider holds the provider name from the BYOK provider header.
	ContextKeyBYOKProvider contextKey = "byok_provider"
	// ContextKeyBYOKAPIKey holds the API key from the BYOK API key header.
	ContextKeyBYOKAPIKey contextKey = "byok_api_key"
	// ContextKeyBYOKModel holds the model name from the BYOK model header.
	ContextKeyBYOKModel contextKey = "byok_model"
	// ContextKeyBYOKBaseURL holds the optional base URL override from the BYOK base-URL header.
	ContextKeyBYOKBaseURL contextKey = "byok_base_url"
)

// BYOK request header names (V2 §5.8). The legacy `X-Model-*` names are
// honoured as fallback so existing clients keep working during the
// transition; new clients should use the canonical `X-BYOK-*` names.
const (
	headerBYOKProvider = "X-BYOK-Provider"
	headerBYOKAPIKey   = "X-BYOK-API-Key"
	headerBYOKModel    = "X-BYOK-Model"
	headerBYOKBaseURL  = "X-BYOK-Base-URL"

	legacyHeaderBYOKProvider = "X-Model-Provider"
	legacyHeaderBYOKAPIKey   = "X-Model-API-Key"
	legacyHeaderBYOKModel    = "X-Model-Name"
)

// BYOKConfig holds BYOK middleware configuration. Loaded from the
// `settings` table at startup and refreshed via SetConfig when the admin
// updates the toggles, so middleware behaviour follows the live config
// without a restart (V2 §5.8).
type BYOKConfig struct {
	Enabled          bool
	AllowedProviders []string // e.g. ["openai", "anthropic", "openrouter"]
}

// byokState is the immutable snapshot stored in the atomic.Value.
// We swap a fresh pointer on each SetConfig — readers see a consistent
// view without a mutex.
type byokState struct {
	enabled          bool
	allowedProviders map[string]struct{}
}

func newByokState(cfg BYOKConfig) *byokState {
	allowed := make(map[string]struct{}, len(cfg.AllowedProviders))
	for _, p := range cfg.AllowedProviders {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		allowed[strings.ToLower(p)] = struct{}{}
	}
	return &byokState{
		enabled:          cfg.Enabled,
		allowedProviders: allowed,
	}
}

// BYOKMiddleware parses BYOK headers and injects them into request context.
// Configuration is held atomically and can be hot-swapped via SetConfig
// (admin toggles → settings table → middleware refresh) without a restart.
type BYOKMiddleware struct {
	state atomic.Pointer[byokState]
}

// NewBYOKMiddleware creates a new BYOKMiddleware seeded with cfg.
func NewBYOKMiddleware(cfg BYOKConfig) *BYOKMiddleware {
	m := &BYOKMiddleware{}
	m.state.Store(newByokState(cfg))
	return m
}

// SetConfig atomically replaces the active configuration. Safe to call
// from any goroutine; in-flight requests keep their snapshot.
func (m *BYOKMiddleware) SetConfig(cfg BYOKConfig) {
	m.state.Store(newByokState(cfg))
}

// firstNonEmpty returns the first non-empty header value, looking at the
// canonical name first then the legacy fallback.
func firstNonEmpty(r *http.Request, names ...string) string {
	for _, n := range names {
		if v := r.Header.Get(n); v != "" {
			return v
		}
	}
	return ""
}

// InjectBYOK is middleware that reads BYOK headers and adds them to context.
// If BYOK is disabled or headers are absent, the request passes through unchanged.
func (m *BYOKMiddleware) InjectBYOK(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider := firstNonEmpty(r, headerBYOKProvider, legacyHeaderBYOKProvider)
		apiKey := firstNonEmpty(r, headerBYOKAPIKey, legacyHeaderBYOKAPIKey)
		modelName := firstNonEmpty(r, headerBYOKModel, legacyHeaderBYOKModel)
		baseURL := firstNonEmpty(r, headerBYOKBaseURL)

		// No BYOK headers present — pass through.
		if provider == "" && apiKey == "" && modelName == "" && baseURL == "" {
			next.ServeHTTP(w, r)
			return
		}

		state := m.state.Load()
		if state == nil || !state.enabled {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "BYOK is disabled"})
			return
		}

		if provider == "" || apiKey == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "X-BYOK-Provider and X-BYOK-API-Key are required for BYOK"})
			return
		}

		providerLower := strings.ToLower(provider)
		if len(state.allowedProviders) > 0 {
			if _, ok := state.allowedProviders[providerLower]; !ok {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "provider not allowed: " + provider})
				return
			}
		}

		ctx := context.WithValue(r.Context(), ContextKeyBYOKProvider, providerLower)
		ctx = context.WithValue(ctx, ContextKeyBYOKAPIKey, apiKey)
		if modelName != "" {
			ctx = context.WithValue(ctx, ContextKeyBYOKModel, modelName)
		}
		if baseURL != "" {
			ctx = context.WithValue(ctx, ContextKeyBYOKBaseURL, baseURL)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
