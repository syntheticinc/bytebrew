package schema_delete

import (
	"context"
	"fmt"
	"testing"
)

type mockRepo struct {
	deleted []uint
	err     error
}

func (m *mockRepo) Delete(_ context.Context, id uint) error {
	if m.err != nil {
		return m.err
	}
	m.deleted = append(m.deleted, id)
	return nil
}

func TestExecute_Success(t *testing.T) {
	repo := &mockRepo{}
	uc := New(repo)

	err := uc.Execute(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != 1 {
		t.Errorf("expected deleted [1], got %v", repo.deleted)
	}
}

func TestExecute_ZeroID(t *testing.T) {
	repo := &mockRepo{}
	uc := New(repo)

	err := uc.Execute(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for zero id")
	}
}

func TestExecute_RepoError(t *testing.T) {
	repo := &mockRepo{err: fmt.Errorf("db failure")}
	uc := New(repo)

	err := uc.Execute(context.Background(), 1)
	if err == nil {
		t.Fatal("expected error")
	}
}
