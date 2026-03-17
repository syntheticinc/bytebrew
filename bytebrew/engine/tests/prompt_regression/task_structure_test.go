//go:build prompt

package prompt_regression

import (
	"context"
	"testing"
	"time"
)

func TestTaskDescriptionREQ3(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	harness, err := NewHarness()
	if err != nil {
		t.Fatalf("NewHarness: %v", err)
	}

	if err := harness.BindSupervisorTools(ctx); err != nil {
		t.Fatalf("BindSupervisorTools: %v", err)
	}

	fixture, err := LoadFixture("task_structure_req3_v1")
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

	// Pre-check: manage_tasks(create) should be present
	AssertHasToolCall(t, result, "manage_tasks")

	// Formal pre-check
	AssertTaskDescriptionHasContext(t, result)

	// LLM Judge: semantic evaluation of task description quality
	description := extractManageTasksCreateDescription(result)
	if description == "" {
		t.Fatalf("could not extract description from manage_tasks(action=create)")
	}

	AssertJudgePass(t, harness, description, TaskDescriptionRubric)
}
