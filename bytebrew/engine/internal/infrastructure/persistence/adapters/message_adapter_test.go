package adapters

import (
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/persistence/models"
	"github.com/google/uuid"
)

func TestMessageToModel_Basic(t *testing.T) {
	msgID := uuid.New().String()
	sessionID := uuid.New().String()
	createdAt := time.Now()

	msg := &domain.Message{
		ID:        msgID,
		SessionID: sessionID,
		Type:      domain.MessageTypeUser,
		Sender:    "user",
		Content:   "Hello world",
		Metadata:  make(map[string]string),
		CreatedAt: createdAt,
	}

	model, err := MessageToModel(msg)
	if err != nil {
		t.Fatalf("MessageToModel() error = %v", err)
	}

	if model == nil {
		t.Fatal("MessageToModel() returned nil")
	}

	if model.MessageType != "user" {
		t.Errorf("MessageToModel() MessageType = %v, want user", model.MessageType)
	}

	if model.Sender != "user" {
		t.Errorf("MessageToModel() Sender = %v, want user", model.Sender)
	}

	if model.Content != "Hello world" {
		t.Errorf("MessageToModel() Content = %v, want 'Hello world'", model.Content)
	}
}

func TestMessageToModel_Nil(t *testing.T) {
	model, err := MessageToModel(nil)
	if err != nil {
		t.Errorf("MessageToModel(nil) error = %v", err)
	}
	if model != nil {
		t.Error("MessageToModel(nil) should return nil")
	}
}

func TestMessageToModel_WithMetadata(t *testing.T) {
	msg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Type:      domain.MessageTypeUser,
		Sender:    "user",
		Content:   "Test",
		Metadata: map[string]string{
			"source":   "web",
			"language": "en",
		},
		CreatedAt: time.Now(),
	}

	model, err := MessageToModel(msg)
	if err != nil {
		t.Fatalf("MessageToModel() error = %v", err)
	}

	if model == nil {
		t.Fatal("MessageToModel() returned nil")
	}

	if len(model.Metadata) == 0 {
		t.Error("MessageToModel() Metadata should not be empty")
	}

	// Verify metadata is serialized (contains expected keys)
	metaStr := string(model.Metadata)
	if metaStr == "" || metaStr == "null" {
		t.Error("MessageToModel() Metadata was not serialized")
	}
}

