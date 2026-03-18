package tools

// SecurityZone classifies tools by security risk for admin dashboard.
type SecurityZone string

const (
	ZoneSafe      SecurityZone = "safe"      // No risk: ask_user, web_search, manage_tasks
	ZoneCaution   SecurityZone = "caution"   // Medium risk: web_fetch (external content)
	ZoneDangerous SecurityZone = "dangerous" // High risk: file system + command execution
)

// ToolMetadata describes a tool's name, purpose, and security classification.
type ToolMetadata struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	SecurityZone SecurityZone `json:"security_zone"`
	RiskWarning  string       `json:"risk_warning,omitempty"`
}

// toolMetadataRegistry holds metadata for all known built-in tools.
var toolMetadataRegistry = map[string]ToolMetadata{
	// === SAFE ZONE (green) ===
	"ask_user": {
		Name:         "ask_user",
		Description:  "Ask the user a question and wait for response",
		SecurityZone: ZoneSafe,
	},
	"web_search": {
		Name:         "web_search",
		Description:  "Search the web using Tavily API",
		SecurityZone: ZoneSafe,
	},
	"manage_tasks": {
		Name:         "manage_tasks",
		Description:  "Create, update, and track tasks",
		SecurityZone: ZoneSafe,
	},
	"manage_subtasks": {
		Name:         "manage_subtasks",
		Description:  "Manage subtasks within a parent task",
		SecurityZone: ZoneSafe,
	},
	"engine_manage_tasks": {
		Name:         "engine_manage_tasks",
		Description:  "Manage tasks in PostgreSQL (DB-backed)",
		SecurityZone: ZoneSafe,
	},
	"spawn_code_agent": {
		Name:         "spawn_code_agent",
		Description:  "Spawn a sub-agent to handle a subtask",
		SecurityZone: ZoneSafe,
	},

	// === CAUTION ZONE (yellow) ===
	"web_fetch": {
		Name:         "web_fetch",
		Description:  "Fetch content from a URL (external data)",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Fetches external content that could contain prompt injections",
	},
	"lsp": {
		Name:         "lsp",
		Description:  "Language Server Protocol diagnostics and code intelligence",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Interacts with language server processes on the host",
	},
	"glob": {
		Name:         "glob",
		Description:  "Find files by pattern (e.g. **/*.go)",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Can enumerate files on the server filesystem",
	},
	"grep_search": {
		Name:         "grep_search",
		Description:  "Search file contents with regex patterns",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Can read file contents matching patterns",
	},
	"get_project_tree": {
		Name:         "get_project_tree",
		Description:  "Show directory tree structure",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Exposes server directory structure",
	},
	"smart_search": {
		Name:         "smart_search",
		Description:  "Semantic code search using embeddings",
		SecurityZone: ZoneCaution,
	},
	"search_code": {
		Name:         "search_code",
		Description:  "Search code by symbol name",
		SecurityZone: ZoneCaution,
	},
	"get_function": {
		Name:         "get_function",
		Description:  "Get function source code by name",
		SecurityZone: ZoneCaution,
	},
	"get_class": {
		Name:         "get_class",
		Description:  "Get class/struct definition by name",
		SecurityZone: ZoneCaution,
	},
	"get_file_structure": {
		Name:         "get_file_structure",
		Description:  "Show symbols in a file (functions, classes, etc.)",
		SecurityZone: ZoneCaution,
	},

	// === DANGEROUS ZONE (red) — disabled by default ===
	"read_file": {
		Name:         "read_file",
		Description:  "Read file contents from the server filesystem",
		SecurityZone: ZoneDangerous,
		RiskWarning:  "FILESYSTEM ACCESS: Agent can read ANY file accessible to the server process. May expose secrets, configs, credentials. Enable only for trusted coding agents.",
	},
	"write_file": {
		Name:         "write_file",
		Description:  "Create or overwrite files on the server filesystem",
		SecurityZone: ZoneDangerous,
		RiskWarning:  "FILESYSTEM WRITE: Agent can create/overwrite ANY file. Could modify configs, inject code, or corrupt data. Enable only for trusted coding agents with restricted project paths.",
	},
	"edit_file": {
		Name:         "edit_file",
		Description:  "Edit existing files with search-and-replace",
		SecurityZone: ZoneDangerous,
		RiskWarning:  "FILESYSTEM WRITE: Agent can modify ANY file contents. Same risks as write_file. Enable only for trusted coding agents.",
	},
	"execute_command": {
		Name:         "execute_command",
		Description:  "Execute shell commands on the server",
		SecurityZone: ZoneDangerous,
		RiskWarning:  "CRITICAL: Agent can execute ARBITRARY shell commands with server process permissions. Can install software, modify system, access network, delete data. NEVER enable for user-facing agents. Only for fully trusted development agents in isolated environments.",
	},
}

// GetToolMetadata returns metadata for a tool by name.
// Returns a default caution-zone entry for unknown tools.
func GetToolMetadata(name string) ToolMetadata {
	if meta, ok := toolMetadataRegistry[name]; ok {
		return meta
	}
	return ToolMetadata{
		Name:         name,
		Description:  "Custom tool",
		SecurityZone: ZoneCaution,
	}
}

// GetAllToolMetadata returns metadata for all known built-in tools.
func GetAllToolMetadata() []ToolMetadata {
	result := make([]ToolMetadata, 0, len(toolMetadataRegistry))
	for _, meta := range toolMetadataRegistry {
		result = append(result, meta)
	}
	return result
}
