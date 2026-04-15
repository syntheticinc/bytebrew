package app

import (
	"context"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/engine/internal/service/guardrail"
	"github.com/syntheticinc/bytebrew/engine/internal/service/policy"
	"github.com/syntheticinc/bytebrew/engine/internal/service/recovery"
	"github.com/syntheticinc/bytebrew/engine/internal/service/resilience"
	"github.com/syntheticinc/bytebrew/engine/internal/service/turnexecutor"
)

// policyEvaluatorAdapter bridges policy.Engine to tools.PolicyEvaluator.
type policyEvaluatorAdapter struct {
	engine *policy.Engine
}

func (a *policyEvaluatorAdapter) EvaluateBefore(ctx context.Context, tc tools.PolicyToolCallContext) tools.PolicyEvalResult {
	result := a.engine.EvaluateBefore(ctx, policy.ToolCallContext{
		AgentName: tc.AgentName,
		ToolName:  tc.ToolName,
		Arguments: tc.Arguments,
		Result:    tc.Result,
		Error:     tc.Error,
		Timestamp: tc.Timestamp,
	})
	return tools.PolicyEvalResult{
		Blocked:      result.Blocked,
		BlockMessage: result.BlockMessage,
	}
}

func (a *policyEvaluatorAdapter) EvaluateAfter(ctx context.Context, tc tools.PolicyToolCallContext) {
	a.engine.EvaluateAfter(ctx, policy.ToolCallContext{
		AgentName: tc.AgentName,
		ToolName:  tc.ToolName,
		Arguments: tc.Arguments,
		Result:    tc.Result,
		Error:     tc.Error,
		Timestamp: tc.Timestamp,
	})
}

// circuitBreakerRegistryAdapter bridges resilience.CircuitBreakerRegistry to tools.CircuitBreakerRegistry.
type circuitBreakerRegistryAdapter struct {
	registry *resilience.CircuitBreakerRegistry
}

func (a *circuitBreakerRegistryAdapter) Get(name string) tools.CircuitBreakerChecker {
	return a.registry.Get(name)
}

// recoveryExecutorAdapter bridges recovery.Executor to tools.RecoveryExecutor.
type recoveryExecutorAdapter struct {
	executor *recovery.Executor
}

func (a *recoveryExecutorAdapter) Execute(ctx context.Context, sessionID string, failureType domain.FailureType, detail string) tools.RecoveryExecResult {
	result := a.executor.Execute(ctx, sessionID, failureType, detail)
	return tools.RecoveryExecResult{
		Recovered: result.Recovered,
		Action:    result.Action,
		Detail:    result.Detail,
	}
}

// guardrailCheckerAdapter bridges guardrail.Pipeline to turnexecutor.GuardrailChecker.
type guardrailCheckerAdapter struct {
	pipeline *guardrail.Pipeline
}

func (a *guardrailCheckerAdapter) Evaluate(ctx context.Context, config *turnexecutor.GuardrailCheckConfig, output string) (*turnexecutor.GuardrailCheckResult, error) {
	if config == nil {
		return &turnexecutor.GuardrailCheckResult{Passed: true}, nil
	}
	grConfig := &guardrail.GuardrailConfig{
		Mode:         guardrail.GuardrailMode(config.Mode),
		OnFailure:    guardrail.OnFailureAction(config.OnFailure),
		MaxRetries:   config.MaxRetries,
		FallbackText: config.FallbackText,
		JSONSchema:   config.JSONSchema,
		JudgePrompt:  config.JudgePrompt,
		JudgeModel:   config.JudgeModel,
		WebhookURL:   config.WebhookURL,
	}
	result, err := a.pipeline.Evaluate(ctx, grConfig, output)
	if err != nil {
		return nil, err
	}
	return &turnexecutor.GuardrailCheckResult{
		Passed: result.Passed,
		Reason: result.Reason,
	}, nil
}

