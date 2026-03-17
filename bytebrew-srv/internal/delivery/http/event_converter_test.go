package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertDomainEvent(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		payload   string
		wantType  string
		wantNil   bool
	}{
		{"MessageStarted to thinking", "MessageStarted", `{"id":1}`, "thinking", false},
		{"StreamingProgress to message", "StreamingProgress", `{"delta":"hi"}`, "message", false},
		{"MessageCompleted to message", "MessageCompleted", `{"text":"done"}`, "message", false},
		{"ToolExecutionStarted to tool_call", "ToolExecutionStarted", `{"tool":"read"}`, "tool_call", false},
		{"ToolExecutionCompleted to tool_result", "ToolExecutionCompleted", `{"result":"ok"}`, "tool_result", false},
		{"ConfirmationRequired to confirmation", "ConfirmationRequired", `{"prompt":"sure?"}`, "confirmation", false},
		{"ProcessingStopped to done", "ProcessingStopped", `{}`, "done", false},
		{"Error to error", "Error", `{"msg":"fail"}`, "error", false},
		{"unknown event returns nil", "SomeRandomEvent", `{}`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertDomainEvent(tt.eventType, tt.payload)
			if tt.wantNil {
				assert.Nil(t, result)
				return
			}
			assert.NotNil(t, result)
			assert.Equal(t, tt.wantType, result.Type)
			assert.Equal(t, tt.payload, result.Data)
		})
	}
}
