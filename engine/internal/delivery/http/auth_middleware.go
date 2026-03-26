package http

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const (
	// ContextKeyActorType holds the actor type: "admin" or "api_token".
	ContextKeyActorType contextKey = "actor_type"
	// ContextKeyActorID holds the actor identifier (subject for JWT, name for API token).
	ContextKeyActorID contextKey = "actor_id"
	// ContextKeyScopes holds the bitmask of allowed scopes.
	ContextKeyScopes contextKey = "scopes"
)

// Scope bitmask constants matching ERD api_tokens.scopes_mask.
const (
	ScopeChat          = 1
	ScopeTasks         = 2
	ScopeAgentsRead    = 4
	ScopeConfig        = 8
	ScopeAdmin         = 16
	ScopeAgentsWrite   = 32
	ScopeModelsRead    = 64
	ScopeModelsWrite   = 128
	ScopeMCPRead       = 256
	ScopeMCPWrite      = 512
	ScopeTriggersRead  = 1024
	ScopeTriggersWrite = 2048
)

// APITokenVerifier looks up API tokens by their SHA-256 hash.
type APITokenVerifier interface {
	VerifyToken(ctx context.Context, tokenHash string) (name string, scopesMask int, err error)
}

// AuthMiddleware handles dual authentication: admin session JWT and API tokens (bb_ prefix).
type AuthMiddleware struct {
	jwtSecret     []byte
	tokenVerifier APITokenVerifier
}

// NewAuthMiddleware creates a new AuthMiddleware.
func NewAuthMiddleware(jwtSecret string, tokenVerifier APITokenVerifier) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret:     []byte(jwtSecret),
		tokenVerifier: tokenVerifier,
	}
}

// Authenticate is the middleware handler that validates Bearer tokens.
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		token := strings.TrimPrefix(authHeader, "Bearer ")

		if strings.HasPrefix(token, "bb_") {
			m.authenticateAPIToken(w, r, next, token)
			return
		}

		m.authenticateJWT(w, r, next, token)
	})
}

func (m *AuthMiddleware) authenticateAPIToken(w http.ResponseWriter, r *http.Request, next http.Handler, token string) {
	hash := sha256Hash(token)
	name, scopes, err := m.tokenVerifier.VerifyToken(r.Context(), hash)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api token"})
		return
	}
	ctx := context.WithValue(r.Context(), ContextKeyActorType, "api_token")
	ctx = context.WithValue(ctx, ContextKeyActorID, name)
	ctx = context.WithValue(ctx, ContextKeyScopes, scopes)
	next.ServeHTTP(w, r.WithContext(ctx))
}

func (m *AuthMiddleware) authenticateJWT(w http.ResponseWriter, r *http.Request, next http.Handler, token string) {
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return m.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}
	ctx := context.WithValue(r.Context(), ContextKeyActorType, "admin")
	ctx = context.WithValue(ctx, ContextKeyActorID, claims.Subject)
	ctx = context.WithValue(ctx, ContextKeyScopes, ScopeAdmin)
	next.ServeHTTP(w, r.WithContext(ctx))
}

// RequireScope returns middleware that checks the authenticated user has the required scope.
func RequireScope(scope int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scopes, _ := r.Context().Value(ContextKeyScopes).(int)
			if scopes&ScopeAdmin != 0 || scopes&scope != 0 {
				next.ServeHTTP(w, r)
				return
			}
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		})
	}
}

// RequireAdminSession ensures only admin JWT (not API token) can access.
func RequireAdminSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actorType, _ := r.Context().Value(ContextKeyActorType).(string)
		if actorType != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin session required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
