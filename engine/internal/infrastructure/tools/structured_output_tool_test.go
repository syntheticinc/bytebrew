package tools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// mockEventEmitter records emitted events for testing
type mockEventEmitter struct {
	events []*domain.AgentEvent
}

func (m *mockEventEmitter) Send(event *domain.AgentEvent) error {
	m.events = append(m.events, event)
	return nil
}

func TestStructuredOutput_SummaryTable(t *testing.T) {
	emitter := &mockEventEmitter{}
	tool := NewStructuredOutputTool(emitter, "sess-1")

	args := `{
		"output_type": "summary_table",
		"title": "Project Overview",
		"description": "Current project configuration",
		"rows": [
			{"label": "Name", "value": "MyProject"},
			{"label": "Language", "value": "Go"},
			{"label": "Version", "value": "1.24"}
		]
	}`

	result, err := tool.InvokableRun(context.Background(), args)
	require.NoError(t, err)
	assert.Equal(t, "Structured output displayed to user.", result)

	// Verify event was emitted
	require.Len(t, emitter.events, 1)
	event := emitter.events[0]
	assert.Equal(t, domain.EventTypeStructuredOutput, event.Type)

	// Verify event content is valid StructuredOutput JSON
	var output domain.StructuredOutput
	err = json.Unmarshal([]byte(event.Content), &output)
	require.NoError(t, err)
	assert.Equal(t, "summary_table", output.OutputType)
	assert.Equal(t, "Project Overview", output.Title)
	assert.Equal(t, "Current project configuration", output.Description)
	require.Len(t, output.Rows, 3)
	assert.Equal(t, "Name", output.Rows[0].Label)
	assert.Equal(t, "MyProject", output.Rows[0].Value)
	assert.Equal(t, "Language", output.Rows[1].Label)
	assert.Equal(t, "Go", output.Rows[1].Value)
	assert.Equal(t, "Version", output.Rows[2].Label)
	assert.Equal(t, "1.24", output.Rows[2].Value)
}

func TestStructuredOutput_WithActions(t *testing.T) {
	emitter := &mockEventEmitter{}
	tool := NewStructuredOutputTool(emitter, "sess-1")

	args := `{
		"output_type": "summary_table",
		"title": "Deployment Ready",
		"rows": [{"label": "Status", "value": "Ready"}],
		"actions": [
			{"label": "Deploy Now", "type": "primary", "value": "deploy"},
			{"label": "Cancel", "type": "secondary", "value": "cancel"}
		]
	}`

	result, err := tool.InvokableRun(context.Background(), args)
	require.NoError(t, err)
	assert.Equal(t, "Structured output displayed to user.", result)

	require.Len(t, emitter.events, 1)
	event := emitter.events[0]

	var output domain.StructuredOutput
	err = json.Unmarshal([]byte(event.Content), &output)
	require.NoError(t, err)
	assert.Equal(t, "summary_table", output.OutputType)
	assert.Equal(t, "Deployment Ready", output.Title)
	require.Len(t, output.Actions, 2)
	assert.Equal(t, "Deploy Now", output.Actions[0].Label)
	assert.Equal(t, "primary", output.Actions[0].Type)
	assert.Equal(t, "deploy", output.Actions[0].Value)
	assert.Equal(t, "Cancel", output.Actions[1].Label)
	assert.Equal(t, "secondary", output.Actions[1].Type)
	assert.Equal(t, "cancel", output.Actions[1].Value)
}

func TestStructuredOutput_MissingOutputType(t *testing.T) {
	emitter := &mockEventEmitter{}
	tool := NewStructuredOutputTool(emitter, "sess-1")

	args := `{"title": "No Type"}`
	result, err := tool.InvokableRun(context.Background(), args)
	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "output_type is required")
	assert.Empty(t, emitter.events)
}

func TestStructuredOutput_InvalidJSON(t *testing.T) {
	emitter := &mockEventEmitter{}
	tool := NewStructuredOutputTool(emitter, "sess-1")

	result, err := tool.InvokableRun(context.Background(), `{invalid}`)
	require.NoError(t, err)
	assert.Contains(t, result, "[ERROR]")
	assert.Empty(t, emitter.events)
}

func TestStructuredOutput_NilEmitter(t *testing.T) {
	// Should not panic when emitter is nil
	tool := NewStructuredOutputTool(nil, "sess-1")

	args := `{"output_type": "summary_table", "title": "Test"}`
	result, err := tool.InvokableRun(context.Background(), args)
	require.NoError(t, err)
	assert.Equal(t, "Structured output displayed to user.", result)
}

func TestStructuredOutput_ToolInfo(t *testing.T) {
	tool := NewStructuredOutputTool(nil, "sess-1")

	info, err := tool.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "show_structured_output", info.Name)
	assert.Contains(t, info.Desc, "structured data")
}
