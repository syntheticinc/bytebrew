package domain

import (
	"testing"
)

func TestNewEdge_Valid(t *testing.T) {
	e, err := NewEdge("schema-1", "agent-a", "agent-b", EdgeTypeFlow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if e.SchemaID != "schema-1" {
		t.Errorf("expected schema_id %q, got %q", "schema-1", e.SchemaID)
	}
	if e.SourceAgentName != "agent-a" {
		t.Errorf("expected source %q, got %q", "agent-a", e.SourceAgentName)
	}
	if e.TargetAgentName != "agent-b" {
		t.Errorf("expected target %q, got %q", "agent-b", e.TargetAgentName)
	}
}

func TestNewEdge_SameSourceTarget(t *testing.T) {
	_, err := NewEdge("s", "agent-a", "agent-a", EdgeTypeFlow)
	if err == nil {
		t.Fatal("expected error for same source and target on non-loop edge")
	}
}

func TestNewEdge_LoopSameSourceTarget(t *testing.T) {
	e, err := NewEdge("s", "agent-a", "agent-a", EdgeTypeLoop)
	if err != nil {
		t.Fatalf("unexpected error for loop edge with same source/target: %v", err)
	}
	if e.Type != EdgeTypeLoop {
		t.Errorf("expected type %q, got %q", EdgeTypeLoop, e.Type)
	}
}

func TestEdge_Validate(t *testing.T) {
	tests := []struct {
		name    string
		edge    Edge
		wantErr bool
	}{
		{"valid flow", Edge{SchemaID: "s", SourceAgentName: "a", TargetAgentName: "b", Type: EdgeTypeFlow}, false},
		{"valid transfer", Edge{SchemaID: "s", SourceAgentName: "a", TargetAgentName: "b", Type: EdgeTypeTransfer}, false},
		{"valid parallel", Edge{SchemaID: "s", SourceAgentName: "a", TargetAgentName: "b", Type: EdgeTypeParallel}, false},
		{"valid gate", Edge{SchemaID: "s", SourceAgentName: "a", TargetAgentName: "b", Type: EdgeTypeGate}, false},
		{"valid loop", Edge{SchemaID: "s", SourceAgentName: "a", TargetAgentName: "a", Type: EdgeTypeLoop}, false},
		{"empty schema_id", Edge{SourceAgentName: "a", TargetAgentName: "b", Type: EdgeTypeFlow}, true},
		{"empty source", Edge{SchemaID: "s", TargetAgentName: "b", Type: EdgeTypeFlow}, true},
		{"empty target", Edge{SchemaID: "s", SourceAgentName: "a", Type: EdgeTypeFlow}, true},
		{"invalid type", Edge{SchemaID: "s", SourceAgentName: "a", TargetAgentName: "b", Type: "bad"}, true},
		{"same src/tgt non-loop", Edge{SchemaID: "s", SourceAgentName: "a", TargetAgentName: "a", Type: EdgeTypeFlow}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.edge.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
