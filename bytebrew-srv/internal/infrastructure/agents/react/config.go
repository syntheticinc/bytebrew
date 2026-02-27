package react

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// PlanManager defines interface for plan orchestration (consumer-side interface)
type PlanManager interface {
	CreatePlan(ctx context.Context, sessionID, goal string, steps []*domain.PlanStep) (*domain.Plan, error)
	GetActivePlan(ctx context.Context, sessionID string) (*domain.Plan, error)
	UpdateStepStatus(ctx context.Context, sessionID string, stepIdx int, status domain.PlanStepStatus, result string) error
	UpdatePlanStatus(ctx context.Context, sessionID string, status domain.PlanStatus) error
	AddStep(ctx context.Context, sessionID, description, reasoning string) error
	RemoveStep(ctx context.Context, sessionID string, stepIndex int) error
	ModifyStep(ctx context.Context, sessionID string, stepIndex int, description, reasoning string) error
}

// ToolCallRecorder defines interface for recording tool calls and results.
// Consumer-side interface: defined here where it's used.
type ToolCallRecorder interface {
	RecordToolCall(sessionID, toolName string)
	RecordToolResult(sessionID, toolName, result string)
}

// AgentConfig holds configuration for ReAct agent
type AgentConfig struct {
	ChatModel                model.ToolCallingChatModel
	Tools                    []tool.BaseTool
	MaxSteps                 int
	SessionID                string
	AgentConfig              *config.AgentConfig
	ModelName                string            // Model name for reasoning extraction
	HistoryMessages          []*schema.Message // Conversation history (user/assistant messages)
	PlanManager              PlanManager       // Plan manager for planning system
	ContextReminderProviders []ContextReminderProvider // External context reminder providers (e.g., WorkContextReminder)
	ToolCallRecorder         ToolCallRecorder  // Records tool calls for efficiency reminders
	AgentID                  string // "supervisor" | "code-agent-xxx" (for log separation)
	ParentAgentID            string // parent agent ID (for Code Agents → "supervisor")
	SubtaskID                string // subtask being executed (for Code Agents)
	SessionDirName           string // shared session dir name (set by parent to keep all logs together)
}
