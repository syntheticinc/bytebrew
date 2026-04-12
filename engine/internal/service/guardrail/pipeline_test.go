package guardrail

import (
	"context"
	"testing"
)

type alwaysPassChecker struct{}

func (c *alwaysPassChecker) Check(_ context.Context, output string) (*CheckResult, error) {
	return &CheckResult{Passed: true, Reason: "always pass"}, nil
}

type alwaysFailChecker struct {
	reason string
}

func (c *alwaysFailChecker) Check(_ context.Context, output string) (*CheckResult, error) {
	return &CheckResult{Passed: false, Reason: c.reason}, nil
}

func TestPipeline_NoConfig(t *testing.T) {
	p := NewPipeline()
	result, err := p.Evaluate(context.Background(), nil, "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass with no config")
	}
}

func TestPipeline_Pass(t *testing.T) {
	p := NewPipeline()
	p.RegisterChecker(ModeJSONSchema, &alwaysPassChecker{})

	result, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:      ModeJSONSchema,
		OnFailure: OnFailureError,
	}, "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass")
	}
}

func TestPipeline_Fail_Error(t *testing.T) {
	p := NewPipeline()
	p.RegisterChecker(ModeJSONSchema, &alwaysFailChecker{reason: "bad output"})

	_, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:      ModeJSONSchema,
		OnFailure: OnFailureError,
	}, "output")
	if err == nil {
		t.Fatal("expected error for failed check with on_failure=error")
	}
}

func TestPipeline_Fail_Fallback(t *testing.T) {
	p := NewPipeline()
	p.RegisterChecker(ModeJSONSchema, &alwaysFailChecker{reason: "bad"})

	result, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:         ModeJSONSchema,
		OnFailure:    OnFailureFallback,
		FallbackText: "Sorry, I could not generate a proper response.",
	}, "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected Passed=false for fallback (caller applies fallback text)")
	}
}

func TestPipeline_Fail_Retry_Exhausted(t *testing.T) {
	p := NewPipeline()
	p.RegisterChecker(ModeJSONSchema, &alwaysFailChecker{reason: "always fails"})

	_, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:       ModeJSONSchema,
		OnFailure:  OnFailureRetry,
		MaxRetries: 3,
	}, "output")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
}

func TestPipeline_UnregisteredMode(t *testing.T) {
	p := NewPipeline()

	_, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode: "unknown",
	}, "output")
	if err == nil {
		t.Fatal("expected error for unregistered mode")
	}
}

func TestPipeline_Integration_JSONSchema(t *testing.T) {
	p := NewPipeline()
	p.RegisterChecker(ModeJSONSchema, NewJSONSchemaValidator(`{"type": "object", "required": ["name"]}`))

	// Pass: has required field
	result, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:      ModeJSONSchema,
		OnFailure: OnFailureError,
	}, `{"name": "Alice"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Errorf("expected pass, got: %s", result.Reason)
	}

	// Fail: missing required field
	_, err = p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:      ModeJSONSchema,
		OnFailure: OnFailureError,
	}, `{"age": 30}`)
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
}

func TestPipeline_Integration_LLMJudge(t *testing.T) {
	p := NewPipeline()
	caller := &mockLLMCaller{response: "Yes, appropriate response."}
	p.RegisterChecker(ModeLLMJudge, NewLLMJudge(caller, "Is this appropriate?", "test"))

	result, err := p.Evaluate(context.Background(), &GuardrailConfig{
		Mode:      ModeLLMJudge,
		OnFailure: OnFailureError,
	}, "Hello!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass")
	}
}
