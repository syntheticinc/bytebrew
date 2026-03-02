package mobile

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// SlidingWindowRateLimiter limits requests per key using a sliding time window.
// Expired entries are cleaned up periodically via StartCleanup.
type SlidingWindowRateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewSlidingWindowRateLimiter creates a rate limiter that allows up to limit
// requests per key within the given time window.
func NewSlidingWindowRateLimiter(limit int, window time.Duration) *SlidingWindowRateLimiter {
	return &SlidingWindowRateLimiter{
		attempts: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks whether a request from the given key is allowed.
// Returns an error if the rate limit is exceeded.
func (r *SlidingWindowRateLimiter) Allow(key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	// Remove expired timestamps for this key
	existing := r.attempts[key]
	valid := existing[:0]
	for _, t := range existing {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= r.limit {
		r.attempts[key] = valid
		return fmt.Errorf("rate limit exceeded: %d requests per %s", r.limit, r.window)
	}

	r.attempts[key] = append(valid, now)
	return nil
}

// StartCleanup runs a background goroutine that removes expired entries
// every 5 minutes. It stops when the context is cancelled.
func (r *SlidingWindowRateLimiter) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.cleanup()
		}
	}
}

// cleanup removes all expired entries from the attempts map.
func (r *SlidingWindowRateLimiter) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-r.window)

	for key, timestamps := range r.attempts {
		valid := timestamps[:0]
		for _, t := range timestamps {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		if len(valid) == 0 {
			delete(r.attempts, key)
			continue
		}
		r.attempts[key] = valid
	}
}
