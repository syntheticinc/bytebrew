package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// PlanManager defines interface for plan orchestration (consumer-side interface)
type PlanManager interface {
	CreatePlan(ctx context.Context, sessionID, goal string, steps []*domain.PlanStep) (*domain.Plan, error)
	GetActivePlan(ctx context.Context, sessionID string) (*domain.Plan, error)
	UpdateStepStatus(ctx context.Context, sessionID string, stepIdx int, status domain.PlanStepStatus, result string) error
	ModifyStep(ctx context.Context, sessionID string, stepIndex int, description, reasoning string) error
}

// ManagePlanArgs represents the full state of a plan
type ManagePlanArgs struct {
	Goal  string           `json:"goal"`
	Steps []PlanStepState `json:"steps"`
}

// PlanStepState represents the state of a single step
type PlanStepState struct {
	Index       int    `json:"index"`
	Description string `json:"description"`
	Reasoning   string `json:"reasoning,omitempty"`
	Status      string `json:"status"` // "pending", "in_progress", "completed"
}

// ManagePlanTool implements stateful plan management (like Windsurf todo_list)
type ManagePlanTool struct {
	planManager PlanManager
	sessionID   string
}

// NewManagePlanTool creates a new stateful plan management tool
func NewManagePlanTool(planManager PlanManager, sessionID string) tool.InvokableTool {
	return &ManagePlanTool{
		planManager: planManager,
		sessionID:   sessionID,
	}
}

// Info returns tool information for LLM
func (t *ManagePlanTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "manage_plan",
		Desc: `Track progress on complex multi-file tasks.

**When to use:**
- Tasks that touch 3+ files (e.g., SOLID review across modules, migration)
- When the user explicitly asks for a plan

**DO NOT use for:**
- Single-file edits or refactoring (just read → write_file → done)
- Simple questions or code reviews

**Usage:** Pass full plan state. System detects changes.
{
  "goal": "Review auth module",
  "steps": [
    {"index": 0, "description": "Check AuthService.ts", "status": "in_progress"},
    {"index": 1, "description": "Check TokenValidator.ts", "status": "pending"}
  ]
}

**CRITICAL RULES:**
1. Indexes MUST start from 0, not 1! First step = index 0.
2. ALWAYS include "description" for EVERY step, even when just updating status.
3. Do NOT remove steps or change their order — only update status.
4. Only mark a step completed AFTER you actually did the work.
5. You CAN mark multiple steps completed at once if you did them together.

**Statuses:** pending → in_progress → completed

**Tips:**
- Keep steps short and specific (reference actual files)
- Do NOT remove or reorder existing steps — only update status
- 3-7 steps is ideal
- Focus on doing the actual work, not on managing the plan`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"goal": {
				Type:     schema.String,
				Desc:     "What you're trying to accomplish",
				Required: true,
			},
			"steps": {
				Type:     schema.Array,
				Desc:     "All steps with: index, description, status (pending/in_progress/completed)",
				Required: true,
			},
		}),
	}, nil
}

// InvokableRun executes the tool with JSON arguments
func (t *ManagePlanTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	slog.InfoContext(ctx, "[manage_plan] tool invoked",
		"session_id", t.sessionID,
		"args_length", len(argumentsInJSON))

	args, err := parseManagePlanArgs(ctx, argumentsInJSON)
	if err != nil {
		slog.ErrorContext(ctx, "[manage_plan] failed to parse arguments", "error", err)
		return fmt.Sprintf("[ERROR] Invalid JSON for manage_plan: %v", err), nil
	}

	// Validate
	if args.Goal == "" {
		return "[ERROR] goal is required", nil
	}
	if len(args.Steps) == 0 {
		return "[ERROR] steps array cannot be empty", nil
	}
	if len(args.Steps) > 10 {
		return fmt.Sprintf("[ERROR] Too many steps (%d). Limit to 3-7 for maintainability.", len(args.Steps)), nil
	}

	// Check if plan exists (needed for fallback)
	existingPlan, _ := t.planManager.GetActivePlan(ctx, t.sessionID)

	// Validate steps and fill missing descriptions from existing plan
	for i, step := range args.Steps {
		if step.Index != i {
			return fmt.Sprintf("[ERROR] Step index mismatch at position %d (expected index %d, got %d). "+
				"Indexes MUST start from 0! First step = {\"index\": 0, ...}, second = {\"index\": 1, ...}", i, i, step.Index), nil
		}
		// Fallback: if description is empty but we have existing plan, use existing description
		if step.Description == "" && existingPlan != nil && i < len(existingPlan.Steps) {
			args.Steps[i].Description = existingPlan.Steps[i].Description
			slog.InfoContext(ctx, "[manage_plan] filled missing description from existing plan",
				"step", i, "description", args.Steps[i].Description)
		}
		// Still validate after fallback
		if args.Steps[i].Description == "" {
			return fmt.Sprintf("[ERROR] Step %d missing description (no existing plan to recover from)", i), nil
		}
		if step.Status != "pending" && step.Status != "in_progress" && step.Status != "completed" {
			return fmt.Sprintf("[ERROR] Step %d has invalid status '%s' (must be pending/in_progress/completed)", i, step.Status), nil
		}
	}

	if existingPlan == nil {
		// CREATE: No existing plan - create new one
		return t.createPlan(ctx, *args)
	} else {
		// UPDATE: Plan exists - update it
		return t.updatePlan(ctx, existingPlan, *args)
	}
}

