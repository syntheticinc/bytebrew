//go:build prompt

package prompt_regression

import (
	"context"
	"testing"
	"time"
)

// TestReadsProtoBeforeTask verifies that when a project tree shows .proto files,
// the agent reads them before creating a task. Catches the bug where agent
// saw .proto in tree but never read it, creating a task with generic description.
func TestReadsProtoBeforeTask(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	harness, err := NewHarness()
	if err != nil {
		t.Fatalf("NewHarness: %v", err)
	}

	if err := harness.BindSupervisorTools(ctx); err != nil {
		t.Fatalf("BindSupervisorTools: %v", err)
	}

	fixture, err := LoadFixture("grpc_client_proto_discovery_v1")
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

	// Should NOT create task yet — need to read proto and source files first
	AssertNoTaskCreation(t, result)

	// MUST read the .proto file — it's visible in the tree
	AssertReadsFileWithExtension(t, result, ".proto")

	// Should read at least 1 source file beyond proto (handler, server, etc.)
	// Note: the count may vary due to LLM non-determinism, but proto itself counts
	sourceCount := AssertReadsSourceFiles(t, result, 1)
	t.Logf("Agent read %d source files", sourceCount)
}

// TestStructuredTaskAfterDeepResearch verifies that after reading proto, server, handler,
// pubspec, main.go, and main.dart — the task description is structured with
// all required sections and references actual file paths from the codebase.
func TestStructuredTaskAfterDeepResearch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	harness, err := NewHarness()
	if err != nil {
		t.Fatalf("NewHarness: %v", err)
	}

	if err := harness.BindSupervisorTools(ctx); err != nil {
		t.Fatalf("BindSupervisorTools: %v", err)
	}

	fixture, err := LoadFixture("task_after_deep_research_v1")
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

	// Should create task now (research is done)
	AssertHasToolCall(t, result, "manage_tasks")

	// Formal check: task has structured sections
	AssertTaskDescriptionHasSections(t, result)

	// Formal check: task has file paths and context
	AssertTaskDescriptionHasContext(t, result)

	// LLM Judge: structured description quality
	description := extractManageTasksCreateDescription(result)
	if description == "" {
		t.Fatalf("could not extract description from manage_tasks(action=create)")
	}
	AssertJudgePass(t, harness, description, TaskDescriptionRubric)

	// LLM Judge: file paths must reference actual code from research
	AssertJudgePass(t, harness, description, TaskDescriptionWithFilePathsRubric)
}
