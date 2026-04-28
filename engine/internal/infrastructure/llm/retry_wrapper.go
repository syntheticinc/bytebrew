package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// RetryWrapper wraps a ToolCallingChatModel with retry logic for transient errors.
type RetryWrapper struct {
	inner      model.ToolCallingChatModel
	maxRetries int
	baseDelay  time.Duration
	timeout    time.Duration
}

// NewRetryWrapper creates a retry wrapper around a chat model.
// maxRetries is the number of retry attempts (0 means no retries).
// baseDelay is the initial delay between retries (exponential backoff).
// timeout is the per-call timeout.
func NewRetryWrapper(inner model.ToolCallingChatModel, maxRetries int, baseDelay, timeout time.Duration) *RetryWrapper {
	return &RetryWrapper{
		inner:      inner,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		timeout:    timeout,
	}
}

// Generate calls the inner model with retry logic for transient errors.
func (w *RetryWrapper) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	var lastErr error
	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			delay := w.baseDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		callCtx, cancel := context.WithTimeout(ctx, w.timeout)
		result, err := w.inner.Generate(callCtx, input, opts...)
		cancel()

		if err == nil {
			return result, nil
		}
		lastErr = err
		if !isRetriable(err) {
			return nil, err
		}
	}
	return nil, fmt.Errorf("all %d retries failed: %w", w.maxRetries+1, lastErr)
}

// Stream delegates directly to the inner model (streaming is stateful, not retriable).
func (w *RetryWrapper) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return w.inner.Stream(ctx, input, opts...)
}

// WithTools returns a new RetryWrapper with the specified tools bound to the inner model.
func (w *RetryWrapper) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	newInner, err := w.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &RetryWrapper{
		inner:      newInner,
		maxRetries: w.maxRetries,
		baseDelay:  w.baseDelay,
		timeout:    w.timeout,
	}, nil
}

// isRetriable determines whether an error is transient and worth retrying.
func isRetriable(err error) bool {
	if err == nil {
		return false
	}

	lower := strings.ToLower(err.Error())

	// Non-retriable patterns (check first to avoid false positives)
	nonRetriable := []string{"400", "401", "403", "404", "invalid"}
	for _, pattern := range nonRetriable {
		if strings.Contains(lower, pattern) {
			return false
		}
	}

	// Retriable patterns
	retriable := []string{
		"503", "service unavailable",
		"429", "too many requests", "rate limit",
		"502", "bad gateway",
		"timeout", "deadline exceeded",
		"connection refused", "connection reset",
		"eof",
	}
	for _, pattern := range retriable {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	// Default: retry on unknown errors
	return true
}
