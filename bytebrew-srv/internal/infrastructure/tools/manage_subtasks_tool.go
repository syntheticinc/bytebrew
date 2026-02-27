package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// SubtaskManager defines operations for subtask management (consumer-side)
type SubtaskManager interface {
	CreateSubtask(ctx context.Context, sessionID, taskID, title, description string, blockedBy, files []string) (*domain.Subtask, error)
	GetSubtask(ctx context.Context, subtaskID string) (*domain.Subtask, error)
	GetSubtasksByTask(ctx context.Context, taskID string) ([]*domain.Subtask, error)
	GetReadySubtasks(ctx context.Context, taskID string) ([]*domain.Subtask, error)
	CompleteSubtask(ctx context.Context, subtaskID, result string) error
	FailSubtask(ctx context.Context, subtaskID, reason string) error
}

type manageSubtasksArgs struct {
	Action      string   `json:"action"` // create, list, get, get_ready, complete, fail
	TaskID      string   `json:"task_id,omitempty"`
	SubtaskID   string   `json:"subtask_id,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	BlockedBy   []string `json:"blocked_by,omitempty"`
	Files       []string `json:"files_involved,omitempty"`
	Result      string   `json:"result,omitempty"`
	Reason      string   `json:"reason,omitempty"`
}

// ManageSubtasksTool implements subtask management for Supervisor
type ManageSubtasksTool struct {
	manager   SubtaskManager
	sessionID string
}

// NewManageSubtasksTool creates a manage_subtasks tool
func NewManageSubtasksTool(manager SubtaskManager, sessionID string) tool.InvokableTool {
	return &ManageSubtasksTool{manager: manager, sessionID: sessionID}
}

func (t *ManageSubtasksTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "manage_subtasks",
		Desc: `Manage subtasks within a task.

IMPORTANT: Subtask description is the ONLY context the Code Agent receives.
Vague descriptions → agent guesses → wrong result. Be specific.

Actions:
- "create": Create subtask (requires task_id, title, description >100 chars; optional: blocked_by, files_involved)
- "list": List all subtasks for a task (requires task_id)
- "get": Get subtask details (requires subtask_id)
- "get_ready": Get ready subtasks — pending with no unfinished blockers (requires task_id)
- "complete": Mark subtask completed (requires subtask_id, result)
- "fail": Mark subtask failed (requires subtask_id, reason)

Subtasks support dependencies via blocked_by.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action":         {Type: schema.String, Desc: "Action to perform", Required: true},
			"task_id":        {Type: schema.String, Desc: "Task ID (for create, list, get_ready)"},
			"subtask_id":     {Type: schema.String, Desc: "Subtask ID (for get, complete, fail)"},
			"title":          {Type: schema.String, Desc: "Short subtask name, 5-10 words (for create)"},
			"description":    {Type: schema.String, Desc: "FULL specification for Code Agent (MUST be >100 chars). Include ALL: (1) what — interfaces, methods, signatures; (2) where — exact file paths; (3) how — existing code to follow as pattern; (4) acceptance criteria. This is the ONLY context the agent receives. Plain text, not JSON."},
			"blocked_by":     {Type: schema.Array, Desc: "Subtask IDs that block this subtask (for create)"},
			"files_involved": {Type: schema.Array, Desc: "EXISTING files the Code Agent should read for context (for create)"},
			"result":         {Type: schema.String, Desc: "Completion result (for complete)"},
			"reason":         {Type: schema.String, Desc: "Failure reason (for fail)"},
		}),
	}, nil
}

