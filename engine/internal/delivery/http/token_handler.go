package http

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// TokenRepository manages API tokens in the database.
type TokenRepository interface {
	Create(ctx context.Context, userSub, name, tokenHash string, scopesMask int) (id string, err error)
	List(ctx context.Context) ([]TokenInfo, error)
	Delete(ctx context.Context, id string) error
}

// TokenInfo is a token record returned by List (no raw token value).
type TokenInfo struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	ScopesMask int        `json:"scopes_mask"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
}

// TokenHandler handles API token CRUD endpoints.
type TokenHandler struct {
	repo TokenRepository
}

// NewTokenHandler creates a new TokenHandler.
func NewTokenHandler(repo TokenRepository) *TokenHandler {
	return &TokenHandler{repo: repo}
}

type createTokenRequest struct {
	Name       string `json:"name"`
	ScopesMask int    `json:"scopes_mask"`
}

type createTokenResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// CreateToken handles POST /auth/tokens.
func (h *TokenHandler) CreateToken(w http.ResponseWriter, r *http.Request) {
	var req createTokenRequest
	if err := readJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}

	rawToken, err := generateAPIToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "generate token failed"})
		return
	}

	userSub := domain.UserSubFromContext(r.Context())
	hash := sha256Hash(rawToken)
	id, err := h.repo.Create(r.Context(), userSub, req.Name, hash, req.ScopesMask)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("create token: %s", err)})
		return
	}

	writeJSON(w, http.StatusCreated, createTokenResponse{
		ID:    id,
		Name:  req.Name,
		Token: rawToken,
	})
}

// ListTokens handles GET /auth/tokens.
func (h *TokenHandler) ListTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := h.repo.List(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("list tokens: %s", err)})
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

// DeleteToken handles DELETE /auth/tokens/{id}.
func (h *TokenHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid token id: must be a UUID"})
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": fmt.Sprintf("delete token: %s", err)})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// generateAPIToken creates a random API token with bb_ prefix.
func generateAPIToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return "bb_" + hex.EncodeToString(b), nil
}
