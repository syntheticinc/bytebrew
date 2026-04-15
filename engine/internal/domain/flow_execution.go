package domain

import (
	"fmt"
	"sync"
	"time"
)

// FlowExecutionStatus represents the status of a flow execution.
type FlowExecutionStatus string

const (
	FlowExecPending   FlowExecutionStatus = "pending"
	FlowExecRunning   FlowExecutionStatus = "running"
	FlowExecCompleted FlowExecutionStatus = "completed"
	FlowExecFailed    FlowExecutionStatus = "failed"
	FlowExecCancelled FlowExecutionStatus = "cancelled"
)

// FlowStepStatus represents the status of a single step in a flow.
type FlowStepStatus string

const (
	StepStatusPending   FlowStepStatus = "pending"
	StepStatusRunning   FlowStepStatus = "running"
	StepStatusCompleted FlowStepStatus = "completed"
	StepStatusFailed    FlowStepStatus = "failed"
	StepStatusSkipped   FlowStepStatus = "skipped"
)

// EdgeRouteMode defines how output is routed between agents along an edge.
type EdgeRouteMode string

const (
	EdgeRouteFull         EdgeRouteMode = "full_output"    // pass entire output
	EdgeRouteFieldMapping EdgeRouteMode = "field_mapping"  // extract specific JSON fields
	EdgeRouteCustomPrompt EdgeRouteMode = "custom_prompt"  // template with {{output}} vars
)

// FlowExecution tracks the state of a schema pipeline execution.
type FlowExecution struct {
	mu        sync.Mutex
	ID        string
	SchemaID  string
	SessionID string
	Status    FlowExecutionStatus
	Steps     []FlowStep
	CreatedAt time.Time
	UpdatedAt time.Time
}

// FlowStep represents a single step in a flow pipeline.
type FlowStep struct {
	AgentName  string
	Status     FlowStepStatus
	Output     string
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time
}

// NewFlowExecution creates a new flow execution.
func NewFlowExecution(schemaID, sessionID string) *FlowExecution {
	return &FlowExecution{
		SchemaID:  schemaID,
		SessionID: sessionID,
		Status:    FlowExecPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Start marks the flow execution as running.
func (fe *FlowExecution) Start() error {
	if fe.Status != FlowExecPending {
		return fmt.Errorf("flow must be pending to start, current: %s", fe.Status)
	}
	fe.Status = FlowExecRunning
	fe.UpdatedAt = time.Now()
	return nil
}

// Complete marks the flow execution as completed.
func (fe *FlowExecution) Complete() {
	fe.Status = FlowExecCompleted
	fe.UpdatedAt = time.Now()
}

// Fail marks the flow execution as failed.
func (fe *FlowExecution) Fail() {
	fe.Status = FlowExecFailed
	fe.UpdatedAt = time.Now()
}

// AddStep adds a step to the flow execution.
func (fe *FlowExecution) AddStep(agentName string) *FlowStep {
	step := FlowStep{
		AgentName: agentName,
		Status:    StepStatusPending,
	}
	fe.Steps = append(fe.Steps, step)
	return &fe.Steps[len(fe.Steps)-1]
}

// MergeSteps appends steps from a completed fork branch. Safe for sequential post-fork use.
func (fe *FlowExecution) MergeSteps(steps []FlowStep) {
	fe.mu.Lock()
	defer fe.mu.Unlock()
	fe.Steps = append(fe.Steps, steps...)
}

// IsTerminal returns true if the execution is in a terminal state.
func (fe *FlowExecution) IsTerminal() bool {
	switch fe.Status {
	case FlowExecCompleted, FlowExecFailed, FlowExecCancelled:
		return true
	}
	return false
}

// --- Flow event types ---

const (
	EventTypeFlowStepStarted   AgentEventType = "flow.step_started"
	EventTypeFlowStepCompleted AgentEventType = "flow.step_completed"
	EventTypeFlowCompleted     AgentEventType = "flow.completed"
	EventTypeFlowFailed        AgentEventType = "flow.failed"
)

// NewFlowStepStartedEvent creates a flow.step_started event.
func NewFlowStepStartedEvent(agentName, sessionID string, stepIndex int) *AgentEvent {
	return &AgentEvent{
		Type:          EventTypeFlowStepStarted,
		SchemaVersion: EventSchemaVersion,
		Timestamp:     time.Now(),
		AgentID:       agentName,
		Metadata: map[string]interface{}{
			"agent_name": agentName,
			"step_index": stepIndex,
		},
	}
}

// NewFlowStepCompletedEvent creates a flow.step_completed event.
func NewFlowStepCompletedEvent(agentName, sessionID string, stepIndex int) *AgentEvent {
	return &AgentEvent{
		Type:          EventTypeFlowStepCompleted,
		SchemaVersion: EventSchemaVersion,
		Timestamp:     time.Now(),
		AgentID:       agentName,
		Metadata: map[string]interface{}{
			"agent_name": agentName,
			"step_index": stepIndex,
		},
	}
}

