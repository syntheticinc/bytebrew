package tools

import (
	"context"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// RecoveryExecutor attempts recovery from tool failures (consumer-side interface).
type RecoveryExecutor interface {
	Execute(ctx context.Context, sessionID string, failureType domain.FailureType, detail string) RecoveryExecResult
}

// RecoveryExecResult contains the outcome of a recovery attempt.
type RecoveryExecResult struct {
	Recovered bool
	Action    domain.RecoveryAction
	Detail    string
}

// RecoveryToolWrapper wraps a tool with automatic failure recovery.
// Used for MCP tools where connection/timeout failures can be retried.
type RecoveryToolWrapper struct {
	inner     tool.InvokableTool
	recovery  RecoveryExecutor
	sessionID string
	toolName  string
}

// NewRecoveryToolWrapper wraps a tool with recovery logic.
func NewRecoveryToolWrapper(inner tool.InvokableTool, recovery RecoveryExecutor, sessionID, toolName string) tool.InvokableTool {
	return &RecoveryToolWrapper{
		inner:     inner,
		recovery:  recovery,
		sessionID: sessionID,
		toolName:  toolName,
	}
}

func (w *RecoveryToolWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return w.inner.Info(ctx)
}

func (w *RecoveryToolWrapper) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	output, err := w.inner.InvokableRun(ctx, argumentsInJSON, opts...)
	if err == nil {
		return output, nil
	}

	// Classify failure and attempt recovery
	failureType := classifyToolFailure(err)
	detail := w.toolName + ": " + err.Error()

	result := w.recovery.Execute(ctx, w.sessionID, failureType, detail)

	if !result.Recovered {
		slog.WarnContext(ctx, "[RecoveryWrapper] recovery failed",
			"tool", w.toolName, "failure_type", failureType, "action", result.Action)
		return output, err
	}

	// Recovery indicated retry — re-execute the tool
	if result.Action == domain.RecoveryRetry {
		slog.InfoContext(ctx, "[RecoveryWrapper] retrying after recovery",
			"tool", w.toolName, "failure_type", failureType)
		return w.inner.InvokableRun(ctx, argumentsInJSON, opts...)
	}

	// Recovery indicated skip — return error as informational message
	if result.Action == domain.RecoverySkip {
		return "[DEGRADED] tool temporarily unavailable, skipping", nil
	}

	return output, err
}

// classifyToolFailure maps an error to a domain.FailureType.
func classifyToolFailure(err error) domain.FailureType {
	if err == nil {
		return domain.FailureMCPConnectionFailed
	}

	errStr := err.Error()
	switch {
	case containsAny(errStr, "connection refused", "connection reset", "no such host", "dial"):
		return domain.FailureMCPConnectionFailed
	case containsAny(errStr, "timeout", "deadline exceeded"):
		return domain.FailureToolTimeout
	case containsAny(errStr, "unauthorized", "forbidden", "auth"):
		return domain.FailureToolAuthFailure
	default:
		return domain.FailureMCPConnectionFailed
	}
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
