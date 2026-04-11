package flow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// --- Mocks ---

type mockAgentRunner struct {
	outputs map[string]string // agentName -> output
	mu      sync.Mutex
	calls   []string // track call order (protected by mu for concurrent fork tests)
	err     error
	counter atomic.Int32
}

func (m *mockAgentRunner) RunAgent(_ context.Context, agentName, input, sessionID string, _ domain.AgentEventStream) (string, error) {
	m.counter.Add(1)
	m.mu.Lock()
	m.calls = append(m.calls, agentName)
	m.mu.Unlock()
	if m.err != nil {
		return "", m.err
	}
	if out, ok := m.outputs[agentName]; ok {
		return out, nil
	}
	return fmt.Sprintf("output from %s", agentName), nil
}

type mockEdgeReader struct {
	edges map[string][]EdgeRecord
}

func (m *mockEdgeReader) ListEdges(_ context.Context, schemaID string) ([]EdgeRecord, error) {
	return m.edges[schemaID], nil
}

type mockGateReader struct {
	gates map[string][]GateRecord
}

func (m *mockGateReader) GetGateByID(_ context.Context, id string) (*GateRecord, error) {
	for _, gates := range m.gates {
		for _, g := range gates {
			if g.ID == id {
				return &g, nil
			}
		}
	}
	return nil, fmt.Errorf("gate %s not found", id)
}

func (m *mockGateReader) ListGates(_ context.Context, schemaID string) ([]GateRecord, error) {
	return m.gates[schemaID], nil
}

type mockEventStream struct {
	events []*domain.AgentEvent
}

func (m *mockEventStream) Send(event *domain.AgentEvent) error {
	m.events = append(m.events, event)
	return nil
}

// --- Tests ---

func TestExecutor_LinearPipeline(t *testing.T) {
	// A → B → C
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": "output-a",
			"agent-b": "output-b",
			"agent-c": "output-c",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-b", Type: "flow"},
				{ID: "2", SchemaID: "1", SourceAgentName: "agent-b", TargetAgentName: "agent-c", Type: "flow"},
			},
		},
	}
	gateReader := &mockGateReader{gates: map[string][]GateRecord{}}
	eventStream := &mockEventStream{}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:    "1",
		SessionID:   "session-1",
		EventStream: eventStream,
	}, "agent-a", "user input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
	if len(exec.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(exec.Steps))
	}
	if exec.Steps[0].AgentName != "agent-a" {
		t.Errorf("step 0 expected agent-a, got %s", exec.Steps[0].AgentName)
	}
	if exec.Steps[2].AgentName != "agent-c" {
		t.Errorf("step 2 expected agent-c, got %s", exec.Steps[2].AgentName)
	}

	// Check SSE events: should have step_started + step_completed for each agent
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
	if stepStarted != 3 || stepCompleted != 3 {
		t.Errorf("expected 3 started + 3 completed events, got %d started, %d completed", stepStarted, stepCompleted)
	}
}

func TestExecutor_TransferEdge(t *testing.T) {
	// A --transfer--> B → C
	// Transfer: A hands off to B, A stops
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": "from-a",
			"agent-b": "from-b",
			"agent-c": "from-c",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-b", Type: "transfer"},
				{ID: "2", SchemaID: "1", SourceAgentName: "agent-b", TargetAgentName: "agent-c", Type: "flow"},
			},
		},
	}
	gateReader := &mockGateReader{gates: map[string][]GateRecord{}}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:  "1",
		SessionID: "session-1",
	}, "agent-a", "input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
	// A runs, transfers to B, B runs, B flows to C
	if len(exec.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(exec.Steps))
	}
}

func TestExecutor_ForkJoin(t *testing.T) {
	// A → [B, C] (parallel fork)
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": "from-a",
			"agent-b": "from-b",
			"agent-c": "from-c",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-b", Type: "flow"},
				{ID: "2", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-c", Type: "flow"},
			},
		},
	}
	gateReader := &mockGateReader{gates: map[string][]GateRecord{}}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:  "1",
		SessionID: "session-1",
	}, "agent-a", "input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
	// A runs, then B and C in parallel = 3 steps total
	if len(exec.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(exec.Steps))
	}
}

func TestExecutor_GatePass(t *testing.T) {
	// A → gate-1 → B
	// Gate passes (contains "SUCCESS")
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": "result: SUCCESS",
			"agent-b": "from-b",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "gate-1", Type: "gate"},
				{ID: "2", SchemaID: "1", SourceAgentName: "gate-1", TargetAgentName: "agent-b", Type: "flow"},
			},
		},
	}
	gateReader := &mockGateReader{
		gates: map[string][]GateRecord{
			"1": {
				{ID: "1", SchemaID: "1", Name: "gate-1", ConditionType: "all",
					Config: map[string]interface{}{"condition": "contains", "text": "SUCCESS"}},
			},
		},
	}
	eventStream := &mockEventStream{}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:    "1",
		SessionID:   "session-1",
		EventStream: eventStream,
	}, "agent-a", "input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}

	// Check gate evaluated event
	gateEvents := 0
	for _, e := range eventStream.events {
		if e.Type == domain.EventTypeFlowGateEvaluated {
			gateEvents++
			if passed, ok := e.Metadata["passed"].(bool); !ok || !passed {
				t.Error("expected gate to pass")
			}
		}
	}
	if gateEvents != 1 {
		t.Errorf("expected 1 gate evaluated event, got %d", gateEvents)
	}
}

