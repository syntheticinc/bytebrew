package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// mockClient implements Client interface for testing
type mockClient struct {
	chatFunc       func(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	chatStreamFunc func(ctx context.Context, req ChatRequest, callback func(ChatMessage) error) error
}

func (m *mockClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &ChatResponse{
		Message: ChatMessage{
			Role:    "assistant",
			Content: "mock response",
		},
	}, nil
}

func (m *mockClient) ChatStream(ctx context.Context, req ChatRequest, callback func(ChatMessage) error) error {
	if m.chatStreamFunc != nil {
		return m.chatStreamFunc(ctx, req, callback)
	}
	// Default: send one chunk
	return callback(ChatMessage{
		Role:    "assistant",
		Content: "mock stream chunk",
	})
}

func (m *mockClient) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Content: "mock generate"}, nil
}

func (m *mockClient) GenerateStream(ctx context.Context, req GenerateRequest, streamFunc func(chunk string) error) error {
	return streamFunc("mock")
}

func (m *mockClient) CreateEmbedding(ctx context.Context, req EmbeddingRequest) (*EmbeddingResponse, error) {
	return &EmbeddingResponse{Embedding: []float32{0.1, 0.2}}, nil
}

func (m *mockClient) Ping(ctx context.Context) error {
	return nil
}

func (m *mockClient) Close() error {
	return nil
}

func TestNewEinoChatModelAdapter(t *testing.T) {
	client := &mockClient{}
	adapter := NewEinoChatModelAdapter(client)

	if adapter == nil {
		t.Fatal("NewEinoChatModelAdapter() returned nil")
	}

	// Check that adapter implements model.ChatModel
	_, ok := adapter.(*EinoChatModelAdapter)
	if !ok {
		t.Error("NewEinoChatModelAdapter() did not return *EinoChatModelAdapter")
	}
}

func TestEinoChatModelAdapter_Generate(t *testing.T) {
	tests := []struct {
		name      string
		input     []*schema.Message
		mockFunc  func(ctx context.Context, req ChatRequest) (*ChatResponse, error)
		wantErr   bool
		wantRole  schema.RoleType
		wantEmpty bool
	}{
		{
			name: "successful generation",
			input: []*schema.Message{
				{
					Role:    schema.User,
					Content: "Hello",
				},
			},
			mockFunc: func(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
				return &ChatResponse{
					Message: ChatMessage{
						Role:    "assistant",
						Content: "Hi there!",
					},
				}, nil
			},
			wantErr:  false,
			wantRole: schema.Assistant,
		},
		{
			name:      "empty input messages",
			input:     []*schema.Message{},
			wantErr:   true,
			wantEmpty: true,
		},
		{
			name: "client error",
			input: []*schema.Message{
				{
					Role:    schema.User,
					Content: "Hello",
				},
			},
			mockFunc: func(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
				return nil, errors.New("client error")
			},
			wantErr: true,
		},
		{
			name: "multiple messages",
			input: []*schema.Message{
				{
					Role:    schema.System,
					Content: "You are a helpful assistant",
				},
				{
					Role:    schema.User,
					Content: "What is Go?",
				},
			},
			mockFunc: func(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
				if len(req.Messages) != 2 {
					t.Errorf("expected 2 messages, got %d", len(req.Messages))
				}
				return &ChatResponse{
					Message: ChatMessage{
						Role:    "assistant",
						Content: "Go is a programming language",
					},
				}, nil
			},
			wantErr:  false,
			wantRole: schema.Assistant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockClient{
				chatFunc: tt.mockFunc,
			}
			adapter := NewEinoChatModelAdapter(client)

			ctx := context.Background()
			result, err := adapter.Generate(ctx, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Generate() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Generate() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Fatal("Generate() returned nil result")
			}

			if result.Role != tt.wantRole {
				t.Errorf("Generate() role = %v, want %v", result.Role, tt.wantRole)
			}

			if result.Content == "" {
				t.Error("Generate() returned empty content")
			}
		})
	}
}

