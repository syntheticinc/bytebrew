package domain

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/utils"
)

// PlanStatus represents the lifecycle stage of a plan
type PlanStatus string

const (
	PlanStatusDraft     PlanStatus = "draft"     // Plan created but not started
	PlanStatusActive    PlanStatus = "active"    // Plan currently being executed
	PlanStatusCompleted PlanStatus = "completed" // Plan finished successfully
	PlanStatusAbandoned PlanStatus = "abandoned" // Plan cancelled or failed
)

// PlanStepStatus represents the status of a single step
type PlanStepStatus string

const (
	StepStatusPending    PlanStepStatus = "pending"     // Not started
	StepStatusInProgress PlanStepStatus = "in_progress" // Currently executing
	StepStatusCompleted  PlanStepStatus = "completed"   // Successfully finished
	StepStatusSkipped    PlanStepStatus = "skipped"     // Skipped by agent
	StepStatusFailed     PlanStepStatus = "failed"      // Failed to complete
)

// PlanStep represents a single step in a plan
type PlanStep struct {
	Index       int            `json:"index"`                  // 0-based step index
	Description string         `json:"description"`            // What to do
	Reasoning   string         `json:"reasoning,omitempty"`    // Why this step is needed
	Status      PlanStepStatus `json:"status"`                 // Current status
	ToolCalls   []string       `json:"tool_calls,omitempty"`   // Tool names called during step
	Result      string         `json:"result,omitempty"`       // Summary of step outcome
	StartedAt   *time.Time     `json:"started_at,omitempty"`   // When step started
	CompletedAt *time.Time     `json:"completed_at,omitempty"` // When step completed
}

// Plan represents a structured execution plan for complex tasks
type Plan struct {
	ID        string       `json:"id"`                   // Unique identifier
	SessionID string       `json:"session_id"`           // Session this plan belongs to
	Goal      string       `json:"goal"`                 // High-level objective
	Steps     []*PlanStep  `json:"steps"`                // Ordered list of steps
	Status    PlanStatus   `json:"status"`               // Current plan status
	Metadata  StringMap    `json:"metadata,omitempty"`   // Additional key-value data
	CreatedAt time.Time    `json:"created_at"`           // When plan was created
	UpdatedAt time.Time    `json:"updated_at"`           // Last modification time
	mu        sync.RWMutex `json:"-"`                    // Thread safety
}

// StringMap is a helper type for metadata
type StringMap map[string]string

// NewPlan creates a new plan with the given parameters
func NewPlan(id, sessionID, goal string, steps []*PlanStep) *Plan {
	now := time.Now()

	// Mark first step as in_progress
	if len(steps) > 0 {
		steps[0].Status = StepStatusInProgress
		steps[0].StartedAt = &now
	}

	return &Plan{
		ID:        id,
		SessionID: sessionID,
		Goal:      goal,
		Steps:     steps,
		Status:    PlanStatusActive, // Start as active
		Metadata:  make(StringMap),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// CurrentStep returns the step currently in progress, or nil if none
func (p *Plan) CurrentStep() *PlanStep {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, step := range p.Steps {
		if step.Status == StepStatusInProgress {
			return step
		}
	}
	return nil
}

// NextPendingStep returns the first pending step, or nil if none
func (p *Plan) NextPendingStep() *PlanStep {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, step := range p.Steps {
		if step.Status == StepStatusPending {
			return step
		}
	}
	return nil
}

// Progress returns (completed steps, total steps)
func (p *Plan) Progress() (completed, total int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total = len(p.Steps)
	for _, step := range p.Steps {
		if step.Status == StepStatusCompleted {
			completed++
		}
	}
	return completed, total
}

// ToCompactContext generates a compact string representation for LLM context
// Returns a formatted string (target: ~150 tokens)
func (p *Plan) ToCompactContext() string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var sb strings.Builder
	completed, total := 0, len(p.Steps)

	// Progress summary
	for _, step := range p.Steps {
		if step.Status == StepStatusCompleted {
			completed++
		}
	}

	sb.WriteString(fmt.Sprintf("**PLAN** [%d/%d steps completed]:\n", completed, total))

	// List all steps with status indicators
	for i, step := range p.Steps {
		var marker string
		switch step.Status {
		case StepStatusCompleted:
			marker = "✓"
		case StepStatusInProgress:
			marker = "→"
		case StepStatusFailed:
			marker = "✗"
		case StepStatusSkipped:
			marker = "⊘"
		default:
			marker = "○"
		}

		sb.WriteString(fmt.Sprintf("%s Step %d: %s", marker, i+1, step.Description))

		// Add reasoning for current/pending steps
		if step.Status == StepStatusInProgress || step.Status == StepStatusPending {
			if step.Reasoning != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", utils.Truncate(step.Reasoning, 50)))
			}
		}

		sb.WriteString("\n")
	}

	return sb.String()
}

