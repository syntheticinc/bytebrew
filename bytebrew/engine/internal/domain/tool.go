package domain

import "context"

// ToolInput represents input for a tool
type ToolInput map[string]interface{}

// ToolOutput represents output from a tool
type ToolOutput struct {
	Result interface{}
	Error  error
}

// Tool defines interface for agent tools
type Tool interface {
	// Name returns the tool name
	Name() string

	// Description returns the tool description
	Description() string

	// Execute executes the tool with given input
	Execute(ctx context.Context, input ToolInput) (*ToolOutput, error)
}

// ToolRegistry defines interface for managing tools
type ToolRegistry interface {
	// Register registers a new tool
	Register(tool Tool) error

	// Unregister removes a tool by name
	Unregister(name string) error

	// Get retrieves a tool by name
	Get(name string) (Tool, error)

	// List returns all registered tools
	List() []Tool

	// Count returns the number of registered tools
	Count() int
}
