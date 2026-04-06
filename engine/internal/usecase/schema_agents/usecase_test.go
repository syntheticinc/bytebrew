package schema_agents

import (
	"context"
	"fmt"
	"testing"
)

type mockRepo struct {
	agents       map[uint][]string // schemaID -> agent names
	schemasByAgent map[string][]string // agentName -> schema names
	addErr       error
	removeErr    error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		agents:         make(map[uint][]string),
		schemasByAgent: make(map[string][]string),
	}
}

func (m *mockRepo) AddAgent(_ context.Context, schemaID uint, agentName string) error {
	if m.addErr != nil {
		return m.addErr
	}
	m.agents[schemaID] = append(m.agents[schemaID], agentName)
	return nil
}

func (m *mockRepo) RemoveAgent(_ context.Context, schemaID uint, agentName string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	agents := m.agents[schemaID]
	for i, a := range agents {
		if a == agentName {
			m.agents[schemaID] = append(agents[:i], agents[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("not found")
}

func (m *mockRepo) ListAgents(_ context.Context, schemaID uint) ([]string, error) {
	return m.agents[schemaID], nil
}

func (m *mockRepo) ListSchemasForAgent(_ context.Context, agentName string) ([]string, error) {
	return m.schemasByAgent[agentName], nil
}

func TestAddAgent_Success(t *testing.T) {
	repo := newMockRepo()
	uc := New(repo)

	err := uc.AddAgent(context.Background(), 1, "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.agents[1]) != 1 || repo.agents[1][0] != "agent-a" {
		t.Errorf("expected [agent-a], got %v", repo.agents[1])
	}
}

func TestAddAgent_EmptyName(t *testing.T) {
	uc := New(newMockRepo())
	err := uc.AddAgent(context.Background(), 1, "")
	if err == nil {
		t.Fatal("expected error for empty agent name")
	}
}

func TestAddAgent_ZeroSchemaID(t *testing.T) {
	uc := New(newMockRepo())
	err := uc.AddAgent(context.Background(), 0, "agent-a")
	if err == nil {
		t.Fatal("expected error for zero schema id")
	}
}

func TestRemoveAgent_Success(t *testing.T) {
	repo := newMockRepo()
	repo.agents[1] = []string{"agent-a", "agent-b"}
	uc := New(repo)

	err := uc.RemoveAgent(context.Background(), 1, "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.agents[1]) != 1 || repo.agents[1][0] != "agent-b" {
		t.Errorf("expected [agent-b], got %v", repo.agents[1])
	}
}

func TestListAgents_Success(t *testing.T) {
	repo := newMockRepo()
	repo.agents[1] = []string{"agent-a", "agent-b"}
	uc := New(repo)

	names, err := uc.ListAgents(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(names))
	}
}

func TestListSchemasForAgent_Success(t *testing.T) {
	repo := newMockRepo()
	repo.schemasByAgent["agent-a"] = []string{"schema-1", "schema-2"}
	uc := New(repo)

	names, err := uc.ListSchemasForAgent(context.Background(), "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(names))
	}
}
