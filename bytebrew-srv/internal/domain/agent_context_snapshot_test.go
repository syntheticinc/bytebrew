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
		FlowType:      FlowType("supervisor"),
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
		FlowType:      FlowType("supervisor"),
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
		FlowType:      FlowType("supervisor"),
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

func TestAgentContextSnapshot_Validate_MissingFlowType(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		SessionID:     "session-1",
		AgentID:       "supervisor",
		SchemaVersion: 1,
		Status:        AgentContextStatusActive,
	}

	err := snapshot.Validate()
	if err == nil {
		t.Error("expected error for missing flow_type, got nil")
	}
	if err.Error() != "flow_type is required" {
		t.Errorf("expected 'flow_type is required', got: %v", err)
	}
}

func TestAgentContextSnapshot_Validate_ZeroSchemaVersion(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		SessionID:     "session-1",
		AgentID:       "supervisor",
		FlowType:      FlowType("supervisor"),
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
		FlowType:      FlowType("supervisor"),
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
		AgentContextStatusSuspended,
		AgentContextStatusCompleted,
		AgentContextStatusInterrupted,
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

func TestAgentContextSnapshot_MarkInterrupted(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		Status:    AgentContextStatusActive,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	oldUpdatedAt := snapshot.UpdatedAt
	snapshot.MarkInterrupted()

	if snapshot.Status != AgentContextStatusInterrupted {
		t.Errorf("expected status to be interrupted, got: %s", snapshot.Status)
	}
	if !snapshot.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestAgentContextSnapshot_MarkSuspended(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		Status:    AgentContextStatusActive,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	oldUpdatedAt := snapshot.UpdatedAt
	snapshot.MarkSuspended()

	if snapshot.Status != AgentContextStatusSuspended {
		t.Errorf("expected status to be suspended, got: %s", snapshot.Status)
	}
	if !snapshot.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestAgentContextSnapshot_MarkCompleted(t *testing.T) {
	snapshot := &AgentContextSnapshot{
		Status:    AgentContextStatusActive,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
	}

	oldUpdatedAt := snapshot.UpdatedAt
	snapshot.MarkCompleted()

	if snapshot.Status != AgentContextStatusCompleted {
		t.Errorf("expected status to be completed, got: %s", snapshot.Status)
	}
	if !snapshot.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}
