package domain

import (
	"testing"
)

func TestNewAgentRelation_Valid(t *testing.T) {
	r, err := NewAgentRelation("schema-1", "agent-a", "agent-b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.SchemaID != "schema-1" {
		t.Errorf("expected schema_id %q, got %q", "schema-1", r.SchemaID)
	}
	if r.SourceAgentID != "agent-a" {
		t.Errorf("expected source %q, got %q", "agent-a", r.SourceAgentID)
	}
	if r.TargetAgentID != "agent-b" {
		t.Errorf("expected target %q, got %q", "agent-b", r.TargetAgentID)
	}
}

func TestNewAgentRelation_SameSourceTarget(t *testing.T) {
	_, err := NewAgentRelation("s", "agent-a", "agent-a")
	if err == nil {
		t.Fatal("expected error for same source and target")
	}
}

func TestAgentRelation_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rel     AgentRelation
		wantErr bool
	}{
		{"valid", AgentRelation{SchemaID: "s", SourceAgentID: "a", TargetAgentID: "b"}, false},
		{"empty schema_id", AgentRelation{SourceAgentID: "a", TargetAgentID: "b"}, true},
		{"empty source", AgentRelation{SchemaID: "s", TargetAgentID: "b"}, true},
		{"empty target", AgentRelation{SchemaID: "s", SourceAgentID: "a"}, true},
		{"same src/tgt", AgentRelation{SchemaID: "s", SourceAgentID: "a", TargetAgentID: "a"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rel.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
