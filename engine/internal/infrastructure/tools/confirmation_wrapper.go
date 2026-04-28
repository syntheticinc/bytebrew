package tools

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ConfirmationRequester asks the user for confirmation before a tool executes.
// Consumer-side interface used by ConfirmationWrapper.
type ConfirmationRequester interface {
	RequestConfirmation(ctx context.Context, toolName string, args string) (bool, error)
}

// NewConfirmationWrapper wraps a tool to require user confirmation before execution.
// Used for tools listed in agent's confirm_before config.
func NewConfirmationWrapper(inner tool.InvokableTool, requester ConfirmationRequester) tool.InvokableTool {
	return &confirmationWrapper{inner: inner, requester: requester}
}

type confirmationWrapper struct {
	inner     tool.InvokableTool
	requester ConfirmationRequester
}

func (w *confirmationWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return w.inner.Info(ctx)
}

func (w *confirmationWrapper) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	info, err := w.inner.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("get tool info: %w", err)
	}

	confirmed, err := w.requester.RequestConfirmation(ctx, info.Name, argumentsInJSON)
	if err != nil {
		return "", fmt.Errorf("request confirmation: %w", err)
	}

	if !confirmed {
		return fmt.Sprintf("Tool %q execution cancelled by user.", info.Name), nil
	}

	return w.inner.InvokableRun(ctx, argumentsInJSON, opts...)
}
