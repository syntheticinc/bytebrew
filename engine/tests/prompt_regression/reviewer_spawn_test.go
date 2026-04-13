//go:build prompt

package prompt_regression

import (
	"context"
	"testing"
	"time"
)

func TestReviewerSpawnAfterCoder(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	harness, err := NewHarness()
	if err != nil {
		t.Fatalf("NewHarness: %v", err)
	}

	if err := harness.BindSupervisorTools(ctx); err != nil {
		t.Fatalf("BindSupervisorTools: %v", err)
	}

	fixture, err := LoadFixture("reviewer_after_coder_v1")
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

	// After coder completes significant changes, supervisor should spawn reviewer
	AssertHasToolCall(t, result, "spawn_agent")
	AssertHasReviewerSpawn(t, result)
}
