package infrastructure

import (
	"context"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	einotool "github.com/cloudwego/eino/components/tool"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	agentservice "github.com/syntheticinc/bytebrew/engine/internal/service/agent"
	"github.com/syntheticinc/bytebrew/engine/internal/service/engine"
	"github.com/syntheticinc/bytebrew/engine/internal/service/orchestrator"
	"github.com/syntheticinc/bytebrew/engine/internal/service/turn_executor"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
)

// AgentModelResolver looks up the ModelID associated with a named agent.
// Returns nil when the agent has no per-agent model configured.
type AgentModelResolver interface {
	ResolveModelID(agentName string) *string
}

// AgentSchemaResolver resolves the primary schema ID (UUID) for an agent.
// Used to propagate SchemaID into tool dependencies for memory/knowledge tools.
type AgentSchemaResolver interface {
	ResolveSchemaID(ctx context.Context, agentName string) (string, error)
}

// EngineTurnExecutorFactory creates EngineAdapter-based TurnExecutors for Supervisor mode.
// Implements grpc.TurnExecutorFactory interface (consumer-side).
type EngineTurnExecutorFactory struct {
	engine        *engine.Engine
	flowManager   turn_executor.FlowProvider
	toolResolver  *tools.AgentToolResolver
	modelSelector *llm.ModelSelector
	modelCache    *llm.ModelCache
	agentResolver AgentModelResolver
	agentConfig   *config.AgentConfig
	// Raw deps for creating per-session ToolDepsProvider
	taskManager    tools.TaskManager
	subtaskManager tools.SubtaskManager
	agentPool      tools.AgentPoolForTool
	webSearchTool  einotool.InvokableTool
	webFetchTool   einotool.InvokableTool
	// Getter for context reminders (from AgentService)
	contextRemindersGetter func() []turn_executor.ContextReminderProvider
	// Memory capability deps (injected via SetMemory — nil = disabled)
	memoryRecaller  tools.MemoryRecaller
	memoryStorer    tools.MemoryStorer
	memoryMaxEntries int
	// Escalation capability deps (injected via SetEscalation — nil = disabled)
	escalationHandler tools.EscalationHandler
	// Schema resolver for memory/knowledge tools (BUG-007)
	schemaResolver AgentSchemaResolver
}

// NewEngineTurnExecutorFactory creates a new factory for Engine-based TurnExecutors.
func NewEngineTurnExecutorFactory(
	engine *engine.Engine,
	flowManager turn_executor.FlowProvider,
	toolResolver *tools.AgentToolResolver,
	modelSelector *llm.ModelSelector,
	agentConfig *config.AgentConfig,
	taskManager tools.TaskManager,
	subtaskManager tools.SubtaskManager,
	agentPool tools.AgentPoolForTool,
	webSearchTool, webFetchTool einotool.InvokableTool,
	contextRemindersGetter func() []turn_executor.ContextReminderProvider,
	modelCache *llm.ModelCache,
	agentResolver AgentModelResolver,
) *EngineTurnExecutorFactory {
	return &EngineTurnExecutorFactory{
		engine:                 engine,
		flowManager:            flowManager,
		toolResolver:           toolResolver,
		modelSelector:          modelSelector,
		modelCache:             modelCache,
		agentResolver:          agentResolver,
		agentConfig:            agentConfig,
		taskManager:            taskManager,
		subtaskManager:         subtaskManager,
		agentPool:              agentPool,
		webSearchTool:          webSearchTool,
		webFetchTool:           webFetchTool,
		contextRemindersGetter: contextRemindersGetter,
	}
}

// SetMemory configures the memory storage for memory_recall/memory_store tools.
// Call after factory creation to enable memory capability tools.
func (f *EngineTurnExecutorFactory) SetMemory(recaller tools.MemoryRecaller, storer tools.MemoryStorer, maxEntries int) {
	f.memoryRecaller = recaller
	f.memoryStorer = storer
	f.memoryMaxEntries = maxEntries
}

// SetEscalation configures the escalation handler for the escalate tool.
func (f *EngineTurnExecutorFactory) SetEscalation(handler tools.EscalationHandler) {
	f.escalationHandler = handler
}

// SetSchemaResolver configures schema lookup for propagating SchemaID to tool deps.
func (f *EngineTurnExecutorFactory) SetSchemaResolver(resolver AgentSchemaResolver) {
	f.schemaResolver = resolver
}

// userMemoryDepsProvider wraps DefaultToolDepsProvider and injects userID + memory + escalation refs per session.
type userMemoryDepsProvider struct {
	base              *tools.DefaultToolDepsProvider
	userID            string
	memoryRecaller    tools.MemoryRecaller
	memoryStorer      tools.MemoryStorer
	memoryMaxEntries  int
	escalationHandler tools.EscalationHandler
}

