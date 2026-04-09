package tools

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/cloudwego/eino/components/tool"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
)

// BuiltinToolFactory creates a tool instance given dependencies.
type BuiltinToolFactory func(deps ToolDependencies) tool.InvokableTool

// BuiltinToolStore stores builtin tool factories by name.
type BuiltinToolStore struct {
	factories map[string]BuiltinToolFactory
}

// NewBuiltinToolStore creates an empty BuiltinToolStore.
func NewBuiltinToolStore() *BuiltinToolStore {
	return &BuiltinToolStore{factories: make(map[string]BuiltinToolFactory)}
}

// Register adds a factory for the given tool name.
func (s *BuiltinToolStore) Register(name string, factory BuiltinToolFactory) {
	s.factories[name] = factory
}

// Get returns the factory for the given name.
func (s *BuiltinToolStore) Get(name string) (BuiltinToolFactory, bool) {
	f, ok := s.factories[name]
	return f, ok
}

// Names returns all registered tool names in alphabetical order.
func (s *BuiltinToolStore) Names() []string {
	names := make([]string, 0, len(s.factories))
	for name := range s.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// KitProvider looks up a kit by name and returns its tools for a session.
type KitProvider interface {
	Get(name string) (Kit, error)
}

// Kit is the consumer-side interface for domain-specific kits.
type Kit interface {
	Tools(session domain.KitSession) []tool.InvokableTool
	PostToolCall(ctx context.Context, session domain.KitSession, toolName string, result string) *domain.Enrichment
}

// MCPClientProvider provides MCP tools for a given MCP server name.
// Defined on the consumer side (AgentToolResolver).
type MCPClientProvider interface {
	// GetMCPTools returns Eino-compatible tools for the named MCP server.
	// Returns nil, nil if the server is not connected.
	GetMCPTools(name string) ([]tool.InvokableTool, error)
}

// CapabilityToolInjector returns additional tool names based on agent capabilities.
type CapabilityToolInjector interface {
	InjectedTools(ctx context.Context, agentName string) ([]string, error)
}

// CircuitBreakerRegistry provides circuit breakers for named resources.
type CircuitBreakerRegistry interface {
	Get(name string) CircuitBreakerChecker
}

// AgentToolResolver composes tools for a specific agent from various sources.
type AgentToolResolver struct {
	builtins          *BuiltinToolStore
	kitProvider       KitProvider
	knowledgeSearcher KnowledgeSearcher
	knowledgeEmbedder KnowledgeEmbedder
	mcpProvider       MCPClientProvider
	spawner           GenericAgentSpawner
	inspector         GenericAgentInspector
	capInjector       CapabilityToolInjector
	policyEvaluator   PolicyEvaluator
	cbRegistry        CircuitBreakerRegistry
	recoveryExecutor  RecoveryExecutor
	toolTimeoutMs     int64 // 0 = disabled
}

// NewAgentToolResolver creates a new AgentToolResolver.
func NewAgentToolResolver(builtins *BuiltinToolStore) *AgentToolResolver {
	return &AgentToolResolver{builtins: builtins}
}

// SetKitProvider configures the kit provider for kit-based tool resolution.
func (r *AgentToolResolver) SetKitProvider(kp KitProvider) {
	r.kitProvider = kp
}

// SetKnowledge configures knowledge search dependencies for auto-injection.
func (r *AgentToolResolver) SetKnowledge(searcher KnowledgeSearcher, embedder KnowledgeEmbedder) {
	r.knowledgeSearcher = searcher
	r.knowledgeEmbedder = embedder
}

// SetMCPProvider configures the MCP client provider for MCP tool resolution.
func (r *AgentToolResolver) SetMCPProvider(provider MCPClientProvider) {
	r.mcpProvider = provider
}

// SetSpawner configures the spawner and inspector for spawn tool resolution via legacy Resolve path.
func (r *AgentToolResolver) SetSpawner(spawner GenericAgentSpawner, inspector GenericAgentInspector) {
	r.spawner = spawner
	r.inspector = inspector
}

// SetCapabilityInjector configures the capability injector for auto-injecting tools based on agent capabilities.
func (r *AgentToolResolver) SetCapabilityInjector(injector CapabilityToolInjector) {
	r.capInjector = injector
}

// SetPolicyEvaluator configures the policy evaluator for wrapping tools with policy checks.
func (r *AgentToolResolver) SetPolicyEvaluator(evaluator PolicyEvaluator) {
	r.policyEvaluator = evaluator
}

// SetCircuitBreakerRegistry configures the circuit breaker registry for MCP tool protection.
func (r *AgentToolResolver) SetCircuitBreakerRegistry(registry CircuitBreakerRegistry) {
	r.cbRegistry = registry
}

// SetRecoveryExecutor configures the recovery executor for MCP tool failure recovery.
func (r *AgentToolResolver) SetRecoveryExecutor(executor RecoveryExecutor) {
	r.recoveryExecutor = executor
}

// SetToolTimeout configures the per-MCP-tool-call timeout in milliseconds (AC-RESIL-05).
func (r *AgentToolResolver) SetToolTimeout(timeoutMs int64) {
	r.toolTimeoutMs = timeoutMs
}

// ResolveContext holds per-agent resolution context.
type ResolveContext struct {
	Agent            *agent_registry.RegisteredAgent
	Deps             ToolDependencies
	KitSession       *domain.KitSession      // nil if agent has no kit
	ConfirmRequester ConfirmationRequester    // nil if no confirmation support
	Spawner          GenericAgentSpawner      // nil if spawn not available
	Inspector        GenericAgentInspector    // nil if inspect not available
	KnowledgeSearcher KnowledgeSearcher       // nil if no knowledge DB
	KnowledgeEmbedder KnowledgeEmbedder       // nil if no embeddings
}

// ResolveForAgent returns tools available to a specific agent.
// Only tools listed in the agent's BuiltinTools whitelist are resolved.
// Unknown tool names produce an error.
func (r *AgentToolResolver) ResolveForAgent(ctx context.Context, rc ResolveContext) ([]tool.InvokableTool, error) {
	var tools []tool.InvokableTool

	// US-001: Inject capability-derived tool names
	builtinTools := rc.Agent.Record.BuiltinTools
	if r.capInjector != nil {
		injected, err := r.capInjector.InjectedTools(ctx, rc.Agent.Record.Name)
		if err != nil {
			slog.WarnContext(ctx, "capability injection failed in ResolveForAgent, continuing",
				"agent", rc.Agent.Record.Name, "error", err)
		} else if len(injected) > 0 {
			existing := make(map[string]bool, len(builtinTools))
			for _, n := range builtinTools {
				existing[n] = true
			}
			for _, n := range injected {
				if !existing[n] {
					builtinTools = append(builtinTools, n)
					existing[n] = true
				}
			}
		}
	}

	for _, name := range builtinTools {
		// knowledge_search is auto-injected below based on KnowledgePath — skip here
		if name == "knowledge_search" {
			continue
		}
		factory, ok := r.builtins.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown builtin tool %q for agent %q", name, rc.Agent.Record.Name)
		}
		t := factory(rc.Deps)
		if t == nil {
			continue // tool disabled (e.g. ask_user in background mode)
		}
		tools = append(tools, t)
	}

	// Phase 2.3: Generate spawn_{name} tools from can_spawn
	if rc.Spawner != nil {
		for _, targetName := range rc.Agent.Record.CanSpawn {
			spawnTool := NewSpawnTool(targetName, rc.Deps.SessionID, rc.Spawner, rc.Inspector)
			tools = append(tools, spawnTool)
		}
	}

	// Phase 2.6: custom declarative tools from agent config
	for _, ct := range rc.Agent.Record.CustomTools {
		cfg := config.CustomToolConfig{Name: ct.Name}
		// ct.Config is JSON — parse if needed. For now, use name-only stub.
		dt := NewDeclarativeTool(cfg)
		tools = append(tools, dt)
	}

	// Phase 2.7: wrap confirm_before tools with ConfirmationWrapper
	if len(rc.Agent.Record.ConfirmBefore) > 0 && rc.ConfirmRequester != nil {
		confirmSet := make(map[string]bool, len(rc.Agent.Record.ConfirmBefore))
		for _, name := range rc.Agent.Record.ConfirmBefore {
			confirmSet[name] = true
		}
		for i, t := range tools {
			info, _ := t.Info(ctx)
			if info != nil && confirmSet[info.Name] {
				tools[i] = NewConfirmationWrapper(t, rc.ConfirmRequester)
			}
		}
	}

	// Knowledge search — auto-inject when agent has KnowledgePath configured
	// Use ResolveContext deps first, fallback to resolver-level deps
	ks := rc.KnowledgeSearcher
	ke := rc.KnowledgeEmbedder
	if ks == nil {
		ks = r.knowledgeSearcher
	}
	if ke == nil {
		ke = r.knowledgeEmbedder
	}
	if rc.Agent.Record.KnowledgePath != "" && ks != nil && ke != nil {
		knowledgeTool := NewKnowledgeSearchTool(rc.Agent.Record.Name, ks, ke)
		tools = append(tools, knowledgeTool)
	} else if hasToolInList(rc.Agent.Record.BuiltinTools, "knowledge_search") {
		slog.WarnContext(ctx, "agent has knowledge_search in tools but knowledge path is not configured — skipping",
			"agent", rc.Agent.Record.Name,
			"knowledge_path", rc.Agent.Record.KnowledgePath,
			"searcher_available", ks != nil,
			"embedder_available", ke != nil)
	}

	// Phase 3: Kit tools — append tools provided by the agent's kit
	kitTools, err := r.resolveKitTools(rc)
	if err != nil {
		return nil, fmt.Errorf("resolve kit tools for agent %q: %w", rc.Agent.Record.Name, err)
	}
	tools = append(tools, kitTools...)

	// MCP tools — append tools from connected MCP servers configured for this agent.
	// Circuit breaker (US-006) and recovery (US-005) wrapping happens inside resolveMCPTools.
	mcpTools, err := r.resolveMCPTools(rc)
	if err != nil {
		return nil, fmt.Errorf("resolve mcp tools for agent %q: %w", rc.Agent.Record.Name, err)
	}
	tools = append(tools, mcpTools...)

	// US-004: Wrap all tools with policy evaluator
	if r.policyEvaluator != nil {
		for i, t := range tools {
			info, _ := t.Info(ctx)
			toolName := "unknown"
			if info != nil {
				toolName = info.Name
			}
			tools[i] = NewPolicyToolWrapper(t, r.policyEvaluator, rc.Agent.Record.Name, toolName)
		}
	}

	return tools, nil
}

