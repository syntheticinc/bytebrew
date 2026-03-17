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

// EscalationAction — what happens on escalation.
const (
	EscalationActionTransferToHuman = "transfer_to_human"
	EscalationActionNotify          = "notify"
)

// ModelProviderType — LLM provider backend.
const (
	ModelProviderOllama           = "ollama"
	ModelProviderOpenAICompatible = "openai_compatible"
	ModelProviderAnthropic        = "anthropic"
)

// ToolType — agent tool classification.
const (
	ToolTypeBuiltin = "builtin"
	ToolTypeCustom  = "custom"
)

// MCPServerType — MCP server transport.
const (
	MCPServerTypeStdio = "stdio"
	MCPServerTypeHTTP  = "http"
	MCPServerTypeSSE   = "sse"
)

// MCPServerStatus — runtime connection status.
const (
	MCPServerStatusConnected    = "connected"
	MCPServerStatusError        = "error"
	MCPServerStatusConnecting   = "connecting"
	MCPServerStatusDisconnected = "disconnected"
)

// TriggerType — trigger source type.
const (
	TriggerTypeCron    = "cron"
	TriggerTypeWebhook = "webhook"
)

// TaskSource — who created the task.
const (
	TaskSourceAgent     = "agent"
	TaskSourceCron      = "cron"
	TaskSourceWebhook   = "webhook"
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
	TaskStatusEscalated  = "escalated"
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
)

// AuditActorType — who performed the action.
const (
	AuditActorAdmin    = "admin"
	AuditActorAPIToken = "api_token"
	AuditActorSystem   = "system"
	AuditActorCron     = "cron"
)
