package guardrail

import (
	"context"
	"fmt"
	"testing"
)

type mockLLMCaller struct {
	response string
	err      error
}

func (m *mockLLMCaller) Call(_ context.Context, model, prompt string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func TestLLMJudge_Pass(t *testing.T) {
	caller := &mockLLMCaller{response: "Yes, this response is appropriate and helpful."}
	judge := NewLLMJudge(caller, "", "test-model")

	result, err := judge.Check(context.Background(), "Hello, how can I help you?")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for 'yes' response")
	}
}

func TestLLMJudge_Fail(t *testing.T) {
	caller := &mockLLMCaller{response: "No, this response contains harmful content."}
	judge := NewLLMJudge(caller, "", "test-model")

	result, err := judge.Check(context.Background(), "bad output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail for 'no' response")
	}
}

func TestLLMJudge_CustomPrompt(t *testing.T) {
	caller := &mockLLMCaller{response: "yes"}
	judge := NewLLMJudge(caller, "Check if response is professional", "test-model")

	result, err := judge.Check(context.Background(), "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass")
	}
}

func TestLLMJudge_LLMError(t *testing.T) {
	caller := &mockLLMCaller{err: fmt.Errorf("LLM unavailable")}
	judge := NewLLMJudge(caller, "", "test-model")

	_, err := judge.Check(context.Background(), "output")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLLMJudge_AmbiguousResponse_Pass(t *testing.T) {
	caller := &mockLLMCaller{response: "The response looks good and is appropriate."}
	judge := NewLLMJudge(caller, "", "test-model")

	result, err := judge.Check(context.Background(), "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for 'good' response")
	}
}

func TestLLMJudge_AmbiguousResponse_Fail(t *testing.T) {
	caller := &mockLLMCaller{response: "I reject this response as it contains bad information."}
	judge := NewLLMJudge(caller, "", "test-model")

	result, err := judge.Check(context.Background(), "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail for 'reject/bad' response")
	}
}

func TestLLMJudge_UnclearDefault_Pass(t *testing.T) {
	caller := &mockLLMCaller{response: "The output seems fine."}
	judge := NewLLMJudge(caller, "", "test-model")

	result, err := judge.Check(context.Background(), "output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "seems fine" doesn't match any keyword — defaults to pass
	if !result.Passed {
		t.Error("expected default pass for unclear judgment")
	}
}
