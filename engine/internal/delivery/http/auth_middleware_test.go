package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "test-secret-key-for-unit-tests"

type mockTokenVerifier struct {
	tokens map[string]struct {
		name   string
		scopes int
	}
}

func newMockTokenVerifier() *mockTokenVerifier {
	return &mockTokenVerifier{
		tokens: make(map[string]struct {
			name   string
			scopes int
		}),
	}
}

func (m *mockTokenVerifier) addToken(rawToken string, name string, scopes int) {
	hash := sha256Hash(rawToken)
	m.tokens[hash] = struct {
		name   string
		scopes int
	}{name: name, scopes: scopes}
}

func (m *mockTokenVerifier) VerifyToken(_ context.Context, tokenHash string) (string, int, error) {
	t, ok := m.tokens[tokenHash]
	if !ok {
		return "", 0, fmt.Errorf("token not found")
	}
	return t.name, t.scopes, nil
}

func generateTestJWT(subject, secret string, expiry time.Duration) string {
	claims := &jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

func TestAuthMiddleware_NoAuthHeader(t *testing.T) {
	verifier := newMockTokenVerifier()
	mw := NewAuthMiddleware(testJWTSecret, verifier)

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "unauthorized")
}

func TestAuthMiddleware_InvalidBearerFormat(t *testing.T) {
	verifier := newMockTokenVerifier()
	mw := NewAuthMiddleware(testJWTSecret, verifier)

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic abc123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ValidJWT(t *testing.T) {
	verifier := newMockTokenVerifier()
	mw := NewAuthMiddleware(testJWTSecret, verifier)

	var capturedActorType, capturedActorID string
	var capturedScopes int

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedActorType, _ = r.Context().Value(ContextKeyActorType).(string)
		capturedActorID, _ = r.Context().Value(ContextKeyActorID).(string)
		capturedScopes, _ = r.Context().Value(ContextKeyScopes).(int)
		w.WriteHeader(http.StatusOK)
	}))

	token := generateTestJWT("admin-user", testJWTSecret, time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "admin", capturedActorType)
	assert.Equal(t, "admin-user", capturedActorID)
	assert.Equal(t, ScopeAdmin, capturedScopes)
}

func TestAuthMiddleware_ExpiredJWT(t *testing.T) {
	verifier := newMockTokenVerifier()
	mw := NewAuthMiddleware(testJWTSecret, verifier)

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token := generateTestJWT("admin-user", testJWTSecret, -time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid token")
}

func TestAuthMiddleware_WrongSecretJWT(t *testing.T) {
	verifier := newMockTokenVerifier()
	mw := NewAuthMiddleware(testJWTSecret, verifier)

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token := generateTestJWT("admin-user", "wrong-secret", time.Hour)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAuthMiddleware_ValidAPIToken(t *testing.T) {
	verifier := newMockTokenVerifier()
	rawToken := "bb_abc123def456"
	verifier.addToken(rawToken, "my-cli-token", ScopeChat|ScopeTasks)

	mw := NewAuthMiddleware(testJWTSecret, verifier)

	var capturedActorType, capturedActorID string
	var capturedScopes int

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedActorType, _ = r.Context().Value(ContextKeyActorType).(string)
		capturedActorID, _ = r.Context().Value(ContextKeyActorID).(string)
		capturedScopes, _ = r.Context().Value(ContextKeyScopes).(int)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "api_token", capturedActorType)
	assert.Equal(t, "my-cli-token", capturedActorID)
	assert.Equal(t, ScopeChat|ScopeTasks, capturedScopes)
}

func TestAuthMiddleware_InvalidAPIToken(t *testing.T) {
	verifier := newMockTokenVerifier()
	mw := NewAuthMiddleware(testJWTSecret, verifier)

	handler := mw.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer bb_unknown_token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid api token")
}

func TestRequireScope_Allowed(t *testing.T) {
	tests := []struct {
		name        string
		scopes      int
		required    int
		expectAllow bool
	}{
		{"admin has all", ScopeAdmin, ScopeChat, true},
		{"admin bypasses agents_write", ScopeAdmin, ScopeAgentsWrite, true},
		{"admin bypasses models_write", ScopeAdmin, ScopeModelsWrite, true},
		{"admin bypasses mcp_write", ScopeAdmin, ScopeMCPWrite, true},
		{"exact scope", ScopeChat, ScopeChat, true},
		{"multiple scopes", ScopeChat | ScopeTasks, ScopeTasks, true},
		{"missing scope", ScopeChat, ScopeTasks, false},
		{"no scopes", 0, ScopeChat, false},
		{"agents_read allows agents_read", ScopeAgentsRead, ScopeAgentsRead, true},
		{"agents_read denies agents_write", ScopeAgentsRead, ScopeAgentsWrite, false},
		{"agents_write allows agents_write", ScopeAgentsWrite, ScopeAgentsWrite, true},
		{"models_read allows models_read", ScopeModelsRead, ScopeModelsRead, true},
		{"models_read denies models_write", ScopeModelsRead, ScopeModelsWrite, false},
		{"models_write allows models_write", ScopeModelsWrite, ScopeModelsWrite, true},
		{"mcp_read allows mcp_read", ScopeMCPRead, ScopeMCPRead, true},
		{"mcp_read denies mcp_write", ScopeMCPRead, ScopeMCPWrite, false},
		{"mcp_write allows mcp_write", ScopeMCPWrite, ScopeMCPWrite, true},
		{"triggers_read allows triggers_read", ScopeTriggersRead, ScopeTriggersRead, true},
		{"triggers_read denies triggers_write", ScopeTriggersRead, ScopeTriggersWrite, false},
		{"triggers_write allows triggers_write", ScopeTriggersWrite, ScopeTriggersWrite, true},
		{"combined read scopes", ScopeAgentsRead | ScopeModelsRead | ScopeMCPRead, ScopeModelsRead, true},
		{"combined read denies write", ScopeAgentsRead | ScopeModelsRead, ScopeAgentsWrite, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			handler := RequireScope(tt.required)(inner)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx := context.WithValue(req.Context(), ContextKeyScopes, tt.scopes)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if tt.expectAllow {
				assert.True(t, called)
				assert.Equal(t, http.StatusOK, rec.Code)
			} else {
				assert.False(t, called)
				assert.Equal(t, http.StatusForbidden, rec.Code)
			}
		})
	}
}

func TestRequireAdminSession(t *testing.T) {
	tests := []struct {
		name        string
		actorType   string
		expectAllow bool
	}{
		{"admin allowed", "admin", true},
		{"api_token denied", "api_token", false},
		{"empty denied", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			handler := RequireAdminSession(inner)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			ctx := context.WithValue(req.Context(), ContextKeyActorType, tt.actorType)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if tt.expectAllow {
				assert.True(t, called)
			} else {
				assert.False(t, called)
				assert.Equal(t, http.StatusForbidden, rec.Code)
			}
		})
	}
}

func TestSha256Hash(t *testing.T) {
	hash := sha256Hash("bb_test123")
	require.NotEmpty(t, hash)
	assert.Len(t, hash, 64) // SHA-256 hex = 64 chars

	// Same input produces same hash
	assert.Equal(t, hash, sha256Hash("bb_test123"))

	// Different input produces different hash
	assert.NotEqual(t, hash, sha256Hash("bb_test456"))
}
