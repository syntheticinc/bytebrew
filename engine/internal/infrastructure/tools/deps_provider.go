package tools

import (
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
)

// DefaultToolDepsProvider creates ToolDependencies for a given session
type DefaultToolDepsProvider struct {
	proxy             ClientOperationsProxy
	agentPool         AgentPoolForTool
	engineTaskManager EngineTaskManager
	projectRoot       string
	chunkStore        *indexing.ChunkStore
	embedder          *indexing.EmbeddingsClient
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

// WithIndexing configures the chunk store and embedder for symbol-based tools.
func (p *DefaultToolDepsProvider) WithIndexing(projectRoot string, store *indexing.ChunkStore, embedder *indexing.EmbeddingsClient) {
	p.projectRoot = projectRoot
	p.chunkStore = store
	p.embedder = embedder
}

// GetDependencies creates ToolDependencies for a session
func (p *DefaultToolDepsProvider) GetDependencies(sessionID, projectKey string) ToolDependencies {
	return ToolDependencies{
		SessionID:         sessionID,
		ProjectKey:        projectKey,
		ProjectRoot:       p.projectRoot,
		Proxy:             p.proxy,
		AgentPool:         p.agentPool,
		EngineTaskManager: p.engineTaskManager,
		ChunkStore:        p.chunkStore,
		Embedder:          p.embedder,
	}
}
