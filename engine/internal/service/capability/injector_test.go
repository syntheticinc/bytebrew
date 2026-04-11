package capability

import (
	"context"
	"testing"
)

type mockCapReader struct {
	caps map[string][]CapabilityRecord
	err  error
}

func (m *mockCapReader) ListEnabledByAgent(_ context.Context, agentName string) ([]CapabilityRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.caps[agentName], nil
}

func TestInjectedTools_Memory(t *testing.T) {
	reader := &mockCapReader{
		caps: map[string][]CapabilityRecord{
			"agent-a": {
				{ID: "1", AgentName: "agent-a", Type: "memory", Enabled: true},
			},
		},
	}
	inj := NewInjector(reader)

	tools, err := inj.InjectedTools(context.Background(), "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d: %v", len(tools), tools)
	}
	if tools[0] != "memory_recall" || tools[1] != "memory_store" {
		t.Errorf("expected [memory_recall, memory_store], got %v", tools)
	}
}

func TestInjectedTools_Multiple(t *testing.T) {
	reader := &mockCapReader{
		caps: map[string][]CapabilityRecord{
			"agent-a": {
				{ID: "1", AgentName: "agent-a", Type: "memory", Enabled: true},
				{ID: "2", AgentName: "agent-a", Type: "knowledge", Enabled: true},
				{ID: "3", AgentName: "agent-a", Type: "escalation", Enabled: true},
			},
		},
	}
	inj := NewInjector(reader)

	tools, err := inj.InjectedTools(context.Background(), "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// memory_recall, memory_store, knowledge_search, escalate = 4
	if len(tools) != 4 {
		t.Fatalf("expected 4 tools, got %d: %v", len(tools), tools)
	}
}

func TestInjectedTools_NoDuplicates(t *testing.T) {
	reader := &mockCapReader{
		caps: map[string][]CapabilityRecord{
			"agent-a": {
				{ID: "1", AgentName: "agent-a", Type: "memory", Enabled: true},
				{ID: "2", AgentName: "agent-a", Type: "memory", Enabled: true}, // duplicate type
			},
		},
	}
	inj := NewInjector(reader)

	tools, err := inj.InjectedTools(context.Background(), "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools (no duplicates), got %d: %v", len(tools), tools)
	}
}

func TestInjectedTools_NoCapabilities(t *testing.T) {
	reader := &mockCapReader{caps: map[string][]CapabilityRecord{}}
	inj := NewInjector(reader)

	tools, err := inj.InjectedTools(context.Background(), "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected 0 tools, got %d: %v", len(tools), tools)
	}
}

func TestInjectedTools_GuardrailNoTools(t *testing.T) {
	reader := &mockCapReader{
		caps: map[string][]CapabilityRecord{
			"agent-a": {
				{ID: "1", AgentName: "agent-a", Type: "guardrail", Enabled: true},
				{ID: "2", AgentName: "agent-a", Type: "output_schema", Enabled: true},
				{ID: "3", AgentName: "agent-a", Type: "policies", Enabled: true},
			},
		},
	}
	inj := NewInjector(reader)

	tools, err := inj.InjectedTools(context.Background(), "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tools) != 0 {
		t.Errorf("expected 0 tools for guardrail/schema/policies, got %d: %v", len(tools), tools)
	}
}
