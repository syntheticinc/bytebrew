package models

// AgentLifecycle — agent lifecycle type.
const (
	AgentLifecyclePersistent = "persistent"
	AgentLifecycleSpawn      = "spawn"
)

// ToolExecutionMode — how agent runs tools.
const (
	ToolExecutionSequential = "sequential"
	ToolExecutionParallel   = "parallel"
)

// ModelProviderType — LLM provider backend.
const (
	ModelProviderOllama           = "ollama"
	ModelProviderOpenAICompatible = "openai_compatible"
	ModelProviderAnthropic        = "anthropic"
	ModelProviderAzureOpenAI      = "azure_openai"
)

// ToolType — agent tool classification.
const (
	ToolTypeBuiltin = "builtin"
	ToolTypeCustom  = "custom"
)

// MCPServerType — MCP server transport.
const (
	MCPServerTypeStdio          = "stdio"
	MCPServerTypeHTTP           = "http"
	MCPServerTypeSSE            = "sse"
	MCPServerTypeStreamableHTTP = "streamable-http"
)

// MCPServerStatus — runtime connection status.
const (
	MCPServerStatusConnected    = "connected"
	MCPServerStatusError        = "error"
	MCPServerStatusConnecting   = "connecting"
	MCPServerStatusDisconnected = "disconnected"
)

// TaskSource — who created the task.
const (
	TaskSourceAgent     = "agent"
	TaskSourceAPI       = "api"
	TaskSourceDashboard = "dashboard"
)

// TaskStatus — task lifecycle state.
const (
	TaskStatusPending    = "pending"
	TaskStatusInProgress = "in_progress"
	TaskStatusCompleted  = "completed"
	TaskStatusFailed     = "failed"
	TaskStatusNeedsInput = "needs_input"
	TaskStatusCancelled  = "cancelled"
)

// TaskMode — task execution mode.
const (
	TaskModeInteractive = "interactive"
	TaskModeBackground  = "background"
)

// SessionStatus — session lifecycle state.
const (
	SessionStatusActive    = "active"
	SessionStatusCompleted = "completed"
	SessionStatusExpired   = "expired"
	SessionStatusFailed    = "failed"
)

// AuditActorType — who performed the action.
const (
	AuditActorAdmin    = "admin"
	AuditActorAPIToken = "api_token"
	AuditActorSystem   = "system"
)

// RuntimeEventType — type of runtime event in the session timeline.
const (
	RuntimeEventUserMessage      = "user_message"
	RuntimeEventAssistantMessage = "assistant_message"
	RuntimeEventToolCall         = "tool_call"
	RuntimeEventToolResult       = "tool_result"
	RuntimeEventReasoning        = "reasoning"
	RuntimeEventSystem           = "system"
)
