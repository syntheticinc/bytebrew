package validate

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/domain"
)

// --- Mocks ---

type mockCloudAPIClient struct {
	result *CloudAPIResult
	err    error
	calls  int
}

func (m *mockCloudAPIClient) ValidateLicense(_ context.Context, _ string) (*CloudAPIResult, error) {
	m.calls++
	return m.result, m.err
}

type mockCache struct {
	entries    map[string]*domain.CachedLicense
	fresh      bool
	withinGrace bool
	syncCalled bool
}

func newMockCache() *mockCache {
	return &mockCache{entries: make(map[string]*domain.CachedLicense)}
}

func (m *mockCache) Get(hash string) (*domain.CachedLicense, bool) {
	e, ok := m.entries[hash]
	return e, ok
}

func (m *mockCache) Set(hash string, l *domain.CachedLicense) {
	m.entries[hash] = l
}

func (m *mockCache) IsFresh(_ *domain.CachedLicense) bool {
	return m.fresh
}

func (m *mockCache) IsWithinGrace() bool {
	return m.withinGrace
}

func (m *mockCache) UpdateSyncTime() {
	m.syncCalled = true
}

type mockHasher struct{}

func (h *mockHasher) Hash(jwt string) string {
	return "hash-" + jwt[:8]
}

// --- Tests ---

func TestExecute_EmptyJWT(t *testing.T) {
	uc := New(&mockCloudAPIClient{}, newMockCache(), &mockHasher{})

	result, err := uc.Execute(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result for empty JWT")
	}
	if result.Message == "" {
		t.Fatal("expected message for empty JWT")
	}
}

func TestExecute_CacheHit_Fresh(t *testing.T) {
	cache := newMockCache()
	cache.fresh = true
	cache.entries["hash-test-jwt"] = &domain.CachedLicense{
		JWT:          "test-jwt-xxxxx",
		Tier:         "personal",
		SeatsAllowed: 1,
		ValidatedAt:  time.Now(),
	}

	client := &mockCloudAPIClient{}
	uc := New(client, cache, &mockHasher{})

	result, err := uc.Execute(context.Background(), "test-jwt-xxxxx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected valid result from cache")
	}
	if result.Tier != "personal" {
		t.Fatalf("expected tier personal, got %s", result.Tier)
	}
	if !result.FromCache {
		t.Fatal("expected FromCache=true")
	}
	if client.calls != 0 {
		t.Fatalf("expected no API calls, got %d", client.calls)
	}
}

func TestExecute_APISuccess(t *testing.T) {
	cache := newMockCache()
	cache.fresh = false

	client := &mockCloudAPIClient{
		result: &CloudAPIResult{
			Tier:         "teams",
			SeatsAllowed: 5,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		},
	}

	uc := New(client, cache, &mockHasher{})

	result, err := uc.Execute(context.Background(), "valid-jw-ttoken")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected valid result")
	}
	if result.Tier != "teams" {
		t.Fatalf("expected tier teams, got %s", result.Tier)
	}
	if result.SeatsAllowed != 5 {
		t.Fatalf("expected 5 seats, got %d", result.SeatsAllowed)
	}
	if result.FromCache {
		t.Fatal("expected FromCache=false for API result")
	}
	if client.calls != 1 {
		t.Fatalf("expected 1 API call, got %d", client.calls)
	}
	if !cache.syncCalled {
		t.Fatal("expected UpdateSyncTime to be called")
	}

	// Verify cached
	cached, ok := cache.entries["hash-valid-jw"]
	if !ok {
		t.Fatal("expected license to be cached")
	}
	if cached.Tier != "teams" {
		t.Fatalf("expected cached tier teams, got %s", cached.Tier)
	}
}

func TestExecute_APIFailure_GracePeriod(t *testing.T) {
	cache := newMockCache()
	cache.fresh = false
	cache.withinGrace = true
	cache.entries["hash-grace-jw"] = &domain.CachedLicense{
		JWT:          "grace-jwt-xxxxx",
		Tier:         "personal",
		SeatsAllowed: 1,
		ValidatedAt:  time.Now().Add(-3 * time.Minute),
	}

	client := &mockCloudAPIClient{
		err: fmt.Errorf("connection refused"),
	}

	uc := New(client, cache, &mockHasher{})

	result, err := uc.Execute(context.Background(), "grace-jwt-xxxxx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Fatal("expected valid result during grace period")
	}
	if result.Tier != "personal" {
		t.Fatalf("expected tier personal, got %s", result.Tier)
	}
	if !result.FromCache {
		t.Fatal("expected FromCache=true during grace")
	}
	if result.Message == "" {
		t.Fatal("expected warning message during grace")
	}
}

func TestExecute_APIFailure_GraceExpired_Blocked(t *testing.T) {
	cache := newMockCache()
	cache.fresh = false
	cache.withinGrace = false
	cache.entries["hash-expired-"] = &domain.CachedLicense{
		JWT:          "expired-jwt-xxx",
		Tier:         "personal",
		SeatsAllowed: 1,
		ValidatedAt:  time.Now().Add(-1 * time.Hour),
	}

	client := &mockCloudAPIClient{
		err: fmt.Errorf("connection refused"),
	}

	uc := New(client, cache, &mockHasher{})

	result, err := uc.Execute(context.Background(), "expired-jwt-xxx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result after grace expired")
	}
	if result.Message == "" {
		t.Fatal("expected error message")
	}
}

func TestExecute_APIFailure_NoCachedEntry_Blocked(t *testing.T) {
	cache := newMockCache()
	cache.fresh = false
	cache.withinGrace = true // grace is ok, but no cached entry

	client := &mockCloudAPIClient{
		err: fmt.Errorf("connection refused"),
	}

	uc := New(client, cache, &mockHasher{})

	result, err := uc.Execute(context.Background(), "unknown-jwt-xxx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result when no cache entry exists")
	}
}
