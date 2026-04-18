package domain

import (
	"fmt"
	"time"
)

// AgentContextStatus represents the lifecycle status of an agent's context.
// Values must match target-schema.dbml agent_context_snapshots.status CHECK:
//
//	active | compacted | expired
type AgentContextStatus string

const (
	AgentContextStatusActive    AgentContextStatus = "active"
	AgentContextStatusCompacted AgentContextStatus = "compacted"
	AgentContextStatusExpired   AgentContextStatus = "expired"
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
	case AgentContextStatusActive, AgentContextStatusCompacted, AgentContextStatusExpired:
		return true
	}
	return false
}

// IsCompatible returns true if snapshot's schema version matches current.
// Incompatible snapshots should be discarded (fresh start).
func (s *AgentContextSnapshot) IsCompatible() bool {
	return s.SchemaVersion == CurrentSchemaVersion
}

// MarkExpired marks the snapshot as expired (paused / interrupted / stale).
// Replaces the previous Suspended/Interrupted semantics — both map to "expired"
// in the DBML-aligned enum (no longer distinguishable at the storage layer).
func (s *AgentContextSnapshot) MarkExpired() {
	s.Status = AgentContextStatusExpired
	s.UpdatedAt = time.Now()
}

// MarkCompacted marks the snapshot as compacted (context was summarized/collapsed).
// Replaces the previous Completed semantics.
func (s *AgentContextSnapshot) MarkCompacted() {
	s.Status = AgentContextStatusCompacted
	s.UpdatedAt = time.Now()
}
