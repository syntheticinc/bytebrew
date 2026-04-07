package domain

import (
	"testing"
)

func TestNewFlowExecution(t *testing.T) {
	fe := NewFlowExecution("schema-1", "session-1")
	if fe.SchemaID != "schema-1" {
		t.Errorf("expected schema_id %q, got %q", "schema-1", fe.SchemaID)
	}
	if fe.Status != FlowExecPending {
		t.Errorf("expected pending, got %s", fe.Status)
	}
}

func TestFlowExecution_Start(t *testing.T) {
	fe := NewFlowExecution("s", "sess")
	if err := fe.Start(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fe.Status != FlowExecRunning {
		t.Errorf("expected running, got %s", fe.Status)
	}
}

func TestFlowExecution_Start_NotPending(t *testing.T) {
	fe := NewFlowExecution("s", "sess")
	fe.Start()
	if err := fe.Start(); err == nil {
		t.Fatal("expected error starting non-pending flow")
	}
}

func TestFlowExecution_Complete(t *testing.T) {
	fe := NewFlowExecution("s", "sess")
	fe.Start()
	fe.Complete()
	if fe.Status != FlowExecCompleted {
		t.Errorf("expected completed, got %s", fe.Status)
	}
}

func TestFlowExecution_Fail(t *testing.T) {
	fe := NewFlowExecution("s", "sess")
	fe.Start()
	fe.Fail()
	if fe.Status != FlowExecFailed {
		t.Errorf("expected failed, got %s", fe.Status)
	}
}

func TestFlowExecution_AddStep(t *testing.T) {
	fe := NewFlowExecution("s", "sess")
	step := fe.AddStep("agent-a")
	if step.AgentName != "agent-a" {
		t.Errorf("expected agent-a, got %s", step.AgentName)
	}
	if step.Status != StepStatusPending {
		t.Errorf("expected pending, got %s", step.Status)
	}
	if len(fe.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(fe.Steps))
	}
}

func TestFlowExecution_IsTerminal(t *testing.T) {
	tests := []struct {
		status   FlowExecutionStatus
		terminal bool
	}{
		{FlowExecPending, false},
		{FlowExecRunning, false},
		{FlowExecCompleted, true},
		{FlowExecFailed, true},
		{FlowExecCancelled, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			fe := &FlowExecution{Status: tt.status}
			if fe.IsTerminal() != tt.terminal {
				t.Errorf("IsTerminal() = %v, want %v", fe.IsTerminal(), tt.terminal)
			}
		})
	}
}

func TestNewFlowStepStartedEvent(t *testing.T) {
	e := NewFlowStepStartedEvent("agent-a", "session-1", 0)
	if e.Type != EventTypeFlowStepStarted {
		t.Errorf("expected type %q, got %q", EventTypeFlowStepStarted, e.Type)
	}
	if e.AgentID != "agent-a" {
		t.Errorf("expected agent_id %q, got %q", "agent-a", e.AgentID)
	}
}

func TestNewFlowGateEvaluatedEvent(t *testing.T) {
	e := NewFlowGateEvaluatedEvent("gate-1", true, "passed")
	if e.Type != EventTypeFlowGateEvaluated {
		t.Errorf("expected type %q, got %q", EventTypeFlowGateEvaluated, e.Type)
	}
	if passed, ok := e.Metadata["passed"].(bool); !ok || !passed {
		t.Error("expected passed=true in metadata")
	}
}
