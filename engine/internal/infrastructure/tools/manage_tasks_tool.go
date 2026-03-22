package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// TaskManager defines operations for task management (consumer-side)
type TaskManager interface {
	CreateTask(ctx context.Context, sessionID, title, description string, criteria []string) (*domain.Task, error)
	ApproveTask(ctx context.Context, taskID string) error
	StartTask(ctx context.Context, taskID string) error
	GetTask(ctx context.Context, taskID string) (*domain.Task, error)
	GetTasks(ctx context.Context, sessionID string) ([]*domain.Task, error)
	CompleteTask(ctx context.Context, taskID string) error
	FailTask(ctx context.Context, taskID, reason string) error
	CancelTask(ctx context.Context, taskID, reason string) error
	SetTaskPriority(ctx context.Context, taskID string, priority int) error
	GetNextTask(ctx context.Context, sessionID string) (*domain.Task, error)
}

type manageTasksArgs struct {
	Action             string   `json:"action"` // create, approve, list, get, complete, fail, cancel, start, set_priority, get_queue
	Title              string   `json:"title,omitempty"`
	Description        string   `json:"description,omitempty"`
	AcceptanceCriteria []string `json:"acceptance_criteria,omitempty"`
	TaskID             string   `json:"task_id,omitempty"`
	Reason             string   `json:"reason,omitempty"`
	Priority           int      `json:"priority,omitempty"` // 0 = normal, 1 = high, 2 = critical
}

// ManageTasksTool implements task management for Supervisor
type ManageTasksTool struct {
	manager   TaskManager
	asker     UserAsker
	sessionID string
}

// NewManageTasksTool creates a manage_tasks tool
func NewManageTasksTool(manager TaskManager, asker UserAsker, sessionID string) tool.InvokableTool {
	return &ManageTasksTool{manager: manager, asker: asker, sessionID: sessionID}
}

