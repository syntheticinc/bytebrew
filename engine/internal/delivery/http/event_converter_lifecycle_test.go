package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertDomainEvent_StateChanged(t *testing.T) {
	// AC-STATE-02: state_changed maps to agent.state_changed SSE event
	event := ConvertDomainEvent("state_changed", `{"agent_name":"test","old_state":"ready","new_state":"running"}`)
	require.NotNil(t, event)
	assert.Equal(t, "agent.state_changed", event.Type)
	assert.Contains(t, event.Data, "ready")
	assert.Contains(t, event.Data, "running")
}

func TestConvertDomainEvent_UnknownType(t *testing.T) {
	// AC-EVT-02: Unknown event types safely ignored
	event := ConvertDomainEvent("future_event_type", `{}`)
	assert.Nil(t, event)
}

func TestConvertDomainEvent_ExistingTypes(t *testing.T) {
	// Verify existing event types still work
	tests := []struct {
		domainType string
		sseType    string
	}{
		{"MessageStarted", "thinking"},
		{"StreamingProgress", "message_delta"},
		{"MessageCompleted", "message"},
		{"ToolExecutionStarted", "tool_call"},
		{"ToolExecutionCompleted", "tool_result"},
		{"Error", "error"},
		{"agent_spawned", "agent_spawn"},
		{"agent_completed", "agent_result"},
		{"user_question", "user_input_required"},
		{"structured_output", "structured_output"},
	}
	for _, tt := range tests {
		t.Run(tt.domainType, func(t *testing.T) {
			event := ConvertDomainEvent(tt.domainType, `{"content":"test"}`)
			require.NotNil(t, event)
			assert.Equal(t, tt.sseType, event.Type)
		})
	}
}
