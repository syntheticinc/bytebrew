package schema_update

import (
	"context"
	"fmt"
	"testing"
)

type mockRepo struct {
	updated map[uint]*SchemaRecord
	err     error
}

func newMockRepo() *mockRepo {
	return &mockRepo{updated: make(map[uint]*SchemaRecord)}
}

func (m *mockRepo) Update(_ context.Context, id uint, record *SchemaRecord) error {
	if m.err != nil {
		return m.err
	}
	m.updated[id] = record
	return nil
}

func TestExecute_Success(t *testing.T) {
	repo := newMockRepo()
	uc := New(repo)

	err := uc.Execute(context.Background(), Input{ID: 1, Name: "updated", Description: "new desc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rec, ok := repo.updated[1]
	if !ok {
		t.Fatal("expected update to be called")
	}
	if rec.Name != "updated" {
		t.Errorf("expected name %q, got %q", "updated", rec.Name)
	}
}

func TestExecute_ZeroID(t *testing.T) {
	uc := New(newMockRepo())
	err := uc.Execute(context.Background(), Input{ID: 0, Name: "test"})
	if err == nil {
		t.Fatal("expected error for zero id")
	}
}

func TestExecute_EmptyName(t *testing.T) {
	uc := New(newMockRepo())
	err := uc.Execute(context.Background(), Input{ID: 1, Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestExecute_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.err = fmt.Errorf("db failure")
	uc := New(repo)

	err := uc.Execute(context.Background(), Input{ID: 1, Name: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
}
