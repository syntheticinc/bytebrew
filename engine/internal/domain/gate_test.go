package domain

import (
	"testing"
)

func TestNewGate_Valid(t *testing.T) {
	g, err := NewGate("schema-1", "join-gate", GateConditionAll)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g.SchemaID != "schema-1" {
		t.Errorf("expected schema_id %q, got %q", "schema-1", g.SchemaID)
	}
	if g.Name != "join-gate" {
		t.Errorf("expected name %q, got %q", "join-gate", g.Name)
	}
	if g.ConditionType != GateConditionAll {
		t.Errorf("expected condition_type %q, got %q", GateConditionAll, g.ConditionType)
	}
}

func TestNewGate_InvalidConditionType(t *testing.T) {
	_, err := NewGate("schema-1", "gate", GateConditionType("invalid"))
	if err == nil {
		t.Fatal("expected error for invalid condition type")
	}
}

func TestGate_Validate(t *testing.T) {
	tests := []struct {
		name    string
		gate    Gate
		wantErr bool
	}{
		{"valid all", Gate{SchemaID: "s", Name: "g", ConditionType: GateConditionAll}, false},
		{"valid any", Gate{SchemaID: "s", Name: "g", ConditionType: GateConditionAny}, false},
		{"valid custom", Gate{SchemaID: "s", Name: "g", ConditionType: GateConditionCustom}, false},
		{"empty schema_id", Gate{Name: "g", ConditionType: GateConditionAll}, true},
		{"empty name", Gate{SchemaID: "s", ConditionType: GateConditionAll}, true},
		{"invalid type", Gate{SchemaID: "s", Name: "g", ConditionType: "bad"}, true},
		{"negative max_iterations", Gate{SchemaID: "s", Name: "g", ConditionType: GateConditionAll, MaxIterations: -1}, true},
		{"negative timeout", Gate{SchemaID: "s", Name: "g", ConditionType: GateConditionAll, Timeout: -1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.gate.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
