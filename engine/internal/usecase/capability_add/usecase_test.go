package capability_add

import (
	"context"
	"fmt"
	"testing"
)

type mockRepo struct {
	caps   []*CapabilityRecord
	nextID uint
	err    error
}

func newMockRepo() *mockRepo {
	return &mockRepo{nextID: 1}
}

func (m *mockRepo) Create(_ context.Context, record *CapabilityRecord) error {
	if m.err != nil {
		return m.err
	}
	record.ID = m.nextID
	m.nextID++
	m.caps = append(m.caps, record)
	return nil
}

func TestExecute_Success(t *testing.T) {
	repo := newMockRepo()
	uc := New(repo)

	out, err := uc.Execute(context.Background(), Input{
		AgentName: "agent-a",
		Type:      "memory",
		Config:    map[string]interface{}{"retention_days": 30},
		Enabled:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if out.Type != "memory" {
		t.Errorf("expected type %q, got %q", "memory", out.Type)
	}
}

func TestExecute_InvalidType(t *testing.T) {
	uc := New(newMockRepo())
	_, err := uc.Execute(context.Background(), Input{
		AgentName: "agent",
		Type:      "invalid",
	})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestExecute_EmptyAgent(t *testing.T) {
	uc := New(newMockRepo())
	_, err := uc.Execute(context.Background(), Input{
		AgentName: "",
		Type:      "memory",
	})
	if err == nil {
		t.Fatal("expected error for empty agent name")
	}
}

func TestExecute_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.err = fmt.Errorf("db failure")
	uc := New(repo)

	_, err := uc.Execute(context.Background(), Input{
		AgentName: "agent",
		Type:      "memory",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
