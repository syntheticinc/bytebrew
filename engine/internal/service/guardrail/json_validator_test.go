package guardrail

import (
	"context"
	"testing"
)

func TestJSONSchemaValidator_ValidJSON(t *testing.T) {
	v := NewJSONSchemaValidator("")
	result, err := v.Check(context.Background(), `{"key": "value"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for valid JSON")
	}
}

func TestJSONSchemaValidator_InvalidJSON(t *testing.T) {
	v := NewJSONSchemaValidator("")
	result, err := v.Check(context.Background(), "not json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail for invalid JSON")
	}
}

func TestJSONSchemaValidator_WithSchema_RequiredFields(t *testing.T) {
	schema := `{"type": "object", "required": ["name", "age"]}`
	v := NewJSONSchemaValidator(schema)

	// Valid: has both required fields
	result, err := v.Check(context.Background(), `{"name": "Alice", "age": 30}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected pass, got fail: %s", result.Reason)
	}

	// Invalid: missing required field
	result, err = v.Check(context.Background(), `{"name": "Alice"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail for missing required field")
	}
}

func TestJSONSchemaValidator_WithSchema_NoRequired(t *testing.T) {
	schema := `{"type": "object"}`
	v := NewJSONSchemaValidator(schema)

	result, err := v.Check(context.Background(), `{"anything": true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for object with no required fields")
	}
}

func TestJSONSchemaValidator_Array(t *testing.T) {
	v := NewJSONSchemaValidator("")
	result, err := v.Check(context.Background(), `[1, 2, 3]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for valid JSON array")
	}
}

func TestJSONSchemaValidator_EmptyString(t *testing.T) {
	v := NewJSONSchemaValidator("")
	result, err := v.Check(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail for empty string")
	}
}
