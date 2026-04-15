package flow

import (
	"context"
	"sort"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// TestParallelExecution_Fork_ThreeAgents verifies that A → [B, C, D] in parallel
// executes all four agents and completes successfully.
func TestParallelExecution_Fork_ThreeAgents(t *testing.T) {
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": "from-a",
			"agent-b": "from-b",
			"agent-c": "from-c",
			"agent-d": "from-d",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-b", Type: "flow"},
				{ID: "2", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-c", Type: "flow"},
				{ID: "3", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-d", Type: "flow"},
			},
		},
	}
	eventStream := &mockEventStream{}

	executor := NewExecutor(runner, edgeReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:    "1",
		SessionID:   "session-fork",
		EventStream: eventStream,
	}, "agent-a", "input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}

	// A runs first, then B, C, D in parallel = 4 steps total
	if len(exec.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(exec.Steps))
	}

	// All 4 agents must have been called
	callCount := int(runner.counter.Load())
	if callCount != 4 {
		t.Errorf("expected 4 runner calls, got %d", callCount)
	}

	// Verify B, C, D are all present in the runner calls
	runner.mu.Lock()
	calls := make([]string, len(runner.calls))
	copy(calls, runner.calls)
	runner.mu.Unlock()

	sort.Strings(calls)
	expected := []string{"agent-a", "agent-b", "agent-c", "agent-d"}
	sort.Strings(expected)

	if len(calls) != len(expected) {
		t.Fatalf("expected calls %v, got %v", expected, calls)
	}
	for i := range expected {
		if calls[i] != expected[i] {
			t.Errorf("call %d: expected %s, got %s", i, expected[i], calls[i])
		}
	}
}
