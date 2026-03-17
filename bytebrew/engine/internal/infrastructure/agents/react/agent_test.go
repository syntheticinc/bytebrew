package react

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/config"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// mockChatModel is a mock implementation of model.ChatModel for testing
type mockChatModel struct {
	generateFunc       func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error)
	streamFunc         func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error)
	bindToolsFunc      func(tools []*schema.ToolInfo) error
	getTypeFunc        func() string
	isCallbacksEnabled bool
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, input, opts...)
	}
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "mock response",
	}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, input, opts...)
	}
	return nil, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	if m.bindToolsFunc != nil {
		return m.bindToolsFunc(tools)
	}
	return nil
}

func (m *mockChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func (m *mockChatModel) GetType() string {
	if m.getTypeFunc != nil {
		return m.getTypeFunc()
	}
	return "mock"
}

func (m *mockChatModel) IsCallbacksEnabled() bool {
	return m.isCallbacksEnabled
}

func TestNewAgent_NilChatModel_ReturnsError(t *testing.T) {
	cfg := AgentConfig{
		ChatModel: nil,
		MaxSteps:  10,
	}

	agent, err := NewAgent(context.Background(), cfg)

	if err == nil {
		t.Error("expected error when ChatModel is nil")
	}
	if agent != nil {
		t.Error("expected nil agent when ChatModel is nil")
	}
}

func TestNewAgent_ZeroMaxSteps_UsesUnlimited(t *testing.T) {
	mockModel := &mockChatModel{}
	cfg := AgentConfig{
		ChatModel: mockModel,
		MaxSteps:  0, // Zero means unlimited (uses 10000 internally)
	}

	agent, err := NewAgent(context.Background(), cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}
}

func TestNewAgent_WithAgentConfig(t *testing.T) {
	mockModel := &mockChatModel{}
	agentConfig := &config.AgentConfig{
		MaxSteps:       10,
		MaxContextSize: 16000,
		ContextLogPath: "./test_logs",
		Prompts: &config.PromptsConfig{
			SystemPrompt:   "Test prompt",
			UrgencyWarning: "Warning: %d steps left",
		},
	}

	cfg := AgentConfig{
		ChatModel:   mockModel,
		MaxSteps:    10,
		SessionID:   "test-session-123",
		AgentConfig: agentConfig,
		ModelName:   "test-model",
	}

	agent, err := NewAgent(context.Background(), cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if agent == nil {
		t.Fatal("expected non-nil agent")
	}

	// Context logger should be created
	if agent.contextLogger == nil {
		t.Error("expected contextLogger to be created")
	}
}

func TestNewAgent_BuildMessagesWithHistory(t *testing.T) {
	mockModel := &mockChatModel{}

	historyMessages := []*schema.Message{
		{Role: schema.User, Content: "Previous question"},
		{Role: schema.Assistant, Content: "Previous answer"},
	}

	cfg := AgentConfig{
		ChatModel:       mockModel,
		MaxSteps:        10,
		HistoryMessages: historyMessages,
	}

	agent, err := NewAgent(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	messages := agent.buildMessagesWithHistory("Current question")

	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}

	if messages[0].Content != "Previous question" {
		t.Errorf("message[0] content: got %q, want %q", messages[0].Content, "Previous question")
	}
	if messages[2].Content != "Current question" {
		t.Errorf("message[2] content: got %q, want %q", messages[2].Content, "Current question")
	}
}

func TestAgentConfig_Structure(t *testing.T) {
	mockModel := &mockChatModel{}

	cfg := AgentConfig{
		ChatModel:       mockModel,
		Tools:           nil,
		MaxSteps:        15,
		SessionID:       "session-123",
		AgentConfig:     nil,
		ModelName:       "test-model",
		HistoryMessages: nil,
	}

	if cfg.MaxSteps != 15 {
		t.Errorf("MaxSteps: got %d, want 15", cfg.MaxSteps)
	}
	if cfg.SessionID != "session-123" {
		t.Errorf("SessionID: got %q, want %q", cfg.SessionID, "session-123")
	}
}

func TestIsRateLimitError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "429 Too Many Requests",
			err:      fmt.Errorf("429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "rate limit exceeded",
			err:      fmt.Errorf("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "Rate Limit (mixed case)",
			err:      fmt.Errorf("Rate Limit exceeded"),
			expected: true,
		},
		{
			name:     "quota exceeded",
			err:      fmt.Errorf("quota exceeded"),
			expected: true,
		},
		{
			name:     "too many requests lowercase",
			err:      fmt.Errorf("too many requests"),
			expected: true,
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("some random error"),
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "unauthorized",
			err:      fmt.Errorf("unauthorized"),
			expected: false,
		},
		{
			name:     "XML error",
			err:      fmt.Errorf("XML syntax error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRateLimitError(tt.err)
			if result != tt.expected {
				t.Errorf("isRateLimitError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsRecoverableAgentError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context canceled - not recoverable",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded - not recoverable",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "rate limit - recoverable (handled separately)",
			err:      fmt.Errorf("rate limit exceeded"),
			expected: true,
		},
		{
			name:     "quota exceeded - recoverable (handled separately)",
			err:      fmt.Errorf("quota exceeded"),
			expected: true,
		},
		{
			name:     "unauthorized - not recoverable",
			err:      fmt.Errorf("unauthorized access"),
			expected: false,
		},
		{
			name:     "XML syntax error - recoverable",
			err:      fmt.Errorf("XML syntax error on line 1"),
			expected: true,
		},
		{
			name:     "JSON unmarshal error - recoverable",
			err:      fmt.Errorf("json: cannot unmarshal string"),
			expected: true,
		},
		{
			name:     "tool not found - recoverable",
			err:      fmt.Errorf("tool not found: unknown_tool"),
			expected: true,
		},
		{
			name:     "GraphRunError - recoverable",
			err:      fmt.Errorf("[GraphRunError] failed to calculate next tasks"),
			expected: true,
		},
		{
			name:     "generic error - recoverable",
			err:      fmt.Errorf("some random error"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRecoverableAgentError(tt.err)
			if result != tt.expected {
				t.Errorf("isRecoverableAgentError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestRateLimitBackoff(t *testing.T) {
	tests := []struct {
		attempt  int
		expected string
	}{
		{0, "2s"},
		{1, "4s"},
		{2, "8s"},
		{3, "16s"},
		{4, "32s"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			result := rateLimitBackoff(tt.attempt)
			if result.String() != tt.expected {
				t.Errorf("rateLimitBackoff(%d) = %v, want %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

func TestFormatAgentErrorFeedback(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "nil error",
			err:      nil,
			contains: "",
		},
		{
			name:     "XML error",
			err:      fmt.Errorf("XML syntax error on line 1"),
			contains: "invalid XML format",
		},
		{
			name:     "JSON error",
			err:      fmt.Errorf("json: cannot unmarshal"),
			contains: "invalid JSON format",
		},
		{
			name:     "tool not found",
			err:      fmt.Errorf("tool not found: my_tool"),
			contains: "does not exist",
		},
		{
			name:     "generic error",
			err:      fmt.Errorf("something went wrong"),
			contains: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatAgentErrorFeedback(tt.err)
			if tt.contains == "" {
				if result != "" {
					t.Errorf("formatAgentErrorFeedback(%v) = %q, want empty", tt.err, result)
				}
			} else if !strings.Contains(result, tt.contains) {
				t.Errorf("formatAgentErrorFeedback(%v) = %q, should contain %q", tt.err, result, tt.contains)
			}
		})
	}
}

func TestSanitizeToolArguments_ValidJSON(t *testing.T) {
	input := `{"file_path": "main.go"}`
	result := sanitizeToolArguments(input)
	if result != input {
		t.Errorf("expected unchanged, got %q", result)
	}
}

func TestSanitizeToolArguments_XMLTags(t *testing.T) {
	input := `<parameter>{"file_path": "main.go"}</parameter>`
	result := sanitizeToolArguments(input)
	if !json.Valid([]byte(result)) {
		t.Errorf("expected valid JSON after sanitization, got %q", result)
	}
	expected := `{"file_path": "main.go"}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSanitizeToolArguments_MixedContent(t *testing.T) {
	input := `some text {"action": "list"} more text`
	result := sanitizeToolArguments(input)
	if !json.Valid([]byte(result)) {
		t.Errorf("expected valid JSON extracted, got %q", result)
	}
	expected := `{"action": "list"}`
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestSanitizeToolArguments_NoJSON(t *testing.T) {
	input := `completely invalid content`
	result := sanitizeToolArguments(input)
	if result != input {
		t.Errorf("expected original returned, got %q", result)
	}
}
