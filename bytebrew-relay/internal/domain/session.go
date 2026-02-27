package domain

import "time"

// ActiveSession represents an active user session tracked by the relay.
type ActiveSession struct {
	ID        string
	UserID    string
	Tier      string
	StartedAt time.Time
	LastPing  time.Time
}

// IsExpired returns true if the session has not received a heartbeat
// within the given timeout duration.
func (s *ActiveSession) IsExpired(timeout time.Duration, now time.Time) bool {
	return now.Sub(s.LastPing) > timeout
}

// CachedLicense holds a validated license cached by the relay.
type CachedLicense struct {
	JWT          string
	Tier         string
	SeatsAllowed int // 1 for Personal, N for Teams
	ValidatedAt  time.Time
	ExpiresAt    time.Time
}

// IsFresh returns true if the cached license is still within TTL.
func (c *CachedLicense) IsFresh(ttl time.Duration, now time.Time) bool {
	return now.Sub(c.ValidatedAt) < ttl
}
