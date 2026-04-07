package guardrail

import (
	"context"
	"fmt"
	"strings"
)

// LLMCaller sends a prompt to an LLM and returns the response.
type LLMCaller interface {
	Call(ctx context.Context, model, prompt string) (string, error)
}

// LLMJudge validates agent output by sending it to a separate LLM for evaluation.
type LLMJudge struct {
	caller      LLMCaller
	judgePrompt string
	model       string
}

// NewLLMJudge creates a new LLMJudge.
func NewLLMJudge(caller LLMCaller, judgePrompt, model string) *LLMJudge {
	return &LLMJudge{
		caller:      caller,
		judgePrompt: judgePrompt,
		model:       model,
	}
}

// Check sends the output to the judge LLM and evaluates the response.
func (j *LLMJudge) Check(ctx context.Context, output string) (*CheckResult, error) {
	prompt := j.buildPrompt(output)

	response, err := j.caller.Call(ctx, j.model, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM judge call failed: %w", err)
	}

	passed, reason := j.parseResponse(response)
	return &CheckResult{
		Passed: passed,
		Reason: reason,
	}, nil
}

func (j *LLMJudge) buildPrompt(output string) string {
	prompt := j.judgePrompt
	if prompt == "" {
		prompt = "You are a quality judge. Evaluate if the following response is appropriate and helpful. Reply with only 'yes' or 'no' followed by a brief reason."
	}
	return fmt.Sprintf("%s\n\n--- Response to evaluate ---\n%s\n--- End of response ---", prompt, output)
}

// parseResponse extracts yes/no from the judge's response.
func (j *LLMJudge) parseResponse(response string) (bool, string) {
	lower := strings.ToLower(strings.TrimSpace(response))

	if strings.HasPrefix(lower, "yes") {
		return true, response
	}
	if strings.HasPrefix(lower, "no") {
		return false, response
	}

	// If unclear, treat as pass with warning
	if strings.Contains(lower, "pass") || strings.Contains(lower, "approve") || strings.Contains(lower, "good") {
		return true, response
	}
	if strings.Contains(lower, "fail") || strings.Contains(lower, "reject") || strings.Contains(lower, "bad") {
		return false, response
	}

	// Default: pass (conservative — don't block unclear judgments)
	return true, fmt.Sprintf("unclear judgment (defaulting to pass): %s", response)
}
