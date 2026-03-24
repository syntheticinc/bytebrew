package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestEEMiddleware(info *domain.LicenseInfo) *EEMiddleware {
	ptr := &atomic.Pointer[domain.LicenseInfo]{}
	if info != nil {
		ptr.Store(info)
	}
	return NewEEMiddleware(ptr)
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})
}

func TestEEMiddleware_NilLicense_Returns403(t *testing.T) {
	mw := newTestEEMiddleware(nil)
	handler := mw.RequireEE(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ee/something", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, eeLicenseRequired, body["error"])
	assert.Equal(t, eeUpgradeURL, body["upgrade_url"])
}

func TestEEMiddleware_ActiveLicense_Allows(t *testing.T) {
	info := &domain.LicenseInfo{
		Status: domain.LicenseActive,
		Tier:   domain.TierPersonal,
	}
	mw := newTestEEMiddleware(info)
	handler := mw.RequireEE(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ee/something", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Header().Get("X-License-Warning"))
}

func TestEEMiddleware_GraceLicense_AllowsWithWarning(t *testing.T) {
	info := &domain.LicenseInfo{
		Status: domain.LicenseGrace,
		Tier:   domain.TierPersonal,
	}
	mw := newTestEEMiddleware(info)
	handler := mw.RequireEE(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ee/something", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, eeLicenseWarning, rec.Header().Get("X-License-Warning"))
}

func TestEEMiddleware_BlockedLicense_Returns403(t *testing.T) {
	info := domain.BlockedLicense()
	mw := newTestEEMiddleware(info)
	handler := mw.RequireEE(okHandler())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ee/something", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, eeLicenseExpired, body["error"])
	assert.Equal(t, eeUpgradeURL, body["upgrade_url"])
}

func TestEEMiddleware_AtomicSwap_ReflectsNewLicense(t *testing.T) {
	ptr := &atomic.Pointer[domain.LicenseInfo]{}
	// Start with no license.
	mw := NewEEMiddleware(ptr)
	handler := mw.RequireEE(okHandler())

	// First request: nil → 403.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)

	// Swap in an active license.
	ptr.Store(&domain.LicenseInfo{Status: domain.LicenseActive, Tier: domain.TierTeams})

	// Second request: active → 200.
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}
