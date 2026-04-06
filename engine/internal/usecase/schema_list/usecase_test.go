package schema_list

import (
	"context"
	"fmt"
	"testing"
)

type mockRepo struct {
	schemas []SchemaRecord
	err     error
}

func (m *mockRepo) List(_ context.Context) ([]SchemaRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.schemas, nil
}

func TestExecute_Success(t *testing.T) {
	repo := &mockRepo{
		schemas: []SchemaRecord{
			{ID: 1, Name: "schema-1", AgentNames: []string{"agent-a"}},
			{ID: 2, Name: "schema-2"},
		},
	}
	uc := New(repo)

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(result))
	}
	if result[0].Name != "schema-1" {
		t.Errorf("expected name %q, got %q", "schema-1", result[0].Name)
	}
	if len(result[0].AgentNames) != 1 {
		t.Errorf("expected 1 agent, got %d", len(result[0].AgentNames))
	}
}

func TestExecute_Empty(t *testing.T) {
	repo := &mockRepo{}
	uc := New(repo)

	result, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 schemas, got %d", len(result))
	}
}

func TestExecute_RepoError(t *testing.T) {
	repo := &mockRepo{err: fmt.Errorf("db failure")}
	uc := New(repo)

	_, err := uc.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
