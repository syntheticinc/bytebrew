package flow

import (
	"testing"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

func TestGateEvaluator_AllCondition_NoConfig(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{ConditionType: domain.GateConditionAll}

	result, err := eval.Evaluate(gate, "any output")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass with no config")
	}
}

func TestGateEvaluator_AllCondition_JSONValid(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionAll,
		Config:        map[string]interface{}{"condition": "json_schema"},
	}

	result, err := eval.Evaluate(gate, `{"key": "value"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for valid JSON")
	}

	result, err = eval.Evaluate(gate, "not json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail for invalid JSON")
	}
}

func TestGateEvaluator_AllCondition_Regex(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionAll,
		Config:        map[string]interface{}{"condition": "regex", "pattern": `\d{3}-\d{4}`},
	}

	result, err := eval.Evaluate(gate, "call 555-1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for matching regex")
	}

	result, err = eval.Evaluate(gate, "no phone")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail for non-matching regex")
	}
}

func TestGateEvaluator_AllCondition_Contains(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionAll,
		Config:        map[string]interface{}{"condition": "contains", "text": "SUCCESS"},
	}

	result, err := eval.Evaluate(gate, "Operation SUCCESS complete")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass when output contains text")
	}

	result, err = eval.Evaluate(gate, "Operation failed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail when output does not contain text")
	}
}

func TestGateEvaluator_AllCondition_EmptyOutput(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionAll,
		Config:        map[string]interface{}{},
	}

	result, err := eval.Evaluate(gate, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail for empty output with default condition")
	}
}

func TestGateEvaluator_AnyCondition(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionAny,
		Config: map[string]interface{}{
			"conditions": []interface{}{
				map[string]interface{}{"condition": "contains", "text": "PASS"},
				map[string]interface{}{"condition": "contains", "text": "OK"},
			},
		},
	}

	result, err := eval.Evaluate(gate, "result is OK")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass when any condition matches")
	}

	result, err = eval.Evaluate(gate, "result is FAIL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail when no conditions match")
	}
}

func TestGateEvaluator_CustomCondition_Contains(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionCustom,
		Config:        map[string]interface{}{"expression": "contains(SUCCESS)"},
	}

	result, err := eval.Evaluate(gate, "operation SUCCESS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass")
	}
}

func TestGateEvaluator_CustomCondition_Regex(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionCustom,
		Config:        map[string]interface{}{"expression": `matches(\d+)`},
	}

	result, err := eval.Evaluate(gate, "count: 42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for matching regex")
	}
}

func TestGateEvaluator_CustomCondition_NonEmpty(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionCustom,
		Config:        map[string]interface{}{"expression": "non_empty"},
	}

	result, err := eval.Evaluate(gate, "has content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for non-empty")
	}

	result, err = eval.Evaluate(gate, "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Passed {
		t.Error("expected fail for whitespace-only")
	}
}

func TestGateEvaluator_CustomCondition_JSONValid(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionCustom,
		Config:        map[string]interface{}{"expression": "json_valid"},
	}

	result, err := eval.Evaluate(gate, `{"ok": true}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Passed {
		t.Error("expected pass for valid JSON")
	}
}

func TestGateEvaluator_Join(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{ConditionType: domain.GateConditionAll}

	result := eval.EvaluateJoin(gate, 2, 3)
	if result.Passed {
		t.Error("expected fail: only 2/3 completed")
	}

	result = eval.EvaluateJoin(gate, 3, 3)
	if !result.Passed {
		t.Error("expected pass: all 3/3 completed")
	}
}

func TestGateEvaluator_InvalidRegex(t *testing.T) {
	eval := NewGateEvaluator()
	gate := &domain.Gate{
		ConditionType: domain.GateConditionAll,
		Config:        map[string]interface{}{"condition": "regex", "pattern": "[invalid"},
	}

	_, err := eval.Evaluate(gate, "test")
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}
