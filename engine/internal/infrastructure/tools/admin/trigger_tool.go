package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// --- admin_list_triggers ---

type adminListTriggersTool struct {
	repo TriggerRepository
}

func NewAdminListTriggersTool(repo TriggerRepository) tool.InvokableTool {
	return &adminListTriggersTool{repo: repo}
}

func (t *adminListTriggersTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_list_triggers",
		Desc: "Lists all triggers. Triggers start agent workflows: cron (scheduled), webhook (HTTP endpoint), or chat (user-initiated).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *adminListTriggersTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	triggers, err := t.repo.List(ctx)
	if err != nil {
		return fmt.Sprintf("[ERROR] Failed to list triggers: %v", err), nil
	}

	if len(triggers) == 0 {
		return "No triggers configured.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %d triggers\n\n", len(triggers)))
	for _, tr := range triggers {
		detail := ""
		switch tr.Type {
		case "cron":
			detail = fmt.Sprintf("schedule=%s", tr.Schedule)
		case "webhook":
			detail = fmt.Sprintf("path=%s", tr.WebhookPath)
		default:
			detail = tr.Type
		}
		sb.WriteString(fmt.Sprintf("- id=%s **%s** (type=%s, %s, enabled=%v)\n",
			tr.ID, tr.Title, tr.Type, detail, tr.Enabled))
	}
	return sb.String(), nil
}

// --- admin_create_trigger ---

type adminCreateTriggerTool struct {
	repo     TriggerRepository
	reloader func()
}

func NewAdminCreateTriggerTool(repo TriggerRepository, reloader func()) tool.InvokableTool {
	return &adminCreateTriggerTool{repo: repo, reloader: reloader}
}

func (t *adminCreateTriggerTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_create_trigger",
		Desc: "Creates a new trigger. For cron: provide schedule. For webhook: provide webhook_path. agent_name is the entry agent.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"type":         {Type: schema.String, Desc: "Trigger type: cron, webhook, or chat", Required: true},
			"title":        {Type: schema.String, Desc: "Trigger title", Required: true},
			"agent_name":   {Type: schema.String, Desc: "Entry agent name", Required: true},
			"schedule":     {Type: schema.String, Desc: "Cron expression (for cron type)", Required: false},
			"webhook_path": {Type: schema.String, Desc: "Webhook URL path (for webhook type)", Required: false},
			"description":  {Type: schema.String, Desc: "Trigger description", Required: false},
			"enabled":      {Type: schema.Boolean, Desc: "Whether trigger is enabled (default: true)", Required: false},
		}),
	}, nil
}

type createTriggerArgs struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	AgentName   string `json:"agent_name"`
	Schedule    string `json:"schedule"`
	WebhookPath string `json:"webhook_path"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
}

func (t *adminCreateTriggerTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args createTriggerArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.Type == "" {
		return "[ERROR] type is required", nil
	}
	if args.Title == "" {
		return "[ERROR] title is required", nil
	}
	enabled := true
	if args.Enabled != nil {
		enabled = *args.Enabled
	}

	record := &TriggerRecord{
		Type:        args.Type,
		Title:       args.Title,
		Schedule:    args.Schedule,
		WebhookPath: args.WebhookPath,
		Description: args.Description,
		Enabled:     enabled,
	}

	if err := t.repo.Create(ctx, record); err != nil {
		return fmt.Sprintf("[ERROR] Failed to create trigger: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminCreateTrigger] created", "title", args.Title, "type", args.Type)
	return fmt.Sprintf("Trigger %q created (id=%s, type=%s).", args.Title, record.ID, args.Type), nil
}

// --- admin_update_trigger ---

type adminUpdateTriggerTool struct {
	repo     TriggerRepository
	reloader func()
}

func NewAdminUpdateTriggerTool(repo TriggerRepository, reloader func()) tool.InvokableTool {
	return &adminUpdateTriggerTool{repo: repo, reloader: reloader}
}

func (t *adminUpdateTriggerTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_update_trigger",
		Desc: "Updates an existing trigger by ID.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"trigger_id":   {Type: schema.String, Desc: "Trigger ID to update", Required: true},
			"type":         {Type: schema.String, Desc: "New type", Required: false},
			"title":        {Type: schema.String, Desc: "New title", Required: false},
			"agent_name":   {Type: schema.String, Desc: "New agent name", Required: false},
			"schedule":     {Type: schema.String, Desc: "New cron schedule", Required: false},
			"webhook_path": {Type: schema.String, Desc: "New webhook path", Required: false},
			"description":  {Type: schema.String, Desc: "New description", Required: false},
			"enabled":      {Type: schema.Boolean, Desc: "Enable or disable", Required: false},
		}),
	}, nil
}

type updateTriggerArgs struct {
	TriggerID   string `json:"trigger_id"`
	Type        string `json:"type"`
	Title       string `json:"title"`
	AgentName   string `json:"agent_name"`
	Schedule    string `json:"schedule"`
	WebhookPath string `json:"webhook_path"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
}

func (t *adminUpdateTriggerTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args updateTriggerArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.TriggerID == "" {
		return "[ERROR] trigger_id is required", nil
	}

	existing, err := t.repo.GetByID(ctx, args.TriggerID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Sprintf("Trigger not found: %s", args.TriggerID), nil
		}
		return fmt.Sprintf("[ERROR] Failed to get trigger: %v", err), nil
	}
	record := &TriggerRecord{
		Type:        coalesce(args.Type, existing.Type),
		Title:       coalesce(args.Title, existing.Title),
		Schedule:    coalesce(args.Schedule, existing.Schedule),
		WebhookPath: coalesce(args.WebhookPath, existing.WebhookPath),
		Description: coalesce(args.Description, existing.Description),
		Enabled:     existing.Enabled,
	}
	if args.Enabled != nil {
		record.Enabled = *args.Enabled
	}

	if err := t.repo.Update(ctx, args.TriggerID, record); err != nil {
		return fmt.Sprintf("[ERROR] Failed to update trigger: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminUpdateTrigger] updated", "id", args.TriggerID)
	return fmt.Sprintf("Trigger %s updated successfully.", args.TriggerID), nil
}

// --- admin_delete_trigger ---

type adminDeleteTriggerTool struct {
	repo     TriggerRepository
	reloader func()
}

func NewAdminDeleteTriggerTool(repo TriggerRepository, reloader func()) tool.InvokableTool {
	return &adminDeleteTriggerTool{repo: repo, reloader: reloader}
}

func (t *adminDeleteTriggerTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_delete_trigger",
		Desc: "Deletes a trigger by ID.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"trigger_id": {Type: schema.String, Desc: "Trigger ID to delete", Required: true},
		}),
	}, nil
}

type deleteTriggerArgs struct {
	TriggerID string `json:"trigger_id"`
}

func (t *adminDeleteTriggerTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args deleteTriggerArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.TriggerID == "" {
		return "[ERROR] trigger_id is required", nil
	}

	if err := t.repo.Delete(ctx, args.TriggerID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Sprintf("Trigger not found: %s", args.TriggerID), nil
		}
		return fmt.Sprintf("[ERROR] Failed to delete trigger: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminDeleteTrigger] deleted", "id", args.TriggerID)
	return fmt.Sprintf("Trigger %s deleted successfully.", args.TriggerID), nil
}