func (t *ManageSubtasksTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args manageSubtasksArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid JSON: %v", err), nil
	}

	slog.InfoContext(ctx, "[manage_subtasks] invoked", "action", args.Action, "task_id", args.TaskID, "subtask_id", args.SubtaskID)

	if args.Action == "" {
		return `[ERROR] "action" field is empty. You MUST specify an action.
Valid actions: create, list, get, get_ready, complete, fail.
Workflow: create subtasks first, then spawn_code_agent for each ready subtask.
Example: {"action": "create", "task_id": "abc", "title": "Fix imports", "description": "Fix broken imports in main.go"}`, nil
	}

	switch args.Action {
	case "create":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for create", nil
		}
		if args.Title == "" {
			return "[ERROR] title is required for create", nil
		}
		if looksLikeJSON(args.Description) {
			slog.WarnContext(ctx, "[manage_subtasks] description is JSON instead of plain text",
				"description_preview", truncateString(args.Description, 200))
			return "[ERROR] Subtask description must be plain text, not JSON. " +
				"Rewrite the description as human-readable text.", nil
		}
		// Description quality validation
		if args.Description == "" {
			return "[ERROR] description is required for create. " +
				"Provide FULL specification: what to implement, where (file paths), " +
				"style reference, acceptance criteria.", nil
		}
		if len(args.Description) < 100 {
			return fmt.Sprintf("[ERROR] Subtask description too short (%d chars, minimum 100). "+
				"Must include: what to implement (interfaces, methods), where (file paths), "+
				"style reference, acceptance criteria. Current: %q", len(args.Description), args.Description), nil
		}
		if strings.EqualFold(strings.TrimSpace(args.Description), strings.TrimSpace(args.Title)) {
			return "[ERROR] Description repeats the title. " +
				"Must be a FULL specification: what/where/how/acceptance criteria.", nil
		}
		descLower := strings.ToLower(args.Description)
		if !strings.Contains(descLower, "acceptance") && !strings.Contains(descLower, "criteria") &&
			!strings.Contains(descLower, "verify") && !strings.Contains(descLower, "must pass") {
			return "[ERROR] Subtask description missing acceptance criteria. " +
				"Add 'Acceptance: <how to verify>' section. Example: 'Acceptance: go build compiles, unit test passes.'", nil
		}
		subtask, err := t.manager.CreateSubtask(ctx, t.sessionID, args.TaskID, args.Title, args.Description, args.BlockedBy, args.Files)
		if err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		return fmt.Sprintf("Subtask created.\nID: %s\nTitle: %s\nDescription: %s\nTask: %s\nBlocked by: %v\nFiles: %v",
			subtask.ID, subtask.Title, subtask.Description, subtask.TaskID, subtask.BlockedBy, subtask.FilesInvolved), nil

	case "list":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for list", nil
		}
		subtasks, err := t.manager.GetSubtasksByTask(ctx, args.TaskID)
		if err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		if len(subtasks) == 0 {
			return "No subtasks found for this task.", nil
		}
		result := fmt.Sprintf("Subtasks for task %s (%d):\n", args.TaskID, len(subtasks))
		for _, subtask := range subtasks {
			agent := ""
			if subtask.AssignedAgentID != "" {
				agent = fmt.Sprintf(" [agent: %s]", subtask.AssignedAgentID)
			}
			result += fmt.Sprintf("  [%s] \"%s\" — %s%s\n", subtask.ID, subtask.Title, subtask.Status, agent)
		}
		return result, nil

	case "get":
		if args.SubtaskID == "" {
			return "[ERROR] subtask_id is required for get", nil
		}
		subtask, err := t.manager.GetSubtask(ctx, args.SubtaskID)
		if err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		if subtask == nil {
			return fmt.Sprintf("[ERROR] subtask not found: %s", args.SubtaskID), nil
		}
		return fmt.Sprintf("Subtask: %s\nTitle: %s\nDescription: %s\nStatus: %s\nAgent: %s\nBlocked by: %v\nFiles: %v\nResult: %s",
			subtask.ID, subtask.Title, subtask.Description, subtask.Status, subtask.AssignedAgentID, subtask.BlockedBy, subtask.FilesInvolved, subtask.Result), nil

	case "get_ready":
		if args.TaskID == "" {
			return "[ERROR] task_id is required for get_ready", nil
		}
		subtasks, err := t.manager.GetReadySubtasks(ctx, args.TaskID)
		if err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		if len(subtasks) == 0 {
			return "No ready subtasks (all pending subtasks have unfinished blockers, or no pending subtasks left).", nil
		}
		result := fmt.Sprintf("Ready subtasks (%d):\n", len(subtasks))
		for _, subtask := range subtasks {
			result += fmt.Sprintf("  [%s] \"%s\"\n", subtask.ID, subtask.Title)
		}
		return result, nil

	case "complete":
		if args.SubtaskID == "" {
			return "[ERROR] subtask_id is required for complete", nil
		}
		subtaskResult := args.Result
		if subtaskResult == "" {
			subtaskResult = "completed"
		}
		if err := t.manager.CompleteSubtask(ctx, args.SubtaskID, subtaskResult); err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		return fmt.Sprintf("Subtask %s completed.", args.SubtaskID), nil

	case "fail":
		if args.SubtaskID == "" {
			return "[ERROR] subtask_id is required for fail", nil
		}
		reason := args.Reason
		if reason == "" {
			reason = "no reason specified"
		}
		if err := t.manager.FailSubtask(ctx, args.SubtaskID, reason); err != nil {
			return fmt.Sprintf("[ERROR] %v", err), nil
		}
		return fmt.Sprintf("Subtask %s marked as failed: %s", args.SubtaskID, reason), nil

	default:
		return fmt.Sprintf("[ERROR] Unknown action: %s. Valid: create, list, get, get_ready, complete, fail", args.Action), nil
	}
}
