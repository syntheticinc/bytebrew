package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeToolCall_NormalCall(t *testing.T) {
	name, args := sanitizeToolCall("manage_tasks", `{"action":"create","title":"Test"}`)
	assert.Equal(t, "manage_tasks", name)
	assert.Equal(t, `{"action":"create","title":"Test"}`, args)
}

func TestSanitizeToolCall_NoParentheses(t *testing.T) {
	name, args := sanitizeToolCall("read_file", `{"path":"main.go"}`)
	assert.Equal(t, "read_file", name)
	assert.Equal(t, `{"path":"main.go"}`, args)
}

func TestSanitizeToolCall_MalformedPythonStyle(t *testing.T) {
	malformedName := `manage_tasks(action=create, title="Mobile app", description="Build mobile app")`
	name, args := sanitizeToolCall(malformedName, "")

	assert.Equal(t, "manage_tasks", name)

	// Args should be valid JSON
	assert.JSONEq(t, `{"action":"create","title":"Mobile app","description":"Build mobile app"}`, args)
}

func TestSanitizeToolCall_MalformedWithArray(t *testing.T) {
	malformedName := `manage_tasks(action=create, title="Test", acceptance_criteria=["a","b","c"])`
	name, args := sanitizeToolCall(malformedName, "")

	assert.Equal(t, "manage_tasks", name)
	assert.Contains(t, args, `"action":"create"`)
	assert.Contains(t, args, `"acceptance_criteria"`)
}

func TestSanitizeToolCall_MalformedWithGarbage(t *testing.T) {
	// qwen-coder-next sometimes appends XML garbage after the closing paren
	malformedName := "manage_tasks(action=create, title=\"Test\")\n</function"
	name, args := sanitizeToolCall(malformedName, "")

	assert.Equal(t, "manage_tasks", name)
	assert.JSONEq(t, `{"action":"create","title":"Test"}`, args)
}

func TestSanitizeToolCall_MalformedNameButValidArgs(t *testing.T) {
	// Name has garbage suffix but arguments are valid JSON
	name, args := sanitizeToolCall("manage_tasks(foo)", `{"action":"create"}`)

	assert.Equal(t, "manage_tasks", name)
	assert.Equal(t, `{"action":"create"}`, args)
}

func TestSanitizeToolCall_EmptyBraces(t *testing.T) {
	malformedName := `read_file(path="internal/server.go")`
	name, args := sanitizeToolCall(malformedName, "{}")

	assert.Equal(t, "read_file", name)
	assert.JSONEq(t, `{"path":"internal/server.go"}`, args)
}

func TestSanitizeToolCall_CommaInQuotedValue(t *testing.T) {
	malformedName := `manage_tasks(action=create, description="Step 1, Step 2, Step 3")`
	name, args := sanitizeToolCall(malformedName, "")

	assert.Equal(t, "manage_tasks", name)
	assert.Contains(t, args, "Step 1, Step 2, Step 3")
}
