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
	users map[string]string // "tenantID:externalID" → userID
}

func newMockUserResolver() *mockUserResolver {
	return &mockUserResolver{users: make(map[string]string)}
}

func (m *mockUserResolver) addUser(tenantID, externalID, userID string) {
	m.users[tenantID+":"+externalID] = userID
}

func (m *mockUserResolver) GetOrCreate(_ context.Context, tenantID, externalID string) (string, error) {
	key := tenantID + ":" + externalID
	if id, ok := m.users[key]; ok {
		return id, nil
	}
	// Simulate lazy creation
	id := "uuid-for-" + externalID
	m.users[key] = id
	return id, nil
}

func TestUserResolveMiddleware_ResolvesUser(t *testing.T) {
	resolver := newMockUserResolver()
	mw := UserResolveMiddleware(resolver)

	var capturedUserID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = domain.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), ContextKeyActorID, "admin-user")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "uuid-for-admin-user", capturedUserID)
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

func TestUserResolveMiddleware_ErrorNonFatal(t *testing.T) {
	resolver := &failingUserResolver{}
	mw := UserResolveMiddleware(resolver)

	var called bool
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), ContextKeyActorID, "admin-user")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.True(t, called, "request must proceed even on resolver error")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestUserResolveMiddleware_UsesTenantFromContext(t *testing.T) {
	resolver := newMockUserResolver()
	customTenant := "00000000-0000-0000-0000-000000000099"
	resolver.addUser(customTenant, "tenant-user", "custom-uuid")

	mw := UserResolveMiddleware(resolver)

	var capturedUserID string
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUserID = domain.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := context.WithValue(req.Context(), ContextKeyActorID, "tenant-user")
	ctx = domain.WithTenantID(ctx, customTenant)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "custom-uuid", capturedUserID)
}

type failingUserResolver struct{}

func (f *failingUserResolver) GetOrCreate(_ context.Context, _, _ string) (string, error) {
	return "", fmt.Errorf("db connection failed")
}
