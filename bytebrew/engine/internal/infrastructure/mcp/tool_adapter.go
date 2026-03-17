package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// AdaptMCPTool converts an MCP tool to Eino InvokableTool.
func AdaptMCPTool(client *Client, mcpTool MCPTool) tool.InvokableTool {
	return &mcpToolAdapter{client: client, mcpTool: mcpTool}
}

type mcpToolAdapter struct {
	client  *Client
	mcpTool MCPTool
}

func (a *mcpToolAdapter) Info(_ context.Context) (*schema.ToolInfo, error) {
	params := parseJSONSchemaToParams(a.mcpTool.InputSchema)
	return &schema.ToolInfo{
		Name:        a.mcpTool.Name,
		Desc:        a.mcpTool.Description,
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}, nil
}

func (a *mcpToolAdapter) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
		return "", fmt.Errorf("parse args: %w", err)
	}
	return a.client.CallTool(ctx, a.mcpTool.Name, args)
}

// parseJSONSchemaToParams converts JSON Schema to Eino params.
// Handles top-level properties only.
func parseJSONSchemaToParams(schemaJSON json.RawMessage) map[string]*schema.ParameterInfo {
	if len(schemaJSON) == 0 {
		return nil
	}

	var s struct {
		Properties map[string]struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(schemaJSON, &s); err != nil {
		return nil
	}

	requiredSet := make(map[string]bool, len(s.Required))
	for _, r := range s.Required {
		requiredSet[r] = true
	}

	params := make(map[string]*schema.ParameterInfo, len(s.Properties))
	for name, prop := range s.Properties {
		params[name] = &schema.ParameterInfo{
			Type:     schema.DataType(prop.Type),
			Desc:     prop.Description,
			Required: requiredSet[name],
		}
	}
	return params
}