func TestExecutor_GateFail_Block(t *testing.T) {
	// A → gate-1 → B
	// Gate fails (does not contain "SUCCESS"), action=block
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": "result: FAILURE",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "gate-1", Type: "gate"},
				{ID: "2", SchemaID: "1", SourceAgentName: "gate-1", TargetAgentName: "agent-b", Type: "flow"},
			},
		},
	}
	gateReader := &mockGateReader{
		gates: map[string][]GateRecord{
			"1": {
				{ID: "1", SchemaID: "1", Name: "gate-1", ConditionType: "all",
					Config: map[string]interface{}{
						"condition":  "contains",
						"text":       "SUCCESS",
						"on_failure": "block",
					}},
			},
		},
	}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:  "1",
		SessionID: "session-1",
	}, "agent-a", "input")

	if err == nil {
		t.Fatal("expected error for gate block")
	}
	if !strings.Contains(err.Error(), "condition failed") {
		t.Errorf("expected condition failed error, got: %v", err)
	}
	if exec.Status != domain.FlowExecFailed {
		t.Errorf("expected failed, got %s", exec.Status)
	}
}

func TestExecutor_GateFail_Skip(t *testing.T) {
	// A → gate-1 → B
	// Gate fails but action=skip → proceeds to B
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": "no match",
			"agent-b": "from-b",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "gate-1", Type: "gate"},
				{ID: "2", SchemaID: "1", SourceAgentName: "gate-1", TargetAgentName: "agent-b", Type: "flow"},
			},
		},
	}
	gateReader := &mockGateReader{
		gates: map[string][]GateRecord{
			"1": {
				{ID: "1", SchemaID: "1", Name: "gate-1", ConditionType: "all",
					Config: map[string]interface{}{
						"condition":  "contains",
						"text":       "MATCH",
						"on_failure": "skip",
					}},
			},
		},
	}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:  "1",
		SessionID: "session-1",
	}, "agent-a", "input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
	// A + B = 2 steps (gate is skipped)
	if len(exec.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(exec.Steps))
	}
}

func TestExecutor_SingleAgent_NoEdges(t *testing.T) {
	runner := &mockAgentRunner{
		outputs: map[string]string{"agent-a": "done"},
	}
	edgeReader := &mockEdgeReader{edges: map[string][]EdgeRecord{}}
	gateReader := &mockGateReader{gates: map[string][]GateRecord{}}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:  "1",
		SessionID: "session-1",
	}, "agent-a", "input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
	if len(exec.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(exec.Steps))
	}
}

func TestExecutor_AgentFailure(t *testing.T) {
	runner := &mockAgentRunner{err: fmt.Errorf("LLM error")}
	edgeReader := &mockEdgeReader{edges: map[string][]EdgeRecord{}}
	gateReader := &mockGateReader{gates: map[string][]GateRecord{}}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:  "1",
		SessionID: "session-1",
	}, "agent-a", "input")

	if err == nil {
		t.Fatal("expected error")
	}
	if exec.Status != domain.FlowExecFailed {
		t.Errorf("expected failed, got %s", exec.Status)
	}
}

func TestExecutor_EdgeRouting_FieldMapping(t *testing.T) {
	runner := &mockAgentRunner{
		outputs: map[string]string{
			"agent-a": `{"task": "build API", "priority": "high"}`,
			"agent-b": "processed",
		},
	}
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-b", Type: "flow",
					Config: map[string]interface{}{
						"mode": "field_mapping",
						"mappings": []interface{}{
							map[string]interface{}{"source": "task", "target": "input.backend_task"},
						},
					}},
			},
		},
	}
	gateReader := &mockGateReader{gates: map[string][]GateRecord{}}

	executor := NewExecutor(runner, edgeReader, gateReader)

	exec, err := executor.Execute(context.Background(), ExecutorConfig{
		SchemaID:  "1",
		SessionID: "session-1",
	}, "agent-a", "input")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exec.Status != domain.FlowExecCompleted {
		t.Errorf("expected completed, got %s", exec.Status)
	}
}

func TestExecutor_HasOutgoingEdges(t *testing.T) {
	edgeReader := &mockEdgeReader{
		edges: map[string][]EdgeRecord{
			"1": {
				{ID: "1", SchemaID: "1", SourceAgentName: "agent-a", TargetAgentName: "agent-b", Type: "flow"},
			},
		},
	}
	gateReader := &mockGateReader{gates: map[string][]GateRecord{}}

	executor := NewExecutor(nil, edgeReader, gateReader)

	has, err := executor.HasOutgoingEdges(context.Background(), "1", "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !has {
		t.Error("expected agent-a to have outgoing edges")
	}

	has, err = executor.HasOutgoingEdges(context.Background(), "1", "agent-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if has {
		t.Error("expected agent-b to have no outgoing edges")
	}
}
