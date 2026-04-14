package turnexecutorfactory

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
	"github.com/syntheticinc/bytebrew/engine/internal/service/turnexecutor"
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

// GuardrailConfigResolver resolves guardrail capability config for an agent.
// Returns nil when the agent has no guardrail capability configured.
type GuardrailConfigResolver interface {
	ResolveGuardrailConfig(ctx context.Context, agentName string) (*turnexecutor.GuardrailCheckConfig, error)
}

// Factory creates EngineAdapter-based TurnExecutors for Supervisor mode.
// Implements grpc.TurnExecutorFactory interface (consumer-side).
type Factory struct {
	engine        *engine.Engine
	flowManager   turnexecutor.FlowProvider
	toolResolver  *tools.AgentToolResolver
	modelSelector *llm.ModelSelector
	modelCache    *llm.ModelCache
	agentResolver AgentModelResolver
	agentConfig   *config.AgentConfig
	// Raw deps for creating per-session ToolDepsProvider
	agentPool     tools.AgentPoolForTool
	webSearchTool einotool.InvokableTool
	webFetchTool  einotool.InvokableTool
	// Getter for context reminders (from AgentService)
	contextRemindersGetter func() []turnexecutor.ContextReminderProvider
	// Memory capability deps (injected via SetMemory — nil = disabled)
	memoryRecaller  tools.MemoryRecaller
	memoryStorer    tools.MemoryStorer
	memoryMaxEntries int
	// Engine task manager (injected via SetEngineTaskManager — nil = old task system fallback)
	engineTaskManager tools.EngineTaskManager
	// Escalation capability deps (injected via SetEscalation — nil = disabled)
	escalationHandler tools.EscalationHandler
	// Schema resolver for memory/knowledge tools (BUG-007)
	schemaResolver AgentSchemaResolver
	// US-003: Guardrail pipeline (injected via SetGuardrail — nil = disabled)
	guardrailChecker        turnexecutor.GuardrailChecker
	guardrailConfigResolver GuardrailConfigResolver
	// Per-agent capability config reader (memory max_entries, etc.)
	capConfigReader tools.CapabilityConfigReader
}

// New creates a new factory for Engine-based TurnExecutors.
func New(
	engine *engine.Engine,
	flowManager turnexecutor.FlowProvider,
	toolResolver *tools.AgentToolResolver,
	modelSelector *llm.ModelSelector,
	agentConfig *config.AgentConfig,
	agentPool tools.AgentPoolForTool,
	webSearchTool, webFetchTool einotool.InvokableTool,
	contextRemindersGetter func() []turnexecutor.ContextReminderProvider,
	modelCache *llm.ModelCache,
	agentResolver AgentModelResolver,
) *Factory {
	return &Factory{
		engine:                 engine,
		flowManager:            flowManager,
		toolResolver:           toolResolver,
		modelSelector:          modelSelector,
		modelCache:             modelCache,
		agentResolver:          agentResolver,
		agentConfig:            agentConfig,
		agentPool:              agentPool,
		webSearchTool:          webSearchTool,
		webFetchTool:           webFetchTool,
		contextRemindersGetter: contextRemindersGetter,
	}
}

// SetMemory configures the memory storage for memory_recall/memory_store tools.
// Call after factory creation to enable memory capability tools.
func (f *Factory) SetMemory(recaller tools.MemoryRecaller, storer tools.MemoryStorer, maxEntries int) {
	f.memoryRecaller = recaller
	f.memoryStorer = storer
	f.memoryMaxEntries = maxEntries
}

// SetEngineTaskManager configures the DB-backed task manager so agents use EngineTask.
func (f *Factory) SetEngineTaskManager(mgr tools.EngineTaskManager) {
	f.engineTaskManager = mgr
}

// SetEscalation configures the escalation handler for the escalate tool.
func (f *Factory) SetEscalation(handler tools.EscalationHandler) {
	f.escalationHandler = handler
}

// SetSchemaResolver configures schema lookup for propagating SchemaID to tool deps.
func (f *Factory) SetSchemaResolver(resolver AgentSchemaResolver) {
	f.schemaResolver = resolver
}

// SetGuardrail configures the guardrail checker and per-agent config resolver.
func (f *Factory) SetGuardrail(checker turnexecutor.GuardrailChecker, resolver GuardrailConfigResolver) {
	f.guardrailChecker = checker
	f.guardrailConfigResolver = resolver
}

