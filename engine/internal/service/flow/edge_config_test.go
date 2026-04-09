package flow

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestEdgeRouter_FieldMapping verifies that output JSON {"task": "analyze"}
// with mapping source=task target=input.result produces JSON containing "analyze".
func TestEdgeRouter_FieldMapping_SourceTarget(t *testing.T) {
	router := NewEdgeRouter()

	output := `{"task": "analyze"}`
	config := map[string]interface{}{
		"mode": "field_mapping",
		"mappings": []interface{}{
			map[string]interface{}{"source": "task", "target": "input.result"},
		},
	}

	result, err := router.RouteOutput(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "analyze") {
		t.Errorf("expected result to contain %q, got %q", "analyze", result)
	}

	// Verify it's valid JSON with the mapped field
	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	if resultData["result"] != "analyze" {
		t.Errorf("expected result field %q, got %v", "analyze", resultData["result"])
	}
}

// TestEdgeRouter_CustomPrompt verifies that output "hello" with template
// "Process: {{output}}" produces "Process: hello".
func TestEdgeRouter_CustomPrompt_OutputVar(t *testing.T) {
	router := NewEdgeRouter()

	output := "hello"
	config := map[string]interface{}{
		"mode":     "custom_prompt",
		"template": "Process: {{output}}",
	}

	result, err := router.RouteOutput(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Process: hello"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// TestEdgeRouter_FullPassthrough verifies that with no config the output
// is returned unchanged.
func TestEdgeRouter_FullPassthrough(t *testing.T) {
	router := NewEdgeRouter()

	output := "raw text"
	result, err := router.RouteOutput(output, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != output {
		t.Errorf("expected %q, got %q", output, result)
	}
}