func TestMessageToModel_WithToolCalls(t *testing.T) {
	toolCalls := []domain.ToolCallInfo{
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

	msg := &domain.Message{
		ID:        uuid.New().String(),
		SessionID: uuid.New().String(),
		Type:      domain.MessageTypeAgent,
		Sender:    "assistant",
		Content:   "Let me search",
		ToolCalls: toolCalls,
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
	}

	model, err := MessageToModel(msg)
	if err != nil {
		t.Fatalf("MessageToModel() error = %v", err)
	}

	if model == nil {
		t.Fatal("MessageToModel() returned nil")
	}

	if len(model.Metadata) == 0 {
		t.Error("MessageToModel() Metadata should contain serialized tool_calls")
	}

	// Verify tool_calls are in metadata
	metaStr := string(model.Metadata)
	if metaStr == "" {
		t.Error("MessageToModel() Metadata should not be empty")
	}

	// Check that tool call info is present
	if !containsString(metaStr, "call-1") {
		t.Error("MessageToModel() Metadata should contain tool call ID 'call-1'")
	}
	if !containsString(metaStr, "search_code") {
		t.Error("MessageToModel() Metadata should contain tool name 'search_code'")
	}
}

func TestMessageToModel_ToolMessage(t *testing.T) {
	msg := &domain.Message{
		ID:         uuid.New().String(),
		SessionID:  uuid.New().String(),
		Type:       domain.MessageTypeTool,
		Sender:     "search_code",
		Content:    "Found 5 results",
		ToolCallID: "call-1",
		ToolName:   "search_code",
		Metadata:   make(map[string]string),
		CreatedAt:  time.Now(),
	}

	model, err := MessageToModel(msg)
	if err != nil {
		t.Fatalf("MessageToModel() error = %v", err)
	}

	if model == nil {
		t.Fatal("MessageToModel() returned nil")
	}

	if model.MessageType != "tool" {
		t.Errorf("MessageToModel() MessageType = %v, want tool", model.MessageType)
	}

	// Verify tool_call_id is in metadata
	metaStr := string(model.Metadata)
	if !containsString(metaStr, "call-1") {
		t.Error("MessageToModel() Metadata should contain tool_call_id 'call-1'")
	}
}

func TestMessageFromModel_Basic(t *testing.T) {
	modelID := uuid.New()
	sessionID := uuid.New()
	createdAt := time.Now()

	model := &models.Message{
		ID:          modelID,
		SessionID:   sessionID,
		MessageType: "user",
		Sender:      "user",
		Content:     "Hello world",
		CreatedAt:   createdAt,
	}

	msg, err := MessageFromModel(model)

	if err != nil {
		t.Errorf("MessageFromModel() error = %v", err)
	}

	if msg == nil {
		t.Fatal("MessageFromModel() returned nil message")
	}

	if msg.ID != modelID.String() {
		t.Errorf("MessageFromModel() ID = %v, want %v", msg.ID, modelID.String())
	}

	if msg.Type != domain.MessageTypeUser {
		t.Errorf("MessageFromModel() Type = %v, want user", msg.Type)
	}

	if msg.Content != "Hello world" {
		t.Errorf("MessageFromModel() Content = %v, want 'Hello world'", msg.Content)
	}
}

func TestMessageFromModel_Nil(t *testing.T) {
	msg, err := MessageFromModel(nil)
	if err != nil {
		t.Errorf("MessageFromModel(nil) error = %v", err)
	}
	if msg != nil {
		t.Error("MessageFromModel(nil) should return nil message")
	}
}

func TestMessageFromModel_WithMetadata(t *testing.T) {
	model := &models.Message{
		ID:          uuid.New(),
		SessionID:   uuid.New(),
		MessageType: "user",
		Sender:      "user",
		Content:     "Test",
		Metadata:    []byte(`{"user_metadata":{"source":"web","language":"en"}}`),
		CreatedAt:   time.Now(),
	}

	msg, err := MessageFromModel(model)

	if err != nil {
		t.Errorf("MessageFromModel() error = %v", err)
	}

	if msg == nil {
		t.Fatal("MessageFromModel() returned nil")
	}

	if msg.Metadata == nil {
		t.Fatal("MessageFromModel() Metadata should not be nil")
	}

	if msg.Metadata["source"] != "web" {
		t.Errorf("MessageFromModel() Metadata[source] = %v, want web", msg.Metadata["source"])
	}

	if msg.Metadata["language"] != "en" {
		t.Errorf("MessageFromModel() Metadata[language] = %v, want en", msg.Metadata["language"])
	}
}

func TestMessageFromModel_WithToolCalls(t *testing.T) {
	model := &models.Message{
		ID:          uuid.New(),
		SessionID:   uuid.New(),
		MessageType: "agent",
		Sender:      "assistant",
		Content:     "Let me search",
		Metadata:    []byte(`{"tool_calls":[{"id":"call-1","name":"search_code","arguments":{"query":"test"}},{"id":"call-2","name":"read_file","arguments":{"path":"/test.go"}}]}`),
		CreatedAt:   time.Now(),
	}

	msg, err := MessageFromModel(model)

	if err != nil {
		t.Errorf("MessageFromModel() error = %v", err)
	}

	if msg == nil {
		t.Fatal("MessageFromModel() returned nil")
	}

	if len(msg.ToolCalls) != 2 {
		t.Errorf("MessageFromModel() ToolCalls length = %d, want 2", len(msg.ToolCalls))
	}

	if msg.ToolCalls[0].ID != "call-1" {
		t.Errorf("MessageFromModel() ToolCalls[0].ID = %v, want call-1", msg.ToolCalls[0].ID)
	}

	if msg.ToolCalls[0].Name != "search_code" {
		t.Errorf("MessageFromModel() ToolCalls[0].Name = %v, want search_code", msg.ToolCalls[0].Name)
	}

	if msg.ToolCalls[1].ID != "call-2" {
		t.Errorf("MessageFromModel() ToolCalls[1].ID = %v, want call-2", msg.ToolCalls[1].ID)
	}
}

func TestMessageFromModel_ToolMessage(t *testing.T) {
	model := &models.Message{
		ID:          uuid.New(),
		SessionID:   uuid.New(),
		MessageType: "tool",
		Sender:      "search_code",
		Content:     "Found 5 results",
		Metadata:    []byte(`{"tool_call_id":"call-1","tool_name":"search_code"}`),
		CreatedAt:   time.Now(),
	}

	msg, err := MessageFromModel(model)

	if err != nil {
		t.Errorf("MessageFromModel() error = %v", err)
	}

	if msg == nil {
		t.Fatal("MessageFromModel() returned nil")
	}

	if msg.Type != domain.MessageTypeTool {
		t.Errorf("MessageFromModel() Type = %v, want tool", msg.Type)
	}

	if msg.ToolCallID != "call-1" {
		t.Errorf("MessageFromModel() ToolCallID = %v, want call-1", msg.ToolCallID)
	}

	if msg.ToolName != "search_code" {
		t.Errorf("MessageFromModel() ToolName = %v, want search_code", msg.ToolName)
	}
}

func TestMessageAdapter_RoundTrip(t *testing.T) {
	// Test user message round trip
	t.Run("user message", func(t *testing.T) {
		original := &domain.Message{
			ID:        uuid.New().String(),
			SessionID: uuid.New().String(),
			Type:      domain.MessageTypeUser,
			Sender:    "user",
			Content:   "What is Go?",
			Metadata: map[string]string{
				"source": "web",
			},
			CreatedAt: time.Now().Truncate(time.Microsecond),
		}

		model, err := MessageToModel(original)
		if err != nil {
			t.Fatalf("MessageToModel() error = %v", err)
		}
		restored, err := MessageFromModel(model)

		if err != nil {
			t.Errorf("Round trip error: %v", err)
		}

		assertMessagesEqual(t, original, restored)
	})

	// Test assistant message with tool calls round trip
	t.Run("assistant with tool calls", func(t *testing.T) {
		original := &domain.Message{
			ID:        uuid.New().String(),
			SessionID: uuid.New().String(),
			Type:      domain.MessageTypeAgent,
			Sender:    "assistant",
			Content:   "Let me search for that",
			ToolCalls: []domain.ToolCallInfo{
				{
					ID:        "call-1",
					Name:      "search_code",
					Arguments: map[string]string{"query": "golang interfaces"},
				},
			},
			Metadata:  make(map[string]string),
			CreatedAt: time.Now().Truncate(time.Microsecond),
		}

		model, err := MessageToModel(original)
		if err != nil {
			t.Fatalf("MessageToModel() error = %v", err)
		}
		restored, err := MessageFromModel(model)

		if err != nil {
			t.Errorf("Round trip error: %v", err)
		}

		assertMessagesEqual(t, original, restored)

		if len(restored.ToolCalls) != 1 {
			t.Errorf("Round trip ToolCalls length = %d, want 1", len(restored.ToolCalls))
		}

		if restored.ToolCalls[0].ID != "call-1" {
			t.Errorf("Round trip ToolCalls[0].ID = %v, want call-1", restored.ToolCalls[0].ID)
		}
	})

	// Test tool result message round trip
	t.Run("tool result", func(t *testing.T) {
		original := &domain.Message{
			ID:         uuid.New().String(),
			SessionID:  uuid.New().String(),
			Type:       domain.MessageTypeTool,
			Sender:     "search_code",
			Content:    "Found 10 results",
			ToolCallID: "call-1",
			ToolName:   "search_code",
			Metadata:   make(map[string]string),
			CreatedAt:  time.Now().Truncate(time.Microsecond),
		}

		model, err := MessageToModel(original)
		if err != nil {
			t.Fatalf("MessageToModel() error = %v", err)
		}
		restored, err := MessageFromModel(model)

		if err != nil {
			t.Errorf("Round trip error: %v", err)
		}

		assertMessagesEqual(t, original, restored)

		if restored.ToolCallID != "call-1" {
			t.Errorf("Round trip ToolCallID = %v, want call-1", restored.ToolCallID)
		}

		if restored.ToolName != "search_code" {
			t.Errorf("Round trip ToolName = %v, want search_code", restored.ToolName)
		}
	})
}

func TestMessageFromModel_InvalidMetadata(t *testing.T) {
	model := &models.Message{
		ID:          uuid.New(),
		SessionID:   uuid.New(),
		MessageType: "user",
		Sender:      "user",
		Content:     "Test",
		Metadata:    []byte(`{invalid json`),
		CreatedAt:   time.Now(),
	}

	msg, err := MessageFromModel(model)

	// Should not fail, just ignore invalid metadata
	if err != nil {
		t.Errorf("MessageFromModel() should not fail on invalid metadata: %v", err)
	}

	if msg == nil {
		t.Fatal("MessageFromModel() returned nil")
	}

	// Metadata should be initialized but empty (or default)
	if msg.Metadata == nil {
		t.Error("MessageFromModel() Metadata should be initialized")
	}
}

// Helper functions

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func assertMessagesEqual(t *testing.T, expected, actual *domain.Message) {
	t.Helper()

	if actual.ID != expected.ID {
		t.Errorf("ID = %v, want %v", actual.ID, expected.ID)
	}

	if actual.SessionID != expected.SessionID {
		t.Errorf("SessionID = %v, want %v", actual.SessionID, expected.SessionID)
	}

	if actual.Type != expected.Type {
		t.Errorf("Type = %v, want %v", actual.Type, expected.Type)
	}

	if actual.Sender != expected.Sender {
		t.Errorf("Sender = %v, want %v", actual.Sender, expected.Sender)
	}

	if actual.Content != expected.Content {
		t.Errorf("Content = %v, want %v", actual.Content, expected.Content)
	}
}
