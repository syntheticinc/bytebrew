package refresh

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/domain"
)

// --- Mocks ---

type mockLicenseCache struct {
	entries    map[string]*domain.CachedLicense
	setCalls   []setCall
	removed    []string
	syncCalled bool
}

type setCall struct {
	hash    string
	license *domain.CachedLicense
}

func newMockCache(entries map[string]*domain.CachedLicense) *mockLicenseCache {
	if entries == nil {
		entries = make(map[string]*domain.CachedLicense)
	}
	return &mockLicenseCache{entries: entries}
}

func (m *mockLicenseCache) Entries() map[string]*domain.CachedLicense {
	result := make(map[string]*domain.CachedLicense, len(m.entries))
	for k, v := range m.entries {
		result[k] = v
	}
	return result
}

func (m *mockLicenseCache) Set(hash string, license *domain.CachedLicense) {
	m.setCalls = append(m.setCalls, setCall{hash: hash, license: license})
	m.entries[hash] = license
}

func (m *mockLicenseCache) Remove(hash string) {
	m.removed = append(m.removed, hash)
	delete(m.entries, hash)
}

func (m *mockLicenseCache) UpdateSyncTime() {
	m.syncCalled = true
}

type mockCloudAPIClient struct {
	refreshResults map[string]string // currentJWT -> newJWT
	refreshErrors  map[string]error  // currentJWT -> error
	validateResults map[string]*ValidationResult // JWT -> result
	validateErrors  map[string]error             // JWT -> error
	refreshCalls    int
	validateCalls   int
}

func newMockCloudAPI() *mockCloudAPIClient {
	return &mockCloudAPIClient{
		refreshResults:  make(map[string]string),
		refreshErrors:   make(map[string]error),
		validateResults: make(map[string]*ValidationResult),
		validateErrors:  make(map[string]error),
	}
}

func (m *mockCloudAPIClient) RefreshLicense(_ context.Context, currentJWT string) (string, error) {
	m.refreshCalls++
	if err, ok := m.refreshErrors[currentJWT]; ok {
		return "", err
	}
	if jwt, ok := m.refreshResults[currentJWT]; ok {
		return jwt, nil
	}
	return "", fmt.Errorf("no mock result for JWT: %s", currentJWT)
}

func (m *mockCloudAPIClient) ValidateLicense(_ context.Context, licenseJWT string) (*ValidationResult, error) {
	m.validateCalls++
	if err, ok := m.validateErrors[licenseJWT]; ok {
		return nil, err
	}
	if result, ok := m.validateResults[licenseJWT]; ok {
		return result, nil
	}
	return nil, fmt.Errorf("no mock result for JWT: %s", licenseJWT)
}

type mockHasher struct {
	hashFunc func(jwt string) string
}

func (m *mockHasher) Hash(jwt string) string {
	if m.hashFunc != nil {
		return m.hashFunc(jwt)
	}
	return "hash-" + jwt
}

// --- Tests ---

