package tools

import "github.com/cloudwego/eino/components/tool"

// RegisterAllBuiltins registers factory functions for all builtin tools.
// Tools that require complex dependencies not available at registration time
// (e.g. AgentPool for spawn_agent) must be registered separately.
func RegisterAllBuiltins(store *BuiltinToolStore) {
	// Unified task management — EngineTask-based, DB-backed, Admin-visible.
	// Subtasks are EngineTask with ParentTaskID set (single entity, no separate manage_subtasks).
	store.Register("manage_tasks", func(deps ToolDependencies) tool.InvokableTool {
		if deps.EngineTaskManager == nil {
			return nil
		}
		return NewEngineManageTasksTool(deps.EngineTaskManager, deps.SessionID)
	})

	// Structured output — display rich data blocks (tables, action buttons) to the user
	// and collect non-blocking form input (output_type=form). Replaces the legacy ask_user tool.
	store.Register("show_structured_output", func(deps ToolDependencies) tool.InvokableTool {
		return NewStructuredOutputTool(deps.EventEmitter, deps.SessionID)
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