// UpdateStepStatus updates a step's status and handles auto-advancement
func (p *Plan) UpdateStepStatus(stepIndex int, status PlanStepStatus, result string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if stepIndex < 0 || stepIndex >= len(p.Steps) {
		return fmt.Errorf("invalid step index: %d", stepIndex)
	}

	step := p.Steps[stepIndex]
	now := time.Now()

	// Update step
	step.Status = status
	step.Result = result
	p.UpdatedAt = now

	// Handle status transitions
	switch status {
	case StepStatusCompleted, StepStatusSkipped, StepStatusFailed:
		if step.CompletedAt == nil {
			step.CompletedAt = &now
		}

		// Auto-advance to next pending step
		if next := p.nextPendingStepLocked(); next != nil {
			next.Status = StepStatusInProgress
			next.StartedAt = &now
		} else {
			// No more pending steps - plan is complete
			p.Status = PlanStatusCompleted
		}
	case StepStatusInProgress:
		if step.StartedAt == nil {
			step.StartedAt = &now
		}
	}

	return nil
}

// nextPendingStepLocked returns the first pending step (assumes lock is held)
func (p *Plan) nextPendingStepLocked() *PlanStep {
	for _, step := range p.Steps {
		if step.Status == StepStatusPending {
			return step
		}
	}
	return nil
}

// SetStatus updates the plan's overall status
func (p *Plan) SetStatus(status PlanStatus) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.Status = status
	p.UpdatedAt = time.Now()
}

// AddStep appends a new step to the plan
func (p *Plan) AddStep(description, reasoning string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	step := &PlanStep{
		Index:       len(p.Steps),
		Description: description,
		Reasoning:   reasoning,
		Status:      StepStatusPending,
	}
	p.Steps = append(p.Steps, step)
	p.UpdatedAt = time.Now()
}

// RemoveStep removes a step by index (not recommended during execution)
func (p *Plan) RemoveStep(stepIndex int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if stepIndex < 0 || stepIndex >= len(p.Steps) {
		return fmt.Errorf("invalid step index: %d", stepIndex)
	}

	// Don't allow removing in-progress or completed steps
	if p.Steps[stepIndex].Status == StepStatusInProgress ||
	   p.Steps[stepIndex].Status == StepStatusCompleted {
		return fmt.Errorf("cannot remove step with status %s", p.Steps[stepIndex].Status)
	}

	// Remove step and reindex
	p.Steps = append(p.Steps[:stepIndex], p.Steps[stepIndex+1:]...)
	for i := range p.Steps {
		p.Steps[i].Index = i
	}
	p.UpdatedAt = time.Now()

	return nil
}

// ModifyStep updates a step's description and reasoning
func (p *Plan) ModifyStep(stepIndex int, description, reasoning string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if stepIndex < 0 || stepIndex >= len(p.Steps) {
		return fmt.Errorf("invalid step index: %d", stepIndex)
	}

	step := p.Steps[stepIndex]

	// Only allow modifying pending steps
	if step.Status != StepStatusPending {
		return fmt.Errorf("can only modify pending steps (current status: %s)", step.Status)
	}

	step.Description = description
	step.Reasoning = reasoning
	p.UpdatedAt = time.Now()

	return nil
}

