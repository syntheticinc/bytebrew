package cloud

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandbox_CEMode_AllAllowed(t *testing.T) {
	sandbox := NewSandbox(false) // CE mode

	// All tools allowed in CE mode
	tools := []string{"read_file", "write_file", "execute_command", "grep_search", "knowledge_search", "ask_user"}
	for _, tool := range tools {
		err := sandbox.ValidateToolAccess(tool)
		assert.NoError(t, err, "tool %s should be allowed in CE mode", tool)
	}
}

func TestSandbox_CloudMode_Tier3Blocked(t *testing.T) {
	// AC-CLOUD-05: Cloud agents cannot use file/shell tools
	sandbox := NewSandbox(true)

	blockedTools := []string{"read_file", "write_file", "edit_file", "execute_command", "glob", "grep_search"}
	for _, tool := range blockedTools {
		err := sandbox.ValidateToolAccess(tool)
		require.Error(t, err, "tool %s should be blocked in Cloud mode", tool)

		var tbe *ToolBlockedError
		require.ErrorAs(t, err, &tbe)
		assert.Equal(t, tool, tbe.ToolName)
		assert.Contains(t, tbe.Reason, "blocked")
	}
}

func TestSandbox_CloudMode_OtherTiersAllowed(t *testing.T) {
	sandbox := NewSandbox(true)

	// Tier 1 (core) allowed
	assert.NoError(t, sandbox.ValidateToolAccess("ask_user"))
	assert.NoError(t, sandbox.ValidateToolAccess("manage_tasks"))
	assert.NoError(t, sandbox.ValidateToolAccess("show_structured_output"))

	// Tier 2 (capability) allowed
	assert.NoError(t, sandbox.ValidateToolAccess("memory_recall"))
	assert.NoError(t, sandbox.ValidateToolAccess("memory_store"))
	assert.NoError(t, sandbox.ValidateToolAccess("knowledge_search"))

	// Tier 4 (MCP) allowed
	assert.NoError(t, sandbox.ValidateToolAccess("tavily_search"))
	assert.NoError(t, sandbox.ValidateToolAccess("custom_mcp_tool"))
}

func TestSandbox_CloudMode_SpawnAllowed(t *testing.T) {
	sandbox := NewSandbox(true)

	// spawn_* tools are Tier 1
	assert.NoError(t, sandbox.ValidateToolAccess("spawn_code_agent"))
	assert.NoError(t, sandbox.ValidateToolAccess("spawn_researcher"))
}

func TestSandbox_FilterTools(t *testing.T) {
	sandbox := NewSandbox(true)

	tools := []string{"ask_user", "read_file", "memory_recall", "execute_command", "tavily_search"}
	allowed, blocked := sandbox.FilterTools(tools)

	assert.Equal(t, []string{"ask_user", "memory_recall", "tavily_search"}, allowed)
	assert.Len(t, blocked, 2) // read_file, execute_command
}
