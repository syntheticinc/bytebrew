package agents

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// ContextLoggerInterface defines the interface for context logging
// Used for dependency injection in MessageModifier and ReActAgent
type ContextLoggerInterface interface {
	// LogContext logs the current context composition to a step-specific file
	LogContext(ctx context.Context, messages []*schema.Message, step int)

	// LogContextSummary logs a summary of the context
	LogContextSummary(ctx context.Context, messages []*schema.Message)
}

// StepContentStoreInterface defines the interface for storing step content
// Used for dependency injection in callback handlers and message modifiers
type StepContentStoreInterface interface {
	// Append adds content to a specific step
	Append(step int, content string)

	// Get returns content for a specific step
	Get(step int) string

	// GetAll returns a copy of all step content
	GetAll() map[int]string

	// ClearBefore removes all content for steps before the given step
	ClearBefore(step int)
}

// Ensure implementations satisfy interfaces
var _ StepContentStoreInterface = (*StepContentStore)(nil)

// Note: ContextLogger implements ContextLoggerInterface but since it uses
// concrete *ContextLogger throughout the codebase, we don't enforce it here
// to avoid breaking changes. In the react/ package refactor, we'll use the interface.
