package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/engine/internal/service/lifecycle"
)

// AgentLifecycleReader reads the lifecycle mode for an agent by name.
type AgentLifecycleReader interface {
	GetLifecycleMode(ctx context.Context, agentName string) domain.LifecycleMode
	GetMaxContextSize(ctx context.Context, agentName string) int
}

// CompositeAgentSpawner routes spawn requests based on the target agent's lifecycle mode.
// For "spawn" agents it delegates to the existing AgentPoolAdapter (unchanged execution path).
// For "persistent" agents it uses lifecycle.Manager to handle context accumulation.
type CompositeAgentSpawner struct {
	pool    tools.GenericAgentSpawner
	manager *lifecycle.Manager
	agents  AgentLifecycleReader
}

// NewCompositeAgentSpawner creates a new CompositeAgentSpawner.
func NewCompositeAgentSpawner(
	pool tools.GenericAgentSpawner,
	manager *lifecycle.Manager,
	agents AgentLifecycleReader,
) *CompositeAgentSpawner {
	return &CompositeAgentSpawner{
		pool:    pool,
		manager: manager,
		agents:  agents,
	}
}

// SpawnAgent implements tools.GenericAgentSpawner by routing based on lifecycle mode.
func (c *CompositeAgentSpawner) SpawnAgent(ctx context.Context, params tools.SpawnParams) (string, error) {
	mode := c.agents.GetLifecycleMode(ctx, params.AgentName)

	if mode != domain.LifecycleModePersistent {
		return c.pool.SpawnAgent(ctx, params)
	}

	slog.InfoContext(ctx, "lifecycle: routing to persistent manager",
		"agent", params.AgentName,
		"session", params.SessionID,
	)

	maxContext := c.agents.GetMaxContextSize(ctx, params.AgentName)

	result, err := c.manager.ExecuteTask(
		ctx,
		params.AgentName,
		params.SessionID,
		params.Description,
		mode,
		maxContext,
		nil, // eventStream — pool handles event emission internally
	)
	if err != nil {
		return "", fmt.Errorf("lifecycle execute task: %w", err)
	}

	return result, nil
}

// WaitForAgent delegates to the underlying pool.
func (c *CompositeAgentSpawner) WaitForAgent(ctx context.Context, sessionID, agentID string) (tools.AgentCompletionInfo, error) {
	return c.pool.WaitForAgent(ctx, sessionID, agentID)
}

// WaitForAllSessionAgents delegates to the underlying pool.
func (c *CompositeAgentSpawner) WaitForAllSessionAgents(ctx context.Context, sessionID string) (tools.WaitResult, error) {
	return c.pool.WaitForAllSessionAgents(ctx, sessionID)
}

// HasBlockingWait delegates to the underlying pool.
func (c *CompositeAgentSpawner) HasBlockingWait(sessionID string) bool {
	return c.pool.HasBlockingWait(sessionID)
}

// NotifyUserMessage delegates to the underlying pool.
func (c *CompositeAgentSpawner) NotifyUserMessage(sessionID, message string) {
	c.pool.NotifyUserMessage(sessionID, message)
}

// StopAgent delegates to the underlying pool.
func (c *CompositeAgentSpawner) StopAgent(agentID string) error {
	return c.pool.StopAgent(agentID)
}

// agentSpawnerWaiter is the minimal interface poolBasedRunner needs:
// spawn an agent and wait for its completion by agentID.
type agentSpawnerWaiter interface {
	SpawnAgent(ctx context.Context, params tools.SpawnParams) (string, error)
	WaitForAgent(ctx context.Context, sessionID, agentID string) (tools.AgentCompletionInfo, error)
}

// poolBasedRunner wraps an agentSpawnerWaiter to implement lifecycle.AgentRunner.
// RunAgent spawns the agent, then blocks until it completes, returning its actual output.
type poolBasedRunner struct {
	pool agentSpawnerWaiter
}

// RunAgent implements lifecycle.AgentRunner.
// Spawns the agent via the pool, then blocks until completion and returns the actual output.
// This is required for the lifecycle.Manager to store real outputs (not agent IDs) in context.
func (r *poolBasedRunner) RunAgent(ctx context.Context, agentName, input, sessionID string, eventStream domain.AgentEventStream) (string, error) {
	agentID, err := r.pool.SpawnAgent(ctx, tools.SpawnParams{
		SessionID:   sessionID,
		AgentName:   agentName,
		Description: input,
		Blocking:    false, // agent runs on session-scoped context; we wait separately
	})
	if err != nil {
		return "", fmt.Errorf("spawn agent: %w", err)
	}

	info, err := r.pool.WaitForAgent(ctx, sessionID, agentID)
	if err != nil {
		return "", fmt.Errorf("wait for agent %s: %w", agentID, err)
	}
	return info.Result, nil
}
