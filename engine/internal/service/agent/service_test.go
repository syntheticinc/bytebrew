package agent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Mock implementations for testing

type mockChatModel struct{}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "mock response",
	}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()
		sw.Send(&schema.Message{
			Role:    schema.Assistant,
			Content: "mock stream response",
		}, nil)
	}()
	return sr, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func (m *mockChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func (m *mockChatModel) GetType() string {
	return "mock_chat_model"
}

func (m *mockChatModel) IsCallbacksEnabled() bool {
	return false
}

// Tests

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				ChatModel: &mockChatModel{},
				MaxSteps:  10,
			},
			wantErr: false,
		},
		{
			name: "missing chat model",
			config: Config{
				MaxSteps: 10,
			},
			wantErr: true,
		},
		{
			name: "zero max steps means no limit",
			config: Config{
				ChatModel: &mockChatModel{},
				MaxSteps:  0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && svc == nil {
				t.Error("New() returned nil service")
			}
			// MaxSteps = 0 means no limit, verify it's preserved as-is
			if !tt.wantErr && tt.config.MaxSteps == 0 && svc.maxSteps != 0 {
				t.Errorf("New() maxSteps = %v, want 0 (no limit)", svc.maxSteps)
			}
		})
	}
}