func TestRefreshAll_UpdatesCache(t *testing.T) {
	// Setup: one license in cache
	oldJWT := "old-jwt-token"
	newJWT := "new-jwt-token"
	oldHash := "hash-old"
	newHash := "hash-new"

	cache := newMockCache(map[string]*domain.CachedLicense{
		oldHash: {
			JWT:          oldJWT,
			Tier:         "personal",
			SeatsAllowed: 1,
			ValidatedAt:  time.Now().Add(-1 * time.Hour),
			ExpiresAt:    time.Now().Add(23 * time.Hour),
		},
	})

	cloudAPI := newMockCloudAPI()
	cloudAPI.refreshResults[oldJWT] = newJWT
	cloudAPI.validateResults[newJWT] = &ValidationResult{
		Tier:         "personal",
		SeatsAllowed: 1,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	hasher := &mockHasher{hashFunc: func(jwt string) string {
		if jwt == newJWT {
			return newHash
		}
		return oldHash
	}}

	uc := New(cache, cloudAPI, hasher)
	fixedNow := time.Date(2026, 2, 23, 12, 0, 0, 0, time.UTC)
	uc.nowFunc = func() time.Time { return fixedNow }

	err := uc.RefreshAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify: CloudAPI was called
	if cloudAPI.refreshCalls != 1 {
		t.Fatalf("expected 1 refresh call, got %d", cloudAPI.refreshCalls)
	}
	if cloudAPI.validateCalls != 1 {
		t.Fatalf("expected 1 validate call, got %d", cloudAPI.validateCalls)
	}

	// Verify: new entry was set in cache
	if len(cache.setCalls) == 0 {
		t.Fatal("expected Set to be called on cache")
	}

	lastSet := cache.setCalls[len(cache.setCalls)-1]
	if lastSet.hash != newHash {
		t.Fatalf("expected set with hash %q, got %q", newHash, lastSet.hash)
	}
	if lastSet.license.JWT != newJWT {
		t.Fatalf("expected set with JWT %q, got %q", newJWT, lastSet.license.JWT)
	}
	if lastSet.license.Tier != "personal" {
		t.Fatalf("expected tier personal, got %s", lastSet.license.Tier)
	}

	// Verify: old entry was removed (hash changed)
	found := false
	for _, h := range cache.removed {
		if h == oldHash {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected old hash to be removed from cache when hash changed")
	}
}

func TestRefreshAll_SameHash_NoRemove(t *testing.T) {
	// When the refreshed JWT produces the same hash, old entry should NOT be removed.
	oldJWT := "old-jwt-token"
	newJWT := "new-jwt-token"
	stableHash := "hash-stable"

	cache := newMockCache(map[string]*domain.CachedLicense{
		stableHash: {
			JWT:          oldJWT,
			Tier:         "personal",
			SeatsAllowed: 1,
			ValidatedAt:  time.Now().Add(-1 * time.Hour),
			ExpiresAt:    time.Now().Add(23 * time.Hour),
		},
	})

	cloudAPI := newMockCloudAPI()
	cloudAPI.refreshResults[oldJWT] = newJWT
	cloudAPI.validateResults[newJWT] = &ValidationResult{
		Tier:         "personal",
		SeatsAllowed: 1,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	// Hasher returns same hash for both JWTs
	hasher := &mockHasher{hashFunc: func(_ string) string {
		return stableHash
	}}

	uc := New(cache, cloudAPI, hasher)

	err := uc.RefreshAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify: Set was called
	if len(cache.setCalls) == 0 {
		t.Fatal("expected Set to be called on cache")
	}

	// Verify: Remove was NOT called (same hash)
	if len(cache.removed) != 0 {
		t.Fatalf("expected no removals when hash unchanged, got %d: %v", len(cache.removed), cache.removed)
	}
}

func TestRefreshAll_RefreshError_ContinuesOtherLicenses(t *testing.T) {
	// Two licenses: first fails refresh, second succeeds.
	// The usecase should continue and not return error.
	failJWT := "fail-jwt-tok"
	goodJWT := "good-jwt-tok"
	newGoodJWT := "new-good-jwt"

	cache := newMockCache(map[string]*domain.CachedLicense{
		"hash-fail": {
			JWT:          failJWT,
			Tier:         "personal",
			SeatsAllowed: 1,
			ValidatedAt:  time.Now().Add(-1 * time.Hour),
		},
		"hash-good": {
			JWT:          goodJWT,
			Tier:         "teams",
			SeatsAllowed: 5,
			ValidatedAt:  time.Now().Add(-1 * time.Hour),
		},
	})

	cloudAPI := newMockCloudAPI()
	cloudAPI.refreshErrors[failJWT] = fmt.Errorf("connection refused")
	cloudAPI.refreshResults[goodJWT] = newGoodJWT
	cloudAPI.validateResults[newGoodJWT] = &ValidationResult{
		Tier:         "teams",
		SeatsAllowed: 5,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	hasher := &mockHasher{hashFunc: func(jwt string) string {
		return "hash-" + jwt
	}}

	uc := New(cache, cloudAPI, hasher)

	err := uc.RefreshAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify: at least the good license was refreshed
	if len(cache.setCalls) == 0 {
		t.Fatal("expected at least one Set call for the successful license")
	}

	// Verify: sync time updated (at least one success)
	if !cache.syncCalled {
		t.Fatal("expected UpdateSyncTime to be called after at least one success")
	}
}

func TestRefreshAll_ValidateError_ContinuesOtherLicenses(t *testing.T) {
	// Refresh succeeds but ValidateLicense fails -- should skip that license.
	jwt1 := "jwt-token-1a"
	newJWT1 := "new-jwt-1aaa"
	jwt2 := "jwt-token-2a"
	newJWT2 := "new-jwt-2aaa"

	cache := newMockCache(map[string]*domain.CachedLicense{
		"hash-1": {
			JWT:  jwt1,
			Tier: "personal",
		},
		"hash-2": {
			JWT:  jwt2,
			Tier: "teams",
		},
	})

	cloudAPI := newMockCloudAPI()
	// First: refresh OK, validate fails
	cloudAPI.refreshResults[jwt1] = newJWT1
	cloudAPI.validateErrors[newJWT1] = fmt.Errorf("license expired")
	// Second: both OK
	cloudAPI.refreshResults[jwt2] = newJWT2
	cloudAPI.validateResults[newJWT2] = &ValidationResult{
		Tier:         "teams",
		SeatsAllowed: 5,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	hasher := &mockHasher{hashFunc: func(jwt string) string {
		return "hash-" + jwt
	}}

	uc := New(cache, cloudAPI, hasher)

	err := uc.RefreshAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify: the successful license was cached
	foundGood := false
	for _, sc := range cache.setCalls {
		if sc.license.Tier == "teams" {
			foundGood = true
		}
	}
	if !foundGood {
		t.Fatal("expected the successful license to be cached")
	}

	// Verify: sync time updated (at least one success)
	if !cache.syncCalled {
		t.Fatal("expected UpdateSyncTime to be called")
	}
}

func TestRefreshAll_UpdatesSyncTime(t *testing.T) {
	jwt := "valid-jwt-to"
	newJWT := "refreshed-jw"

	cache := newMockCache(map[string]*domain.CachedLicense{
		"hash-valid": {
			JWT:  jwt,
			Tier: "personal",
		},
	})

	cloudAPI := newMockCloudAPI()
	cloudAPI.refreshResults[jwt] = newJWT
	cloudAPI.validateResults[newJWT] = &ValidationResult{
		Tier:         "personal",
		SeatsAllowed: 1,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	hasher := &mockHasher{hashFunc: func(j string) string {
		return "hash-" + j
	}}

	uc := New(cache, cloudAPI, hasher)

	err := uc.RefreshAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !cache.syncCalled {
		t.Fatal("expected UpdateSyncTime to be called after successful refresh")
	}
}

func TestRefreshAll_EmptyCache(t *testing.T) {
	cache := newMockCache(nil)
	cloudAPI := newMockCloudAPI()
	hasher := &mockHasher{}

	uc := New(cache, cloudAPI, hasher)

	err := uc.RefreshAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify: no API calls made
	if cloudAPI.refreshCalls != 0 {
		t.Fatalf("expected 0 refresh calls for empty cache, got %d", cloudAPI.refreshCalls)
	}
	if cloudAPI.validateCalls != 0 {
		t.Fatalf("expected 0 validate calls for empty cache, got %d", cloudAPI.validateCalls)
	}

	// Verify: sync time NOT updated (nothing refreshed)
	if cache.syncCalled {
		t.Fatal("expected UpdateSyncTime NOT to be called for empty cache")
	}
}

func TestRefreshAll_AllFail_NoSyncTimeUpdate(t *testing.T) {
	// All licenses fail to refresh -- sync time should NOT be updated.
	cache := newMockCache(map[string]*domain.CachedLicense{
		"hash-1": {JWT: "jwt-fail-1a", Tier: "personal"},
		"hash-2": {JWT: "jwt-fail-2a", Tier: "teams"},
	})

	cloudAPI := newMockCloudAPI()
	cloudAPI.refreshErrors["jwt-fail-1a"] = fmt.Errorf("timeout")
	cloudAPI.refreshErrors["jwt-fail-2a"] = fmt.Errorf("timeout")

	hasher := &mockHasher{}

	uc := New(cache, cloudAPI, hasher)

	err := uc.RefreshAll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify: no Set calls
	if len(cache.setCalls) != 0 {
		t.Fatalf("expected 0 Set calls when all fail, got %d", len(cache.setCalls))
	}

	// Verify: sync time NOT updated
	if cache.syncCalled {
		t.Fatal("expected UpdateSyncTime NOT to be called when all licenses fail")
	}
}

func TestRefreshAll_RespectsContextCancellation(t *testing.T) {
	cache := newMockCache(map[string]*domain.CachedLicense{
		"hash-1": {JWT: "jwt-ctx-1ab", Tier: "personal"},
		"hash-2": {JWT: "jwt-ctx-2ab", Tier: "teams"},
	})

	cloudAPI := newMockCloudAPI()
	cloudAPI.refreshResults["jwt-ctx-1ab"] = "new-jwt-1ab"
	cloudAPI.validateResults["new-jwt-1ab"] = &ValidationResult{
		Tier:         "personal",
		SeatsAllowed: 1,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
	cloudAPI.refreshResults["jwt-ctx-2ab"] = "new-jwt-2ab"
	cloudAPI.validateResults["new-jwt-2ab"] = &ValidationResult{
		Tier:         "teams",
		SeatsAllowed: 5,
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}

	hasher := &mockHasher{hashFunc: func(jwt string) string {
		return "hash-" + jwt
	}}

	// Cancel context before calling RefreshAll
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	uc := New(cache, cloudAPI, hasher)

	err := uc.RefreshAll(ctx)
	// Should return early without error (or with context error)
	// The key assertion: it should not process all licenses
	_ = err

	// With cancelled context, we expect either no calls or early exit.
	// The exact behavior depends on implementation, but at minimum
	// it should not panic and should respect cancellation.
	totalAPICalls := cloudAPI.refreshCalls + cloudAPI.validateCalls
	if totalAPICalls > 2 {
		// At most one license could be processed before checking ctx
		t.Fatalf("expected early exit on cancelled context, got %d total API calls", totalAPICalls)
	}
}
