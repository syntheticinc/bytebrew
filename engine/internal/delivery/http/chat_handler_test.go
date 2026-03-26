package http

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleNonStreaming_EmptyMessageEventDoesNotOverwrite(t *testing.T) {
	tests := []struct {
		name     string
		events   []SSEEvent
		wantMsg  string
		wantSID  string
		wantTool int
	}{
		{
			name: "trailing empty message event does not erase answer",
			events: []SSEEvent{
				sseEvent("message_delta", map[string]interface{}{"content": "Hello"}),
				sseEvent("message_delta", map[string]interface{}{"content": " world!"}),
				sseEvent("message", map[string]interface{}{"content": "Hello world!"}),
				// Engine sends this trailing "completion signal" with empty content
				sseEvent("message", map[string]interface{}{"content": ""}),
				sseEvent("done", map[string]interface{}{"session_id": "sess-1"}),
			},
			wantMsg: "Hello world!",
			wantSID: "sess-1",
		},
		{
			name: "single message event with content works normally",
			events: []SSEEvent{
				sseEvent("message", map[string]interface{}{"content": "Hi there"}),
				sseEvent("done", map[string]interface{}{"session_id": "sess-2"}),
			},
			wantMsg: "Hi there",
			wantSID: "sess-2",
		},
		{
			name: "only deltas without final message works",
			events: []SSEEvent{
				sseEvent("message_delta", map[string]interface{}{"content": "chunk1"}),
				sseEvent("message_delta", map[string]interface{}{"content": "chunk2"}),
				sseEvent("done", map[string]interface{}{"session_id": "sess-3"}),
			},
			wantMsg: "chunk1chunk2",
			wantSID: "sess-3",
		},
		{
			name: "message replaces accumulated deltas",
			events: []SSEEvent{
				sseEvent("message_delta", map[string]interface{}{"content": "chunk1"}),
				sseEvent("message_delta", map[string]interface{}{"content": "chunk2"}),
				sseEvent("message", map[string]interface{}{"content": "full answer"}),
				sseEvent("done", map[string]interface{}{"session_id": "sess-4"}),
			},
			wantMsg: "full answer",
			wantSID: "sess-4",
		},
		{
			name: "tool calls are collected",
			events: []SSEEvent{
				sseEvent("tool_call", map[string]interface{}{"tool": "search", "content": "query"}),
				sseEvent("tool_result", map[string]interface{}{"tool": "search", "content": "results"}),
				sseEvent("message", map[string]interface{}{"content": "Done"}),
				sseEvent("done", map[string]interface{}{"session_id": "sess-5"}),
			},
			wantMsg:  "Done",
			wantSID:  "sess-5",
			wantTool: 1,
		},
		{
			name: "only empty message events result in empty message",
			events: []SSEEvent{
				sseEvent("message", map[string]interface{}{"content": ""}),
				sseEvent("done", map[string]interface{}{"session_id": "sess-6"}),
			},
			wantMsg: "",
			wantSID: "sess-6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := make(chan SSEEvent, len(tt.events))
			for _, e := range tt.events {
				ch <- e
			}
			close(ch)

			w := httptest.NewRecorder()
			h := &ChatHandler{}
			h.handleNonStreaming(w, "test-agent", ch)

			var resp nonStreamResponse
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)

			assert.Equal(t, tt.wantMsg, resp.Message)
			assert.Equal(t, tt.wantSID, resp.SessionID)
			assert.Equal(t, "test-agent", resp.Agent)
			assert.Len(t, resp.ToolCalls, tt.wantTool)
		})
	}
}

// sseEvent creates an SSEEvent with JSON-encoded data.
func sseEvent(eventType string, data map[string]interface{}) SSEEvent {
	jsonBytes, _ := json.Marshal(data)
	return SSEEvent{
		Type: eventType,
		Data: string(jsonBytes),
	}
}