// Resolve implements the legacy ToolResolver interface (Resolve by tool names + deps).
// This allows AgentToolResolver to be used as a drop-in replacement for DefaultToolResolver
// in the turn_executor pipeline where RegisteredAgent is not yet available.
func (r *AgentToolResolver) Resolve(ctx context.Context, toolNames []string, deps ToolDependencies) ([]tool.InvokableTool, error) {
	// US-001: Inject capability-derived tool names before resolution
	allToolNames := toolNames
	if r.capInjector != nil && deps.AgentName != "" {
		injected, err := r.capInjector.InjectedTools(ctx, deps.AgentName)
		if err != nil {
			slog.WarnContext(ctx, "capability injection failed, continuing without injected tools",
				"agent", deps.AgentName, "error", err)
		} else if len(injected) > 0 {
			// Deduplicate: only add tools not already in the list
			existing := make(map[string]bool, len(toolNames))
			for _, n := range toolNames {
				existing[n] = true
			}
			for _, n := range injected {
				if !existing[n] {
					allToolNames = append(allToolNames, n)
					existing[n] = true
				}
			}
		}
	}

	var resolved []tool.InvokableTool

	for _, name := range allToolNames {
		// knowledge_search is auto-injected below based on KnowledgePath — skip here
		if name == "knowledge_search" {
			continue
		}
		factory, ok := r.builtins.Get(name)
		if !ok {
			return nil, fmt.Errorf("resolve tool %s: unknown builtin tool", name)
		}
		t := factory(deps)
		if t == nil {
			continue
		}
		riskLevel := GetContentRiskLevel(name)
		t = NewSafeToolWrapper(t, name, riskLevel)
		t = NewCancellableToolWrapper(t)
		resolved = append(resolved, t)
	}

	// Knowledge auto-injection via legacy Resolve path
	if deps.KnowledgePath != "" && deps.AgentName != "" && r.knowledgeSearcher != nil && r.knowledgeEmbedder != nil {
		knowledgeTool := NewKnowledgeSearchTool(deps.AgentName, r.knowledgeSearcher, r.knowledgeEmbedder)
		resolved = append(resolved, knowledgeTool)
	} else if hasToolInList(allToolNames, "knowledge_search") {
		slog.WarnContext(ctx, "knowledge_search in tool list but knowledge path not configured — skipping",
			"agent", deps.AgentName,
			"knowledge_path", deps.KnowledgePath)
	}

	// Spawn tools via legacy Resolve path
	if r.spawner != nil && len(deps.CanSpawn) > 0 {
		for _, targetName := range deps.CanSpawn {
			spawnTool := NewSpawnTool(targetName, deps.SessionID, r.spawner, r.inspector)
			resolved = append(resolved, spawnTool)
		}
	}

	// MCP tools via legacy Resolve path
	if r.mcpProvider != nil && len(deps.MCPServers) > 0 {
		for _, serverName := range deps.MCPServers {
			mcpTools, err := r.mcpProvider.GetMCPTools(serverName)
			if err != nil {
				slog.WarnContext(ctx, "failed to get MCP tools in legacy Resolve, skipping",
					"server", serverName, "error", err)
				continue
			}
			// AC-RESIL-05: Timeout is innermost — fires first, feeds timeout error to CB
			// US-006: Circuit breaker wraps timeout
			// US-005: Recovery wraps circuit breaker
			for i, mt := range mcpTools {
				if r.toolTimeoutMs > 0 {
					mcpTools[i] = NewTimeoutToolWrapper(mt, r.toolTimeoutMs)
					mt = mcpTools[i]
				}
				if r.cbRegistry != nil {
					mcpTools[i] = NewCircuitBreakerToolWrapper(mt, r.cbRegistry.Get(serverName))
					mt = mcpTools[i]
				}
				if r.recoveryExecutor != nil {
					info, _ := mt.Info(ctx)
					toolName := serverName
					if info != nil {
						toolName = info.Name
					}
					mcpTools[i] = NewRecoveryToolWrapper(mt, r.recoveryExecutor, deps.SessionID, toolName)
				}
			}
			resolved = append(resolved, mcpTools...)
		}
	}

	// US-004: Wrap all tools with policy evaluator
	if r.policyEvaluator != nil && deps.AgentName != "" {
		for i, t := range resolved {
			info, _ := t.Info(ctx)
			toolName := "unknown"
			if info != nil {
				toolName = info.Name
			}
			resolved[i] = NewPolicyToolWrapper(t, r.policyEvaluator, deps.AgentName, toolName)
		}
	}

	return resolved, nil
}