func (t *ManageTasksTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "manage_tasks",
		Desc: `Manage work tasks (high-level work items requiring user approval).

Actions:
- "create": Create a new task and ask user for approval (requires title, description, acceptance_criteria, optional priority). This BLOCKS until user responds — returns approved/rejected status.
- "start": Start an approved task (mark as in_progress, requires task_id)
- "list": List all tasks for current session
- "get": Get task details (requires task_id)
- "get_queue": Get tasks ordered by priority (highest first) — use this to pick next task to work on
- "set_priority": Set task priority (requires task_id, priority: 0=normal, 1=high, 2=critical)
- "complete": Mark task as completed when all subtasks done (requires task_id)
- "fail": Mark task as failed (requires task_id, reason)
- "cancel": Cancel a task (requires task_id, optional reason)

NOTE: "create" automatically asks the user for approval. Do NOT call ask_user separately for task approval.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action":              {Type: schema.String, Desc: "Action to perform", Required: true},
			"title":               {Type: schema.String, Desc: "Task title (for create)"},
			"description":         {Type: schema.String, Desc: "Full task specification: goal, current state (file paths), changes required, constraints. Survives context compression — must be self-contained. See supervisor_prompt for templates."},
			"acceptance_criteria": {Type: schema.Array, Desc: "Acceptance criteria list (for create)"},
			"task_id":             {Type: schema.String, Desc: "Task ID (for start, get, complete, fail, set_priority)"},
			"reason":              {Type: schema.String, Desc: "Failure reason (for fail)"},
			"priority":            {Type: schema.Integer, Desc: "Task priority: 0=normal, 1=high, 2=critical (for create, set_priority)"},
		}),
	}, nil
}

func (t *ManageTasksTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args manageTasksArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid JSON: %v", err), nil
	}

	slog.InfoContext(ctx, "[manage_tasks] invoked", "action", args.Action, "task_id", args.TaskID)

	if args.Action == "" {
		return `[ERROR] "action" field is empty. You MUST specify an action.
Valid actions: create, start, list, get, get_queue, set_priority, complete, fail, cancel.
Workflow: create task (asks user for approval) → start → create subtasks → spawn agents.
Example: {"action": "create", "title": "Fix compilation errors", "description": "Fix all compilation errors in server.go", "priority": 1}`, nil
	}

	switch args.Action {
	case "create":
		if args.Title == "" {
			return "[ERROR] title is required for create", nil
		}
		if looksLikeJSON(args.Description) {
			slog.WarnContext(ctx, "[manage_tasks] description is JSON instead of plain text",
				"description_preview", truncateString(args.Description, 200))
			return "[ERROR] Task description must be plain text (markdown allowed), not JSON. " +
				"Rewrite the description as human-readable text.", nil
		}
		task, err := t.manager.CreateTask(ctx, t.sessionID, args.Title, args.Description, args.AcceptanceCriteria)
		if err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}

		// Set priority if provided
		if args.Priority > 0 {
			if err := task.SetPriority(args.Priority); err != nil {
				return fmt.Sprintf("[ERROR] invalid priority: %v", err), nil
			}
			if err := t.manager.SetTaskPriority(ctx, task.ID, args.Priority); err != nil {
				slog.WarnContext(ctx, "[manage_tasks] failed to set priority", "error", err)
			}
		}

		// Generate MD file for the task (server-side reference)
		t.generateTaskMD(task)

		// Build formatted markdown for user approval
		userContent := t.buildTaskMarkdown(task)
		approvalQuestion := fmt.Sprintf("%s\nDo you approve this Task?", userContent)

		// Build questionnaire JSON with a single approval question
		approvalQuestions := []Question{
			{
				Text:    approvalQuestion,
				Options: []QuestionOption{{Label: "approved"}, {Label: "rejected"}},
				Default: "approved",
			},
		}
		questionsJSON, jsonErr := json.Marshal(approvalQuestions)
		if jsonErr != nil {
			return fmt.Sprintf("Task created (ID: %s, title: %s, status: %s). "+
				"Failed to serialize approval question: %v. Call ask_user manually to get approval.",
				task.ID, task.Title, task.Status, jsonErr), nil
		}

		// Directly ask user for approval (bypasses LLM reformatting issues)
		slog.InfoContext(ctx, "[manage_tasks] asking user for task approval", "task_id", task.ID)
		answersJSON, askErr := t.asker.AskUserQuestionnaire(ctx, t.sessionID, string(questionsJSON))
		if askErr != nil {
			slog.ErrorContext(ctx, "[manage_tasks] ask_user failed", "error", askErr)
			return fmt.Sprintf("Task created (ID: %s, title: %s, status: %s). "+
				"Failed to get user approval: %v. Call ask_user manually to get approval.",
				task.ID, task.Title, task.Status, askErr), nil
		}

		// Parse the answer from questionnaire response
		answer := extractFirstAnswer(answersJSON)

		// Process user response
		isApproved := strings.EqualFold(strings.TrimSpace(answer), "approved") ||
			strings.EqualFold(strings.TrimSpace(answer), "yes") ||
			strings.EqualFold(strings.TrimSpace(answer), "да")

		if isApproved {
			if err := t.manager.ApproveTask(ctx, task.ID); err != nil {
				return fmt.Sprintf("Task created (ID: %s) but approval failed: %v", task.ID, err), nil
			}
			return fmt.Sprintf("Task %s created and approved by user. You can now start it and create subtasks.", task.ID), nil
		}

		// User rejected or gave feedback
		if strings.EqualFold(strings.TrimSpace(answer), "cancelled") {
			if err := t.manager.CancelTask(ctx, task.ID, "rejected by user"); err != nil {
				slog.ErrorContext(ctx, "[manage_tasks] cancel failed", "error", err)
			}
			return fmt.Sprintf("Task %s was rejected by user and cancelled.", task.ID), nil
		}

		// User gave custom feedback — cancel the draft and return feedback
		if err := t.manager.CancelTask(ctx, task.ID, "user provided feedback"); err != nil {
			slog.ErrorContext(ctx, "[manage_tasks] cancel failed", "error", err)
		}
		return fmt.Sprintf("Task %s was NOT approved. User feedback: %s\nRevise the task based on this feedback and create a new one.", task.ID, answer), nil

	case "approve":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for approve", nil
		}
		if err := t.manager.ApproveTask(ctx, args.TaskID); err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		return fmt.Sprintf("Task %s approved. You can now create subtasks for it.", args.TaskID), nil

	case "start":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for start", nil
		}
		if err := t.manager.StartTask(ctx, args.TaskID); err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		return fmt.Sprintf("Task %s started (in_progress).", args.TaskID), nil

	case "list":
		tasks, err := t.manager.GetTasks(ctx, t.sessionID)
		if err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		if len(tasks) == 0 {
			return "No tasks found.", nil
		}
		result := fmt.Sprintf("Tasks (%d):\n", len(tasks))
		for _, tk := range tasks {
			result += fmt.Sprintf("  [%s] \"%s\" — %s\n", tk.ID, tk.Title, tk.Status)
		}
		return result, nil

	case "get":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for get", nil
		}
		task, err := t.manager.GetTask(ctx, args.TaskID)
		if err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		if task == nil {
			return fmt.Sprintf("[ERROR] task not found: %s", args.TaskID), nil
		}
		return fmt.Sprintf("Task: %s\nTitle: %s\nDescription: %s\nStatus: %s\nCriteria: %v",
			task.ID, task.Title, task.Description, task.Status, task.AcceptanceCriteria), nil

	case "complete":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for complete", nil
		}
		if err := t.manager.CompleteTask(ctx, args.TaskID); err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		return fmt.Sprintf("Task %s completed.", args.TaskID), nil

	case "fail":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for fail", nil
		}
		reason := args.Reason
		if reason == "" {
			reason = "no reason specified"
		}
		if err := t.manager.FailTask(ctx, args.TaskID, reason); err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		return fmt.Sprintf("Task %s marked as failed: %s", args.TaskID, reason), nil

	case "cancel":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for cancel", nil
		}
		reason := args.Reason
		if reason == "" {
			reason = "cancelled by user"
		}
		if err := t.manager.CancelTask(ctx, args.TaskID, reason); err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		return fmt.Sprintf("Task %s cancelled: %s", args.TaskID, reason), nil

	case "set_priority":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for set_priority", nil
		}
		if args.Priority < 0 || args.Priority > 2 {
			return "[ERROR] priority must be 0 (normal), 1 (high), or 2 (critical)", nil
		}
		if err := t.manager.SetTaskPriority(ctx, args.TaskID, args.Priority); err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		priorityLabels := []string{"normal", "high", "critical"}
		return fmt.Sprintf("Task %s priority set to %d (%s)", args.TaskID, args.Priority, priorityLabels[args.Priority]), nil

	case "get_queue":
		nextTask, err := t.manager.GetNextTask(ctx, t.sessionID)
		if err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		if nextTask == nil {
			return "No tasks available for work (no approved or in_progress tasks)", nil
		}
		priorityLabels := []string{"normal", "high", "critical"}
		return fmt.Sprintf("Next task: [%s] \"%s\" (priority: %d=%s, status: %s)\nDescription: %s",
			nextTask.ID, nextTask.Title, nextTask.Priority, priorityLabels[nextTask.Priority],
			nextTask.Status, nextTask.Description), nil

	default:
		return fmt.Sprintf("[ERROR] Unknown action: %s. Valid: create, approve, start, list, get, get_queue, set_priority, complete, fail, cancel", args.Action), nil
	}
}

// buildTaskMarkdown generates markdown content for a task (without writing to disk)
func (t *ManageTasksTool) buildTaskMarkdown(task *domain.Task) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Task: %s\n\n", task.Title))
	sb.WriteString(fmt.Sprintf("## Status: %s\n\n", task.Status))

	if task.Description != "" {
		sb.WriteString("## Description\n")
		sb.WriteString(task.Description)
		sb.WriteString("\n\n")
	}

	if len(task.AcceptanceCriteria) > 0 {
		sb.WriteString("## Acceptance Criteria\n")
		for _, criterion := range task.AcceptanceCriteria {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", criterion))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildTaskMarkdownWithMetadata generates full markdown including internal metadata (for disk/agent)
func (t *ManageTasksTool) buildTaskMarkdownWithMetadata(task *domain.Task) string {
	content := t.buildTaskMarkdown(task)
	content += "## Metadata\n"
	content += fmt.Sprintf("- ID: %s\n", task.ID)
	content += fmt.Sprintf("- Created: %s\n", time.Now().Format(time.RFC3339))
	content += fmt.Sprintf("- Session: %s\n", t.sessionID)
	return content
}

// generateTaskMD creates a markdown file for the task on disk
func (t *ManageTasksTool) generateTaskMD(task *domain.Task) string {
	taskDir := filepath.Join("temp", "tasks")
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		slog.Error("failed to create tasks directory", "error", err, "path", taskDir)
		return ""
	}

	content := t.buildTaskMarkdownWithMetadata(task)

	filename := fmt.Sprintf("task-%s.md", task.ID)
	mdPath := filepath.Join(taskDir, filename)

	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		slog.Error("failed to write task MD", "error", err, "path", mdPath)
		return ""
	}

	slog.Info("task MD file created", "path", mdPath, "task_id", task.ID)
	return mdPath
}

// extractFirstAnswer parses questionnaire JSON response and returns the first answer.
// Falls back to returning the raw string if JSON parsing fails.
func extractFirstAnswer(answersJSON string) string {
	var answers []QuestionAnswer
	if err := json.Unmarshal([]byte(answersJSON), &answers); err != nil {
		return strings.TrimSpace(answersJSON)
	}
	if len(answers) == 0 {
		return strings.TrimSpace(answersJSON)
	}
	return answers[0].Answer
}
