package domain

import (
	"context"
	"fmt"
	"regexp"
)

// AgentNameRe validates agent names: lowercase letters, digits, and hyphens; must start with a letter.
var AgentNameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateAgentName returns an error if the name does not match AgentNameRe.
func ValidateAgentName(name string) error {
	if !AgentNameRe.MatchString(name) {
		return fmt.Errorf("agent name must match %s (lowercase letters, digits, hyphens; must start with a letter)", AgentNameRe.String())
	}
	return nil
}

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
