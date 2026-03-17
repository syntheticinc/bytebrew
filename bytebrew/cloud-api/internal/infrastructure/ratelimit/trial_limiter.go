package ratelimit

import (
	"fmt"
	"sync"
	"time"
)

// TrialLimiter is an in-memory sliding window rate limiter per user.
// It limits the number of proxy steps a trial user can perform per hour.
type TrialLimiter struct {
	mu      sync.Mutex
	windows map[string][]time.Time
	limit   int
	window  time.Duration
}

// NewTrialLimiter creates a new TrialLimiter with the given per-hour limit.
func NewTrialLimiter(stepsPerHour int) *TrialLimiter {
	return &TrialLimiter{
		windows: make(map[string][]time.Time),
		limit:   stepsPerHour,
		window:  time.Hour,
	}
}

// Check verifies the rate limit for a user and records the current timestamp.
// Returns error if the limit is exceeded.
func (l *TrialLimiter) Check(userID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-l.window)

	// Prune expired entries
	timestamps := l.windows[userID]
	valid := timestamps[:0]
	for _, t := range timestamps {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}

	if len(valid) >= l.limit {
		return fmt.Errorf("rate limit exceeded: %d steps/hour for trial", l.limit)
	}

	l.windows[userID] = append(valid, now)
	return nil
}
