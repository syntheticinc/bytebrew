package tools

import (
	"context"
	"fmt"
	"sort"

	"github.com/cloudwego/eino/components/tool"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/agent_registry"
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

// AgentToolResolver composes tools for a specific agent from various sources.
type AgentToolResolver struct {
	builtins    *BuiltinToolStore
	kitProvider KitProvider
}

// NewAgentToolResolver creates a new AgentToolResolver.
func NewAgentToolResolver(builtins *BuiltinToolStore) *AgentToolResolver {
	return &AgentToolResolver{builtins: builtins}
}

// SetKitProvider configures the kit provider for kit-based tool resolution.
func (r *AgentToolResolver) SetKitProvider(kp KitProvider) {
	r.kitProvider = kp
}

// ResolveContext holds per-agent resolution context.
type ResolveContext struct {
	Agent      *agent_registry.RegisteredAgent
	Deps       ToolDependencies
	KitSession *domain.KitSession // nil if agent has no kit
}

// ResolveForAgent returns tools available to a specific agent.
// Only tools listed in the agent's BuiltinTools whitelist are resolved.
// Unknown tool names produce an error.
func (r *AgentToolResolver) ResolveForAgent(ctx context.Context, rc ResolveContext) ([]tool.InvokableTool, error) {
	var tools []tool.InvokableTool

	for _, name := range rc.Agent.Record.BuiltinTools {
		factory, ok := r.builtins.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown builtin tool %q for agent %q", name, rc.Agent.Record.Name)
		}
		tools = append(tools, factory(rc.Deps))
	}

	// Phase 2.6: custom declarative tools
	// Phase 2.8-2.9: MCP tools

	// Phase 3: Kit tools — append tools provided by the agent's kit
	kitTools, err := r.resolveKitTools(rc)
	if err != nil {
		return nil, fmt.Errorf("resolve kit tools for agent %q: %w", rc.Agent.Record.Name, err)
	}
	tools = append(tools, kitTools...)

	return tools, nil
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
