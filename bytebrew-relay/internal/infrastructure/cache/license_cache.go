package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/domain"
)

// LicenseCache provides in-memory license caching with disk persistence.
type LicenseCache struct {
	mu       sync.RWMutex
	entries  map[string]*domain.CachedLicense // key: sha256(jwt)[:16]
	filePath string
	ttl      time.Duration
	grace    time.Duration
	lastSync time.Time // last successful Cloud API sync
	nowFunc  func() time.Time
}

// New creates a new LicenseCache.
func New(filePath string, ttl, grace time.Duration) *LicenseCache {
	return &LicenseCache{
		entries:  make(map[string]*domain.CachedLicense),
		filePath: filePath,
		ttl:      ttl,
		grace:    grace,
		nowFunc:  time.Now,
	}
}

// HashJWT returns a short hash of the JWT for use as cache key.
func HashJWT(jwt string) string {
	h := sha256.Sum256([]byte(jwt))
	return hex.EncodeToString(h[:8]) // 16 hex chars
}

// Get returns a cached license by JWT hash, or nil if not found.
func (c *LicenseCache) Get(jwtHash string) (*domain.CachedLicense, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[jwtHash]
	if !ok {
		return nil, false
	}
	return entry, true
}

// Set stores a license in the cache.
func (c *LicenseCache) Set(jwtHash string, license *domain.CachedLicense) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[jwtHash] = license
}

// IsFresh returns true if the cached entry is within TTL.
func (c *LicenseCache) IsFresh(license *domain.CachedLicense) bool {
	return license.IsFresh(c.ttl, c.now())
}

// IsWithinGrace returns true if the last successful sync is within grace period.
func (c *LicenseCache) IsWithinGrace() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.lastSync.IsZero() {
		return false
	}
	return c.now().Sub(c.lastSync) < c.grace
}

// UpdateSyncTime records a successful Cloud API sync.
func (c *LicenseCache) UpdateSyncTime() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.lastSync = c.now()
}

// LastSyncTime returns the time of the last successful sync.
func (c *LicenseCache) LastSyncTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.lastSync
}

// Entries returns a copy of all cache entries (for background refresh).
func (c *LicenseCache) Entries() map[string]*domain.CachedLicense {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*domain.CachedLicense, len(c.entries))
	for k, v := range c.entries {
		result[k] = v
	}
	return result
}

// Remove deletes a cache entry by JWT hash.
func (c *LicenseCache) Remove(jwtHash string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, jwtHash)
}

// Count returns the number of cached licenses.
func (c *LicenseCache) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// PersistToDisk writes the cache to disk as JSON.
func (c *LicenseCache) PersistToDisk() error {
	if c.filePath == "" {
		return nil
	}

	c.mu.RLock()
	data := persistData{
		Entries:  c.entries,
		LastSync: c.lastSync,
	}
	c.mu.RUnlock()

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal cache: %w", err)
	}

	dir := filepath.Dir(c.filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	if err := os.WriteFile(c.filePath, jsonData, 0600); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	slog.Info("cache persisted to disk", "path", c.filePath, "entries", len(data.Entries))
	return nil
}

// LoadFromDisk restores the cache from disk.
func (c *LicenseCache) LoadFromDisk() error {
	if c.filePath == "" {
		return nil
	}

	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Info("no cache file found, starting fresh", "path", c.filePath)
			return nil
		}
		return fmt.Errorf("read cache file: %w", err)
	}

	var pd persistData
	if err := json.Unmarshal(data, &pd); err != nil {
		slog.Warn("corrupt cache file, starting fresh", "path", c.filePath, "error", err)
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if pd.Entries != nil {
		c.entries = pd.Entries
	}
	c.lastSync = pd.LastSync

	slog.Info("cache loaded from disk", "path", c.filePath, "entries", len(c.entries))
	return nil
}

func (c *LicenseCache) now() time.Time {
	if c.nowFunc != nil {
		return c.nowFunc()
	}
	return time.Now()
}

// persistData is the on-disk representation of the cache.
type persistData struct {
	Entries  map[string]*domain.CachedLicense `json:"entries"`
	LastSync time.Time                        `json:"last_sync"`
}
