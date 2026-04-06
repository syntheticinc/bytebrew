package capability_list

import (
	"context"
	"fmt"
	"testing"
)

type mockRepo struct {
	caps map[string][]CapabilityRecord
	err  error
}

func (m *mockRepo) ListByAgent(_ context.Context, agentName string) ([]CapabilityRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.caps[agentName], nil
}

func TestExecute_Success(t *testing.T) {
	repo := &mockRepo{
		caps: map[string][]CapabilityRecord{
			"agent-a": {
				{ID: 1, Type: "memory", Enabled: true},
				{ID: 2, Type: "knowledge", Enabled: false},
			},
		},
	}
	uc := New(repo)

	result, err := uc.Execute(context.Background(), "agent-a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 capabilities, got %d", len(result))
	}
	if result[0].Type != "memory" {
		t.Errorf("expected type %q, got %q", "memory", result[0].Type)
	}
}

func TestExecute_EmptyAgent(t *testing.T) {
	uc := New(&mockRepo{})
	_, err := uc.Execute(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty agent name")
	}
}

func TestExecute_RepoError(t *testing.T) {
	repo := &mockRepo{err: fmt.Errorf("db failure")}
	uc := New(repo)

	_, err := uc.Execute(context.Background(), "agent-a")
	if err == nil {
		t.Fatal("expected error")
	}
}
