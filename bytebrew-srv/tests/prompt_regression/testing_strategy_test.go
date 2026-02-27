//go:build prompt

package prompt_regression

import (
	"context"
	"testing"
	"time"
)

func TestTestingStrategyInSubtasks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	harness, err := NewHarness()
	if err != nil {
		t.Fatalf("NewHarness: %v", err)
	}

	if err := harness.BindSupervisorTools(ctx); err != nil {
		t.Fatalf("BindSupervisorTools: %v", err)
	}

	fixture, err := LoadFixture("testing_strategy_subtask_v1")
	if err != nil {
		t.Fatalf("LoadFixture: %v", err)
	}

	supervisorPrompt, err := LoadCurrentSupervisorPrompt()
	if err != nil {
		t.Fatalf("LoadCurrentSupervisorPrompt: %v", err)
	}

	// Append testing strategy context — simulates what TestingStrategyReminder injects
	testingStrategyContext := "\n\n**PROJECT TESTING STRATEGY:**\n" +
		"- Build: go build ./...\n" +
		"- Unit tests: go test ./... (pattern: *_test.go, framework: testing)\n" +
		"- Lint: golangci-lint run ./...\n" +
		"\nWhen creating subtask acceptance criteria, use these commands. Do not invent test commands.\n"
	combinedPrompt := supervisorPrompt + testingStrategyContext

	messages := harness.ReconstructMessages(&fixture.Snapshot, combinedPrompt)

	result, err := harness.Generate(ctx, messages)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Log all tool calls for visibility
	for _, tc := range result.ToolCalls {
		t.Logf("Tool call: %s\nArguments: %s\n", tc.Function.Name, tc.Function.Arguments)
	}

	// Supervisor should create subtasks with project-specific testing commands
	AssertHasToolCall(t, result, "manage_subtasks")
	AssertSubtaskUsesTestingCommands(t, result, []string{
		"go test",
		"go build",
		"golangci-lint",
	})
}
