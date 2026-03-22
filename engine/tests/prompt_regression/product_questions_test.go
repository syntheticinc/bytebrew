//go:build prompt

package prompt_regression

import (
	"context"
	"testing"
	"time"
)

func TestAskUserForAmbiguousRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	harness, err := NewHarness()
	if err != nil {
		t.Fatalf("NewHarness: %v", err)
	}

	if err := harness.BindSupervisorTools(ctx); err != nil {
		t.Fatalf("BindSupervisorTools: %v", err)
	}

	fixture, err := LoadFixture("ambiguous_product_request_v1")
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

	// Pre-check: should NOT create task immediately for ambiguous product request
	AssertNoTaskCreation(t, result)

	// LLM Judge: evaluate appropriateness of agent's response
	responseText := formatResponseForJudge(result)
	AssertJudgePass(t, harness, responseText, DiscoveryDepthRubric)
}
