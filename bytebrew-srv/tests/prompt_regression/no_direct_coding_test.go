//go:build prompt

package prompt_regression

import (
	"context"
	"testing"
	"time"
)

// TestNoDirectCodingForComplexTask verifies that the supervisor does NOT use write_file/edit_file
// directly for complex tasks. Instead, it should create a task via manage_tasks and delegate
// to code agents via spawn_code_agent.
//
// This catches the real bug: supervisor reads existing files, then starts writing code directly
// (write_file main.dart, edit_file pubspec.yaml) instead of creating a structured task.
func TestNoDirectCodingForComplexTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	harness, err := NewHarness()
	if err != nil {
		t.Fatalf("NewHarness: %v", err)
	}

	if err := harness.BindSupervisorTools(ctx); err != nil {
		t.Fatalf("BindSupervisorTools: %v", err)
	}

	fixture, err := LoadFixture("complex_task_no_direct_code_v1")
	if err != nil {
		t.Fatalf("LoadFixture: %v", err)
	}

	supervisorPrompt, err := LoadCurrentSupervisorPrompt()
	if err != nil {
		t.Fatalf("LoadCurrentSupervisorPrompt: %v", err)
	}

	messages := harness.ReconstructMessages(&fixture.Snapshot, supervisorPrompt)

	result, err := harness.Generate(ctx, messages)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Log all tool calls for visibility
	for _, tc := range result.ToolCalls {
		t.Logf("Tool call: %s\nArguments: %s\n", tc.Function.Name, tc.Function.Arguments)
	}

	// MUST NOT write code directly — complex task requires task management
	AssertNoDirectCoding(t, result)

	// SHOULD create a task via manage_tasks
	AssertHasToolCall(t, result, "manage_tasks")

	// Task should have structured description
	AssertTaskDescriptionHasSections(t, result)
}
