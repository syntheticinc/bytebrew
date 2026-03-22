package tools

import (
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/cloudwego/eino/components/tool"
)

// ToolDependencies holds all dependencies needed by tools
type ToolDependencies struct {
	SessionID          string
	AgentName          string
	KnowledgePath      string
	ProjectKey         string
	ProjectRoot        string
	BackgroundMode     bool               // true for cron/webhook/API tasks (no user interaction)
	Proxy              ClientOperationsProxy
	TaskManager        TaskManager
	SubtaskManager     SubtaskManager
	AgentPool          AgentPoolForTool
	EngineTaskManager  EngineTaskManager  // Phase 4: engine task CRUD
	WebSearchTool      tool.InvokableTool // pre-created (depends on API key)
	WebFetchTool       tool.InvokableTool // pre-created
	ChunkStore         *indexing.ChunkStore
	Embedder           *indexing.EmbeddingsClient
	MCPServers         []string           // MCP server names for legacy Resolve path
	CanSpawn           []string           // target agent names this agent can spawn (legacy Resolve path)
}

