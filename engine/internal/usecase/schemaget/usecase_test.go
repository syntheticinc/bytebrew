package schemaget

import (
	"context"
	"fmt"
	"testing"
)

type mockRepo struct {
	schemas map[uint]*SchemaRecord
	err     error
}

func (m *mockRepo) GetByID(_ context.Context, id uint) (*SchemaRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	rec, ok := m.schemas[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return rec, nil
}

func TestExecute_Success(t *testing.T) {
	repo := &mockRepo{schemas: map[uint]*SchemaRecord{
		1: {ID: 1, Name: "schema-1", AgentNames: []string{"agent-a"}},
	}}
	uc := New(repo)

	out, err := uc.Execute(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Name != "schema-1" {
		t.Errorf("expected name %q, got %q", "schema-1", out.Name)
	}
}

func TestExecute_ZeroID(t *testing.T) {
	repo := &mockRepo{schemas: map[uint]*SchemaRecord{}}
	uc := New(repo)

	_, err := uc.Execute(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for zero id")
	}
}

func TestExecute_NotFound(t *testing.T) {
	repo := &mockRepo{schemas: map[uint]*SchemaRecord{}}
	uc := New(repo)

	_, err := uc.Execute(context.Background(), 99)
	if err == nil {
		t.Fatal("expected error for missing schema")
	}
}
