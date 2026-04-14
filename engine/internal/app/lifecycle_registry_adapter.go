package app

import (
	"context"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agentregistry"
)

// agentRegistryLifecycleAdapter adapts AgentRegistry to the AgentLifecycleReader interface.
type agentRegistryLifecycleAdapter struct {
	registry *agentregistry.AgentRegistry
}

// newAgentRegistryLifecycleAdapter creates a new adapter.
func newAgentRegistryLifecycleAdapter(registry *agentregistry.AgentRegistry) *agentRegistryLifecycleAdapter {
	return &agentRegistryLifecycleAdapter{registry: registry}
}

// GetLifecycleMode returns the lifecycle mode for the given agent.
// Falls back to "spawn" if the agent is not found.
func (a *agentRegistryLifecycleAdapter) GetLifecycleMode(_ context.Context, agentName string) domain.LifecycleMode {
	agent, err := a.registry.Get(agentName)
	if err != nil {
		return domain.LifecycleModeSpawn
	}

	switch agent.Record.Lifecycle {
	case "persistent":
		return domain.LifecycleModePersistent
	default:
		return domain.LifecycleModeSpawn
	}
}

// GetMaxContextSize returns the max context size for the given agent.
// Falls back to 16000 if the agent is not found.
func (a *agentRegistryLifecycleAdapter) GetMaxContextSize(_ context.Context, agentName string) int {
	agent, err := a.registry.Get(agentName)
	if err != nil {
		return 16000
	}

	if agent.Record.MaxContextSize > 0 {
		return agent.Record.MaxContextSize
	}
	return 16000
}
