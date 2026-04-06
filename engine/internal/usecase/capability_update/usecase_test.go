package capability_update

import (
	"context"
	"fmt"
	"testing"
)

type mockRepo struct {
	updated map[uint]*CapabilityRecord
	err     error
}

func newMockRepo() *mockRepo {
	return &mockRepo{updated: make(map[uint]*CapabilityRecord)}
}

func (m *mockRepo) Update(_ context.Context, id uint, record *CapabilityRecord) error {
	if m.err != nil {
		return m.err
	}
	m.updated[id] = record
	return nil
}

func TestExecute_Success(t *testing.T) {
	repo := newMockRepo()
	uc := New(repo)

	err := uc.Execute(context.Background(), Input{
		ID:      1,
		Type:    "knowledge",
		Config:  map[string]interface{}{"top_k": 5},
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rec, ok := repo.updated[1]
	if !ok {
		t.Fatal("expected update to be called")
	}
	if rec.Type != "knowledge" {
		t.Errorf("expected type %q, got %q", "knowledge", rec.Type)
	}
}

func TestExecute_ZeroID(t *testing.T) {
	uc := New(newMockRepo())
	err := uc.Execute(context.Background(), Input{ID: 0, Type: "memory"})
	if err == nil {
		t.Fatal("expected error for zero id")
	}
}

func TestExecute_InvalidType(t *testing.T) {
	uc := New(newMockRepo())
	err := uc.Execute(context.Background(), Input{ID: 1, Type: "invalid"})
	if err == nil {
		t.Fatal("expected error for invalid type")
	}
}

func TestExecute_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.err = fmt.Errorf("db failure")
	uc := New(repo)

	err := uc.Execute(context.Background(), Input{ID: 1, Type: "memory"})
	if err == nil {
		t.Fatal("expected error")
	}
}
