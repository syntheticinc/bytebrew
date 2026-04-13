//go:build prompt

package prompt_regression

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// supervisorToolNames lists all supervisor tools from flows.yaml
// Excludes web_search and web_fetch (require external dependencies)
var supervisorToolNames = []string{
	"read_file",
	"write_file",
	"edit_file",
	"grep_search",
	"glob",
	"lsp",
	"get_project_tree",
	"execute_command",
	"manage_tasks",
	"manage_subtasks",
	"spawn_agent",
	"ask_user",
}

// getToolSchemas returns tool schemas for given tool names
func getToolSchemas(ctx context.Context, names []string) ([]*schema.ToolInfo, error) {
	schemas := make([]*schema.ToolInfo, 0, len(names))

	for _, name := range names {
		toolInstance := createToolForSchema(name)
		if toolInstance == nil {
			return nil, fmt.Errorf("unknown tool: %s", name)
		}

		info, err := toolInstance.Info(ctx)
		if err != nil {
			return nil, fmt.Errorf("get info for tool %s: %w", name, err)
		}

		schemas = append(schemas, info)
	}

	return schemas, nil
}

// createToolForSchema creates a tool instance for schema extraction
// Tools are created with nil dependencies (safe for Info() calls)
func createToolForSchema(name string) tool.InvokableTool {
	switch name {
	case "read_file":
		return tools.NewReadFileTool(nil, "")
	case "write_file":
		return tools.NewWriteFileTool(nil, "")
	case "edit_file":
		return tools.NewEditFileTool(nil, "")
	case "grep_search":
		return tools.NewGrepSearchTool(nil, "")
	case "glob":
		return tools.NewGlobTool(nil, "")
	case "lsp":
		return tools.NewLspTool(nil, "")
	case "get_project_tree":
		return tools.NewGetProjectTreeTool(nil, "", "")
	case "execute_command":
		return tools.NewExecuteCommandTool(nil, "")
	case "manage_tasks":
		return tools.NewManageTasksTool(nil, nil, "")
	case "manage_subtasks":
		return tools.NewManageSubtasksTool(nil, "")
	case "spawn_agent":
		return tools.NewSpawnAgentTool(nil, "", "")
	case "ask_user":
		return tools.NewAskUserTool(nil, "")
	default:
		return nil
	}
}