func TestEinoChatModelAdapter_Stream(t *testing.T) {
	tests := []struct {
		name       string
		input      []*schema.Message
		mockFunc   func(ctx context.Context, req ChatRequest, callback func(ChatMessage) error) error
		wantErr    bool
		wantChunks int
	}{
		{
			name: "successful streaming",
			input: []*schema.Message{
				{
					Role:    schema.User,
					Content: "Hello",
				},
			},
			mockFunc: func(ctx context.Context, req ChatRequest, callback func(ChatMessage) error) error {
				chunks := []string{"Hello", " there", "!"}
				for _, chunk := range chunks {
					if err := callback(ChatMessage{
						Role:    "assistant",
						Content: chunk,
					}); err != nil {
						return err
					}
				}
				return nil
			},
			wantErr:    false,
			wantChunks: 3,
		},
		{
			name:    "empty input messages",
			input:   []*schema.Message{},
			wantErr: true,
		},
		{
			name: "client stream error",
			input: []*schema.Message{
				{
					Role:    schema.User,
					Content: "Hello",
				},
			},
			mockFunc: func(ctx context.Context, req ChatRequest, callback func(ChatMessage) error) error {
				return errors.New("stream error")
			},
			wantErr: false, // Stream() itself doesn't error, error comes through pipe
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockClient{
				chatStreamFunc: tt.mockFunc,
			}
			adapter := NewEinoChatModelAdapter(client)

			ctx := context.Background()
			sr, err := adapter.Stream(ctx, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("Stream() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Stream() unexpected error: %v", err)
				return
			}

			if sr == nil {
				t.Fatal("Stream() returned nil StreamReader")
			}

			// Read chunks from stream
			chunks := 0
			for {
				msg, err := sr.Recv()
				if err != nil {
					break
				}
				if msg == nil {
					break
				}
				chunks++

				if msg.Role != schema.Assistant {
					t.Errorf("Stream() chunk role = %v, want %v", msg.Role, schema.Assistant)
				}
			}

			if tt.wantChunks > 0 && chunks != tt.wantChunks {
				t.Errorf("Stream() received %d chunks, want %d", chunks, tt.wantChunks)
			}
		})
	}
}

func TestEinoChatModelAdapter_BindTools(t *testing.T) {
	client := &mockClient{}
	adapter := NewEinoChatModelAdapter(client).(*EinoChatModelAdapter)

	tools := []*schema.ToolInfo{
		{
			Name: "test_tool",
			Desc: "A test tool",
		},
	}

	err := adapter.BindTools(tools)
	if err != nil {
		t.Errorf("BindTools() unexpected error: %v", err)
	}

	if len(adapter.tools) != 1 {
		t.Errorf("BindTools() stored %d tools, want 1", len(adapter.tools))
	}

	if adapter.tools[0].Name != "test_tool" {
		t.Errorf("BindTools() tool name = %v, want test_tool", adapter.tools[0].Name)
	}
}

func TestEinoChatModelAdapter_MessageConversion(t *testing.T) {
	client := &mockClient{
		chatFunc: func(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
			// Verify message conversion
			if len(req.Messages) != 2 {
				t.Errorf("expected 2 messages, got %d", len(req.Messages))
			}

			if req.Messages[0].Role != "system" {
				t.Errorf("first message role = %v, want system", req.Messages[0].Role)
			}
			if req.Messages[0].Content != "You are helpful" {
				t.Errorf("first message content = %v, want 'You are helpful'", req.Messages[0].Content)
			}

			if req.Messages[1].Role != "user" {
				t.Errorf("second message role = %v, want user", req.Messages[1].Role)
			}
			if req.Messages[1].Content != "Hello" {
				t.Errorf("second message content = %v, want 'Hello'", req.Messages[1].Content)
			}

			return &ChatResponse{
				Message: ChatMessage{
					Role:    "assistant",
					Content: "Response",
				},
			}, nil
		},
	}

	adapter := NewEinoChatModelAdapter(client)

	input := []*schema.Message{
		{
			Role:    schema.System,
			Content: "You are helpful",
		},
		{
			Role:    schema.User,
			Content: "Hello",
		},
	}

	ctx := context.Background()
	_, err := adapter.Generate(ctx, input)
	if err != nil {
		t.Errorf("Generate() unexpected error: %v", err)
	}
}