// SetCapabilityConfigReader configures per-agent capability config resolution.
func (f *Factory) SetCapabilityConfigReader(reader tools.CapabilityConfigReader) {
	f.capConfigReader = reader
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
func (f *Factory) CreateForSession(
	proxy tools.ClientOperationsProxy,
	sessionID, projectKey string,
	projectRoot, platform, agentName, userID string,
) orchestrator.TurnExecutor {
	// Create per-session ToolDepsProvider with proxy for this session
	baseDeps := tools.NewDefaultToolDepsProvider(
		proxy,
		f.agentPool,
		f.webSearchTool,
		f.webFetchTool,
	)
	if f.engineTaskManager != nil {
		baseDeps.SetEngineTaskManager(f.engineTaskManager)
	}
	// Resolve per-agent memory max_entries from capability config
	memMaxEntries := f.memoryMaxEntries
	if f.capConfigReader != nil {
		if cfg, err := f.capConfigReader.ReadConfig(context.Background(), agentName, "memory"); err == nil && cfg != nil {
			unlimitedEntries, _ := cfg["unlimited_entries"].(bool)
			if !unlimitedEntries {
				if me, ok := cfg["max_entries"].(float64); ok && int(me) > 0 {
					memMaxEntries = int(me)
				}
			}
		}
	}

	// Wrap with per-user memory + escalation deps
	toolDeps := &userMemoryDepsProvider{
		base:              baseDeps,
		userID:            userID,
		memoryRecaller:    f.memoryRecaller,
		memoryStorer:      f.memoryStorer,
		memoryMaxEntries:  memMaxEntries,
		escalationHandler: f.escalationHandler,
	}

	// Get context reminders from getter (if provided)
	var contextReminders []turnexecutor.ContextReminderProvider
	if f.contextRemindersGetter != nil {
		contextReminders = f.contextRemindersGetter()
	}

	// Create per-request EnvironmentContextReminder (replaces any global one from getter)
	if projectRoot != "" || platform != "" {
		envReminder := agentservice.NewEnvironmentContextReminder(projectRoot, platform)
		var filtered []turnexecutor.ContextReminderProvider
		for _, r := range contextReminders {
			if _, ok := r.(*agentservice.EnvironmentContextReminder); !ok {
				filtered = append(filtered, r)
			}
		}
		contextReminders = append(filtered, envReminder)
	}

	// Append capability prompt hints so the agent knows about its capabilities.
	if f.capConfigReader != nil {
		var hints []string
		for _, cap := range []struct{ name, hint string }{
			{"memory", "You have Memory capability. Use memory_recall at the start of conversations to check for prior context about this user. Use memory_store to save important facts for future conversations."},
			{"knowledge", "You have Knowledge capability. Use knowledge_search to find relevant information from your knowledge base before answering questions."},
			{"escalation", "You have Escalation capability. Use escalate when a request is beyond your scope or requires human intervention."},
		} {
			if cfg, err := f.capConfigReader.ReadConfig(context.Background(), agentName, cap.name); err == nil && cfg != nil {
				hints = append(hints, cap.hint)
			}
		}
		if len(hints) > 0 {
			contextReminders = append(contextReminders, &capabilityHintReminder{hints: hints})
		}
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

	// US-003: Resolve guardrail config for this agent (nil = no guardrails).
	var guardrailConfig *turnexecutor.GuardrailCheckConfig
	if f.guardrailConfigResolver != nil {
		if cfg, err := f.guardrailConfigResolver.ResolveGuardrailConfig(context.Background(), agentName); err == nil && cfg != nil {
			guardrailConfig = cfg
		}
	}

	// Create EngineAdapter (implements TurnExecutor interface)
	adapter, err := turnexecutor.NewEngineAdapter(turnexecutor.Config{
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
		Guardrail:        f.guardrailChecker,
		GuardrailConfig:  guardrailConfig,
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
func (f *Factory) resolveModel(agentName string) (model.ToolCallingChatModel, string) {
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

// capabilityHintReminder injects capability usage hints into the agent's context.
type capabilityHintReminder struct {
	hints []string
}

func (r *capabilityHintReminder) GetContextReminder(_ context.Context, _ string) (string, int, bool) {
	if len(r.hints) == 0 {
		return "", 0, false
	}
	content := "## Capabilities\n"
	for _, h := range r.hints {
		content += "- " + h + "\n"
	}
	return content, 5, true // priority 5: after env, before tasks
}
