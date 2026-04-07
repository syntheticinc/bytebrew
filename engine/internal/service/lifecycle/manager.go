package lifecycle

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// AgentRunner executes an agent with input and returns output.
type AgentRunner interface {
	RunAgent(ctx context.Context, agentName, input, sessionID string, eventStream domain.AgentEventStream) (string, error)
}

// ContextCompactor compacts agent context when it overflows.
type ContextCompactor interface {
	Compact(ctx context.Context, agentName, sessionID string) error
}

// Manager tracks agent instances and their lifecycle (spawn vs persistent).
type Manager struct {
	mu        sync.RWMutex
	instances map[string]*domain.AgentInstance // key: "agentName:sessionID"
	contexts  map[string][]string             // key: "agentName:sessionID" -> accumulated context messages
	runner    AgentRunner
	compactor ContextCompactor // optional, nil-safe
}

// NewManager creates a new lifecycle Manager.
func NewManager(runner AgentRunner) *Manager {
	return &Manager{
		instances: make(map[string]*domain.AgentInstance),
		contexts:  make(map[string][]string),
		runner:    runner,
	}
}

// SetCompactor sets the context compactor (optional).
func (m *Manager) SetCompactor(c ContextCompactor) {
	m.compactor = c
}

// ExecuteTask executes a task on an agent, handling spawn vs persistent lifecycle.
func (m *Manager) ExecuteTask(ctx context.Context, agentName, sessionID, input string,
	mode domain.LifecycleMode, maxContext int, eventStream domain.AgentEventStream) (string, error) {

	key := instanceKey(agentName, sessionID)

	instance := m.getOrCreateInstance(key, agentName, mode, maxContext)

	// For spawn agents, always reset context
	if mode == domain.LifecycleModeSpawn {
		m.mu.Lock()
		instance.ResetContext()
		m.contexts[key] = nil
		m.mu.Unlock()
	}

	// Transition: initializing → ready → running
	if instance.State() == domain.LifecycleInitializing {
		if err := instance.MarkReady(); err != nil {
			return "", fmt.Errorf("mark ready: %w", err)
		}
	}
	if instance.State() == domain.LifecycleReady {
		if err := instance.MarkRunning(); err != nil {
			return "", fmt.Errorf("mark running: %w", err)
		}
	}

	// Check if persistent agent needs compaction
	if instance.IsPersistent() && instance.NeedsCompaction() {
		slog.InfoContext(ctx, "lifecycle: auto-compacting context", "agent", agentName, "tokens", instance.ContextTokens)
		if m.compactor != nil {
			if err := m.compactor.Compact(ctx, agentName, sessionID); err != nil {
				slog.ErrorContext(ctx, "lifecycle: compaction failed", "error", err, "agent", agentName)
			}
		}
	}

	// Build full input for persistent agents (include previous context)
	fullInput := input
	if instance.IsPersistent() {
		m.mu.RLock()
		prevContext := m.contexts[key]
		m.mu.RUnlock()
		if len(prevContext) > 0 {
			fullInput = buildContextualInput(prevContext, input)
		}
	}

	// Execute the agent
	output, err := m.runner.RunAgent(ctx, agentName, fullInput, sessionID, eventStream)
	if err != nil {
		_ = instance.MarkBlocked()
		return "", fmt.Errorf("agent %q execution failed: %w", agentName, err)
	}

	// Store context for persistent agents
	if instance.IsPersistent() {
		m.mu.Lock()
		m.contexts[key] = append(m.contexts[key], "User: "+input, "Agent: "+output)
		instance.ContextTokens += estimateTokens(input) + estimateTokens(output)
		m.mu.Unlock()
	}

	// Finish task: spawn → finished, persistent → ready
	if err := instance.FinishTask(); err != nil {
		return output, fmt.Errorf("finish task: %w", err)
	}

	// For spawn agents, clean up instance
	if mode == domain.LifecycleModeSpawn {
		m.mu.Lock()
		delete(m.instances, key)
		delete(m.contexts, key)
		m.mu.Unlock()
	}

	return output, nil
}

// GetInstance returns the current agent instance state, if it exists.
func (m *Manager) GetInstance(agentName, sessionID string) (*domain.AgentInstance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	inst, ok := m.instances[instanceKey(agentName, sessionID)]
	return inst, ok
}

// ResetAgent resets a persistent agent's context explicitly.
func (m *Manager) ResetAgent(agentName, sessionID string) {
	key := instanceKey(agentName, sessionID)
	m.mu.Lock()
	defer m.mu.Unlock()

	if inst, ok := m.instances[key]; ok {
		inst.ResetContext()
	}
	m.contexts[key] = nil
}

// ContextSize returns the number of context entries for a persistent agent.
func (m *Manager) ContextSize(agentName, sessionID string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.contexts[instanceKey(agentName, sessionID)])
}

func (m *Manager) getOrCreateInstance(key, agentName string, mode domain.LifecycleMode, maxContext int) *domain.AgentInstance {
	m.mu.Lock()
	defer m.mu.Unlock()

	if inst, ok := m.instances[key]; ok {
		// For spawn agents, recreate instance for fresh state
		if mode == domain.LifecycleModeSpawn {
			inst = domain.NewAgentInstance(agentName, mode, maxContext)
			m.instances[key] = inst
		}
		return inst
	}

	inst := domain.NewAgentInstance(agentName, mode, maxContext)
	m.instances[key] = inst
	return inst
}

func instanceKey(agentName, sessionID string) string {
	return agentName + ":" + sessionID
}

func buildContextualInput(prevContext []string, newInput string) string {
	result := "Previous context:\n"
	for _, msg := range prevContext {
		result += msg + "\n"
	}
	result += "\nNew task:\n" + newInput
	return result
}

// estimateTokens provides a rough token count estimate (1 token ≈ 4 chars).
func estimateTokens(text string) int {
	return len(text) / 4
}
