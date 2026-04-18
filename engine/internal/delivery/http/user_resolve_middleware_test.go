package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

type mockUserResolver struct {
	users map[string]string // id → userID
}

func newMockUserResolver() *mockUserResolver {
	return &mockUserResolver{users: make(map[string]string)}
}

func (m *mockUserResolver) addUser(id, userID string) {
	m.users[id] = userID
}

func (m *mockUserResolver) ResolveByID(_ context.Context, id string) (string, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return "", nil
}

func TestUserResolveMiddleware_ResolvesUser(t *testing.T) {
	resolver := newMockUserResolver()
	resolver.addUser("uuid-admin", "uuid-admin")
	mw := UserResolveMiddleware(resolver)

	var capturedUserID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = domain.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), ContextKeyActorID, "uuid-admin")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "uuid-admin", capturedUserID)
}

func TestUserResolveMiddleware_NoActorSkips(t *testing.T) {
	resolver := newMockUserResolver()
	mw := UserResolveMiddleware(resolver)

	var capturedUserID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = domain.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, capturedUserID, "no actor → no user ID in context")
}

func TestUserResolveMiddleware_UnknownActorSkips(t *testing.T) {
	resolver := newMockUserResolver()
	mw := UserResolveMiddleware(resolver)

	var capturedUserID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = domain.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	// e.g. API token name (not a user UUID) — resolver returns empty, no error.
	ctx := context.WithValue(req.Context(), ContextKeyActorID, "api-token-name")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, capturedUserID, "unknown actor → no user ID in context")
}

func TestUserResolveMiddleware_ErrorNonFatal(t *testing.T) {
	resolver := &failingUserResolver{}
	mw := UserResolveMiddleware(resolver)

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), ContextKeyActorID, "uuid-admin")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.True(t, called, "request must proceed even on resolver error")
	assert.Equal(t, http.StatusOK, rec.Code)
}

type failingUserResolver struct{}

func (f *failingUserResolver) ResolveByID(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("db connection failed")
}
