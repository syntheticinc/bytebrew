package assistant

import (
	"context"
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

type mockAdminOps struct {
	schemas  []string
	agents   []string
	edges    []struct{ source, target, edgeType string }
	triggers []string
	err      error
}

func (m *mockAdminOps) CreateSchema(_ context.Context, name, description string) (uint, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.schemas = append(m.schemas, name)
	return uint(len(m.schemas)), nil
}

func (m *mockAdminOps) CreateAgent(_ context.Context, name, systemPrompt, model string) error {
	if m.err != nil {
		return m.err
	}
	m.agents = append(m.agents, name)
	return nil
}

func (m *mockAdminOps) AddAgentToSchema(_ context.Context, schemaID uint, agentName string) error {
	return m.err
}

func (m *mockAdminOps) CreateEdge(_ context.Context, schemaID uint, source, target, edgeType string) error {
	if m.err != nil {
		return m.err
	}
	m.edges = append(m.edges, struct{ source, target, edgeType string }{source, target, edgeType})
	return nil
}

func (m *mockAdminOps) CreateTrigger(_ context.Context, agentName, triggerType string) error {
	m.triggers = append(m.triggers, agentName)
	return m.err
}

type mockEventStream struct {
	events []*domain.AgentEvent
}

func (m *mockEventStream) Send(event *domain.AgentEvent) error {
	m.events = append(m.events, event)
	return nil
}

func TestAssembler_PlanFromInterview_Simple(t *testing.T) {
	assembler := NewAssembler(&mockAdminOps{})

	interview := NewInterviewState()
	interview.Channels = []string{"chat"}
	interview.Queries = []string{"delivery", "returns"}
	interview.SchemaName = "Support Flow"

	plan := assembler.PlanFromInterview(interview)

	if plan.SchemaName != "support-flow" {
		t.Errorf("expected slugified name %q, got %q", "support-flow", plan.SchemaName)
	}
	// 2 queries → single agent (≤2)
	if len(plan.Agents) != 1 {
		t.Errorf("expected 1 agent for simple flow, got %d", len(plan.Agents))
	}
}

func TestAssembler_PlanFromInterview_Complex(t *testing.T) {
	assembler := NewAssembler(&mockAdminOps{})

	interview := NewInterviewState()
	interview.Channels = []string{"chat", "email"}
	interview.Queries = []string{"delivery", "returns", "sizing"}
	interview.Integrations = []string{"Google Sheets"}
	interview.SchemaName = "Support"

	plan := assembler.PlanFromInterview(interview)

	// 3 queries → classifier + 3 handlers + escalation = 5
	if len(plan.Agents) != 5 {
		t.Errorf("expected 5 agents (classifier + 3 handlers + escalation), got %d", len(plan.Agents))
	}

	// 3 edges (classifier → each handler)
	if len(plan.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(plan.Edges))
	}

	if plan.Trigger == nil {
		t.Error("expected trigger to be created")
	}
}

func TestAssembler_Execute(t *testing.T) {
	ops := &mockAdminOps{}
	assembler := NewAssembler(ops)
	stream := &mockEventStream{}

	plan := &AssemblyPlan{
		SchemaName:  "test-schema",
		Description: "Test",
		Agents: []PlannedAgent{
			{Name: "agent-a", SystemPrompt: "You are A", Role: "handler"},
			{Name: "agent-b", SystemPrompt: "You are B", Role: "handler"},
		},
		Edges: []PlannedEdge{
			{Source: "agent-a", Target: "agent-b", Type: "flow"},
		},
		Trigger: &PlannedTrigger{AgentName: "agent-a", Type: "webhook"},
	}

	err := assembler.Execute(context.Background(), plan, stream)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ops.schemas) != 1 {
		t.Errorf("expected 1 schema, got %d", len(ops.schemas))
	}
	if len(ops.agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(ops.agents))
	}
	if len(ops.edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(ops.edges))
	}

	// Check SSE events: 2 node_create + 1 edge_create = 3
	nodeCreates := 0
	edgeCreates := 0
	for _, e := range stream.events {
		switch AdminEventType(e.Type) {
		case AdminEventNodeCreate:
			nodeCreates++
		case AdminEventEdgeCreate:
			edgeCreates++
		}
	}
	if nodeCreates != 2 {
		t.Errorf("expected 2 node_create events, got %d", nodeCreates)
	}
	if edgeCreates != 1 {
		t.Errorf("expected 1 edge_create event, got %d", edgeCreates)
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Support Flow", "support-flow"},
		{"My Schema 123", "my-schema-123"},
		{"test", "test"},
		{"Hello World!", "hello-world"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.expected {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
