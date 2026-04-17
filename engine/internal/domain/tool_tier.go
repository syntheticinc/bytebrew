package domain

import "strings"

// ToolTier represents the tier classification of a tool.
type ToolTier int

const (
	// ToolTierCore (Tier 1) — always available: ask_user, show_structured_output, spawn_*, manage_tasks, wait
	ToolTierCore ToolTier = 1

	// ToolTierCapability (Tier 2) — auto-injected by capabilities: memory_recall, memory_store, knowledge_search
	ToolTierCapability ToolTier = 2

	// ToolTierSelfHosted (Tier 3) — CE only, blocked in Cloud: read_file, write_file, execute_command, etc.
	ToolTierSelfHosted ToolTier = 3

	// ToolTierMCP (Tier 4) — from connected MCP servers: web_search, external APIs, etc.
	ToolTierMCP ToolTier = 4
)

// CoreToolNames returns the Tier 1 tool names that are always available.
func CoreToolNames() []string {
	return []string{
		"ask_user",
		"show_structured_output",
		"manage_tasks",
		"manage_subtasks",
		"wait",
	}
}

// CapabilityToolNames returns the Tier 2 tool names injected by capabilities.
func CapabilityToolNames() []string {
	return []string{
		"memory_recall",
		"memory_store",
		"knowledge_search",
	}
}

// SelfHostedToolNames returns the Tier 3 tool names blocked in Cloud.
func SelfHostedToolNames() []string {
	return []string{
		"read_file",
		"write_file",
		"edit_file",
		"glob",
		"grep_search",
		"search_code",
		"smart_search",
		"get_project_tree",
		"get_function",
		"get_class",
		"get_file_structure",
		"lsp",
		"execute_command",
	}
}

// ClassifyToolTier returns the tier for a given tool name.
func ClassifyToolTier(toolName string) ToolTier {
	for _, name := range CoreToolNames() {
		if name == toolName {
			return ToolTierCore
		}
	}
	for _, name := range CapabilityToolNames() {
		if name == toolName {
			return ToolTierCapability
		}
	}
	for _, name := range SelfHostedToolNames() {
		if name == toolName {
			return ToolTierSelfHosted
		}
	}
	// spawn_* tools are also Tier 1
	if strings.HasPrefix(toolName, "spawn_") {
		return ToolTierCore
	}
	// admin_* tools — orchestration over other platform objects. Treated as
	// self-hosted so that Cloud tenants don't grant arbitrary admin tool use
	// to agents through the default MCP fallthrough. Admin HTTP layer still
	// rejects these names at agent create/update time; this extra guard keeps
	// seed agents / runtime-built tool lists honest.
	if strings.HasPrefix(toolName, "admin_") {
		return ToolTierSelfHosted
	}
	return ToolTierMCP
}
