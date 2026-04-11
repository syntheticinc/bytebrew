package flow

import (
	"context"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// TestFlowEdge_AgentA_to_AgentB tests a simple A → B flow edge:
// agentA is called with the user input, agentB is called with agentA's output.
func TestFlowEdge_AgentA_to_AgentB(t *testing.T) {
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agentA": "output from A",
			"agentB": "output from B",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agentA", TargetAgentName: "agentB", Type: "flow"},
			},
		},
	}
	gateReader := &mockGateReader{gates: map[string][]GateRecord{}}
	eventStream := &mockEventStream{}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:    "1",
		SessionID:   "test",
		EventStream: eventStream,
	}, "agentA", "hello")

	// No error expected
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Result must be non-nil
	if exec == nil {
		t.Fatal("expected non-nil FlowExecution result")
	}

	// Status must be completed
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected status %q, got %q", domain.FlowExecCompleted, exec.Status)
	}

	// Both agents must have been called
	if len(exec.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(exec.Steps))
	}

	// agentA was called first
	if exec.Steps[0].AgentName != "agentA" {
		t.Errorf("step 0: expected agentA, got %s", exec.Steps[0].AgentName)
	}

	// agentB was called second
	if exec.Steps[1].AgentName != "agentB" {
		t.Errorf("step 1: expected agentB, got %s", exec.Steps[1].AgentName)
	}

	// Verify call order recorded by mock runner
	if len(runner.calls) != 2 {
		t.Fatalf("expected 2 agent calls, got %d", len(runner.calls))
	}
	if runner.calls[0] != "agentA" {
		t.Errorf("first call: expected agentA, got %s", runner.calls[0])
	}
	if runner.calls[1] != "agentB" {
		t.Errorf("second call: expected agentB, got %s", runner.calls[1])
	}

	// agentA's step output must equal "output from A"
	if exec.Steps[0].Output != "output from A" {
		t.Errorf("step 0 output: expected %q, got %q", "output from A", exec.Steps[0].Output)
	}

	// agentB's step output must equal "output from B"
	if exec.Steps[1].Output != "output from B" {
		t.Errorf("step 1 output: expected %q, got %q", "output from B", exec.Steps[1].Output)
	}
}

// TestFlowEdge_EventStream verifies that flow.step_started and flow.step_completed
// events are emitted for each agent in the A → B pipeline.
func TestFlowEdge_EventStream(t *testing.T) {
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agentA": "output from A",
			"agentB": "output from B",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agentA", TargetAgentName: "agentB", Type: "flow"},
			},
		},
	}
	gateReader := &mockGateReader{gates: map[string][]GateRecord{}}
	eventStream := &mockEventStream{}

	executor := NewExecutor(runner, edgeReader, gateReader)

	_, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:    "1",
		SessionID:   "test",
		EventStream: eventStream,
	}, "agentA", "hello")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count flow step events
	stepStarted := 0
	stepCompleted := 0
	for _, e := range eventStream.events {
		if e.Type == domain.EventTypeFlowStepStarted {
			stepStarted++
		}
		if e.Type == domain.EventTypeFlowStepCompleted {
			stepCompleted++
		}
	}

	// Expect one started + one completed event per agent (2 agents total)
	if stepStarted < 1 {
		t.Errorf("expected at least 1 flow.step_started event, got %d", stepStarted)
	}
	if stepCompleted < 1 {
		t.Errorf("expected at least 1 flow.step_completed event, got %d", stepCompleted)
	}

	// Exact count: 2 started + 2 completed for A and B
	if stepStarted != 2 {
		t.Errorf("expected 2 flow.step_started events (one per agent), got %d", stepStarted)
	}
	if stepCompleted != 2 {
		t.Errorf("expected 2 flow.step_completed events (one per agent), got %d", stepCompleted)
	}
}
