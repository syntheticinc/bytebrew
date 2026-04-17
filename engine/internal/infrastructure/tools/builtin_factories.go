package tools

import "github.com/cloudwego/eino/components/tool"

// RegisterAllBuiltins registers factory functions for all builtin tools.
// Tools that require complex dependencies not available at registration time
// (e.g. AgentPool for spawn_agent) must be registered separately.
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

	// Unified task management — EngineTask-based, DB-backed, Admin-visible.
	// Subtasks are EngineTask with ParentTaskID set (single entity, no separate manage_subtasks).
	store.Register("manage_tasks", func(deps ToolDependencies) tool.InvokableTool {
		if deps.EngineTaskManager == nil {
			return nil
		}
		return NewEngineManageTasksTool(deps.EngineTaskManager, deps.SessionID)
	})

	// User interaction — disabled in background mode (cron/webhook tasks have no user)
	store.Register("ask_user", func(deps ToolDependencies) tool.InvokableTool {
		if deps.BackgroundMode {
			return nil // tool not available in background mode
		}
		return NewAskUserTool(deps.Proxy, deps.SessionID)
	})

	// Structured output — display rich data blocks (tables, action buttons) to the user
	store.Register("show_structured_output", func(deps ToolDependencies) tool.InvokableTool {
		return NewStructuredOutputTool(deps.EventEmitter, deps.SessionID)
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

	// Web search is available only via MCP servers (Tavily, Brave, Exa, etc.) — attach via Admin UI.

	// Memory capability tools (US-001: auto-injected by capability injector when agent has Memory)
	store.Register("memory_recall", func(deps ToolDependencies) tool.InvokableTool {
		if deps.MemoryRecaller == nil || deps.SchemaID == "" {
			return nil // disabled when no storage or schema context
		}
		return NewMemoryRecallTool(deps.SchemaID, deps.UserID, deps.MemoryRecaller)
	})
	store.Register("memory_store", func(deps ToolDependencies) tool.InvokableTool {
		if deps.MemoryStorer == nil || deps.SchemaID == "" {
			return nil // disabled when no storage or schema context
		}
		return NewMemoryStoreTool(deps.SchemaID, deps.UserID, deps.MemoryStorer, deps.MemoryMaxEntries)
	})

	// spawn_agent — not registered here.
	// Requires AgentPool which is created after tool store initialization.
	// Register separately: store.Register("spawn_agent", ...)
}
