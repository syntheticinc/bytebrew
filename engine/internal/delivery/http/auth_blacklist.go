package http

import (
	"sync"
	"time"
)

// TokenBlacklist is a concurrent-safe in-memory set of revoked JWT token hashes.
// Entries are evicted automatically after their expiry time to prevent unbounded growth.
type TokenBlacklist struct {
	mu      sync.RWMutex
	revoked map[string]time.Time // tokenHash → token expiresAt
}

// NewTokenBlacklist creates a TokenBlacklist with a background cleanup goroutine.
func NewTokenBlacklist() *TokenBlacklist {
	b := &TokenBlacklist{revoked: make(map[string]time.Time)}
	go b.cleanup()
	return b
}

// Revoke adds a token hash to the blacklist until expiresAt.
func (b *TokenBlacklist) Revoke(tokenHash string, expiresAt time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.revoked[tokenHash] = expiresAt
}

// IsRevoked reports whether a token hash is on the blacklist and not yet expired.
func (b *TokenBlacklist) IsRevoked(tokenHash string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	exp, ok := b.revoked[tokenHash]
	if !ok {
		return false
	}
	return time.Now().Before(exp)
}

// cleanup removes expired entries every hour.
func (b *TokenBlacklist) cleanup() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		b.mu.Lock()
		for hash, exp := range b.revoked {
			if now.After(exp) {
				delete(b.revoked, hash)
			}
		}
		b.mu.Unlock()
	}
}
