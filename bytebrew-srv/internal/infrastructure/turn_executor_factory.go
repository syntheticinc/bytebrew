package infrastructure

import (
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/engine"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/orchestrator"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/turn_executor"
	agentservice "github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/agent"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/config"
	einotool "github.com/cloudwego/eino/components/tool"
)

// EngineTurnExecutorFactory creates EngineAdapter-based TurnExecutors for Supervisor mode.
// Implements grpc.TurnExecutorFactory interface (consumer-side).
type EngineTurnExecutorFactory struct {
	engine        *engine.Engine
	flowManager   *agentservice.FlowManager
	toolResolver  *tools.DefaultToolResolver
	modelSelector *llm.ModelSelector
	agentConfig   *config.AgentConfig
	// Raw deps for creating per-session ToolDepsProvider
	taskManager    tools.TaskManager
	subtaskManager tools.SubtaskManager
	agentPool      tools.AgentPoolForTool
	webSearchTool  einotool.InvokableTool
	webFetchTool   einotool.InvokableTool
	// Getter for context reminders (from AgentService)
	contextRemindersGetter func() []turn_executor.ContextReminderProvider
}

// NewEngineTurnExecutorFactory creates a new factory for Engine-based TurnExecutors.
func NewEngineTurnExecutorFactory(
	engine *engine.Engine,
	flowManager *agentservice.FlowManager,
	toolResolver *tools.DefaultToolResolver,
	modelSelector *llm.ModelSelector,
	agentConfig *config.AgentConfig,
	taskManager tools.TaskManager,
	subtaskManager tools.SubtaskManager,
	agentPool tools.AgentPoolForTool,
	webSearchTool, webFetchTool einotool.InvokableTool,
	contextRemindersGetter func() []turn_executor.ContextReminderProvider,
) *EngineTurnExecutorFactory {
	return &EngineTurnExecutorFactory{
		engine:                 engine,
		flowManager:            flowManager,
		toolResolver:           toolResolver,
		modelSelector:          modelSelector,
		agentConfig:            agentConfig,
		taskManager:            taskManager,
		subtaskManager:         subtaskManager,
		agentPool:              agentPool,
		webSearchTool:          webSearchTool,
		webFetchTool:           webFetchTool,
		contextRemindersGetter: contextRemindersGetter,
	}
}

// CreateForSession creates a TurnExecutor for the given session.
// Implements grpc.TurnExecutorFactory interface.
func (f *EngineTurnExecutorFactory) CreateForSession(
	proxy tools.ClientOperationsProxy,
	sessionID, projectKey string,
) orchestrator.TurnExecutor {
	// Create per-session ToolDepsProvider with proxy for this session
	toolDeps := tools.NewDefaultToolDepsProvider(
		proxy,
		f.taskManager,
		f.subtaskManager,
		f.agentPool,
		f.webSearchTool,
		f.webFetchTool,
	)

	// Get context reminders from getter (if provided)
	var contextReminders []turn_executor.ContextReminderProvider
	if f.contextRemindersGetter != nil {
		contextReminders = f.contextRemindersGetter()
	}

	// Create EngineAdapter (implements TurnExecutor interface)
	adapter, err := turn_executor.NewEngineAdapter(turn_executor.Config{
		Engine:           f.engine,
		FlowProvider:     f.flowManager,
		ToolResolver:     f.toolResolver,
		ToolDeps:         toolDeps,
		ChatModel:        f.modelSelector.Select(domain.FlowTypeSupervisor),
		AgentConfig:      f.agentConfig,
		ModelName:        f.modelSelector.ModelName(domain.FlowTypeSupervisor),
		ContextReminders: contextReminders,
	})

	if err != nil {
		// Shouldn't happen if factory was created successfully.
		// If this occurs, Orchestrator will fail gracefully with nil TurnExecutor.
		return nil
	}

	return adapter
}
