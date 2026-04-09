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

// --- admin_list_edges ---

type adminListEdgesTool struct {
	repo EdgeRepository
}

func NewAdminListEdgesTool(repo EdgeRepository) tool.InvokableTool {
	return &adminListEdgesTool{repo: repo}
}

func (t *adminListEdgesTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_list_edges",
		Desc: "Lists all edges in a schema. Edges connect agents: flow (sequential), transfer (hand-off), loop (cycle), spawn (parallel).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"schema_id": {Type: schema.Integer, Desc: "Schema ID", Required: true},
		}),
	}, nil
}

type listEdgesArgs struct {
	SchemaID uint `json:"schema_id"`
}

func (t *adminListEdgesTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args listEdgesArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.SchemaID == 0 {
		return "[ERROR] schema_id is required", nil
	}

	edges, err := t.repo.List(ctx, args.SchemaID)
	if err != nil {
		return fmt.Sprintf("[ERROR] Failed to list edges: %v", err), nil
	}

	if len(edges) == 0 {
		return fmt.Sprintf("No edges in schema %d.", args.SchemaID), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## %d edges in schema %d\n\n", len(edges), args.SchemaID))
	for _, e := range edges {
		label := ""
		if e.Label != "" {
			label = fmt.Sprintf(" [%s]", e.Label)
		}
		sb.WriteString(fmt.Sprintf("- id=%d: %s --%s--> %s%s\n", e.ID, e.FromAgent, e.Type, e.ToAgent, label))
	}
	return sb.String(), nil
}

// --- admin_create_edge ---

type adminCreateEdgeTool struct {
	repo     EdgeRepository
	reloader func()
}

func NewAdminCreateEdgeTool(repo EdgeRepository, reloader func()) tool.InvokableTool {
	return &adminCreateEdgeTool{repo: repo, reloader: reloader}
}

func (t *adminCreateEdgeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_create_edge",
		Desc: "Creates an edge between two agents in a schema. Types: flow (sequential), transfer (hand-off), loop (cycle back), spawn (parallel execution).",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"schema_id":  {Type: schema.Integer, Desc: "Schema ID", Required: true},
			"from_agent": {Type: schema.String, Desc: "Source agent name", Required: true},
			"to_agent":   {Type: schema.String, Desc: "Target agent name", Required: true},
			"type":       {Type: schema.String, Desc: "Edge type: flow, transfer, loop, or spawn", Required: true},
			"label":      {Type: schema.String, Desc: "Optional label for the edge", Required: false},
		}),
	}, nil
}

type createEdgeArgs struct {
	SchemaID  uint   `json:"schema_id"`
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Type      string `json:"type"`
	Label     string `json:"label"`
}

func (t *adminCreateEdgeTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args createEdgeArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.SchemaID == 0 {
		return "[ERROR] schema_id is required", nil
	}
	if args.FromAgent == "" {
		return "[ERROR] from_agent is required", nil
	}
	if args.ToAgent == "" {
		return "[ERROR] to_agent is required", nil
	}
	if args.Type == "" {
		return "[ERROR] type is required", nil
	}

	validTypes := map[string]bool{"flow": true, "transfer": true, "loop": true, "spawn": true}
	if !validTypes[args.Type] {
		return fmt.Sprintf("[ERROR] Invalid edge type %q. Must be: flow, transfer, loop, or spawn.", args.Type), nil
	}

	record := &EdgeRecord{
		SchemaID:  args.SchemaID,
		FromAgent: args.FromAgent,
		ToAgent:   args.ToAgent,
		Type:      args.Type,
		Label:     args.Label,
	}

	if err := t.repo.Create(ctx, record); err != nil {
		return fmt.Sprintf("[ERROR] Failed to create edge: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminCreateEdge] created", "schema_id", args.SchemaID, "from", args.FromAgent, "to", args.ToAgent, "type", args.Type)
	return fmt.Sprintf("Edge created (id=%d): %s --%s--> %s in schema %d.", record.ID, args.FromAgent, args.Type, args.ToAgent, args.SchemaID), nil
}

// --- admin_delete_edge ---

type adminDeleteEdgeTool struct {
	repo     EdgeRepository
	reloader func()
}

func NewAdminDeleteEdgeTool(repo EdgeRepository, reloader func()) tool.InvokableTool {
	return &adminDeleteEdgeTool{repo: repo, reloader: reloader}
}

func (t *adminDeleteEdgeTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "admin_delete_edge",
		Desc: "Deletes an edge by ID.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"edge_id": {Type: schema.Integer, Desc: "Edge ID to delete", Required: true},
		}),
	}, nil
}

type deleteEdgeArgs struct {
	EdgeID uint `json:"edge_id"`
}

func (t *adminDeleteEdgeTool) InvokableRun(ctx context.Context, argsJSON string, _ ...tool.Option) (string, error) {
	var args deleteEdgeArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return fmt.Sprintf("[ERROR] Invalid arguments: %v", err), nil
	}
	if args.EdgeID == 0 {
		return "[ERROR] edge_id is required", nil
	}

	if err := t.repo.Delete(ctx, args.EdgeID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return fmt.Sprintf("Edge not found: %d", args.EdgeID), nil
		}
		return fmt.Sprintf("[ERROR] Failed to delete edge: %v", err), nil
	}

	if t.reloader != nil {
		t.reloader()
	}

	slog.InfoContext(ctx, "[AdminDeleteEdge] deleted", "edge_id", args.EdgeID)
	return fmt.Sprintf("Edge %d deleted successfully.", args.EdgeID), nil
}
