package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// EngineTaskManager defines operations for managing engine tasks (consumer-side).
type EngineTaskManager interface {
	CreateTask(ctx context.Context, params CreateEngineTaskParams) (string, error)
	UpdateTask(ctx context.Context, id string, title, description string) error
	SetTaskStatus(ctx context.Context, id string, status string, result string) error
	ListTasks(ctx context.Context, sessionID string) ([]EngineTaskSummary, error)
	CreateSubTask(ctx context.Context, parentID string, params CreateEngineTaskParams) (string, error)
}

// CreateEngineTaskParams holds parameters for creating an engine task.
type CreateEngineTaskParams struct {
	Title       string
	Description string
	AgentName   string
	SessionID   string
	Source      string
	UserID      string
}

// EngineTaskSummary is a lightweight view of an engine task.
type EngineTaskSummary struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Status    string  `json:"status"`
	AgentName string  `json:"agent_name"`
	ParentID  *string `json:"parent_id,omitempty"`
}

type engineManageTasksArgs struct {
	Action       string                   `json:"action"`
	Tasks        []engineManageTaskCreate `json:"tasks,omitempty"`
	TaskID       string                   `json:"task_id,omitempty"`
	ParentTaskID string                   `json:"parent_task_id,omitempty"`
	Title        string                   `json:"title,omitempty"`
	Description  string                   `json:"description,omitempty"`
	Status       string                   `json:"status,omitempty"`
	Result       string                   `json:"result,omitempty"`
}

type engineManageTaskCreate struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// EngineManageTasksTool implements engine task management as an Eino tool.
type EngineManageTasksTool struct {
	manager   EngineTaskManager
	sessionID string
}

// NewEngineManageTasksTool creates a new engine_manage_tasks tool.
func NewEngineManageTasksTool(manager EngineTaskManager, sessionID string) tool.InvokableTool {
	return &EngineManageTasksTool{manager: manager, sessionID: sessionID}
}

