package resilience

import (
	"context"
	"fmt"
	"time"
)

// ToolTimeoutConfig holds per-tool-call timeout settings (AC-RESIL-05, AC-RESIL-06).
type ToolTimeoutConfig struct {
	DefaultTimeout time.Duration // default per-tool-call timeout (30s)
}

// DefaultToolTimeoutConfig returns default tool timeout configuration.
func DefaultToolTimeoutConfig() ToolTimeoutConfig {
	return ToolTimeoutConfig{
		DefaultTimeout: 30 * time.Second,
	}
}

// ToolTimeoutError is the structured error returned to the agent when a tool times out.
type ToolTimeoutError struct {
	ToolName  string `json:"tool"`
	TimeoutMs int64  `json:"timeout_ms"`
}

func (e *ToolTimeoutError) Error() string {
	return fmt.Sprintf("tool_timeout: %s exceeded %dms", e.ToolName, e.TimeoutMs)
}

// WithToolTimeout wraps a context with a tool call timeout.
// Returns the wrapped context and a cancel function.
func WithToolTimeout(ctx context.Context, toolName string, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}

// NewToolTimeoutError creates a structured timeout error (AC-RESIL-05).
func NewToolTimeoutError(toolName string, timeout time.Duration) *ToolTimeoutError {
	return &ToolTimeoutError{
		ToolName:  toolName,
		TimeoutMs: timeout.Milliseconds(),
	}
}
