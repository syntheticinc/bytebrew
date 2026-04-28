package tools

// DefaultToolDepsProvider creates ToolDependencies for a given session.
type DefaultToolDepsProvider struct {
	proxy             ClientOperationsProxy
	agentPool         AgentPoolForTool
	engineTaskManager EngineTaskManager
}

// NewDefaultToolDepsProvider creates a new provider.
func NewDefaultToolDepsProvider(
	proxy ClientOperationsProxy,
	agentPool AgentPoolForTool,
) *DefaultToolDepsProvider {
	return &DefaultToolDepsProvider{
		proxy:     proxy,
		agentPool: agentPool,
	}
}

// SetEngineTaskManager configures the unified EngineTask-based manager.
func (p *DefaultToolDepsProvider) SetEngineTaskManager(mgr EngineTaskManager) {
	p.engineTaskManager = mgr
}

// GetDependencies creates ToolDependencies for a session.
func (p *DefaultToolDepsProvider) GetDependencies(sessionID, projectKey string) ToolDependencies {
	return ToolDependencies{
		SessionID:         sessionID,
		ProjectKey:        projectKey,
		Proxy:             p.proxy,
		AgentPool:         p.agentPool,
		EngineTaskManager: p.engineTaskManager,
	}
}