// resolveKitTools returns tools from the agent's kit, if configured.
func (r *AgentToolResolver) resolveKitTools(rc ResolveContext) ([]tool.InvokableTool, error) {
	kitName := rc.Agent.Record.Kit
	if kitName == "" || r.kitProvider == nil || rc.KitSession == nil {
		return nil, nil
	}

	kit, err := r.kitProvider.Get(kitName)
	if err != nil {
		return nil, fmt.Errorf("get kit %q: %w", kitName, err)
	}

	return kit.Tools(*rc.KitSession), nil
}

// resolveMCPTools returns tools from MCP servers configured for the agent.
// US-005/US-006: MCP tools are wrapped with circuit breaker and recovery if configured.
func (r *AgentToolResolver) resolveMCPTools(rc ResolveContext) ([]tool.InvokableTool, error) {
	if r.mcpProvider == nil || len(rc.Agent.Record.MCPServers) == 0 {
		return nil, nil
	}

	ctx := context.Background()
	var result []tool.InvokableTool
	for _, serverName := range rc.Agent.Record.MCPServers {
		mcpTools, err := r.mcpProvider.GetMCPTools(serverName)
		if err != nil {
			slog.Warn("failed to get MCP tools, skipping server",
				"server", serverName, "agent", rc.Agent.Record.Name, "error", err)
			continue
		}
		// AC-RESIL-05: Timeout is innermost — fires first, feeds timeout error to CB
		// US-006: Circuit breaker wraps timeout
		// US-005: Recovery wraps circuit breaker
		for i, mt := range mcpTools {
			if r.toolTimeoutMs > 0 {
				mcpTools[i] = NewTimeoutToolWrapper(mt, r.toolTimeoutMs)
				mt = mcpTools[i]
			}
			if r.cbRegistry != nil {
				mcpTools[i] = NewCircuitBreakerToolWrapper(mt, r.cbRegistry.Get(serverName))
				mt = mcpTools[i]
			}
			if r.recoveryExecutor != nil {
				info, _ := mt.Info(ctx)
				toolName := serverName
				if info != nil {
					toolName = info.Name
				}
				mcpTools[i] = NewRecoveryToolWrapper(mt, r.recoveryExecutor, rc.Deps.SessionID, toolName)
			}
		}
		result = append(result, mcpTools...)
	}
	return result, nil
}

// hasToolInList checks if a tool name exists in the given list.
func hasToolInList(tools []string, name string) bool {
	for _, t := range tools {
		if t == name {
			return true
		}
	}
	return false
}
