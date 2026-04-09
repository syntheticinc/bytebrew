package guardrail

import (
	"context"
	"testing"
)

// TestGuardrailPipeline_JSONSchema_Pass verifies that valid JSON output
// passes the JSON Schema guardrail check.
func TestGuardrailPipeline_JSONSchema_Pass(t *testing.T) {
	p := NewPipeline()
	schema := `{"type": "object", "required": ["name", "age"]}`
	p.RegisterChecker(ModeJSONSchema, NewJSONSchemaValidator(schema))

	result, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:      ModeJSONSchema,
		OnFailure: OnFailureError,
	}, `{"name": "Bob", "age": 25}`)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected pass, got fail with reason: %s", result.Reason)
	}
}

// TestGuardrailPipeline_JSONSchema_Fail_OnFailureError verifies that invalid JSON
// (missing required field) with OnFailure=error returns an error.
func TestGuardrailPipeline_JSONSchema_Fail_OnFailureError(t *testing.T) {
	p := NewPipeline()
	schema := `{"type": "object", "required": ["name", "age"]}`
	p.RegisterChecker(ModeJSONSchema, NewJSONSchemaValidator(schema))

	result, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:      ModeJSONSchema,
		OnFailure: OnFailureError,
	}, `{"name": "Bob"}`)

	// With on_failure=error, either an error is returned or the result is not passed
	if err != nil {
		// Expected: error returned for failed check
		return
	}
	if result.Passed {
		t.Error("expected check to fail for missing required field 'age'")
	}
}

// TestGuardrailPipeline_JSONSchema_Fail_InvalidJSON verifies that non-JSON output
// fails the JSON Schema check.
func TestGuardrailPipeline_JSONSchema_Fail_InvalidJSON(t *testing.T) {
	p := NewPipeline()
	p.RegisterChecker(ModeJSONSchema, NewJSONSchemaValidator(""))

	result, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:      ModeJSONSchema,
		OnFailure: OnFailureError,
	}, "this is not json")

	if err != nil {
		return // error path is acceptable
	}
	if result.Passed {
		t.Error("expected check to fail for non-JSON output")
	}
}