func (p *userMemoryDepsProvider) GetDependencies(sessionID, projectKey string) tools.ToolDependencies {
	deps := p.base.GetDependencies(sessionID, projectKey)
	deps.UserID = p.userID
	deps.MemoryRecaller = p.memoryRecaller
	deps.MemoryStorer = p.memoryStorer
	deps.MemoryMaxEntries = p.memoryMaxEntries
	deps.EscalationHandler = p.escalationHandler
	return deps
}

// CreateForSession creates a TurnExecutor for the given session.
// Implements grpc.TurnExecutorFactory interface.
func (f *EngineTurnExecutorFactory) CreateForSession(
	proxy tools.ClientOperationsProxy,
	sessionID, projectKey string,
	projectRoot, platform, agentName, userID string,
) orchestrator.TurnExecutor {
	// Create per-session ToolDepsProvider with proxy for this session
	baseDeps := tools.NewDefaultToolDepsProvider(
		proxy,
		f.taskManager,
		f.subtaskManager,
		f.agentPool,
		f.webSearchTool,
		f.webFetchTool,
	)
	// Wrap with per-user memory + escalation deps
	toolDeps := &userMemoryDepsProvider{
		base:              baseDeps,
		userID:            userID,
		memoryRecaller:    f.memoryRecaller,
		memoryStorer:      f.memoryStorer,
		memoryMaxEntries:  f.memoryMaxEntries,
		escalationHandler: f.escalationHandler,
	}

	// Get context reminders from getter (if provided)
	var contextReminders []turn_executor.ContextReminderProvider
	if f.contextRemindersGetter != nil {
		contextReminders = f.contextRemindersGetter()
	}

	// Create per-request EnvironmentContextReminder (replaces any global one from getter)
	if projectRoot != "" || platform != "" {
		envReminder := agentservice.NewEnvironmentContextReminder(projectRoot, platform)
		var filtered []turn_executor.ContextReminderProvider
		for _, r := range contextReminders {
			if _, ok := r.(*agentservice.EnvironmentContextReminder); !ok {
				filtered = append(filtered, r)
			}
		}
		contextReminders = append(filtered, envReminder)
	}

	// Resolve model: try per-agent DB model first, fall back to ModelSelector.
	chatModel, modelName := f.resolveModel(agentName)
	if chatModel == nil {
		slog.Error("no model available for agent — add a model via Admin Dashboard",
			"agent", agentName)
		return nil
	}

	// BUG-007: Resolve agent's schema for memory/knowledge tool deps.
	var schemaID string
	if f.schemaResolver != nil {
		sid, err := f.schemaResolver.ResolveSchemaID(context.Background(), agentName)
		if err != nil {
			slog.Warn("failed to resolve schema for agent, memory tools may be disabled",
				"agent", agentName, "error", err)
		} else {
			schemaID = sid
		}
	}

	// Create EngineAdapter (implements TurnExecutor interface)
	adapter, err := turn_executor.NewEngineAdapter(turn_executor.Config{
		Engine:           f.engine,
		FlowProvider:     f.flowManager,
		ToolResolver:     f.toolResolver,
		ToolDeps:         toolDeps,
		ChatModel:        chatModel,
		AgentConfig:      f.agentConfig,
		ModelName:        modelName,
		AgentName:        agentName,
		SchemaID:         schemaID,
		ContextReminders: contextReminders,
	})

	if err != nil {
		// Shouldn't happen if factory was created successfully.
		// If this occurs, Orchestrator will fail gracefully with nil TurnExecutor.
		return nil
	}

	return adapter
}

// resolveModel tries to resolve a model from the DB cache via the agent's ModelID.
// Falls back to the static ModelSelector when no per-agent model is configured
// or when the cache is not available.
func (f *EngineTurnExecutorFactory) resolveModel(agentName string) (model.ToolCallingChatModel, string) {
	if f.modelCache != nil && f.agentResolver != nil {
		modelID := f.agentResolver.ResolveModelID(agentName)
		if modelID != nil {
			client, name, err := f.modelCache.Get(context.Background(), *modelID)
			if err != nil {
				slog.Error("failed to resolve model from cache, falling back to selector",
					"agent", agentName, "model_id", *modelID, "error", err)
			} else {
				return client, name
			}
		}
	}

	// Fallback: static ModelSelector (legacy config or no per-agent model)
	flowType := domain.FlowType(agentName)
	return f.modelSelector.Select(flowType), f.modelSelector.ModelName(flowType)
}
