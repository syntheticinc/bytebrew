package domain

import "fmt"

// FlowType represents the type of flow (agent role)
type FlowType string

const (
	FlowTypeSupervisor FlowType = "supervisor"
	FlowTypeCoder      FlowType = "coder"
	FlowTypeReviewer   FlowType = "reviewer"
	FlowTypeResearcher FlowType = "researcher"
)

// LifecyclePolicy defines when a flow should suspend and where to report
type LifecyclePolicy struct {
	SuspendOn []string // events that cause suspension: "final_answer", "ask_user"
	ReportTo  string   // "user" or "parent_agent"
}

// SpawnPolicy defines which flows can be spawned by this flow
type SpawnPolicy struct {
	AllowedFlows  []FlowType
	MaxConcurrent int // 0 = no limit (backward compatibility)
}

// Flow represents a flow configuration (agent behavior template)
type Flow struct {
	Type           FlowType
	Name           string
	SystemPrompt   string
	ToolNames      []string
	MaxSteps       int
	MaxContextSize int
	Lifecycle      LifecyclePolicy
	Spawn          SpawnPolicy
}

// Validate validates the Flow configuration
func (f *Flow) Validate() error {
	if f.Type == "" {
		return fmt.Errorf("flow type is required")
	}
	if f.Name == "" {
		return fmt.Errorf("flow name is required")
	}
	if f.SystemPrompt == "" {
		return fmt.Errorf("system prompt is required")
	}
	if f.MaxSteps < 0 {
		return fmt.Errorf("max_steps must be non-negative (0 = unlimited)")
	}
	if f.MaxContextSize <= 0 {
		return fmt.Errorf("max_context_size must be positive")
	}
	return nil
}

// CanSpawn returns true if this flow can spawn the specified flow type
func (f *Flow) CanSpawn(flowType FlowType) bool {
	for _, allowed := range f.Spawn.AllowedFlows {
		if allowed == flowType {
			return true
		}
	}
	return false
}

// ShouldSuspendOn returns true if this flow should suspend on the specified event
func (f *Flow) ShouldSuspendOn(event string) bool {
	for _, e := range f.Lifecycle.SuspendOn {
		if e == event {
			return true
		}
	}
	return false
}
