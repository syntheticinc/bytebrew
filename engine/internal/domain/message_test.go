package domain

import (
	"testing"
	"time"
)

func TestNewMessage(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		msgType   MessageType
		sender    string
		content   string
		wantErr   bool
	}{
		{
			name:      "valid user message",
			sessionID: "session-1",
			msgType:   MessageTypeUser,
			sender:    "user",
			content:   "Hello",
			wantErr:   false,
		},
		{
			name:      "valid agent message",
			sessionID: "session-1",
			msgType:   MessageTypeAgent,
			sender:    "assistant",
			content:   "Hi there",
			wantErr:   false,
		},
		{
			name:      "empty session_id",
			sessionID: "",
			msgType:   MessageTypeUser,
			sender:    "user",
			content:   "Hello",
			wantErr:   true,
		},
		{
			name:      "empty content",
			sessionID: "session-1",
			msgType:   MessageTypeUser,
			sender:    "user",
			content:   "",
			wantErr:   true,
		},
		{
			name:      "invalid message type",
			sessionID: "session-1",
			msgType:   MessageType("invalid"),
			sender:    "user",
			content:   "Hello",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NewMessage(tt.sessionID, tt.msgType, tt.sender, tt.content)

			if tt.wantErr {
				if err == nil {
					t.Error("NewMessage() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewMessage() unexpected error: %v", err)
				return
			}

			if msg == nil {
				t.Error("NewMessage() returned nil message")
				return
			}

			if msg.SessionID != tt.sessionID {
				t.Errorf("NewMessage() SessionID = %v, want %v", msg.SessionID, tt.sessionID)
			}
			if msg.Type != tt.msgType {
				t.Errorf("NewMessage() Type = %v, want %v", msg.Type, tt.msgType)
			}
			if msg.Content != tt.content {
				t.Errorf("NewMessage() Content = %v, want %v", msg.Content, tt.content)
			}
		})
	}
}

