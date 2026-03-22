package tools

import (
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
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
	projectRoot    string
	chunkStore     *indexing.ChunkStore
	embedder       *indexing.EmbeddingsClient
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

// WithIndexing configures the chunk store and embedder for symbol-based tools.
func (p *DefaultToolDepsProvider) WithIndexing(projectRoot string, store *indexing.ChunkStore, embedder *indexing.EmbeddingsClient) {
	p.projectRoot = projectRoot
	p.chunkStore = store
	p.embedder = embedder
}

// GetDependencies creates ToolDependencies for a session
func (p *DefaultToolDepsProvider) GetDependencies(sessionID, projectKey string) ToolDependencies {
	return ToolDependencies{
		SessionID:      sessionID,
		ProjectKey:     projectKey,
		ProjectRoot:    p.projectRoot,
		Proxy:          p.proxy,
		TaskManager:    p.taskManager,
		SubtaskManager: p.subtaskManager,
		AgentPool:      p.agentPool,
		WebSearchTool:  p.webSearchTool,
		WebFetchTool:   p.webFetchTool,
		ChunkStore:     p.chunkStore,
		Embedder:       p.embedder,
	}
}
