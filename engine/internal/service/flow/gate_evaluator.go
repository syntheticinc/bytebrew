package flow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// GateEvalResult represents the result of gate evaluation.
type GateEvalResult struct {
	Passed    bool
	Reason    string
	Iteration int
}

// GateEvaluator evaluates gate conditions.
type GateEvaluator struct{}

// NewGateEvaluator creates a new GateEvaluator.
func NewGateEvaluator() *GateEvaluator {
	return &GateEvaluator{}
}

// Evaluate evaluates a gate's condition against the agent output.
func (ge *GateEvaluator) Evaluate(gate *domain.Gate, output string) (*GateEvalResult, error) {
	switch gate.ConditionType {
	case domain.GateConditionAll:
		return ge.evaluateAutoCondition(gate, output)
	case domain.GateConditionAny:
		return ge.evaluateAnyCondition(gate, output)
	case domain.GateConditionCustom:
		return ge.evaluateCustomCondition(gate, output)
	default:
		return nil, fmt.Errorf("unknown gate condition type: %s", gate.ConditionType)
	}
}

// EvaluateJoin evaluates a join gate (all_completed) — returns true when all inputs are ready.
func (ge *GateEvaluator) EvaluateJoin(gate *domain.Gate, completedInputs, totalInputs int) *GateEvalResult {
	if completedInputs >= totalInputs {
		return &GateEvalResult{
			Passed: true,
			Reason: fmt.Sprintf("all %d inputs completed", totalInputs),
		}
	}
	return &GateEvalResult{
		Passed: false,
		Reason: fmt.Sprintf("waiting for inputs: %d/%d completed", completedInputs, totalInputs),
	}
}

// evaluateAutoCondition checks output against auto conditions: json_schema, regex, contains.
func (ge *GateEvaluator) evaluateAutoCondition(gate *domain.Gate, output string) (*GateEvalResult, error) {
	if gate.Config == nil {
		// No condition configured — always pass
		return &GateEvalResult{Passed: true, Reason: "no condition configured"}, nil
	}

	condType, _ := gate.Config["condition"].(string)

	switch condType {
	case "json_schema":
		return ge.checkJSONValid(output), nil
	case "regex":
		pattern, _ := gate.Config["pattern"].(string)
		return ge.checkRegex(output, pattern)
	case "contains":
		text, _ := gate.Config["text"].(string)
		return ge.checkContains(output, text), nil
	default:
		// No specific condition — check if output is non-empty
		if strings.TrimSpace(output) == "" {
			return &GateEvalResult{Passed: false, Reason: "output is empty"}, nil
		}
		return &GateEvalResult{Passed: true, Reason: "output is non-empty"}, nil
	}
}

// evaluateAnyCondition passes if the output matches any configured condition.
func (ge *GateEvaluator) evaluateAnyCondition(gate *domain.Gate, output string) (*GateEvalResult, error) {
	if gate.Config == nil {
		return &GateEvalResult{Passed: true, Reason: "no conditions configured"}, nil
	}

	// Try each condition — pass on first match
	conditions, ok := gate.Config["conditions"].([]interface{})
	if !ok {
		// Single condition fallback
		return ge.evaluateAutoCondition(gate, output)
	}

	for _, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			continue
		}
		tempGate := &domain.Gate{
			Config:        condMap,
			ConditionType: domain.GateConditionAll,
		}
		result, err := ge.evaluateAutoCondition(tempGate, output)
		if err != nil {
			continue
		}
		if result.Passed {
			return result, nil
		}
	}

	return &GateEvalResult{Passed: false, Reason: "no conditions matched"}, nil
}

// evaluateCustomCondition evaluates a simple custom expression.
func (ge *GateEvaluator) evaluateCustomCondition(gate *domain.Gate, output string) (*GateEvalResult, error) {
	if gate.Config == nil {
		return &GateEvalResult{Passed: true, Reason: "no custom expression"}, nil
	}

	expr, _ := gate.Config["expression"].(string)
	if expr == "" {
		return &GateEvalResult{Passed: true, Reason: "empty expression"}, nil
	}

	// Simple expression evaluator: supports "contains(text)", "matches(regex)", "non_empty"
	switch {
	case strings.HasPrefix(expr, "contains(") && strings.HasSuffix(expr, ")"):
		text := expr[9 : len(expr)-1]
		return ge.checkContains(output, text), nil
	case strings.HasPrefix(expr, "matches(") && strings.HasSuffix(expr, ")"):
		pattern := expr[8 : len(expr)-1]
		return ge.checkRegex(output, pattern)
	case expr == "non_empty":
		if strings.TrimSpace(output) != "" {
			return &GateEvalResult{Passed: true, Reason: "output is non-empty"}, nil
		}
		return &GateEvalResult{Passed: false, Reason: "output is empty"}, nil
	case expr == "json_valid":
		return ge.checkJSONValid(output), nil
	default:
		return &GateEvalResult{Passed: false, Reason: fmt.Sprintf("unsupported expression: %s", expr)}, nil
	}
}

func (ge *GateEvaluator) checkJSONValid(output string) *GateEvalResult {
	var js json.RawMessage
	if json.Unmarshal([]byte(output), &js) == nil {
		return &GateEvalResult{Passed: true, Reason: "valid JSON"}
	}
	return &GateEvalResult{Passed: false, Reason: "invalid JSON"}
}

func (ge *GateEvaluator) checkRegex(output, pattern string) (*GateEvalResult, error) {
	if pattern == "" {
		return &GateEvalResult{Passed: true, Reason: "no pattern"}, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regex pattern %q: %w", pattern, err)
	}
	if re.MatchString(output) {
		return &GateEvalResult{Passed: true, Reason: fmt.Sprintf("matches pattern %q", pattern)}, nil
	}
	return &GateEvalResult{Passed: false, Reason: fmt.Sprintf("does not match pattern %q", pattern)}, nil
}

func (ge *GateEvaluator) checkContains(output, text string) *GateEvalResult {
	if text == "" {
		return &GateEvalResult{Passed: true, Reason: "no text to check"}
	}
	if strings.Contains(output, text) {
		return &GateEvalResult{Passed: true, Reason: fmt.Sprintf("contains %q", text)}
	}
	return &GateEvalResult{Passed: false, Reason: fmt.Sprintf("does not contain %q", text)}
}
