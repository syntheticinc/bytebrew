package flow

import (
	"encoding/json"
	"testing"
)

func TestEdgeRouter_FullOutput(t *testing.T) {
	router := NewEdgeRouter()

	output := "Hello from agent A"
	result, err := router.RouteOutput(output, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != output {
		t.Errorf("expected %q, got %q", output, result)
	}
}

func TestEdgeRouter_FullOutputExplicit(t *testing.T) {
	router := NewEdgeRouter()

	output := "Hello"
	config := map[string]interface{}{"mode": "full_output"}
	result, err := router.RouteOutput(output, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != output {
		t.Errorf("expected %q, got %q", output, result)
	}
}

func TestEdgeRouter_FieldMapping(t *testing.T) {
	router := NewEdgeRouter()

	outputData := map[string]interface{}{
		"task":     "implement feature",
		"priority": "high",
		"details":  "some details",
	}
	outputJSON, _ := json.Marshal(outputData)

	config := map[string]interface{}{
		"mode": "field_mapping",
		"mappings": []interface{}{
			map[string]interface{}{"source": "task", "target": "input.backend_task"},
			map[string]interface{}{"source": "priority", "target": "input.prio"},
		},
	}

	result, err := router.RouteOutput(string(outputJSON), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}

	if resultData["backend_task"] != "implement feature" {
		t.Errorf("expected backend_task %q, got %v", "implement feature", resultData["backend_task"])
	}
	if resultData["prio"] != "high" {
		t.Errorf("expected prio %q, got %v", "high", resultData["prio"])
	}
}

func TestEdgeRouter_FieldMapping_NonJSON(t *testing.T) {
	router := NewEdgeRouter()

	config := map[string]interface{}{
		"mode": "field_mapping",
		"mappings": []interface{}{
			map[string]interface{}{"source": "output", "target": "input.content"},
		},
	}

	result, err := router.RouteOutput("plain text output", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("result is not JSON: %v", err)
	}

	if resultData["content"] != "plain text output" {
		t.Errorf("expected content %q, got %v", "plain text output", resultData["content"])
	}
}

func TestEdgeRouter_CustomPrompt(t *testing.T) {
	router := NewEdgeRouter()

	outputData := map[string]interface{}{
		"task":    "build API",
		"context": "Go project",
	}
	outputJSON, _ := json.Marshal(outputData)

	config := map[string]interface{}{
		"mode":     "custom_prompt",
		"template": "Summarize: Task={{output.task}}, Context={{output.context}}",
	}

	result, err := router.RouteOutput(string(outputJSON), config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Summarize: Task=build API, Context=Go project"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEdgeRouter_CustomPrompt_FullOutput(t *testing.T) {
	router := NewEdgeRouter()

	config := map[string]interface{}{
		"mode":     "custom_prompt",
		"template": "Process this: {{output}}",
	}

	result, err := router.RouteOutput("raw output text", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Process this: raw output text"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEdgeRouter_UnknownMode(t *testing.T) {
	router := NewEdgeRouter()

	config := map[string]interface{}{"mode": "unknown_mode"}
	result, err := router.RouteOutput("output", config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "output" {
		t.Errorf("expected fallback to full output, got %q", result)
	}
}
