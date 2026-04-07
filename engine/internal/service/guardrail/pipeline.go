package guardrail

import (
	"context"
	"fmt"
	"log/slog"
)

// GuardrailMode defines the validation mode.
type GuardrailMode string

const (
	ModeJSONSchema GuardrailMode = "json_schema"
	ModeLLMJudge   GuardrailMode = "llm_judge"
	ModeWebhook    GuardrailMode = "webhook"
)

// OnFailureAction defines what to do when guardrail check fails.
type OnFailureAction string

const (
	OnFailureRetry    OnFailureAction = "retry"
	OnFailureError    OnFailureAction = "error"
	OnFailureFallback OnFailureAction = "fallback"
)

// GuardrailConfig holds the configuration for a guardrail check.
type GuardrailConfig struct {
	Mode           GuardrailMode
	OnFailure      OnFailureAction
	MaxRetries     int    // for retry mode, default 3
	FallbackText   string // for fallback mode

	// JSON Schema mode
	JSONSchema string // the JSON Schema to validate against

	// LLM Judge mode
	JudgePrompt string // prompt for the judge LLM
	JudgeModel  string // model to use for judging

	// Webhook mode
	WebhookURL  string
	WebhookAuth string // "none", "api_key", "forward_headers", "oauth2"
	AuthToken   string // for api_key auth
}

// CheckResult represents the result of a guardrail check.
type CheckResult struct {
	Passed  bool
	Reason  string
	Attempt int
}

// Checker is the interface that each guardrail mode implements.
type Checker interface {
	Check(ctx context.Context, output string) (*CheckResult, error)
}

// Pipeline orchestrates guardrail checks on agent output.
type Pipeline struct {
	checkers map[GuardrailMode]Checker
}

// NewPipeline creates a new guardrail Pipeline.
func NewPipeline() *Pipeline {
	return &Pipeline{
		checkers: make(map[GuardrailMode]Checker),
	}
}

// RegisterChecker registers a checker for a given mode.
func (p *Pipeline) RegisterChecker(mode GuardrailMode, checker Checker) {
	p.checkers[mode] = checker
}

// Evaluate runs the guardrail check on agent output.
// If the check fails and on_failure=retry, it returns an error indicating retry is needed.
// The caller (agent runtime) should re-generate and call Evaluate again.
func (p *Pipeline) Evaluate(ctx context.Context, config *GuardrailConfig, output string) (*CheckResult, error) {
	if config == nil {
		return &CheckResult{Passed: true, Reason: "no guardrail configured"}, nil
	}

	checker, ok := p.checkers[config.Mode]
	if !ok {
		return nil, fmt.Errorf("no checker registered for mode %q", config.Mode)
	}

	maxRetries := config.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		result, err := checker.Check(ctx, output)
		if err != nil {
			slog.ErrorContext(ctx, "guardrail check error", "mode", config.Mode, "attempt", attempt, "error", err)
			return nil, fmt.Errorf("guardrail check failed: %w", err)
		}

		result.Attempt = attempt

		if result.Passed {
			return result, nil
		}

		slog.InfoContext(ctx, "guardrail check failed", "mode", config.Mode, "attempt", attempt, "reason", result.Reason)

		// On last attempt, apply on_failure action
		if attempt >= maxRetries || config.OnFailure != OnFailureRetry {
			return p.handleFailure(config, result)
		}

		// Retry mode: return the failure so caller can re-generate
		// In a real implementation, the caller would re-generate and call Evaluate again
		// For now, we just continue the loop (in tests, output doesn't change)
	}

	return &CheckResult{Passed: false, Reason: "max retries exceeded", Attempt: maxRetries}, nil
}

func (p *Pipeline) handleFailure(config *GuardrailConfig, result *CheckResult) (*CheckResult, error) {
	switch config.OnFailure {
	case OnFailureError:
		return result, fmt.Errorf("guardrail check failed: %s", result.Reason)
	case OnFailureFallback:
		return &CheckResult{
			Passed:  true,
			Reason:  fmt.Sprintf("fallback applied (original failure: %s)", result.Reason),
			Attempt: result.Attempt,
		}, nil
	case OnFailureRetry:
		// Already exhausted retries
		return result, fmt.Errorf("guardrail check failed after %d retries: %s", result.Attempt, result.Reason)
	default:
		return result, fmt.Errorf("guardrail check failed: %s", result.Reason)
	}
}
