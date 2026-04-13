package tools

import "github.com/syntheticinc/bytebrew/engine/internal/domain"

// DefaultToolClassifier implements domain.ToolClassifier
// It classifies tools as proxied (client-side) or server-side based on their names
type DefaultToolClassifier struct {
	proxiedTools    map[string]bool
	serverSideTools map[string]bool
}

// NewToolClassifier creates a new DefaultToolClassifier with predefined tool classifications
func NewToolClassifier() *DefaultToolClassifier {
	return &DefaultToolClassifier{
		proxiedTools: map[string]bool{
			"read_file":        true,
			"write_file":       true,
			"edit_file":        true,
			"search_code":      true,
			"get_project_tree": true,
			"execute_command":  true,
			"ask_user":         true, // Uses proxy.AskUserQuestionnaire() → executeToolCallCore → stream TOOL_CALL
			"smart_search":     true, // Uses proxy.ExecuteSubQueries() → stream TOOL_CALL with subQueries
			"grep_search":      true, // Uses proxy.GrepSearch() → executeToolCall → stream TOOL_CALL
			"glob":             true, // Uses proxy.Glob() → executeToolCall → stream TOOL_CALL
			"lsp":              true, // Uses proxy → client LSP servers
		},
		serverSideTools: map[string]bool{
			"web_search":       true,
			"web_fetch":        true,
			"manage_tasks":     true,
			"manage_subtasks":  true,
			"spawn_code_agent": true,
		},
	}
}

// ClassifyTool returns the type of the given tool
func (c *DefaultToolClassifier) ClassifyTool(toolName string) domain.ToolType {
	if c.proxiedTools[toolName] {
		return domain.ToolTypeProxied
	}
	if c.serverSideTools[toolName] {
		return domain.ToolTypeServerSide
	}
	// Default to server-side for unknown tools
	return domain.ToolTypeServerSide
}

// IsProxied returns true if the tool is executed on the client side
func (c *DefaultToolClassifier) IsProxied(toolName string) bool {
	return c.ClassifyTool(toolName) == domain.ToolTypeProxied
}

// IsServerSide returns true if the tool is executed on the server side
func (c *DefaultToolClassifier) IsServerSide(toolName string) bool {
	return c.ClassifyTool(toolName) == domain.ToolTypeServerSide
}
