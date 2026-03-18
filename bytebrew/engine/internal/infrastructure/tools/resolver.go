package tools

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/indexing"
	"github.com/cloudwego/eino/components/tool"
)

// ToolDependencies holds all dependencies needed by tools
type ToolDependencies struct {
	SessionID          string
	ProjectKey         string
	ProjectRoot        string
	Proxy              ClientOperationsProxy
	TaskManager        TaskManager
	SubtaskManager     SubtaskManager
	AgentPool          AgentPoolForTool
	EngineTaskManager  EngineTaskManager  // Phase 4: engine task CRUD
	WebSearchTool      tool.InvokableTool // pre-created (depends on API key)
	WebFetchTool       tool.InvokableTool // pre-created
	ChunkStore         *indexing.ChunkStore
	Embedder           *indexing.EmbeddingsClient
}

// DefaultToolResolver resolves tool names to tool instances
type DefaultToolResolver struct{}

// NewDefaultToolResolver creates a new resolver
func NewDefaultToolResolver() *DefaultToolResolver {
	return &DefaultToolResolver{}
}

// Resolve creates tool instances by name
func (r *DefaultToolResolver) Resolve(ctx context.Context, toolNames []string, deps ToolDependencies) ([]tool.InvokableTool, error) {
	var resolved []tool.InvokableTool

	for _, name := range toolNames {
		t, err := r.resolveOne(ctx, name, deps)
		if err != nil {
			return nil, fmt.Errorf("resolve tool %s: %w", name, err)
		}
		if t != nil { // nil = optional tool not available
			riskLevel := GetContentRiskLevel(name)
			t = NewSafeToolWrapper(t, name, riskLevel)
			t = NewCancellableToolWrapper(t)
			resolved = append(resolved, t)
		}
	}

	return resolved, nil
}

func (r *DefaultToolResolver) resolveOne(ctx context.Context, name string, deps ToolDependencies) (tool.InvokableTool, error) {
	switch name {
	case "read_file":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewReadFileTool(deps.Proxy, deps.SessionID), nil
	case "write_file":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewWriteFileTool(deps.Proxy, deps.SessionID), nil
	case "edit_file":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewEditFileTool(deps.Proxy, deps.SessionID), nil
	case "search_code":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewSearchCodeTool(deps.Proxy, deps.SessionID, deps.ProjectKey), nil
	case "grep_search":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewGrepSearchTool(deps.Proxy, deps.SessionID), nil
	case "glob":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewGlobTool(deps.Proxy, deps.SessionID), nil
	case "smart_search":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewSmartSearchTool(deps.Proxy, deps.SessionID, deps.ProjectKey), nil
	case "get_project_tree":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewGetProjectTreeTool(deps.Proxy, deps.SessionID, deps.ProjectKey), nil
	case "execute_command":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewExecuteCommandTool(deps.Proxy, deps.SessionID), nil
	case "web_search":
		return deps.WebSearchTool, nil // nil if not configured
	case "web_fetch":
		return deps.WebFetchTool, nil // nil if not configured
	case "manage_tasks":
		if deps.TaskManager == nil || deps.Proxy == nil {
			return nil, nil // optional
		}
		return NewManageTasksTool(deps.TaskManager, deps.Proxy, deps.SessionID), nil
	case "manage_subtasks":
		if deps.SubtaskManager == nil {
			return nil, nil
		}
		return NewManageSubtasksTool(deps.SubtaskManager, deps.SessionID), nil
	case "spawn_code_agent":
		if deps.AgentPool == nil {
			return nil, nil
		}
		return NewSpawnCodeAgentTool(deps.AgentPool, deps.SessionID, deps.ProjectKey), nil
	case "ask_user":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewAskUserTool(deps.Proxy, deps.SessionID), nil
	case "lsp":
		if deps.Proxy == nil {
			return nil, nil
		}
		return NewLspTool(deps.Proxy, deps.SessionID), nil
	case "get_function":
		if deps.ChunkStore == nil {
			return nil, nil
		}
		return NewGetFunctionTool(deps.ChunkStore, deps.Embedder), nil
	case "get_class":
		if deps.ChunkStore == nil {
			return nil, nil
		}
		return NewGetClassTool(deps.ChunkStore, deps.Embedder), nil
	case "get_file_structure":
		if deps.ChunkStore == nil {
			return nil, nil
		}
		return NewGetFileStructureTool(deps.ChunkStore, deps.ProjectRoot), nil
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}
