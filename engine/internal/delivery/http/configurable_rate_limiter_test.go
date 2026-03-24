package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

func newTestLicense(status domain.LicenseStatus) *atomic.Pointer[domain.LicenseInfo] {
	p := &atomic.Pointer[domain.LicenseInfo]{}
	p.Store(&domain.LicenseInfo{Status: status})
	return p
}

func newNilLicense() *atomic.Pointer[domain.LicenseInfo] {
	return &atomic.Pointer[domain.LicenseInfo]{}
}

func testRules() []RateLimitRule {
	return []RateLimitRule{
		{
			Name:      "per-org",
			KeyHeader: "X-Org-Id",
			TierHeader: "X-Org-Tier",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free":       {Requests: 5, Window: "1h"},
				"pro":        {Requests: 500, Window: "24h"},
				"enterprise": {Unlimited: true},
			},
		},
	}
}

func rlOKHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	})
}

func TestConfigurableRL_ExceedsLimit_Returns429(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 3, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-1")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass", i+1)
	}

	// 4th request should be denied
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	var body map[string]interface{}
	err := json.Unmarshal(rr.Body.Bytes(), &body)
	require.NoError(t, err)
	assert.Equal(t, "Rate limit exceeded", body["error"])
	assert.NotEmpty(t, body["retry_after"])
}

func TestConfigurableRL_DifferentKeys_Independent(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 2, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// Use up org-1's quota
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-1")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// org-1 should be denied
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// org-2 should still be allowed
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-2")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestConfigurableRL_TierPro_500PerDay(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := testRules()
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// Pro tier with X-Org-Tier: pro
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-pro")
		req.Header.Set("X-Org-Tier", "pro")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "pro tier request %d should pass", i+1)
	}

	// Check headers show correct limit
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-pro")
	req.Header.Set("X-Org-Tier", "pro")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "500", rr.Header().Get("X-RateLimit-Limit"))
}

func TestConfigurableRL_TierUnlimited_NoLimit(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := testRules()
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// Enterprise tier — unlimited
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-enterprise")
		req.Header.Set("X-Org-Tier", "enterprise")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "enterprise request %d should pass", i+1)
	}
}

func TestConfigurableRL_MissingTierHeader_UsesDefault(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			TierHeader:  "X-Org-Tier",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 2, Window: "1h"},
				"pro":  {Requests: 100, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// No X-Org-Tier header → defaults to "free" (limit=2)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-default")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-default")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestConfigurableRL_MissingKeyHeader_SkipsRule(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 1, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// No X-Org-Id header → rule skipped, request passes through
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d without key header should pass", i+1)
	}
}

func TestConfigurableRL_MultipleRules_FirstDeny(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			DefaultTier: "default",
			Tiers: map[string]RateLimitTier{
				"default": {Requests: 2, Window: "1h"},
			},
		},
		{
			Name:        "per-user",
			KeyHeader:   "X-User-Id",
			DefaultTier: "default",
			Tiers: map[string]RateLimitTier{
				"default": {Requests: 100, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// Exhaust per-org limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-1")
		req.Header.Set("X-User-Id", "user-1")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Next request should be denied by per-org rule even though per-user is fine
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-1")
	req.Header.Set("X-User-Id", "user-1")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestConfigurableRL_NoEELicense_Skips(t *testing.T) {
	tests := []struct {
		name    string
		license *atomic.Pointer[domain.LicenseInfo]
	}{
		{"nil license", newNilLicense()},
		{"blocked license", newTestLicense(domain.LicenseBlocked)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := []RateLimitRule{
				{
					Name:        "per-org",
					KeyHeader:   "X-Org-Id",
					DefaultTier: "free",
					Tiers: map[string]RateLimitTier{
						"free": {Requests: 1, Window: "1h"},
					},
				},
			}
			rl := NewConfigurableRateLimiter(rules, tt.license)
			handler := rl.Middleware(rlOKHandler())

			// Without EE license, rate limiting is skipped
			for i := 0; i < 10; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				req.Header.Set("X-Org-Id", "org-1")
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass without EE", i+1)
			}
		})
	}
}

func TestConfigurableRL_Headers_Present(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 10, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-headers")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "10", rr.Header().Get("X-RateLimit-Limit"))

	remaining, err := strconv.Atoi(rr.Header().Get("X-RateLimit-Remaining"))
	require.NoError(t, err)
	assert.Equal(t, 9, remaining)

	resetStr := rr.Header().Get("X-RateLimit-Reset")
	assert.NotEmpty(t, resetStr)
	resetUnix, err := strconv.ParseInt(resetStr, 10, 64)
	require.NoError(t, err)
	assert.Greater(t, resetUnix, time.Now().Unix())
}

func TestConfigurableRL_SlidingWindow_Resets(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 2, Window: "100ms"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// Use up the limit
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-reset")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	// Should be denied
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-reset")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-reset")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestConfigurableRL_Concurrent_ThreadSafe(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 50, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	var wg sync.WaitGroup
	results := make([]int, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("X-Org-Id", "org-concurrent")
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			results[idx] = rr.Code
		}(i)
	}

	wg.Wait()

	okCount := 0
	deniedCount := 0
	for _, code := range results {
		switch code {
		case http.StatusOK:
			okCount++
		case http.StatusTooManyRequests:
			deniedCount++
		default:
			t.Fatalf("unexpected status code: %d", code)
		}
	}

	assert.Equal(t, 50, okCount, "exactly 50 requests should be allowed")
	assert.Equal(t, 50, deniedCount, "exactly 50 requests should be denied")
}

func TestConfigurableRL_NoRules_Passthrough(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rl := NewConfigurableRateLimiter(nil, license)
	handler := rl.Middleware(rlOKHandler())

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-1")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d should pass with no rules", i+1)
	}
}

func TestConfigurableRL_GraceLicense_Works(t *testing.T) {
	license := newTestLicense(domain.LicenseGrace)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 2, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// Grace license should still enforce rate limits
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-grace")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-grace")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}

func TestConfigurableRL_RequestContext_Headers(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			TierHeader:  "X-Org-Tier",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 1, Window: "1h"},
				"pro":  {Requests: 100, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// Set headers via RequestContext (simulating proxy/MCP forwarding)
	rc := &domain.RequestContext{
		Headers: map[string]string{
			"X-Org-Id":   "org-ctx",
			"X-Org-Tier": "pro",
		},
	}

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		ctx := domain.WithRequestContext(req.Context(), rc)
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code, "request %d with RequestContext should pass", i+1)
		assert.Equal(t, "100", rr.Header().Get("X-RateLimit-Limit"))
	}
}

func TestConfigurableRL_UnknownTier_FallsBackToDefault(t *testing.T) {
	license := newTestLicense(domain.LicenseActive)
	rules := []RateLimitRule{
		{
			Name:        "per-org",
			KeyHeader:   "X-Org-Id",
			TierHeader:  "X-Org-Tier",
			DefaultTier: "free",
			Tiers: map[string]RateLimitTier{
				"free": {Requests: 2, Window: "1h"},
			},
		},
	}
	rl := NewConfigurableRateLimiter(rules, license)
	handler := rl.Middleware(rlOKHandler())

	// Unknown tier "platinum" falls back to default "free" (limit=2)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Org-Id", "org-unknown-tier")
		req.Header.Set("X-Org-Tier", "platinum")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		assert.Equal(t, http.StatusOK, rr.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Org-Id", "org-unknown-tier")
	req.Header.Set("X-Org-Tier", "platinum")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)
}
