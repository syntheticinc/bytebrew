package http

import (
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// AuthHandler handles authentication endpoints (login).
type AuthHandler struct {
	adminUser     string
	adminPassword string
	jwtSecret     []byte
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(adminUser, adminPassword, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		adminUser:     adminUser,
		adminPassword: adminPassword,
		jwtSecret:     []byte(jwtSecret),
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username and password required"})
		return
	}

	if req.Username != h.adminUser || req.Password != h.adminPassword {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	claims := &jwt.RegisteredClaims{
		Subject:   req.Username,
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(h.jwtSecret)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "generate token failed"})
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Token:     tokenString,
		ExpiresAt: expiresAt.Format(time.RFC3339),
	})
}
