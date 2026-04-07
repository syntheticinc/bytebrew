package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// PolicyEvaluator evaluates policies before/after tool calls (consumer-side interface).
type PolicyEvaluator interface {
	EvaluateBefore(ctx context.Context, tc PolicyToolCallContext) PolicyEvalResult
	EvaluateAfter(ctx context.Context, tc PolicyToolCallContext)
}

// PolicyToolCallContext holds context for policy evaluation around a tool call.
type PolicyToolCallContext struct {
	AgentName string
	ToolName  string
	Arguments string
	Result    string
	Error     error
	Timestamp time.Time
}

// PolicyEvalResult holds the result of a before-tool-call policy check.
type PolicyEvalResult struct {
	Blocked      bool
	BlockMessage string
}

// PolicyToolWrapper wraps a tool with policy evaluation before and after execution.
type PolicyToolWrapper struct {
	inner     tool.InvokableTool
	evaluator PolicyEvaluator
	agentName string
	toolName  string
}

// NewPolicyToolWrapper wraps a tool with policy checks.
func NewPolicyToolWrapper(inner tool.InvokableTool, evaluator PolicyEvaluator, agentName, toolName string) tool.InvokableTool {
	return &PolicyToolWrapper{
		inner:     inner,
		evaluator: evaluator,
		agentName: agentName,
		toolName:  toolName,
	}
}

func (w *PolicyToolWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return w.inner.Info(ctx)
}

func (w *PolicyToolWrapper) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	tc := PolicyToolCallContext{
		AgentName: w.agentName,
		ToolName:  w.toolName,
		Arguments: argumentsInJSON,
		Timestamp: time.Now(),
	}

	// Before: check policy
	result := w.evaluator.EvaluateBefore(ctx, tc)
	if result.Blocked {
		return fmt.Sprintf("[POLICY BLOCKED] %s", result.BlockMessage), nil
	}

	// Execute tool
	output, err := w.inner.InvokableRun(ctx, argumentsInJSON, opts...)

	// After: evaluate post-call policies
	tc.Result = output
	tc.Error = err
	w.evaluator.EvaluateAfter(ctx, tc)

	return output, err
}
