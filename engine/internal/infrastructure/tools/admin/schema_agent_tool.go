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

// --- admin_add_agent_to_schema ---

type adminAddAgentToSchemaTool struct {
	repo     SchemaRepository
	reloader func()
}

func NewAdminAddAgentToSchemaTool(repo SchemaRepository, reloader func()) tool.InvokableTool {
	return &adminAddAgentToSchemaTool{repo: repo, reloader: reloader}
}

func (t *adminAddAgentToSchemaTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_add_agent_to_schema",
		Desc: "Adds an agent to a schema. The agent must exist. After adding, create edges to connect agents.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"schema_id":  {Type: schema.String, Desc: "Schema ID", Required: true},
			"agent_name": {Type: schema.String, Desc: "Agent name to add", Required: true},
		}),
	}, nil
}

type schemaAgentArgs struct {
	SchemaID  string `json:"schema_id"`
	AgentName string `json:"agent_name"`
}

func (t *adminAddAgentToSchemaTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args schemaAgentArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.SchemaID == "" {
		return "[ERROR] schema_id is required", nil
	}
	if args.AgentName == "" {
		return "[ERROR] agent_name is required", nil
	}

	if err := t.repo.AddAgent(ctx, args.SchemaID, args.AgentName); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Sprintf("Schema or agent not found (schema_id=%s, agent=%s).", args.SchemaID, args.AgentName), nil
		}
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Sprintf("Agent %q is already in schema %s.", args.AgentName, args.SchemaID), nil
		}
		return fmt.Sprintf("[ERROR] Failed to add agent to schema: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminAddAgentToSchema] added", "schema_id", args.SchemaID, "agent", args.AgentName)
	return fmt.Sprintf("Agent %q added to schema %s.", args.AgentName, args.SchemaID), nil
}

// --- admin_remove_agent_from_schema ---

type adminRemoveAgentFromSchemaTool struct {
	repo     SchemaRepository
	reloader func()
}

func NewAdminRemoveAgentFromSchemaTool(repo SchemaRepository, reloader func()) tool.InvokableTool {
	return &adminRemoveAgentFromSchemaTool{repo: repo, reloader: reloader}
}

func (t *adminRemoveAgentFromSchemaTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_remove_agent_from_schema",
		Desc: "Removes an agent from a schema. Does not delete the agent itself.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"schema_id":  {Type: schema.String, Desc: "Schema ID", Required: true},
			"agent_name": {Type: schema.String, Desc: "Agent name to remove", Required: true},
		}),
	}, nil
}

func (t *adminRemoveAgentFromSchemaTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args schemaAgentArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.SchemaID == "" {
		return "[ERROR] schema_id is required", nil
	}
	if args.AgentName == "" {
		return "[ERROR] agent_name is required", nil
	}

	if err := t.repo.RemoveAgent(ctx, args.SchemaID, args.AgentName); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Sprintf("Agent %q not found in schema %s.", args.AgentName, args.SchemaID), nil
		}
		return fmt.Sprintf("[ERROR] Failed to remove agent from schema: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminRemoveAgentFromSchema] removed", "schema_id", args.SchemaID, "agent", args.AgentName)
	return fmt.Sprintf("Agent %q removed from schema %s.", args.AgentName, args.SchemaID), nil
}