func (t *EngineManageTasksTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "manage_tasks",
		Desc: `Manage engine tasks (universal work units).

Actions:
- "create": Create one or more tasks. Requires tasks array: [{title, description}].
- "update": Update task title/description. Requires task_id, optional title and description.
- "set_status": Set task status. Requires task_id, status (pending/in_progress/completed/failed/cancelled). Optional result.
- "list": List all tasks for current session.
- "create_subtask": Create a sub-task. Requires parent_task_id, title, optional description.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"action":         {Type: schema.String, Desc: "Action to perform: create, update, set_status, list, create_subtask", Required: true},
			"tasks":          {Type: schema.Array, Desc: "Array of {title, description} objects (for create)"},
			"task_id":        {Type: schema.String, Desc: "Task ID (for update, set_status)"},
			"parent_task_id": {Type: schema.String, Desc: "Parent task ID (for create_subtask)"},
			"title":          {Type: schema.String, Desc: "Task title (for update, create_subtask)"},
			"description":    {Type: schema.String, Desc: "Task description (for update, create_subtask)"},
			"status":         {Type: schema.String, Desc: "Target status (for set_status)"},
			"result":         {Type: schema.String, Desc: "Task result text (for set_status with completed)"},
		}),
	}, nil
}

func (t *EngineManageTasksTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var args engineManageTasksArgs
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid JSON: %v", err), nil
	}

	slog.InfoContext(ctx, "[engine_manage_tasks] invoked", "action", args.Action, "session_id", t.sessionID)

	switch args.Action {
	case "create":
		return t.handleCreate(ctx, args)
	case "update":
		return t.handleUpdate(ctx, args)
	case "set_status":
		return t.handleSetStatus(ctx, args)
	case "list":
		return t.handleList(ctx)
	case "create_subtask":
		return t.handleCreateSubtask(ctx, args)
	default:
		return fmt.Sprintf("[ERROR] Unknown action: %q. Valid: create, update, set_status, list, create_subtask", args.Action), nil
	}
}

func (t *EngineManageTasksTool) handleCreate(ctx context.Context, args engineManageTasksArgs) (string, error) {
	if len(args.Tasks) == 0 {
		return "[ERROR] tasks array is required and must not be empty for create action", nil
	}

	var ids []string
	for _, task := range args.Tasks {
		if task.Title == "" {
			return "[ERROR] each task must have a title", nil
		}
		id, err := t.manager.CreateTask(ctx, CreateEngineTaskParams{
			Title:       task.Title,
			Description: task.Description,
			SessionID:   t.sessionID,
			Source:      "agent",
		})
		if err != nil {
			return fmt.Sprintf("[ERROR] failed to create task %q: %v", task.Title, err), nil
		}
		ids = append(ids, id)
	}

	if len(ids) == 1 {
		return fmt.Sprintf("Task created (ID: %s).", ids[0]), nil
	}

	result := fmt.Sprintf("%d tasks created:", len(ids))
	for i, id := range ids {
		result += fmt.Sprintf("\n  [%s] %s", id, args.Tasks[i].Title)
	}
	return result, nil
}

func (t *EngineManageTasksTool) handleUpdate(ctx context.Context, args engineManageTasksArgs) (string, error) {
	if args.TaskID == "" {
		return "[ERROR] task_id is required for update", nil
	}
	if args.Title == "" && args.Description == "" {
		return "[ERROR] at least one of title or description must be provided for update", nil
	}
	if err := t.manager.UpdateTask(ctx, args.TaskID, args.Title, args.Description); err != nil {
		return fmt.Sprintf("[ERROR] %v", err), nil
	}
	return fmt.Sprintf("Task %s updated.", args.TaskID), nil
}

func (t *EngineManageTasksTool) handleSetStatus(ctx context.Context, args engineManageTasksArgs) (string, error) {
	if args.TaskID == "" {
		return "[ERROR] task_id is required for set_status", nil
	}
	if args.Status == "" {
		return "[ERROR] status is required for set_status", nil
	}
	if err := t.manager.SetTaskStatus(ctx, args.TaskID, args.Status, args.Result); err != nil {
		return fmt.Sprintf("[ERROR] %v", err), nil
	}
	return fmt.Sprintf("Task %s status set to %q.", args.TaskID, args.Status), nil
}

func (t *EngineManageTasksTool) handleList(ctx context.Context) (string, error) {
	tasks, err := t.manager.ListTasks(ctx, t.sessionID)
	if err != nil {
		return fmt.Sprintf("[ERROR] %v", err), nil
	}
	if len(tasks) == 0 {
		return "No tasks found for this session.", nil
	}

	result := fmt.Sprintf("Tasks (%d):\n", len(tasks))
	for _, tk := range tasks {
		line := fmt.Sprintf("  [%s] %q — %s", tk.ID, tk.Title, tk.Status)
		if tk.ParentID != nil {
			line += fmt.Sprintf(" (parent: %s)", *tk.ParentID)
		}
		result += line + "\n"
	}
	return result, nil
}

func (t *EngineManageTasksTool) handleCreateSubtask(ctx context.Context, args engineManageTasksArgs) (string, error) {
	if args.ParentTaskID == "" {
		return "[ERROR] parent_task_id is required for create_subtask", nil
	}
	if args.Title == "" {
		return "[ERROR] title is required for create_subtask", nil
	}

	id, err := t.manager.CreateSubTask(ctx, args.ParentTaskID, CreateEngineTaskParams{
		Title:       args.Title,
		Description: args.Description,
		SessionID:   t.sessionID,
		Source:      "agent",
	})
	if err != nil {
		return fmt.Sprintf("[ERROR] %v", err), nil
	}
	return fmt.Sprintf("Sub-task created (ID: %s, parent: %s).", id, args.ParentTaskID), nil
}
