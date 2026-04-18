package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// mockUserAuthenticator is an in-memory UserAuthenticator for tests.
type mockUserAuthenticator struct {
	users map[string]*models.UserModel // key: "tenantID:username"
	err   error
}

func newMockUserAuthenticator() *mockUserAuthenticator {
	return &mockUserAuthenticator{users: make(map[string]*models.UserModel)}
}

func (m *mockUserAuthenticator) addUser(t *testing.T, tenantID, id, username, password, role string, disabled bool) {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)
	m.users[tenantID+":"+username] = &models.UserModel{
		ID:           id,
		TenantID:     tenantID,
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		Disabled:     disabled,
	}
}

func (m *mockUserAuthenticator) GetByUsername(_ context.Context, tenantID, username string) (*models.UserModel, error) {
	if m.err != nil {
		return nil, m.err
	}
	u, ok := m.users[tenantID+":"+username]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func TestAuthHandler_Login_Valid(t *testing.T) {
	repo := newMockUserAuthenticator()
	repo.addUser(t, DefaultAdminTenantID, "user-uuid-1", "admin", "secret123", "admin", false)
	h := NewAuthHandlerWithRepo(repo, testJWTSecret)

	body := `{"username":"admin","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp loginResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.NotEmpty(t, resp.ExpiresAt)

	// Verify the token is valid and carries user UUID as subject + custom claims.
	claims := &adminClaims{}
	parsed, err := jwt.ParseWithClaims(resp.Token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err)
	assert.True(t, parsed.Valid)
	assert.Equal(t, "user-uuid-1", claims.Subject)
	assert.Equal(t, "admin", claims.Role)
	assert.Equal(t, DefaultAdminTenantID, claims.TenantID)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	repo := newMockUserAuthenticator()
	repo.addUser(t, DefaultAdminTenantID, "user-uuid-1", "admin", "secret123", "admin", false)
	h := NewAuthHandlerWithRepo(repo, testJWTSecret)

	tests := []struct {
		name string
		body string
	}{
		{"wrong password", `{"username":"admin","password":"wrong"}`},
		{"unknown username", `{"username":"hacker","password":"secret123"}`},
		{"both wrong", `{"username":"hacker","password":"wrong"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.Login(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Contains(t, rec.Body.String(), "invalid credentials")
		})
	}
}

func TestAuthHandler_Login_DisabledUser(t *testing.T) {
	repo := newMockUserAuthenticator()
	repo.addUser(t, DefaultAdminTenantID, "user-uuid-d", "disabled-user", "secret123", "admin", true)
	h := NewAuthHandlerWithRepo(repo, testJWTSecret)

	body := `{"username":"disabled-user","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	// Disabled users produce the same uniform 401 (no info leak).
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid credentials")
}

func TestAuthHandler_Login_EmptyFields(t *testing.T) {
	repo := newMockUserAuthenticator()
	h := NewAuthHandlerWithRepo(repo, testJWTSecret)

	tests := []struct {
		name string
		body string
	}{
		{"empty username", `{"username":"","password":"secret123"}`},
		{"empty password", `{"username":"admin","password":""}`},
		{"both empty", `{"username":"","password":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			h.Login(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "username and password required")
		})
	}
}

func TestAuthHandler_Login_InvalidJSON(t *testing.T) {
	repo := newMockUserAuthenticator()
	h := NewAuthHandlerWithRepo(repo, testJWTSecret)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}