// createPlan creates a new plan
func (t *ManagePlanTool) createPlan(ctx context.Context, args ManagePlanArgs) (string, error) {
	// Convert to domain model
	domainSteps := make([]*domain.PlanStep, len(args.Steps))
	for i, step := range args.Steps {
		status := parseStatus(step.Status)
		domainSteps[i] = &domain.PlanStep{
			Index:       i,
			Description: step.Description,
			Reasoning:   step.Reasoning,
			Status:      status,
		}
	}

	// Create plan
	plan, err := t.planManager.CreatePlan(ctx, t.sessionID, args.Goal, domainSteps)
	if err != nil {
		slog.ErrorContext(ctx, "[manage_plan] failed to create plan", "error", err)
		return fmt.Sprintf("[ERROR] Failed to create plan: %v", err), nil
	}

	slog.InfoContext(ctx, "[manage_plan] plan created",
		"plan_id", plan.ID,
		"goal", args.Goal,
		"steps", len(args.Steps))

	// Return formatted response
	return formatPlanResponse(plan, "created"), nil
}

// updatePlan updates existing plan
func (t *ManagePlanTool) updatePlan(ctx context.Context, existingPlan *domain.Plan, args ManagePlanArgs) (string, error) {
	// Detect structural changes (warnings for agent)
	var warnings []string

	// Check if steps were added or removed
	if len(args.Steps) != len(existingPlan.Steps) {
		if len(args.Steps) > len(existingPlan.Steps) {
			warnings = append(warnings, fmt.Sprintf("⚠️ You added %d new step(s). Avoid changing plan structure - only update status.", len(args.Steps)-len(existingPlan.Steps)))
		} else {
			warnings = append(warnings, fmt.Sprintf("⚠️ You removed %d step(s). Do NOT remove steps - mark them completed instead.", len(existingPlan.Steps)-len(args.Steps)))
		}
		slog.WarnContext(ctx, "[manage_plan] structural change detected",
			"old_steps", len(existingPlan.Steps),
			"new_steps", len(args.Steps))
	}

	// Check for description changes (suspicious - agent should only update status)
	for i, newStep := range args.Steps {
		if i < len(existingPlan.Steps) {
			oldStep := existingPlan.Steps[i]
			if oldStep.Description != newStep.Description && newStep.Description != "" {
				warnings = append(warnings, fmt.Sprintf("⚠️ Step %d description changed. Avoid rewriting steps - only update status.", i+1))
				slog.WarnContext(ctx, "[manage_plan] description change detected",
					"step", i,
					"old_desc", oldStep.Description,
					"new_desc", newStep.Description)
			}
		}
	}

	// Update goal if changed
	if args.Goal != existingPlan.Goal {
		slog.InfoContext(ctx, "[manage_plan] goal changed",
			"old", existingPlan.Goal,
			"new", args.Goal)
		existingPlan.Goal = args.Goal
	}

	// Process step changes
	changes := []string{}

	// Update existing steps and detect changes
	for i, newStep := range args.Steps {
		newStatus := parseStatus(newStep.Status)

		if i < len(existingPlan.Steps) {
			oldStep := existingPlan.Steps[i]

			// Description change - use ModifyStep to persist
			if oldStep.Description != newStep.Description {
				if err := t.planManager.ModifyStep(ctx, t.sessionID, i, newStep.Description, newStep.Reasoning); err != nil {
					slog.ErrorContext(ctx, "[manage_plan] failed to modify step", "error", err, "step", i)
				} else {
					changes = append(changes, fmt.Sprintf("Step %d: description updated", i+1))
				}
			}

			// Status change
			if oldStep.Status != newStatus {
				if err := t.planManager.UpdateStepStatus(ctx, t.sessionID, i, newStatus, ""); err != nil {
					slog.ErrorContext(ctx, "[manage_plan] failed to update step status", "error", err, "step", i)
				} else {
					changes = append(changes, fmt.Sprintf("Step %d: %s → %s", i+1, oldStep.Status, newStatus))
				}
			}
		} else {
			// New step added
			changes = append(changes, fmt.Sprintf("Step %d: added", i+1))
		}
	}

	// Get updated plan
	updatedPlan, err := t.planManager.GetActivePlan(ctx, t.sessionID)
	if err != nil || updatedPlan == nil {
		updatedPlan = existingPlan
	}

	slog.InfoContext(ctx, "[manage_plan] plan updated",
		"plan_id", existingPlan.ID,
		"changes", len(changes),
		"warnings", len(warnings))

	// Include warnings in response so agent sees them
	response := formatPlanResponse(updatedPlan, "updated")
	if len(warnings) > 0 {
		response = strings.Join(warnings, "\n") + "\n\n" + response
	}
	return response, nil
}

