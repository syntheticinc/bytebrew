package sessions

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/domain"
)

// Usecase manages active sessions and enforces seat limits.
type Usecase struct {
	mu               sync.Mutex
	sessions         map[string]*domain.ActiveSession // sessionID -> session
	heartbeatTimeout time.Duration
	nowFunc          func() time.Time
}

// New creates a new sessions usecase.
func New(heartbeatTimeout time.Duration) *Usecase {
	return &Usecase{
		sessions:         make(map[string]*domain.ActiveSession),
		heartbeatTimeout: heartbeatTimeout,
		nowFunc:          time.Now,
	}
}

// Register creates a new session. Returns error if the user is at their seat limit.
func (u *Usecase) Register(userID, sessionID, tier string, seatsAllowed int) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	// Clean expired sessions first (under lock)
	u.cleanExpiredLocked()

	// Count active sessions for this user
	activeCount := u.activeCountLocked(userID)
	if activeCount >= seatsAllowed {
		return fmt.Errorf("seat limit reached: %d/%d active sessions for user %s", activeCount, seatsAllowed, userID)
	}

	now := u.now()
	u.sessions[sessionID] = &domain.ActiveSession{
		ID:        sessionID,
		UserID:    userID,
		Tier:      tier,
		StartedAt: now,
		LastPing:  now,
	}

	slog.Info("session registered",
		"session_id", sessionID,
		"user_id", userID,
		"tier", tier,
		"active", activeCount+1,
		"limit", seatsAllowed,
	)
	return nil
}

// Heartbeat updates the last ping time for a session.
func (u *Usecase) Heartbeat(sessionID string) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	session, ok := u.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.LastPing = u.now()
	return nil
}

// Release removes a session.
func (u *Usecase) Release(sessionID string) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	session, ok := u.sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	slog.Info("session released",
		"session_id", sessionID,
		"user_id", session.UserID,
	)
	delete(u.sessions, sessionID)
	return nil
}

// ActiveCount returns the number of active sessions for a user.
func (u *Usecase) ActiveCount(userID string) int {
	u.mu.Lock()
	defer u.mu.Unlock()

	return u.activeCountLocked(userID)
}

// CleanExpired removes sessions that have not sent a heartbeat within the timeout.
func (u *Usecase) CleanExpired() {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.cleanExpiredLocked()
}

// TotalActive returns the total number of active sessions.
func (u *Usecase) TotalActive() int {
	u.mu.Lock()
	defer u.mu.Unlock()

	return len(u.sessions)
}

func (u *Usecase) activeCountLocked(userID string) int {
	count := 0
	for _, s := range u.sessions {
		if s.UserID == userID {
			count++
		}
	}
	return count
}

func (u *Usecase) cleanExpiredLocked() {
	now := u.now()
	for id, s := range u.sessions {
		if s.IsExpired(u.heartbeatTimeout, now) {
			slog.Info("session expired, removing",
				"session_id", id,
				"user_id", s.UserID,
				"last_ping", s.LastPing,
			)
			delete(u.sessions, id)
		}
	}
}

func (u *Usecase) now() time.Time {
	if u.nowFunc != nil {
		return u.nowFunc()
	}
	return time.Now()
}
