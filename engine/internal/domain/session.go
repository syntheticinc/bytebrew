package domain

import (
	"fmt"
	"time"
)

// SessionStatus represents the lifecycle stage of a session
type SessionStatus string

const (
	SessionActive    SessionStatus = "active"
	SessionSuspended SessionStatus = "suspended"
	SessionCompleted SessionStatus = "completed"
)

// Session represents a user session that can persist across server restarts
type Session struct {
	ID             string
	ProjectKey     string
	Status         SessionStatus
	TenantID       string
	SchemaID       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastActivityAt time.Time
}

// NewSession creates a new Session with validation
func NewSession(id, projectKey string) (*Session, error) {
	now := time.Now()
	session := &Session{
		ID:             id,
		ProjectKey:     projectKey,
		Status:         SessionActive,
		CreatedAt:      now,
		UpdatedAt:      now,
		LastActivityAt: now,
	}

	if err := session.Validate(); err != nil {
		return nil, err
	}

	return session, nil
}

// Validate validates the Session
func (s *Session) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("session id is required")
	}
	if s.ProjectKey == "" {
		return fmt.Errorf("project_key is required")
	}

	switch s.Status {
	case SessionActive, SessionSuspended, SessionCompleted:
		// Valid
	default:
		return fmt.Errorf("invalid session status: %s", s.Status)
	}

	return nil
}

// Activate transitions session to active status
func (s *Session) Activate() {
	s.Status = SessionActive
	s.UpdatedAt = time.Now()
	s.LastActivityAt = time.Now()
}

// Suspend transitions session to suspended status
func (s *Session) Suspend() {
	s.Status = SessionSuspended
	s.UpdatedAt = time.Now()
}

// Complete transitions session to completed status
func (s *Session) Complete() {
	s.Status = SessionCompleted
	s.UpdatedAt = time.Now()
}

// TouchActivity updates last activity timestamp
func (s *Session) TouchActivity() {
	s.LastActivityAt = time.Now()
	s.UpdatedAt = time.Now()
}

// IsTerminal returns true if the session is in a terminal state
func (s *Session) IsTerminal() bool {
	return s.Status == SessionCompleted
}
