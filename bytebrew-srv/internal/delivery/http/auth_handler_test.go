package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandler_Login_Valid(t *testing.T) {
	h := NewAuthHandler("admin", "secret123", testJWTSecret)

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

	// Verify the token is valid
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(resp.Token, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err)
	assert.True(t, parsed.Valid)
	assert.Equal(t, "admin", claims.Subject)
}

func TestAuthHandler_Login_InvalidCredentials(t *testing.T) {
	h := NewAuthHandler("admin", "secret123", testJWTSecret)

	tests := []struct {
		name string
		body string
	}{
		{"wrong password", `{"username":"admin","password":"wrong"}`},
		{"wrong username", `{"username":"hacker","password":"secret123"}`},
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

func TestAuthHandler_Login_EmptyFields(t *testing.T) {
	h := NewAuthHandler("admin", "secret123", testJWTSecret)

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
	h := NewAuthHandler("admin", "secret123", testJWTSecret)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid request body")
}
