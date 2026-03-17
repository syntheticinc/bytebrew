package http

import (
	"context"
	"net/http"
	"strings"
)

const (
	// ContextKeyBYOKProvider holds the provider name from X-Model-Provider header.
	ContextKeyBYOKProvider contextKey = "byok_provider"
	// ContextKeyBYOKAPIKey holds the API key from X-Model-API-Key header.
	ContextKeyBYOKAPIKey contextKey = "byok_api_key"
	// ContextKeyBYOKModel holds the model name from X-Model-Name header.
	ContextKeyBYOKModel contextKey = "byok_model"
)

// BYOKConfig holds BYOK middleware configuration.
type BYOKConfig struct {
	Enabled          bool
	AllowedProviders []string // e.g. ["openai", "anthropic", "openrouter"]
}

// BYOKMiddleware parses BYOK headers and injects them into request context.
type BYOKMiddleware struct {
	config           BYOKConfig
	allowedProviders map[string]struct{}
}

// NewBYOKMiddleware creates a new BYOKMiddleware.
func NewBYOKMiddleware(cfg BYOKConfig) *BYOKMiddleware {
	allowed := make(map[string]struct{}, len(cfg.AllowedProviders))
	for _, p := range cfg.AllowedProviders {
		allowed[strings.ToLower(p)] = struct{}{}
	}
	return &BYOKMiddleware{
		config:           cfg,
		allowedProviders: allowed,
	}
}

// InjectBYOK is middleware that reads BYOK headers and adds them to context.
// If BYOK is disabled or headers are absent, the request passes through unchanged.
func (m *BYOKMiddleware) InjectBYOK(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provider := r.Header.Get("X-Model-Provider")
		apiKey := r.Header.Get("X-Model-API-Key")
		modelName := r.Header.Get("X-Model-Name")

		// No BYOK headers present — pass through.
		if provider == "" && apiKey == "" && modelName == "" {
			next.ServeHTTP(w, r)
			return
		}

		if !m.config.Enabled {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "BYOK is disabled"})
			return
		}

		if provider == "" || apiKey == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "X-Model-Provider and X-Model-API-Key are required for BYOK"})
			return
		}

		providerLower := strings.ToLower(provider)
		if len(m.allowedProviders) > 0 {
			if _, ok := m.allowedProviders[providerLower]; !ok {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "provider not allowed: " + provider})
				return
			}
		}

		ctx := context.WithValue(r.Context(), ContextKeyBYOKProvider, providerLower)
		ctx = context.WithValue(ctx, ContextKeyBYOKAPIKey, apiKey)
		if modelName != "" {
			ctx = context.WithValue(ctx, ContextKeyBYOKModel, modelName)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
