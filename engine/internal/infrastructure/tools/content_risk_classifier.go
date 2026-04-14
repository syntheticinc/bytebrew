package tools

// ContentRiskLevel represents the risk level of content returned by a tool
type ContentRiskLevel int

const (
	RiskNone     ContentRiskLevel = iota // Internal tools (manage_plan, write_file) — no wrapping
	RiskLow                              // Structural tools (glob, get_project_tree) — light prefix
	RiskHigh                             // File/search tools (read_file, grep_search) — content markers
	RiskCritical                         // External content (execute_command, MCP web tools) — strong markers
)

// GetContentRiskLevel returns the risk level for a given tool name
func GetContentRiskLevel(toolName string) ContentRiskLevel {
	switch toolName {
	// Critical: shell execution. External web content comes via MCP tools and is classified
	// by MCP-specific rules elsewhere; unknown tools default to RiskHigh below.
	case "execute_command":
		return RiskCritical
	// High: project content that could contain injections
	case "read_file", "grep_search", "smart_search", "search_code",
		"get_function", "get_class":
		return RiskHigh
	// Low: structural/metadata tools
	case "glob", "get_project_tree", "lsp", "get_file_structure":
		return RiskLow
	// None: internal tools that don't return untrusted content
	case "manage_tasks", "manage_subtasks", "spawn_agent",
		"write_file", "edit_file", "ask_user":
		return RiskNone
	default:
		// Unknown tools default to high risk
		return RiskHigh
	}
}
