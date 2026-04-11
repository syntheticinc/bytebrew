package agent_registry

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
)

// AgentReader is the consumer-side interface for reading agent records.
type AgentReader interface {
	List(ctx context.Context) ([]config_repo.AgentRecord, error)
	GetByName(ctx context.Context, name string) (*config_repo.AgentRecord, error)
	Count(ctx context.Context) (int64, error)
}

// RegisteredAgent holds a domain Flow and its original DB record.
type RegisteredAgent struct {
	Flow   *domain.Flow
	Record config_repo.AgentRecord
}

// AgentRegistry loads agents from DB and caches them in memory.
type AgentRegistry struct {
	mu     sync.RWMutex
	agents map[string]*RegisteredAgent
	repo   AgentReader
}

// New creates a new AgentRegistry.
func New(repo AgentReader) *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]*RegisteredAgent),
		repo:   repo,
	}
}

// Load reads all agents from DB and caches them in memory.
func (r *AgentRegistry) Load(ctx context.Context) error {
	records, err := r.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("load agents: %w", err)
	}

	agents := make(map[string]*RegisteredAgent, len(records))
	for _, rec := range records {
		flow := toFlow(rec)
		agents[rec.Name] = &RegisteredAgent{
			Flow:   flow,
			Record: rec,
		}
	}

	r.mu.Lock()
	r.agents = agents
	r.mu.Unlock()
	return nil
}

// Reload reloads all agents from DB (hot-reload support).
func (r *AgentRegistry) Reload(ctx context.Context) error {
	return r.Load(ctx)
}

// Get returns a registered agent by name.
func (r *AgentRegistry) Get(name string) (*RegisteredAgent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[name]
	if !ok {
		return nil, fmt.Errorf("agent %q not found", name)
	}
	return agent, nil
}

// GetFlow implements the FlowProvider interface used by EngineAdapter and AgentPool.
// This allows AgentRegistry to be a drop-in replacement for FlowManager.
func (r *AgentRegistry) GetFlow(_ context.Context, flowType domain.FlowType) (*domain.Flow, error) {
	agent, err := r.Get(string(flowType))
	if err != nil {
		return nil, err
	}
	return agent.Flow, nil
}

// List returns all registered agent names in alphabetical order.
func (r *AgentRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.agents))
	for name := range r.agents {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// GetAll returns a copy of all registered agents.
func (r *AgentRegistry) GetAll() map[string]*RegisteredAgent {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]*RegisteredAgent, len(r.agents))
	for k, v := range r.agents {
		result[k] = v
	}
	return result
}

// GetDefault returns the first agent alphabetically.
func (r *AgentRegistry) GetDefault() (*RegisteredAgent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.agents) == 0 {
		return nil, fmt.Errorf("no agents configured")
	}

	var firstName string
	for name := range r.agents {
		if firstName == "" || name < firstName {
			firstName = name
		}
	}
	return r.agents[firstName], nil
}

// ResolveModelID returns the ModelID for the given agent name, or nil if not found.
// Implements infrastructure.AgentModelResolver interface.
func (r *AgentRegistry) ResolveModelID(agentName string) *string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	agent, ok := r.agents[agentName]
	if !ok {
		return nil
	}
	return agent.Record.ModelID
}

// Count returns the number of registered agents.
func (r *AgentRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.agents)
}

// toFlow converts an AgentRecord into a domain.Flow.
func toFlow(rec config_repo.AgentRecord) *domain.Flow {
	spawn := domain.SpawnPolicy{
		AllowedFlows: make([]domain.FlowType, 0, len(rec.CanSpawn)),
	}
	for _, name := range rec.CanSpawn {
		spawn.AllowedFlows = append(spawn.AllowedFlows, domain.FlowType(name))
	}

	// Collect all tool names (builtin + custom)
	toolNames := make([]string, 0, len(rec.BuiltinTools)+len(rec.CustomTools))
	toolNames = append(toolNames, rec.BuiltinTools...)
	for _, ct := range rec.CustomTools {
		toolNames = append(toolNames, ct.Name)
	}

	lifecycle := domain.LifecyclePolicy{}
	switch rec.Lifecycle {
	case "persistent":
		lifecycle.SuspendOn = []string{"final_answer", "ask_user"}
		lifecycle.ReportTo = "user"
	case "spawn":
		lifecycle.ReportTo = "parent_agent"
	}

	// Append confirm_before instruction to system prompt (mirrors prompt_builder.go logic).
	// This ensures DB-configured confirm_before is applied in the SSE/HTTP path.
	systemPrompt := rec.SystemPrompt
	if len(rec.ConfirmBefore) > 0 {
		systemPrompt += "\n\n## Confirmation required\nAsk user before calling: " +
			strings.Join(rec.ConfirmBefore, ", ") +
			"\nWhen asking for confirmation, include the tool_name parameter in the ask_user call."
	}

	return &domain.Flow{
		Type:           domain.FlowType(rec.Name),
		Name:           rec.Name,
		SystemPrompt:   systemPrompt,
		ToolNames:      toolNames,
		MaxSteps:       rec.MaxSteps,
		MaxContextSize:  rec.MaxContextSize,
		MaxTurnDuration: rec.MaxTurnDuration,
		ToolExecution:   rec.ToolExecution,
		Lifecycle:      lifecycle,
		Spawn:          spawn,
		KnowledgePath:  rec.KnowledgePath,
		MCPServers:     rec.MCPServers,
		ConfirmBefore:  rec.ConfirmBefore,
	}
}
