package domain

import "context"

// AgentConfig represents configuration for an agent
type AgentConfig struct {
	LLMProvider LLMProvider
	Tools       []Tool
	MaxSteps    int
}

// AgentResult represents the result of agent execution
type AgentResult struct {
	Output string
	Steps  int
	Error  error
}

// Agent defines interface for AI agents
type Agent interface {
	// Run executes the agent with given task
	Run(ctx context.Context, task string) (string, error)

	// RunStream executes the agent with streaming output
	RunStream(ctx context.Context, task string, streamFunc func(chunk string) error) error
}