// formatPlanResponse formats the plan as human-readable text
func formatPlanResponse(plan *domain.Plan, action string) string {
	completed, total := plan.Progress()

	result := fmt.Sprintf("Plan %s successfully (ID: %s)\n\n", action, plan.ID)
	result += fmt.Sprintf("Goal: %s\n\n", plan.Goal)
	result += fmt.Sprintf("Progress: [%d/%d steps completed]\n\n", completed, total)
	result += "Steps:\n"

	for _, step := range plan.Steps {
		var marker string
		switch step.Status {
		case domain.StepStatusCompleted:
			marker = "✓"
		case domain.StepStatusInProgress:
			marker = "→"
		default:
			marker = "○"
		}

		result += fmt.Sprintf("%s Step %d: %s", marker, step.Index+1, step.Description)
		if step.Reasoning != "" {
			result += fmt.Sprintf(" (%s)", step.Reasoning)
		}
		result += "\n"
	}

	return result
}

// parseManagePlanArgs parses arguments with fallback for when LLM sends steps as string
func parseManagePlanArgs(ctx context.Context, argumentsInJSON string) (*ManagePlanArgs, error) {
	// First try normal parsing (steps as array)
	var args ManagePlanArgs
	err := json.Unmarshal([]byte(argumentsInJSON), &args)
	if err == nil {
		return &args, nil
	}

	// Check if error is due to steps being a string instead of array
	if !strings.Contains(err.Error(), "cannot unmarshal string into Go struct field ManagePlanArgs.steps") {
		return nil, err
	}

	slog.WarnContext(ctx, "[manage_plan] LLM sent steps as string, attempting to parse")

	// Parse with steps as raw JSON to handle string case
	var rawArgs struct {
		Goal  string          `json:"goal"`
		Steps json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &rawArgs); err != nil {
		return nil, fmt.Errorf("failed to parse raw args: %w", err)
	}

	// Check if steps is a string (starts with quote)
	stepsBytes := []byte(rawArgs.Steps)
	if len(stepsBytes) == 0 {
		return nil, fmt.Errorf("steps is empty")
	}

	var steps []PlanStepState

	if stepsBytes[0] == '"' {
		// Steps is a JSON string - unmarshal to get the string value, then parse as array
		var stepsStr string
		if err := json.Unmarshal(stepsBytes, &stepsStr); err != nil {
			return nil, fmt.Errorf("failed to parse steps as string: %w", err)
		}
		if err := json.Unmarshal([]byte(stepsStr), &steps); err != nil {
			return nil, fmt.Errorf("failed to parse steps string content as array: %w", err)
		}
	} else {
		// Steps is already an array - parse directly
		if err := json.Unmarshal(stepsBytes, &steps); err != nil {
			return nil, fmt.Errorf("failed to parse steps as array: %w", err)
		}
	}

	slog.InfoContext(ctx, "[manage_plan] successfully parsed steps from string",
		"goal", rawArgs.Goal,
		"steps_count", len(steps))

	return &ManagePlanArgs{
		Goal:  rawArgs.Goal,
		Steps: steps,
	}, nil
}

// parseStatus converts string status to domain status
func parseStatus(status string) domain.PlanStepStatus {
	switch status {
	case "completed":
		return domain.StepStatusCompleted
	case "in_progress":
		return domain.StepStatusInProgress
	case "pending":
		return domain.StepStatusPending
	default:
		return domain.StepStatusPending
	}
}
