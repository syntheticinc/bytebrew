package tools

import (
	"context"
	"testing"
)

func TestParseManagePlanArgs_StepsAsArray(t *testing.T) {
	ctx := context.Background()

	// Normal case: steps as array
	input := `{
		"goal": "Test goal",
		"steps": [
			{"index": 0, "description": "Step 1", "status": "pending"},
			{"index": 1, "description": "Step 2", "status": "in_progress"}
		]
	}`

	args, err := parseManagePlanArgs(ctx, input)
	if err != nil {
		t.Fatalf("parseManagePlanArgs() unexpected error: %v", err)
	}

	if args.Goal != "Test goal" {
		t.Errorf("parseManagePlanArgs() Goal = %q, want 'Test goal'", args.Goal)
	}

	if len(args.Steps) != 2 {
		t.Fatalf("parseManagePlanArgs() Steps length = %d, want 2", len(args.Steps))
	}

	if args.Steps[0].Description != "Step 1" {
		t.Errorf("parseManagePlanArgs() Steps[0].Description = %q, want 'Step 1'", args.Steps[0].Description)
	}

	if args.Steps[1].Status != "in_progress" {
		t.Errorf("parseManagePlanArgs() Steps[1].Status = %q, want 'in_progress'", args.Steps[1].Status)
	}
}

func TestParseManagePlanArgs_StepsAsString(t *testing.T) {
	ctx := context.Background()

	// LLM sometimes sends steps as JSON string instead of array
	input := `{
		"goal": "Test goal",
		"steps": "[{\"index\": 0, \"description\": \"Step 1\", \"status\": \"pending\"}, {\"index\": 1, \"description\": \"Step 2\", \"status\": \"completed\"}]"
	}`

	args, err := parseManagePlanArgs(ctx, input)
	if err != nil {
		t.Fatalf("parseManagePlanArgs() unexpected error: %v", err)
	}

	if args.Goal != "Test goal" {
		t.Errorf("parseManagePlanArgs() Goal = %q, want 'Test goal'", args.Goal)
	}

	if len(args.Steps) != 2 {
		t.Fatalf("parseManagePlanArgs() Steps length = %d, want 2", len(args.Steps))
	}

	if args.Steps[0].Index != 0 {
		t.Errorf("parseManagePlanArgs() Steps[0].Index = %d, want 0", args.Steps[0].Index)
	}

	if args.Steps[0].Description != "Step 1" {
		t.Errorf("parseManagePlanArgs() Steps[0].Description = %q, want 'Step 1'", args.Steps[0].Description)
	}

	if args.Steps[1].Status != "completed" {
		t.Errorf("parseManagePlanArgs() Steps[1].Status = %q, want 'completed'", args.Steps[1].Status)
	}
}

func TestParseManagePlanArgs_StepsAsStringWithUnicode(t *testing.T) {
	ctx := context.Background()

	// Real-world case: LLM sends steps as string with unicode escapes
	input := `{
		"goal": "\u041d\u0430\u0439\u0442\u0438 \u043d\u0430\u0440\u0443\u0448\u0435\u043d\u0438\u044f",
		"steps": "[{\"index\": 0, \"description\": \"\u0410\u043d\u0430\u043b\u0438\u0437 SRP\", \"status\": \"pending\"}]"
	}`

	args, err := parseManagePlanArgs(ctx, input)
	if err != nil {
		t.Fatalf("parseManagePlanArgs() unexpected error: %v", err)
	}

	// Goal should be decoded from unicode
	expectedGoal := "Найти нарушения"
	if args.Goal != expectedGoal {
		t.Errorf("parseManagePlanArgs() Goal = %q, want %q", args.Goal, expectedGoal)
	}

	if len(args.Steps) != 1 {
		t.Fatalf("parseManagePlanArgs() Steps length = %d, want 1", len(args.Steps))
	}

	expectedDesc := "Анализ SRP"
	if args.Steps[0].Description != expectedDesc {
		t.Errorf("parseManagePlanArgs() Steps[0].Description = %q, want %q", args.Steps[0].Description, expectedDesc)
	}
}

func TestParseManagePlanArgs_InvalidJSON(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "completely invalid JSON",
			input: `not json at all`,
		},
		{
			name:  "missing closing brace",
			input: `{"goal": "Test"`,
		},
		{
			name:  "invalid steps format",
			input: `{"goal": "Test", "steps": 123}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseManagePlanArgs(ctx, tt.input)
			if err == nil {
				t.Error("parseManagePlanArgs() expected error, got nil")
			}
		})
	}
}

func TestParseManagePlanArgs_EmptySteps(t *testing.T) {
	ctx := context.Background()

	input := `{"goal": "Test", "steps": []}`

	args, err := parseManagePlanArgs(ctx, input)
	if err != nil {
		t.Fatalf("parseManagePlanArgs() unexpected error: %v", err)
	}

	if len(args.Steps) != 0 {
		t.Errorf("parseManagePlanArgs() Steps length = %d, want 0", len(args.Steps))
	}
}

func TestParseManagePlanArgs_StepsAsEmptyString(t *testing.T) {
	ctx := context.Background()

	input := `{"goal": "Test", "steps": "[]"}`

	args, err := parseManagePlanArgs(ctx, input)
	if err != nil {
		t.Fatalf("parseManagePlanArgs() unexpected error: %v", err)
	}

	if len(args.Steps) != 0 {
		t.Errorf("parseManagePlanArgs() Steps length = %d, want 0", len(args.Steps))
	}
}

func TestParseManagePlanArgs_StepsStringInvalidContent(t *testing.T) {
	ctx := context.Background()

	// Steps is a string but contains invalid JSON
	input := `{"goal": "Test", "steps": "not valid json array"}`

	_, err := parseManagePlanArgs(ctx, input)
	if err == nil {
		t.Error("parseManagePlanArgs() expected error for invalid steps content, got nil")
	}
}

func TestParseManagePlanArgs_WithReasoning(t *testing.T) {
	ctx := context.Background()

	// Steps with reasoning field
	input := `{
		"goal": "Refactor code",
		"steps": [
			{"index": 0, "description": "Extract method", "status": "pending", "reasoning": "Reduces complexity"}
		]
	}`

	args, err := parseManagePlanArgs(ctx, input)
	if err != nil {
		t.Fatalf("parseManagePlanArgs() unexpected error: %v", err)
	}

	if args.Steps[0].Reasoning != "Reduces complexity" {
		t.Errorf("parseManagePlanArgs() Steps[0].Reasoning = %q, want 'Reduces complexity'", args.Steps[0].Reasoning)
	}
}

func TestParseStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pending", "pending"},
		{"in_progress", "in_progress"},
		{"completed", "completed"},
		{"invalid", "pending"}, // defaults to pending
		{"", "pending"},        // empty defaults to pending
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseStatus(tt.input)
			if string(result) != tt.expected {
				t.Errorf("parseStatus(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
