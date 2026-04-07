package domain

// LifecycleMode defines whether an agent is spawn (ephemeral) or persistent.
type LifecycleMode string

const (
	LifecycleModeSpawn      LifecycleMode = "spawn"      // execute task → destroy context
	LifecycleModePersistent LifecycleMode = "persistent"  // execute task → keep context → await next task
)

// AgentInstance represents a running instance of an agent with lifecycle tracking.
// Uses the existing LifecycleState from lifecycle.go for state management.
type AgentInstance struct {
	AgentName     string
	Mode          LifecycleMode
	Lifecycle     *AgentLifecycle
	ContextTokens int    // current context size in tokens (approximate)
	MaxContext    int    // max context window size
	TasksHandled  int    // number of tasks executed
}

// NewAgentInstance creates a new agent instance in the initializing state.
func NewAgentInstance(agentName string, mode LifecycleMode, maxContext int) *AgentInstance {
	return &AgentInstance{
		AgentName: agentName,
		Mode:      mode,
		Lifecycle: NewAgentLifecycle(agentName, ""),
		MaxContext: maxContext,
	}
}

// MarkReady transitions to ready state.
func (ai *AgentInstance) MarkReady() error {
	return ai.Lifecycle.TransitionTo(LifecycleReady)
}

// MarkRunning transitions to running state.
func (ai *AgentInstance) MarkRunning() error {
	return ai.Lifecycle.TransitionTo(LifecycleRunning)
}

// MarkBlocked transitions to blocked state.
func (ai *AgentInstance) MarkBlocked() error {
	return ai.Lifecycle.TransitionTo(LifecycleBlocked)
}

// FinishTask completes the current task. For spawn agents, transitions to finished.
// For persistent agents, transitions to finished then back to ready.
func (ai *AgentInstance) FinishTask() error {
	ai.TasksHandled++
	if err := ai.Lifecycle.TransitionTo(LifecycleFinished); err != nil {
		return err
	}
	if ai.Mode == LifecycleModePersistent {
		return ai.Lifecycle.TransitionTo(LifecycleReady)
	}
	return nil
}

// NeedsCompaction returns true if the context is near overflow.
func (ai *AgentInstance) NeedsCompaction() bool {
	if ai.MaxContext <= 0 {
		return false
	}
	return ai.ContextTokens > (ai.MaxContext * 80 / 100) // 80% threshold
}

// ResetContext clears the context (for spawn agents on re-spawn).
func (ai *AgentInstance) ResetContext() {
	ai.ContextTokens = 0
}

// IsFinished returns true if the agent instance is finished.
func (ai *AgentInstance) IsFinished() bool {
	return ai.Lifecycle.State == LifecycleFinished
}

// IsPersistent returns true if the agent uses persistent lifecycle.
func (ai *AgentInstance) IsPersistent() bool {
	return ai.Mode == LifecycleModePersistent
}

// State returns the current lifecycle state.
func (ai *AgentInstance) State() LifecycleState {
	return ai.Lifecycle.State
}
