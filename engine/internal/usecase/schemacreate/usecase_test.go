package schemacreate

import (
	"context"
	"fmt"
	"testing"
)

type mockRepo struct {
	schemas map[string]*SchemaRecord
	nextID  uint
	err     error
}

func newMockRepo() *mockRepo {
	return &mockRepo{schemas: make(map[string]*SchemaRecord), nextID: 1}
}

func (m *mockRepo) Create(_ context.Context, record *SchemaRecord) error {
	if m.err != nil {
		return m.err
	}
	if _, exists := m.schemas[record.Name]; exists {
		return fmt.Errorf("UNIQUE constraint: schema with name %q already exists", record.Name)
	}
	record.ID = m.nextID
	m.nextID++
	m.schemas[record.Name] = record
	return nil
}

func TestExecute_Success(t *testing.T) {
	repo := newMockRepo()
	uc := New(repo)

	out, err := uc.Execute(context.Background(), Input{Name: "test-schema", Description: "desc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if out.Name != "test-schema" {
		t.Errorf("expected name %q, got %q", "test-schema", out.Name)
	}
}

func TestExecute_EmptyName(t *testing.T) {
	repo := newMockRepo()
	uc := New(repo)

	_, err := uc.Execute(context.Background(), Input{Name: ""})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestExecute_DuplicateName(t *testing.T) {
	repo := newMockRepo()
	uc := New(repo)

	_, err := uc.Execute(context.Background(), Input{Name: "dup"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = uc.Execute(context.Background(), Input{Name: "dup"})
	if err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestExecute_RepoError(t *testing.T) {
	repo := newMockRepo()
	repo.err = fmt.Errorf("db failure")
	uc := New(repo)

	_, err := uc.Execute(context.Background(), Input{Name: "test"})
	if err == nil {
		t.Fatal("expected error")
	}
}
