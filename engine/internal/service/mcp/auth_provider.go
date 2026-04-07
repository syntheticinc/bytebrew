package mcp

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// AuthProvider applies authentication to MCP server HTTP requests
// based on the server's auth configuration.
type AuthProvider struct{}

// NewAuthProvider creates a new MCP auth provider.
func NewAuthProvider() *AuthProvider {
	return &AuthProvider{}
}

// ApplyAuth applies the configured auth to an HTTP request for an MCP server call.
// incomingHeaders are the headers from the original user request (for forward_headers).
func (p *AuthProvider) ApplyAuth(req *http.Request, config domain.MCPAuthConfig, incomingHeaders map[string]string) error {
	switch config.Type {
	case domain.MCPAuthNone:
		return nil

	case domain.MCPAuthAPIKey:
		// AC-AUTH-03: API key read from env variable, not stored in plain text
		key := os.Getenv(config.KeyEnv)
		if key == "" {
			return fmt.Errorf("env var %s not set for MCP auth", config.KeyEnv)
		}
		req.Header.Set("Authorization", "Bearer "+key)
		return nil

	case domain.MCPAuthForwardHeaders:
		// AC-AUTH-02: forward_headers proxied from incoming request
		for name, value := range incomingHeaders {
			req.Header.Set(name, value)
		}
		return nil

	case domain.MCPAuthServiceAccount:
		token := os.Getenv(config.TokenEnv)
		if token == "" {
			return fmt.Errorf("env var %s not set for MCP service account auth", config.TokenEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil

	case domain.MCPAuthOAuth2:
		// OAuth2 token refresh is complex; for V2, treat like service_account
		// with the token stored in TokenEnv after external refresh.
		slog.Warn("[MCPAuth] OAuth2 using static token — full refresh not yet implemented")
		token := os.Getenv(config.TokenEnv)
		if token == "" {
			return fmt.Errorf("oauth2 token env var %s not set", config.TokenEnv)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		return nil

	default:
		return fmt.Errorf("unsupported MCP auth type: %s", config.Type)
	}
}
