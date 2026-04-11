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

// --- admin_list_schemas ---

type adminListSchemasTool struct {
	repo SchemaRepository
}

func NewAdminListSchemasTool(repo SchemaRepository) tool.InvokableTool {
	return &adminListSchemasTool{repo: repo}
}

func (t *adminListSchemasTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_list_schemas",
		Desc: "Lists all schemas. A schema groups agents into a workflow with edges and triggers.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *adminListSchemasTool) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	schemas, err := t.repo.List(ctx)
	if err != nil {
		return fmt.Sprintf("[ERROR] Failed to list schemas: %v", err), nil
	}

	if len(schemas) == 0 {
		return "No schemas configured.", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %d schemas\n\n", len(schemas)))
	for _, s := range schemas {
		agents := "none"
		if len(s.AgentNames) > 0 {
			agents = strings.Join(s.AgentNames, ", ")
		}
		sb.WriteString(fmt.Sprintf("- **%s** (id=%s, agents=[%s]) — %s\n", s.Name, s.ID, agents, s.Description))
	}
	return sb.String(), nil
}

// --- admin_get_schema ---

type adminGetSchemaTool struct {
	repo SchemaRepository
}

func NewAdminGetSchemaTool(repo SchemaRepository) tool.InvokableTool {
	return &adminGetSchemaTool{repo: repo}
}

func (t *adminGetSchemaTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_get_schema",
		Desc: "Returns full details of a schema by ID, including assigned agents.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"schema_id": {Type: schema.String, Desc: "Schema ID", Required: true},
		}),
	}, nil
}

type getSchemaArgs struct {
	SchemaID string `json:"schema_id"`
}

func (t *adminGetSchemaTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args getSchemaArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.SchemaID == "" {
		return "[ERROR] schema_id is required", nil
	}

	s, err := t.repo.GetByID(ctx, args.SchemaID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Sprintf("Schema not found: %s", args.SchemaID), nil
		}
		return fmt.Sprintf("[ERROR] Failed to get schema: %v", err), nil
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Sprintf("[ERROR] failed to serialize result: %v", err), nil
	}
	return string(data), nil
}

// --- admin_create_schema ---

type adminCreateSchemaTool struct {
	repo     SchemaRepository
	reloader func()
}

func NewAdminCreateSchemaTool(repo SchemaRepository, reloader func()) tool.InvokableTool {
	return &adminCreateSchemaTool{repo: repo, reloader: reloader}
}

func (t *adminCreateSchemaTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_create_schema",
		Desc: "Creates a new schema (workflow). Requires name. Optional: description.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"name":        {Type: schema.String, Desc: "Schema name", Required: true},
			"description": {Type: schema.String, Desc: "Schema description", Required: false},
		}),
	}, nil
}

type createSchemaArgs struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (t *adminCreateSchemaTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args createSchemaArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.Name == "" {
		return "[ERROR] name is required", nil
	}

	record := &SchemaRecord{
		Name:        args.Name,
		Description: args.Description,
	}

	if err := t.repo.Create(ctx, record); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Sprintf("Schema with name %q already exists.", args.Name), nil
		}
		return fmt.Sprintf("[ERROR] Failed to create schema: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminCreateSchema] created schema", "name", args.Name, "id", record.ID)
	return fmt.Sprintf("Schema %q created (id=%s).", args.Name, record.ID), nil
}

// --- admin_update_schema ---

type adminUpdateSchemaTool struct {
	repo     SchemaRepository
	reloader func()
}

func NewAdminUpdateSchemaTool(repo SchemaRepository, reloader func()) tool.InvokableTool {
	return &adminUpdateSchemaTool{repo: repo, reloader: reloader}
}

func (t *adminUpdateSchemaTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_update_schema",
		Desc: "Updates an existing schema by ID.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"schema_id":   {Type: schema.String, Desc: "Schema ID to update", Required: true},
			"name":        {Type: schema.String, Desc: "New name", Required: false},
			"description": {Type: schema.String, Desc: "New description", Required: false},
		}),
	}, nil
}

type updateSchemaArgs struct {
	SchemaID    string `json:"schema_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (t *adminUpdateSchemaTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args updateSchemaArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.SchemaID == "" {
		return "[ERROR] schema_id is required", nil
	}

	record := &SchemaRecord{
		Name:        args.Name,
		Description: args.Description,
	}

	if err := t.repo.Update(ctx, args.SchemaID, record); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Sprintf("Schema not found: %s", args.SchemaID), nil
		}
		return fmt.Sprintf("[ERROR] Failed to update schema: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminUpdateSchema] updated schema", "id", args.SchemaID)
	return fmt.Sprintf("Schema %s updated successfully.", args.SchemaID), nil
}

// --- admin_delete_schema ---

type adminDeleteSchemaTool struct {
	repo     SchemaRepository
	reloader func()
}

func NewAdminDeleteSchemaTool(repo SchemaRepository, reloader func()) tool.InvokableTool {
	return &adminDeleteSchemaTool{repo: repo, reloader: reloader}
}

func (t *adminDeleteSchemaTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_delete_schema",
		Desc: "Deletes a schema by ID. WARNING: This removes all edges and agent associations in the schema.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"schema_id": {Type: schema.String, Desc: "Schema ID to delete", Required: true},
		}),
	}, nil
}

type deleteSchemaArgs struct {
	SchemaID string `json:"schema_id"`
}

func (t *adminDeleteSchemaTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args deleteSchemaArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.SchemaID == "" {
		return "[ERROR] schema_id is required", nil
	}

	if err := t.repo.Delete(ctx, args.SchemaID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Sprintf("Schema not found: %s", args.SchemaID), nil
		}
		return fmt.Sprintf("[ERROR] Failed to delete schema: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminDeleteSchema] deleted schema", "id", args.SchemaID)
	return fmt.Sprintf("Schema %s deleted successfully.", args.SchemaID), nil
}
