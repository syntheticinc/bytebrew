package guardrail

import (
	"context"
	"encoding/json"
	"fmt"
)

// JSONSchemaValidator validates agent output against a JSON Schema.
type JSONSchemaValidator struct {
	schema string
}

// NewJSONSchemaValidator creates a new JSONSchemaValidator.
func NewJSONSchemaValidator(schema string) *JSONSchemaValidator {
	return &JSONSchemaValidator{schema: schema}
}

// Check validates the output is valid JSON and optionally matches a schema.
func (v *JSONSchemaValidator) Check(_ context.Context, output string) (*CheckResult, error) {
	// Step 1: Check if output is valid JSON
	var parsed interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		return &CheckResult{
			Passed: false,
			Reason: fmt.Sprintf("output is not valid JSON: %s", err.Error()),
		}, nil
	}

	// Step 2: If a schema is provided, validate against it
	if v.schema != "" {
		if err := v.validateAgainstSchema(output); err != nil {
			return &CheckResult{
				Passed: false,
				Reason: fmt.Sprintf("JSON Schema validation failed: %s", err.Error()),
			}, nil
		}
	}

	return &CheckResult{Passed: true, Reason: "valid JSON"}, nil
}

// validateAgainstSchema validates output against the configured JSON Schema.
// For V2, we do basic structural validation: check required fields exist.
func (v *JSONSchemaValidator) validateAgainstSchema(output string) error {
	var schemaObj map[string]interface{}
	if err := json.Unmarshal([]byte(v.schema), &schemaObj); err != nil {
		return fmt.Errorf("invalid JSON Schema: %w", err)
	}

	var outputObj map[string]interface{}
	if err := json.Unmarshal([]byte(output), &outputObj); err != nil {
		return fmt.Errorf("output is not a JSON object: %w", err)
	}

	// Check type
	if expectedType, ok := schemaObj["type"].(string); ok {
		if expectedType == "object" {
			// Check required fields
			if requiredRaw, ok := schemaObj["required"]; ok {
				required, ok := requiredRaw.([]interface{})
				if !ok {
					return nil
				}
				for _, fieldRaw := range required {
					field, ok := fieldRaw.(string)
					if !ok {
						continue
					}
					if _, exists := outputObj[field]; !exists {
						return fmt.Errorf("missing required field: %s", field)
					}
				}
			}
		}
	}

	return nil
}
