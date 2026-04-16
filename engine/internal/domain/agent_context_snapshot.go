package domain

import (
	"fmt"
	"time"
)

// AgentContextStatus represents the lifecycle status of an agent's context
type AgentContextStatus string

const (
	AgentContextStatusActive      AgentContextStatus = "active"
	AgentContextStatusSuspended   AgentContextStatus = "suspended"
	AgentContextStatusCompleted   AgentContextStatus = "completed"
	AgentContextStatusInterrupted AgentContextStatus = "interrupted"
)

// CurrentSchemaVersion is the version of the context data format.
// Increment when eino library updates break serialization compatibility.
const CurrentSchemaVersion = 1

// AgentContextSnapshot represents a serialized snapshot of an agent's full context.
// ContextData contains []*schema.Message as JSON blob — lossless, no conversion.
type AgentContextSnapshot struct {
	ID            string
	SessionID     string
	AgentID       string             // agent name or "supervisor"
	SchemaVersion int                // For detecting incompatible snapshots after eino updates
	ContextData   []byte             // JSON blob: serialized []*schema.Message
	StepNumber    int
	TokenCount    int
	Status        AgentContextStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Validate validates the AgentContextSnapshot
func (s *AgentContextSnapshot) Validate() error {
	if s.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if s.AgentID == "" {
		return fmt.Errorf("agent_id is required")
	}
	if s.SchemaVersion <= 0 {
		return fmt.Errorf("schema_version must be positive")
	}
	if !s.Status.IsValid() {
		return fmt.Errorf("invalid status: %s", s.Status)
	}
	return nil
}

// IsValid returns true if the status is valid
func (s AgentContextStatus) IsValid() bool {
	switch s {
	case AgentContextStatusActive, AgentContextStatusSuspended,
		AgentContextStatusCompleted, AgentContextStatusInterrupted:
		return true
	}
	return false
}

// IsCompatible returns true if snapshot's schema version matches current.
// Incompatible snapshots should be discarded (fresh start).
func (s *AgentContextSnapshot) IsCompatible() bool {
	return s.SchemaVersion == CurrentSchemaVersion
}

// MarkInterrupted marks the snapshot as interrupted (server crash recovery).
func (s *AgentContextSnapshot) MarkInterrupted() {
	s.Status = AgentContextStatusInterrupted
	s.UpdatedAt = time.Now()
}

// MarkSuspended marks the snapshot as suspended (agent paused).
func (s *AgentContextSnapshot) MarkSuspended() {
	s.Status = AgentContextStatusSuspended
	s.UpdatedAt = time.Now()
}

// MarkCompleted marks the snapshot as completed.
func (s *AgentContextSnapshot) MarkCompleted() {
	s.Status = AgentContextStatusCompleted
	s.UpdatedAt = time.Now()
}