func TestNewMessage_WithToolCalls(t *testing.T) {
	toolCalls := []ToolCallInfo{
		{
			ID:        "call-1",
			Name:      "search_code",
			Arguments: map[string]string{"query": "test"},
		},
	}

	msg, err := NewAssistantMessageWithToolCalls("session-1", "", toolCalls)
	if err != nil {
		t.Errorf("NewAssistantMessageWithToolCalls() unexpected error: %v", err)
		return
	}

	if len(msg.ToolCalls) != 1 {
		t.Errorf("NewAssistantMessageWithToolCalls() ToolCalls length = %d, want 1", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].ID != "call-1" {
		t.Errorf("NewAssistantMessageWithToolCalls() ToolCalls[0].ID = %v, want call-1", msg.ToolCalls[0].ID)
	}

	if msg.ToolCalls[0].Name != "search_code" {
		t.Errorf("NewAssistantMessageWithToolCalls() ToolCalls[0].Name = %v, want search_code", msg.ToolCalls[0].Name)
	}
}

func TestNewAssistantMessageWithToolCalls_Validation(t *testing.T) {
	tests := []struct {
		name      string
		sessionID string
		content   string
		toolCalls []ToolCallInfo
		wantErr   bool
	}{
		{
			name:      "valid with tool calls and no content",
			sessionID: "session-1",
			content:   "",
			toolCalls: []ToolCallInfo{{ID: "call-1", Name: "test"}},
			wantErr:   false,
		},
		{
			name:      "valid with content and no tool calls",
			sessionID: "session-1",
			content:   "Some answer",
			toolCalls: nil,
			wantErr:   false,
		},
		{
			name:      "valid with both content and tool calls",
			sessionID: "session-1",
			content:   "Let me search",
			toolCalls: []ToolCallInfo{{ID: "call-1", Name: "test"}},
			wantErr:   false,
		},
		{
			name:      "invalid empty session_id",
			sessionID: "",
			content:   "test",
			toolCalls: nil,
			wantErr:   true,
		},
		{
			name:      "invalid no content and no tool calls",
			sessionID: "session-1",
			content:   "",
			toolCalls: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAssistantMessageWithToolCalls(tt.sessionID, tt.content, tt.toolCalls)

			if tt.wantErr && err == nil {
				t.Error("NewAssistantMessageWithToolCalls() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("NewAssistantMessageWithToolCalls() unexpected error: %v", err)
			}
		})
	}
}

func TestNewToolMessage(t *testing.T) {
	tests := []struct {
		name       string
		sessionID  string
		toolCallID string
		toolName   string
		content    string
		wantErr    bool
	}{
		{
			name:       "valid tool message",
			sessionID:  "session-1",
			toolCallID: "call-1",
			toolName:   "search_code",
			content:    "Found 5 results",
			wantErr:    false,
		},
		{
			name:       "valid tool message with empty content",
			sessionID:  "session-1",
			toolCallID: "call-1",
			toolName:   "search_code",
			content:    "",
			wantErr:    false,
		},
		{
			name:       "missing tool_call_id",
			sessionID:  "session-1",
			toolCallID: "",
			toolName:   "search_code",
			content:    "Result",
			wantErr:    true,
		},
		{
			name:       "missing session_id",
			sessionID:  "",
			toolCallID: "call-1",
			toolName:   "search_code",
			content:    "Result",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := NewToolMessage(tt.sessionID, tt.toolCallID, tt.toolName, tt.content)

			if tt.wantErr {
				if err == nil {
					t.Error("NewToolMessage() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewToolMessage() unexpected error: %v", err)
				return
			}

			if msg.Type != MessageTypeTool {
				t.Errorf("NewToolMessage() Type = %v, want %v", msg.Type, MessageTypeTool)
			}
			if msg.ToolCallID != tt.toolCallID {
				t.Errorf("NewToolMessage() ToolCallID = %v, want %v", msg.ToolCallID, tt.toolCallID)
			}
			if msg.ToolName != tt.toolName {
				t.Errorf("NewToolMessage() ToolName = %v, want %v", msg.ToolName, tt.toolName)
			}
		})
	}
}

func TestMessage_ToHistoryMessage_PreservesToolCalls(t *testing.T) {
	toolCalls := []ToolCallInfo{
		{
			ID:        "call-1",
			Name:      "search_code",
			Arguments: map[string]string{"query": "test"},
		},
		{
			ID:        "call-2",
			Name:      "read_file",
			Arguments: map[string]string{"path": "/test.go"},
		},
	}

	msg := &Message{
		ID:        "msg-1",
		SessionID: "session-1",
		Type:      MessageTypeAgent,
		Sender:    "assistant",
		Content:   "Let me search for that",
		ToolCalls: toolCalls,
		CreatedAt: time.Now(),
	}

	history := msg.ToHistoryMessage()

	if history.Role != "assistant" {
		t.Errorf("ToHistoryMessage() Role = %v, want assistant", history.Role)
	}

	if len(history.ToolCalls) != 2 {
		t.Errorf("ToHistoryMessage() ToolCalls length = %d, want 2", len(history.ToolCalls))
	}

	if history.ToolCalls[0].ID != "call-1" {
		t.Errorf("ToHistoryMessage() ToolCalls[0].ID = %v, want call-1", history.ToolCalls[0].ID)
	}

	if history.ToolCalls[1].Name != "read_file" {
		t.Errorf("ToHistoryMessage() ToolCalls[1].Name = %v, want read_file", history.ToolCalls[1].Name)
	}
}

func TestMessage_ToHistoryMessage_ToolResult(t *testing.T) {
	msg := &Message{
		ID:         "msg-1",
		SessionID:  "session-1",
		Type:       MessageTypeTool,
		Sender:     "search_code",
		Content:    "Found 5 results: file1.go, file2.go...",
		ToolCallID: "call-1",
		ToolName:   "search_code",
		CreatedAt:  time.Now(),
	}

	history := msg.ToHistoryMessage()

	if history.Role != "tool" {
		t.Errorf("ToHistoryMessage() Role = %v, want tool", history.Role)
	}

	if history.ToolCallID != "call-1" {
		t.Errorf("ToHistoryMessage() ToolCallID = %v, want call-1", history.ToolCallID)
	}

	if history.ToolName != "search_code" {
		t.Errorf("ToHistoryMessage() ToolName = %v, want search_code", history.ToolName)
	}

	if history.Content != "Found 5 results: file1.go, file2.go..." {
		t.Errorf("ToHistoryMessage() Content = %v, want 'Found 5 results...'", history.Content)
	}
}

func TestMessage_ToHistoryMessage_AllRoles(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		wantRole string
	}{
		{
			name:     "user message",
			msgType:  MessageTypeUser,
			wantRole: "user",
		},
		{
			name:     "agent message",
			msgType:  MessageTypeAgent,
			wantRole: "assistant",
		},
		{
			name:     "system message",
			msgType:  MessageTypeSystem,
			wantRole: "system",
		},
		{
			name:     "tool message",
			msgType:  MessageTypeTool,
			wantRole: "tool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &Message{
				ID:         "msg-1",
				SessionID:  "session-1",
				Type:       tt.msgType,
				Sender:     "test",
				Content:    "test content",
				ToolCallID: "call-1", // For tool messages
				CreatedAt:  time.Now(),
			}

			history := msg.ToHistoryMessage()

			if history.Role != tt.wantRole {
				t.Errorf("ToHistoryMessage() Role = %v, want %v", history.Role, tt.wantRole)
			}
		})
	}
}

func TestMessage_Validate_ToolMessageRequiresToolCallID(t *testing.T) {
	msg := &Message{
		SessionID:  "session-1",
		Type:       MessageTypeTool,
		Sender:     "search_code",
		Content:    "Result",
		ToolCallID: "", // Missing!
		CreatedAt:  time.Now(),
	}

	err := msg.Validate()
	if err == nil {
		t.Error("Validate() expected error for tool message without ToolCallID, got nil")
	}
}

func TestMessage_Metadata(t *testing.T) {
	msg, _ := NewMessage("session-1", MessageTypeUser, "user", "Hello")

	// Add metadata
	msg.AddMetadata("source", "web")
	msg.AddMetadata("language", "en")

	// Get metadata
	source, ok := msg.GetMetadata("source")
	if !ok {
		t.Error("GetMetadata() did not find 'source'")
	}
	if source != "web" {
		t.Errorf("GetMetadata() source = %v, want web", source)
	}

	// Get non-existent metadata
	_, ok = msg.GetMetadata("nonexistent")
	if ok {
		t.Error("GetMetadata() found non-existent key")
	}
}
