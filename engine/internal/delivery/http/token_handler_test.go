package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTokenRepository struct {
	tokens    []TokenInfo
	nextID    int
	createErr error
	deleteErr error
}

func newMockTokenRepository() *mockTokenRepository {
	return &mockTokenRepository{nextID: 1}
}

func (m *mockTokenRepository) Create(_ context.Context, name, tokenHash string, scopesMask int) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	id := fmt.Sprintf("%d", m.nextID)
	m.nextID++
	m.tokens = append(m.tokens, TokenInfo{
		ID:         id,
		Name:       name,
		ScopesMask: scopesMask,
		CreatedAt:  time.Now(),
	})
	return id, nil
}

func (m *mockTokenRepository) List(_ context.Context) ([]TokenInfo, error) {
	return m.tokens, nil
}

func (m *mockTokenRepository) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	for i, t := range m.tokens {
		if t.ID == id {
			m.tokens = append(m.tokens[:i], m.tokens[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("token not found")
}

func TestTokenHandler_CreateToken(t *testing.T) {
	repo := newMockTokenRepository()
	h := NewTokenHandler(repo)

	body := `{"name":"my-token","scopes_mask":3}`
	req := httptest.NewRequest(http.MethodPost, "/auth/tokens", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.CreateToken(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp createTokenResponse
	err := json.NewDecoder(rec.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "1", resp.ID)
	assert.Equal(t, "my-token", resp.Name)
	assert.True(t, strings.HasPrefix(resp.Token, "bb_"))
	assert.Len(t, resp.Token, 3+64) // "bb_" + 32 bytes hex

	// Verify token stored in repo
	assert.Len(t, repo.tokens, 1)
	assert.Equal(t, "my-token", repo.tokens[0].Name)
	assert.Equal(t, 3, repo.tokens[0].ScopesMask)
}

func TestTokenHandler_CreateToken_EmptyName(t *testing.T) {
	repo := newMockTokenRepository()
	h := NewTokenHandler(repo)

	body := `{"name":"","scopes_mask":1}`
	req := httptest.NewRequest(http.MethodPost, "/auth/tokens", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.CreateToken(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "name required")
}

func TestTokenHandler_CreateToken_DuplicateName(t *testing.T) {
	repo := newMockTokenRepository()
	repo.createErr = fmt.Errorf("duplicate name")
	h := NewTokenHandler(repo)

	body := `{"name":"dup","scopes_mask":1}`
	req := httptest.NewRequest(http.MethodPost, "/auth/tokens", strings.NewReader(body))
	rec := httptest.NewRecorder()

	h.CreateToken(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestTokenHandler_ListTokens(t *testing.T) {
	repo := newMockTokenRepository()
	repo.tokens = []TokenInfo{
		{ID: "1", Name: "token-1", ScopesMask: 1, CreatedAt: time.Now()},
		{ID: "2", Name: "token-2", ScopesMask: 3, CreatedAt: time.Now()},
	}
	h := NewTokenHandler(repo)

	req := httptest.NewRequest(http.MethodGet, "/auth/tokens", nil)
	rec := httptest.NewRecorder()

	h.ListTokens(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var tokens []TokenInfo
	err := json.NewDecoder(rec.Body).Decode(&tokens)
	require.NoError(t, err)
	assert.Len(t, tokens, 2)
	assert.Equal(t, "token-1", tokens[0].Name)
	assert.Equal(t, "token-2", tokens[1].Name)
}

func TestTokenHandler_DeleteToken(t *testing.T) {
	repo := newMockTokenRepository()
	repo.tokens = []TokenInfo{
		{ID: "1", Name: "to-delete", ScopesMask: 1, CreatedAt: time.Now()},
	}
	h := NewTokenHandler(repo)

	// Use chi router to extract URL param
	r := chi.NewRouter()
	r.Delete("/auth/tokens/{id}", h.DeleteToken)

	req := httptest.NewRequest(http.MethodDelete, "/auth/tokens/1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Len(t, repo.tokens, 0)
}

func TestTokenHandler_DeleteToken_NotFound(t *testing.T) {
	repo := newMockTokenRepository()
	h := NewTokenHandler(repo)

	r := chi.NewRouter()
	r.Delete("/auth/tokens/{id}", h.DeleteToken)

	req := httptest.NewRequest(http.MethodDelete, "/auth/tokens/999", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
