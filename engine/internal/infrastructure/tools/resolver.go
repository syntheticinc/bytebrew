package tools

import (
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/cloudwego/eino/components/tool"
)

// ToolEventEmitter sends agent events from tools (e.g. structured output).
type ToolEventEmitter interface {
	Send(event *domain.AgentEvent) error
}

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
	EventEmitter       ToolEventEmitter   // event stream for tools that emit events
	WebSearchTool      tool.InvokableTool // pre-created (depends on API key)
	WebFetchTool       tool.InvokableTool // pre-created
	ChunkStore         *indexing.ChunkStore
	Embedder           *indexing.EmbeddingsClient
	MCPServers          []string           // MCP server names for legacy Resolve path
	CanSpawn            []string           // target agent names this agent can spawn (legacy Resolve path)
	// Memory capability deps (US-001: injected when agent has Memory capability)
	SchemaID            string             // agent's schema ID for memory scoping
	UserID              string             // end-user ID for memory scoping
	MemoryRecaller      MemoryRecaller     // nil → memory_recall disabled
	MemoryStorer        MemoryStorer       // nil → memory_store disabled
	MemoryMaxEntries    int                // 0 → unlimited
	ConfirmBefore      []string              // tools requiring user confirmation before execution
	ConfirmRequester   ConfirmationRequester // confirmation handler for confirm_before tools (nil = no wrapping)
}

