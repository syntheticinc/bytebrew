package tools

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// TimeoutToolWrapper wraps an MCP tool with a per-call timeout (AC-RESIL-05).
// It must be the innermost wrapper so that timeouts feed as failures into the circuit breaker.
type TimeoutToolWrapper struct {
	inner   tool.InvokableTool
	timeout int64 // milliseconds
}

// NewTimeoutToolWrapper wraps inner with a timeout in milliseconds.
func NewTimeoutToolWrapper(inner tool.InvokableTool, timeoutMs int64) tool.InvokableTool {
	return &TimeoutToolWrapper{inner: inner, timeout: timeoutMs}
}

func (w *TimeoutToolWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return w.inner.Info(ctx)
}

func (w *TimeoutToolWrapper) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(w.timeout)*time.Millisecond)
	defer cancel()

	result, err := w.inner.InvokableRun(timeoutCtx, argumentsInJSON, opts...)
	if err != nil && (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) {
		info, _ := w.inner.Info(ctx)
		name := "tool"
		if info != nil {
			name = info.Name
		}
		return "", fmt.Errorf("tool_timeout: %s exceeded %dms", name, w.timeout)
	}
	return result, err
}
