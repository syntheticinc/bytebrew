package tools

import "github.com/cloudwego/eino/components/tool"

// RegisterAllBuiltins registers factory functions for all builtin tools.
// Tools that require complex dependencies not available at registration time
// (e.g. AgentPool for spawn_code_agent) must be registered separately.
func RegisterAllBuiltins(store *BuiltinToolStore) {
	// File operations (proxied to client)
	store.Register("read_file", func(deps ToolDependencies) tool.InvokableTool {
		return NewReadFileTool(deps.Proxy, deps.SessionID)
	})
	store.Register("write_file", func(deps ToolDependencies) tool.InvokableTool {
		return NewWriteFileTool(deps.Proxy, deps.SessionID)
	})
	store.Register("edit_file", func(deps ToolDependencies) tool.InvokableTool {
		return NewEditFileTool(deps.Proxy, deps.SessionID)
	})

	// Search tools
	store.Register("search_code", func(deps ToolDependencies) tool.InvokableTool {
		return NewSearchCodeTool(deps.Proxy, deps.SessionID, deps.ProjectKey)
	})
	store.Register("grep_search", func(deps ToolDependencies) tool.InvokableTool {
		return NewGrepSearchTool(deps.Proxy, deps.SessionID)
	})
	store.Register("glob", func(deps ToolDependencies) tool.InvokableTool {
		return NewGlobTool(deps.Proxy, deps.SessionID)
	})
	store.Register("smart_search", func(deps ToolDependencies) tool.InvokableTool {
		return NewSmartSearchTool(deps.Proxy, deps.SessionID, deps.ProjectKey)
	})
	store.Register("get_project_tree", func(deps ToolDependencies) tool.InvokableTool {
		return NewGetProjectTreeTool(deps.Proxy, deps.SessionID, deps.ProjectKey)
	})

	// Command execution
	store.Register("execute_command", func(deps ToolDependencies) tool.InvokableTool {
		return NewExecuteCommandTool(deps.Proxy, deps.SessionID)
	})

	// Task management
	store.Register("manage_tasks", func(deps ToolDependencies) tool.InvokableTool {
		return NewManageTasksTool(deps.TaskManager, deps.Proxy, deps.SessionID)
	})
	store.Register("manage_subtasks", func(deps ToolDependencies) tool.InvokableTool {
		return NewManageSubtasksTool(deps.SubtaskManager, deps.SessionID)
	})

	// User interaction
	store.Register("ask_user", func(deps ToolDependencies) tool.InvokableTool {
		return NewAskUserTool(deps.Proxy, deps.SessionID)
	})

	// LSP
	store.Register("lsp", func(deps ToolDependencies) tool.InvokableTool {
		return NewLspTool(deps.Proxy, deps.SessionID)
	})

	// Indexing-based symbol tools
	store.Register("get_function", func(deps ToolDependencies) tool.InvokableTool {
		return NewGetFunctionTool(deps.ChunkStore, deps.Embedder)
	})
	store.Register("get_class", func(deps ToolDependencies) tool.InvokableTool {
		return NewGetClassTool(deps.ChunkStore, deps.Embedder)
	})
	store.Register("get_file_structure", func(deps ToolDependencies) tool.InvokableTool {
		return NewGetFileStructureTool(deps.ChunkStore, deps.ProjectRoot)
	})

	// Engine task management (Phase 4: replaces old manage_tasks for DB-backed tasks)
	store.Register("engine_manage_tasks", func(deps ToolDependencies) tool.InvokableTool {
		if deps.EngineTaskManager == nil {
			return nil // not available without PostgreSQL
		}
		return NewEngineManageTasksTool(deps.EngineTaskManager, deps.SessionID)
	})

	// Web tools (pre-created instances passed via deps)
	store.Register("web_search", func(deps ToolDependencies) tool.InvokableTool {
		return deps.WebSearchTool
	})
	store.Register("web_fetch", func(deps ToolDependencies) tool.InvokableTool {
		return deps.WebFetchTool
	})

	// spawn_code_agent — not registered here.
	// Requires AgentPool which is created after tool store initialization.
	// Register separately: store.Register("spawn_code_agent", ...)
}
