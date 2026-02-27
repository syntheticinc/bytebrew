package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/domain"
)

func TestGetSet(t *testing.T) {
	c := New("", 5*time.Minute, 30*time.Minute)

	// Get nonexistent
	_, ok := c.Get("nonexistent")
	if ok {
		t.Fatal("expected not found for nonexistent key")
	}

	// Set and get
	license := &domain.CachedLicense{
		JWT:          "test-jwt",
		Tier:         "personal",
		SeatsAllowed: 1,
		ValidatedAt:  time.Now(),
	}
	c.Set("key-1", license)

	got, ok := c.Get("key-1")
	if !ok {
		t.Fatal("expected to find key-1")
	}
	if got.Tier != "personal" {
		t.Fatalf("expected personal, got %s", got.Tier)
	}
}

func TestIsFresh(t *testing.T) {
	ttl := 5 * time.Minute
	c := New("", ttl, 30*time.Minute)

	now := time.Now()
	c.nowFunc = func() time.Time { return now }

	// Fresh license
	fresh := &domain.CachedLicense{
		ValidatedAt: now.Add(-2 * time.Minute),
	}
	if !c.IsFresh(fresh) {
		t.Fatal("expected fresh for 2-minute-old entry")
	}

	// Stale license
	stale := &domain.CachedLicense{
		ValidatedAt: now.Add(-10 * time.Minute),
	}
	if c.IsFresh(stale) {
		t.Fatal("expected stale for 10-minute-old entry")
	}
}

func TestIsWithinGrace(t *testing.T) {
	grace := 30 * time.Minute
	c := New("", 5*time.Minute, grace)

	now := time.Now()
	c.nowFunc = func() time.Time { return now }

	// No sync yet
	if c.IsWithinGrace() {
		t.Fatal("expected not within grace when no sync has occurred")
	}

	// Sync just happened
	c.UpdateSyncTime()
	if !c.IsWithinGrace() {
		t.Fatal("expected within grace right after sync")
	}

	// Move past grace
	now = now.Add(31 * time.Minute)
	if c.IsWithinGrace() {
		t.Fatal("expected not within grace after 31 minutes")
	}
}

func TestPersistAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	// Create cache with data
	c1 := New(path, 5*time.Minute, 30*time.Minute)
	c1.Set("key-1", &domain.CachedLicense{
		JWT:          "jwt-1",
		Tier:         "personal",
		SeatsAllowed: 1,
		ValidatedAt:  time.Now(),
	})
	c1.Set("key-2", &domain.CachedLicense{
		JWT:          "jwt-2",
		Tier:         "teams",
		SeatsAllowed: 5,
		ValidatedAt:  time.Now(),
	})
	c1.UpdateSyncTime()

	if err := c1.PersistToDisk(); err != nil {
		t.Fatalf("persist failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file not found: %v", err)
	}

	// Load into new cache
	c2 := New(path, 5*time.Minute, 30*time.Minute)
	if err := c2.LoadFromDisk(); err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if c2.Count() != 2 {
		t.Fatalf("expected 2 entries, got %d", c2.Count())
	}

	got, ok := c2.Get("key-1")
	if !ok {
		t.Fatal("expected to find key-1 after load")
	}
	if got.Tier != "personal" {
		t.Fatalf("expected personal, got %s", got.Tier)
	}

	got2, ok := c2.Get("key-2")
	if !ok {
		t.Fatal("expected to find key-2 after load")
	}
	if got2.Tier != "teams" {
		t.Fatalf("expected teams, got %s", got2.Tier)
	}
}

func TestLoadFromDisk_NoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	c := New(path, 5*time.Minute, 30*time.Minute)

	err := c.LoadFromDisk()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if c.Count() != 0 {
		t.Fatalf("expected 0 entries, got %d", c.Count())
	}
}

func TestLoadFromDisk_CorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.json")
	if err := os.WriteFile(path, []byte("not valid json"), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	c := New(path, 5*time.Minute, 30*time.Minute)
	err := c.LoadFromDisk()
	if err != nil {
		t.Fatalf("expected no error for corrupt file, got: %v", err)
	}
	if c.Count() != 0 {
		t.Fatalf("expected 0 entries for corrupt file, got %d", c.Count())
	}
}

func TestPersistToDisk_EmptyPath(t *testing.T) {
	c := New("", 5*time.Minute, 30*time.Minute)
	c.Set("key-1", &domain.CachedLicense{Tier: "personal"})

	err := c.PersistToDisk()
	if err != nil {
		t.Fatalf("expected no error for empty path, got: %v", err)
	}
}

func TestHashJWT(t *testing.T) {
	hash1 := HashJWT("jwt-token-1")
	hash2 := HashJWT("jwt-token-2")

	if hash1 == hash2 {
		t.Fatal("expected different hashes for different JWTs")
	}

	// Same input should produce same hash
	if HashJWT("jwt-token-1") != hash1 {
		t.Fatal("expected consistent hash")
	}

	// Should be 16 hex chars (8 bytes)
	if len(hash1) != 16 {
		t.Fatalf("expected 16 char hash, got %d", len(hash1))
	}
}

func TestCount(t *testing.T) {
	c := New("", 5*time.Minute, 30*time.Minute)

	if c.Count() != 0 {
		t.Fatalf("expected 0, got %d", c.Count())
	}

	c.Set("k1", &domain.CachedLicense{})
	c.Set("k2", &domain.CachedLicense{})

	if c.Count() != 2 {
		t.Fatalf("expected 2, got %d", c.Count())
	}
}

func TestRemove(t *testing.T) {
	c := New("", 5*time.Minute, 30*time.Minute)
	c.Set("k1", &domain.CachedLicense{Tier: "personal"})

	c.Remove("k1")

	_, ok := c.Get("k1")
	if ok {
		t.Fatal("expected key to be removed")
	}
	if c.Count() != 0 {
		t.Fatalf("expected 0 entries, got %d", c.Count())
	}
}

func TestEntries_ReturnsCopy(t *testing.T) {
	c := New("", 5*time.Minute, 30*time.Minute)
	c.Set("k1", &domain.CachedLicense{Tier: "personal"})

	entries := c.Entries()
	entries["k2"] = &domain.CachedLicense{Tier: "teams"}

	// Original should not be affected
	if c.Count() != 1 {
		t.Fatalf("expected 1 entry in original cache, got %d", c.Count())
	}
}
