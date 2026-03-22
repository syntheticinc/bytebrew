//go:build prompt

package prompt_regression

import (
	"context"
	"testing"
	"time"
)

// TestSingleAskUserForAmbiguousRequest verifies that for an ambiguous product request:
// 1. Agent calls ask_user AT MOST once (not 2-3 times)
// 2. If ask_user is called, it combines multiple questions into one call
// 3. Questions are PRODUCT-level (what to build), not TECH-level (which framework)
// 4. Agent does NOT create task immediately without research
//
// This test catches the real bug: agent calling ask_user 3 times
// ("What platform?", "What features?", "What stack?") instead of one combined call.
func TestSingleAskUserForAmbiguousRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	harness, err := NewHarness()
	if err != nil {
		t.Fatalf("NewHarness: %v", err)
	}

	if err := harness.BindSupervisorTools(ctx); err != nil {
		t.Fatalf("BindSupervisorTools: %v", err)
	}

	fixture, err := LoadFixture("mobile_app_ambiguous_v1")
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
	if result.Content != "" {
		t.Logf("Content: %s\n", result.Content)
	}

	// HARD CHECK: at most 1 ask_user call
	AssertMaxOneAskUser(t, result)

	// Should NOT create task without research
	AssertNoTaskCreation(t, result)

	// If ask_user is present, verify quality
	question := AssertAskUserHasMultipleQuestions(t, result)
	if question != "" {
		// LLM Judge: questions should be product-level, not tech decisions
		AssertJudgePass(t, harness, question, AskUserQualityRubric)
	}
}
