package domain

import (
	"testing"
	"time"
)

func TestAgentContextSnapshot_Validate(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		ID:            "snap-1",
		SessionID:     "session-1",
		AgentID:       "supervisor",
		SchemaVersion: CurrentSchemaVersion,
		ContextData:   []byte(`[{"role":"user","content":"test"}]`),
		StepNumber:    5,
		TokenCount:    100,
		Status:        AgentContextStatusActive,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := snapshot.Validate(); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestAgentContextSnapshot_Validate_MissingSessionID(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		AgentID:       "supervisor",
		SchemaVersion: 1,
		Status:        AgentContextStatusActive,
	}

	err := snapshot.Validate()
	if err == nil {
		t.Error("expected error for missing session_id, got nil")
	}
	if err.Error() != "session_id is required" {
		t.Errorf("expected 'session_id is required', got: %v", err)
	}
}

func TestAgentContextSnapshot_Validate_MissingAgentID(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		SessionID:     "session-1",
		SchemaVersion: 1,
		Status:        AgentContextStatusActive,
	}

	err := snapshot.Validate()
	if err == nil {
		t.Error("expected error for missing agent_id, got nil")
	}
	if err.Error() != "agent_id is required" {
		t.Errorf("expected 'agent_id is required', got: %v", err)
	}
}

func TestAgentContextSnapshot_Validate_ZeroSchemaVersion(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		SessionID:     "session-1",
		AgentID:       "supervisor",
		SchemaVersion: 0,
		Status:        AgentContextStatusActive,
	}

	err := snapshot.Validate()
	if err == nil {
		t.Error("expected error for zero schema_version, got nil")
	}
	if err.Error() != "schema_version must be positive" {
		t.Errorf("expected 'schema_version must be positive', got: %v", err)
	}
}

func TestAgentContextSnapshot_Validate_InvalidStatus(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		SessionID:     "session-1",
		AgentID:       "supervisor",
		SchemaVersion: 1,
		Status:        AgentContextStatus("invalid"),
	}

	err := snapshot.Validate()
	if err == nil {
		t.Error("expected error for invalid status, got nil")
	}
	if err.Error() != "invalid status: invalid" {
		t.Errorf("expected 'invalid status: invalid', got: %v", err)
	}
}

func TestAgentContextStatus_IsValid(t *testing.T) {
	validStatuses := []AgentContextStatus{
		AgentContextStatusActive,
		AgentContextStatusCompacted,
		AgentContextStatusExpired,
	}

	for _, status := range validStatuses {
		if !status.IsValid() {
			t.Errorf("expected status %s to be valid", status)
		}
	}
}

func TestAgentContextStatus_IsValid_Invalid(t *testing.T) {
	invalidStatus := AgentContextStatus("invalid")
	if invalidStatus.IsValid() {
		t.Error("expected invalid status to return false")
	}
}

func TestAgentContextSnapshot_IsCompatible(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		SchemaVersion: CurrentSchemaVersion,
	}

	if !snapshot.IsCompatible() {
		t.Error("expected snapshot with current version to be compatible")
	}
}

func TestAgentContextSnapshot_IsCompatible_Mismatch(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		SchemaVersion: CurrentSchemaVersion + 1,
	}

	if snapshot.IsCompatible() {
		t.Error("expected snapshot with different version to be incompatible")
	}
}

func TestAgentContextSnapshot_MarkExpired(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		Status:    AgentContextStatusActive,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	oldUpdatedAt := snapshot.UpdatedAt
	snapshot.MarkExpired()

	if snapshot.Status != AgentContextStatusExpired {
		t.Errorf("expected status to be expired, got: %s", snapshot.Status)
	}
	if !snapshot.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestAgentContextSnapshot_MarkCompacted(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		Status:    AgentContextStatusActive,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	oldUpdatedAt := snapshot.UpdatedAt
	snapshot.MarkCompacted()

	if snapshot.Status != AgentContextStatusCompacted {
		t.Errorf("expected status to be compacted, got: %s", snapshot.Status)
	}
	if !snapshot.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}
