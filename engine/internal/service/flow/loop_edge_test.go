package flow

import (
	"context"
	"strings"
	"testing"
)

// TestLoopEdge_MaxIterations_Respected verifies that a loop edge back to itself
// is bounded by the executor's depth guard (>50) and terminates with an error.
// The recursive nature of executeAgent → executeLoop → executeAgent means that
// MaxIterations on the gate controls the inner loop, while the depth guard
// prevents infinite recursion.
func TestLoopEdge_MaxIterations_Respected(t *testing.T) {
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": "not passing output",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[uint][]EdgeRecord{
			1: {
				{ID: 1, SchemaID: 1, SourceAgentName: "agent-a", TargetAgentName: "agent-a", Type: "loop"},
			},
		},
	}
	gateReader := &mockGateReader{
		gates: map[uint][]GateRecord{
			1: {
				{
					ID:            1,
					SchemaID:      1,
					Name:          "loop-gate",
					ConditionType: "all",
					MaxIterations: 3,
					Config: map[string]interface{}{
						"condition": "regex",
						"pattern":   "PASS",
					},
				},
			},
		},
	}
	eventStream := &mockEventStream{}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:    1,
		SessionID:   "session-loop",
		EventStream: eventStream,
	}, "agent-a", "start")

	// Must terminate with an error (depth exceeded or loop max iterations)
	if err == nil {
		t.Fatal("expected error when loop terminates")
	}

	// Error should mention depth or max iterations
	errMsg := err.Error()
	if !strings.Contains(errMsg, "depth exceeded") && !strings.Contains(errMsg, "max iterations") {
		t.Errorf("expected error about depth or max iterations, got: %s", errMsg)
	}

	// Execution should exist and have recorded steps
	if exec == nil {
		t.Fatal("expected non-nil execution even on loop failure")
	}
	if len(exec.Steps) == 0 {
		t.Error("expected at least 1 step recorded")
	}

	// The depth guard (50) should limit total calls
	callCount := int(runner.counter.Load())
	if callCount > 51 {
		t.Errorf("expected at most 51 runner calls (depth guard), got %d", callCount)
	}
}
