package domain

import (
	"testing"
)

func TestNewSchema_Valid(t *testing.T) {
	s, err := NewSchema("my-schema", "A test schema")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Name != "my-schema" {
		t.Errorf("expected name %q, got %q", "my-schema", s.Name)
	}
	if s.Description != "A test schema" {
		t.Errorf("expected description %q, got %q", "A test schema", s.Description)
	}
	if s.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestNewSchema_EmptyName(t *testing.T) {
	_, err := NewSchema("", "desc")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestSchema_Validate(t *testing.T) {
	tests := []struct {
		name    string
		schema  Schema
		wantErr bool
	}{
		{"valid", Schema{Name: "test"}, false},
		{"empty name", Schema{Name: ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.schema.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
