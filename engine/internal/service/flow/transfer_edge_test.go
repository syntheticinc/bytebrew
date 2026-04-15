package flow

import (
	"context"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// TestTransferEdge_AgentA_HandsOff_To_AgentB verifies that a transfer edge
// causes agent A to terminate and agent B to continue with A's output.
func TestTransferEdge_AgentA_HandsOff_To_AgentB(t *testing.T) {
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": "handoff-payload",
			"agent-b": "final-result",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-b", Type: "transfer"},
			},
		},
	}
	eventStream := &mockEventStream{}

	executor := NewExecutor(runner, edgeReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:    "1",
		SessionID:   "session-transfer",
		EventStream: eventStream,
	}, "agent-a", "user input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}

	// Both A and B must have been called
	if len(exec.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(exec.Steps))
	}
	if exec.Steps[0].AgentName != "agent-a" {
		t.Errorf("step 0: expected agent-a, got %s", exec.Steps[0].AgentName)
	}
	if exec.Steps[1].AgentName != "agent-b" {
		t.Errorf("step 1: expected agent-b, got %s", exec.Steps[1].AgentName)
	}

	// Verify runner call order
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 runner calls, got %d", len(runner.calls))
	}
	if runner.calls[0] != "agent-a" {
		t.Errorf("first call: expected agent-a, got %s", runner.calls[0])
	}
	if runner.calls[1] != "agent-b" {
		t.Errorf("second call: expected agent-b, got %s", runner.calls[1])
	}

	// Verify A's output is recorded
	if exec.Steps[0].Output != "handoff-payload" {
		t.Errorf("step 0 output: expected %q, got %q", "handoff-payload", exec.Steps[0].Output)
	}

	// Verify B produced the final result
	if exec.Steps[1].Output != "final-result" {
		t.Errorf("step 1 output: expected %q, got %q", "final-result", exec.Steps[1].Output)
	}
}
