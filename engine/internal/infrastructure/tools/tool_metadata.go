package tools

// SecurityZone classifies tools by security risk for admin dashboard.
type SecurityZone string

const (
	ZoneSafe      SecurityZone = "safe"      // No risk: ask_user, manage_tasks, memory, knowledge
	ZoneCaution   SecurityZone = "caution"   // Medium risk: MCP / unknown external tools
	ZoneDangerous SecurityZone = "dangerous" // High risk: self-hosted file system and shell (not available after parking)
)

// ToolMetadata describes a tool's name, purpose, and security classification.
type ToolMetadata struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	SecurityZone SecurityZone `json:"security_zone"`
	RiskWarning  string       `json:"risk_warning,omitempty"`
	Hint         string       `json:"hint,omitempty"`
	Companion    string       `json:"companion,omitempty"`
}

// toolMetadataRegistry holds metadata for all built-in tools currently
// exposed by the engine. Self-hosted file/shell/LSP tools were parked into
// bytebrew-archive; their metadata was removed along with the tool code.
var toolMetadataRegistry = map[string]ToolMetadata{
	// === SAFE ZONE — coordination and capability tools ===
	"ask_user": {
		Name:         "ask_user",
		Description:  "Asks the user a question and waits for their response. Used for clarification, confirmation, or gathering input during task execution.",
		SecurityZone: ZoneSafe,
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
	"spawn_agent": {
		Name:         "spawn_agent",
		Description:  "Spawns a specialized sub-agent (e.g. code-agent, reviewer) to handle a specific subtask. The sub-agent works independently and returns a summary when done.",
		SecurityZone: ZoneSafe,
	},
	"show_structured_output": {
		Name:         "show_structured_output",
		Description:  "Displays a structured output block (table, action buttons) to the user during a conversation.",
		SecurityZone: ZoneSafe,
	},
	"wait": {
		Name:         "wait",
		Description:  "Pauses the agent for a defined duration. Used for pacing, polling, or scheduled follow-ups.",
		SecurityZone: ZoneSafe,
	},
	"memory_recall": {
		Name:         "memory_recall",
		Description:  "Recalls stored memories for the current schema/user pair. Auto-injected when the Memory capability is enabled.",
		SecurityZone: ZoneSafe,
	},
	"memory_store": {
		Name:         "memory_store",
		Description:  "Stores a new memory entry for the current schema/user pair. Auto-injected when the Memory capability is enabled.",
		SecurityZone: ZoneSafe,
	},
	"knowledge_search": {
		Name:         "knowledge_search",
		Description:  "Searches the agent's knowledge base using semantic similarity. Finds relevant information from indexed documents (markdown, text) even when exact keywords don't match.",
		SecurityZone: ZoneSafe,
	},
}

// GetToolMetadata returns metadata for a tool by name.
// Returns a default caution-zone entry for unknown tools (MCP tools).
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
