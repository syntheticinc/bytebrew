package tools

import (
	"github.com/cloudwego/eino/components/tool"
)

// DefaultToolDepsProvider creates ToolDependencies for a given session
type DefaultToolDepsProvider struct {
	proxy          ClientOperationsProxy
	taskManager    TaskManager
	subtaskManager SubtaskManager
	agentPool      AgentPoolForTool
	webSearchTool  tool.InvokableTool
	webFetchTool   tool.InvokableTool
}

// NewDefaultToolDepsProvider creates a new provider
func NewDefaultToolDepsProvider(
	proxy ClientOperationsProxy,
	taskManager TaskManager,
	subtaskManager SubtaskManager,
	agentPool AgentPoolForTool,
	webSearchTool, webFetchTool tool.InvokableTool,
) *DefaultToolDepsProvider {
	return &DefaultToolDepsProvider{
		proxy:          proxy,
		taskManager:    taskManager,
		subtaskManager: subtaskManager,
		agentPool:      agentPool,
		webSearchTool:  webSearchTool,
		webFetchTool:   webFetchTool,
	}
}

// GetDependencies creates ToolDependencies for a session
func (p *DefaultToolDepsProvider) GetDependencies(sessionID, projectKey string) ToolDependencies {
	return ToolDependencies{
		SessionID:      sessionID,
		ProjectKey:     projectKey,
		Proxy:          p.proxy,
		TaskManager:    p.taskManager,
		SubtaskManager: p.subtaskManager,
		AgentPool:      p.agentPool,
		WebSearchTool:  p.webSearchTool,
		WebFetchTool:   p.webFetchTool,
	}
}
