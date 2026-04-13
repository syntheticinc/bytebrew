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
	Hint         string       `json:"hint,omitempty"`         // contextual hint shown when tool is selected
	Companion    string       `json:"companion,omitempty"`    // suggested companion tool name
}

// toolMetadataRegistry holds metadata for all known built-in tools.
var toolMetadataRegistry = map[string]ToolMetadata{
	// === SAFE ZONE ===
	"ask_user": {
		Name:         "ask_user",
		Description:  "Asks the user a question and waits for their response. Used for clarification, confirmation, or gathering input during task execution.",
		SecurityZone: ZoneSafe,
	},
	"web_search": {
		Name:         "web_search",
		Description:  "Performs web search using Tavily API and returns relevant results with snippets. Useful for finding documentation, solutions, and current information.",
		SecurityZone: ZoneSafe,
		Hint:         "Returns search snippets only. Consider enabling web_fetch (Caution zone) to allow the agent to read full page content from search results.",
		Companion:    "web_fetch",
	},
	"manage_tasks": {
		Name:         "manage_tasks",
		Description:  "Creates, updates, and manages work tasks. The agent uses this to plan work, track progress, and organize subtasks for delegation to other agents.",
		SecurityZone: ZoneSafe,
	},
	"manage_subtasks": {
		Name:         "manage_subtasks",
		Description:  "Manages subtasks within a parent task — create, update status, mark complete. Used for granular progress tracking.",
		SecurityZone: ZoneSafe,
	},
	"engine_manage_tasks": {
		Name:         "engine_manage_tasks",
		Description:  "Manages tasks stored in PostgreSQL. Provides persistent, DB-backed task tracking that survives server restarts.",
		SecurityZone: ZoneSafe,
	},
	"spawn_agent": {
		Name:         "spawn_agent",
		Description:  "Spawns a specialized sub-agent (e.g. code-agent, reviewer) to handle a specific subtask. The sub-agent works independently and returns a summary when done.",
		SecurityZone: ZoneSafe,
	},

	// === CAUTION ZONE ===
	"web_fetch": {
		Name:         "web_fetch",
		Description:  "Fetches the content of a URL and returns the response body. Used to retrieve API responses, documentation pages, or external data.",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Fetches external content that could contain prompt injection attacks. The returned content is processed by the LLM and could influence agent behavior.",
	},
	"lsp": {
		Name:         "lsp",
		Description:  "Interacts with Language Server Protocol servers for code intelligence — diagnostics, hover info, go-to-definition, completions. Provides real-time code analysis.",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Starts and communicates with LSP server processes on the host machine.",
	},
	"glob": {
		Name:         "glob",
		Description:  "Finds files matching a glob pattern (e.g. **/*.go, src/**/*.ts). Returns a list of matching file paths. Used for discovering project structure.",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Can enumerate file paths on the server, revealing directory structure and file names.",
	},
	"grep_search": {
		Name:         "grep_search",
		Description:  "Searches file contents using regex patterns across the project. Returns matching lines with file paths and line numbers.",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Can read portions of file contents matching the pattern. May expose sensitive data if patterns match credential files.",
	},
	"get_project_tree": {
		Name:         "get_project_tree",
		Description:  "Shows the directory tree structure of the project, listing files and folders up to a configurable depth.",
		SecurityZone: ZoneCaution,
		RiskWarning:  "Exposes the server's directory structure which may reveal sensitive paths or project layout.",
	},
	"smart_search": {
		Name:         "smart_search",
		Description:  "Performs semantic code search using embeddings. Finds conceptually related code even when exact keywords don't match.",
		SecurityZone: ZoneCaution,
	},
	"knowledge_search": {
		Name:         "knowledge_search",
		Description:  "Searches the agent's knowledge base using semantic similarity. Finds relevant information from indexed documents (markdown, text) even when exact keywords don't match.",
		SecurityZone: ZoneSafe,
	},
	"search_code": {
		Name:         "search_code",
		Description:  "Searches code by symbol name across the project. Finds function, class, and variable definitions by name or pattern.",
		SecurityZone: ZoneCaution,
	},
	"get_function": {
		Name:         "get_function",
		Description:  "Retrieves the full source code of a function by name. Includes the function signature, body, and surrounding context.",
		SecurityZone: ZoneCaution,
	},
	"get_class": {
		Name:         "get_class",
		Description:  "Retrieves a class or struct definition by name, including fields, methods, and embedded types.",
		SecurityZone: ZoneCaution,
	},
	"get_file_structure": {
		Name:         "get_file_structure",
		Description:  "Shows all symbols defined in a file — functions, classes, interfaces, constants. Provides a quick overview of file contents without reading the full source.",
		SecurityZone: ZoneCaution,
	},

	// === DANGEROUS ZONE — disabled by default ===
	"read_file": {
		Name:         "read_file",
		Description:  "Reads the full contents of a file from the server filesystem. The agent can access any file readable by the server process, including configs, source code, and potentially sensitive files like .env or credentials.",
		SecurityZone: ZoneDangerous,
		RiskWarning:  "FILESYSTEM ACCESS: The agent can read ANY file accessible to the server process. This includes environment variables, database credentials, API keys, and other secrets. Enable only for trusted coding agents operating on their own project files.",
	},
	"write_file": {
		Name:         "write_file",
		Description:  "Creates a new file or completely overwrites an existing file on the server filesystem. The agent can write to any path writable by the server process.",
		SecurityZone: ZoneDangerous,
		RiskWarning:  "FILESYSTEM WRITE: The agent can create or overwrite ANY writable file. This could modify application configs, inject malicious code, overwrite backups, or corrupt data. Enable only for trusted coding agents with a restricted working directory.",
	},
	"edit_file": {
		Name:         "edit_file",
		Description:  "Modifies an existing file using search-and-replace operations. More precise than write_file — changes specific parts of a file rather than overwriting entirely.",
		SecurityZone: ZoneDangerous,
		RiskWarning:  "FILESYSTEM MODIFY: Same risks as write_file. The agent can alter any writable file's contents. Enable only for trusted coding agents.",
	},
	"execute_command": {
		Name:         "execute_command",
		Description:  "Executes arbitrary shell commands on the server with the permissions of the server process. Can run any program, script, or system command available on the host.",
		SecurityZone: ZoneDangerous,
		RiskWarning:  "CRITICAL SECURITY RISK: This tool grants the agent full shell access. The agent can install packages, modify system files, access the network, read/write any file, start/stop services, and execute destructive commands. A malicious or misled agent prompt could compromise the entire server. NEVER enable for user-facing agents. Only for fully trusted development agents running in isolated/sandboxed environments.",
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
