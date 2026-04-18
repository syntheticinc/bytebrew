package http

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// UserAuthenticator looks up a user by tenant + username for login.
// Consumer-side interface — the http package owns the shape it needs.
type UserAuthenticator interface {
	GetByUsername(ctx context.Context, tenantID, username string) (*models.UserModel, error)
}

// AuthHandler handles authentication endpoints (login).
// Credentials are verified against the `users` table (bcrypt password_hash).
// Admin/system users must be created out-of-band via the `ce admin` CLI.
type AuthHandler struct {
	repo      UserAuthenticator
	jwtSecret []byte
	tenantID  string
}

// DefaultAdminTenantID is the single-tenant default for CE installations.
const DefaultAdminTenantID = "00000000-0000-0000-0000-000000000001"

// NewAuthHandler creates a new AuthHandler backed by the users table.
// Pass *gorm.DB directly — the handler wraps it in a minimal repo adapter.
func NewAuthHandler(db *gorm.DB, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		repo:      &gormUserAuthenticator{db: db},
		jwtSecret: []byte(jwtSecret),
		tenantID:  DefaultAdminTenantID,
	}
}

// NewAuthHandlerWithRepo creates a new AuthHandler with an injected authenticator.
// Useful for tests.
func NewAuthHandlerWithRepo(repo UserAuthenticator, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
		tenantID:  DefaultAdminTenantID,
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

// adminClaims carries the role + tenant alongside standard registered claims.
type adminClaims struct {
	Role     string `json:"role"`
	TenantID string `json:"tenant_id"`
	jwt.RegisteredClaims
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

	user, err := h.repo.GetByUsername(r.Context(), h.tenantID, req.Username)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "authentication failed"})
		return
	}
	// Uniform error — do not reveal whether the user exists.
	if user == nil || user.Disabled {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
		return
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	claims := adminClaims{
		Role:     user.Role,
		TenantID: user.TenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
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

// gormUserAuthenticator is the default DB-backed implementation of UserAuthenticator.
// Defined here (not in configrepo) to keep the AuthHandler constructor simple —
// the handler accepts *gorm.DB and wraps it internally.
type gormUserAuthenticator struct {
	db *gorm.DB
}

func (a *gormUserAuthenticator) GetByUsername(ctx context.Context, tenantID, username string) (*models.UserModel, error) {
	var user models.UserModel
	err := a.db.WithContext(ctx).
		Where("tenant_id = ? AND username = ?", tenantID, username).
		First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}
