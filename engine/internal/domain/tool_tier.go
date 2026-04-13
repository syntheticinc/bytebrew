package domain

// ToolTier represents the tier classification of a tool.
type ToolTier int

const (
	// ToolTierCore (Tier 1) — always available: ask_user, show_structured_output, spawn_*, wait
	ToolTierCore ToolTier = 1

	// ToolTierCapability (Tier 2) — auto-injected by capabilities: memory_recall, memory_store, knowledge_search, escalate
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
		"wait",
	}
}

// CapabilityToolNames returns the Tier 2 tool names injected by capabilities.
func CapabilityToolNames() []string {
	return []string{
		"memory_recall",
		"memory_store",
		"knowledge_search",
		"escalate",
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
	if len(toolName) > 6 && toolName[:6] == "spawn_" {
		return ToolTierCore
	}
	return ToolTierMCP
}
