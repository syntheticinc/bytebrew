package llm

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockVerifyModel struct {
	generateFunc func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error)
	streamFunc   func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error)
	withToolsErr error
	toolResponse *schema.Message
	toolErr      error
}

func (m *mockVerifyModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, input, opts...)
	}
	return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
}

func (m *mockVerifyModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	if m.streamFunc != nil {
		return m.streamFunc(ctx, input, opts...)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockVerifyModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func (m *mockVerifyModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	if m.withToolsErr != nil {
		return nil, m.withToolsErr
	}
	// Return a new mock that returns toolResponse
	return &mockVerifyModel{
		generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
			if m.toolErr != nil {
				return nil, m.toolErr
			}
			if m.toolResponse != nil {
				return m.toolResponse, nil
			}
			return &schema.Message{Role: schema.Assistant, Content: "4"}, nil
		},
	}, nil
}

func TestVerifyModel(t *testing.T) {
	tests := []struct {
		name             string
		provider         string
		mock             *mockVerifyModel
		wantConnectivity string
		wantToolCalling  string
		wantError        bool
	}{
		{
			name:     "successful ping with known provider skips tool probe",
			provider: "openai",
			mock: &mockVerifyModel{
				generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
					return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
				},
			},
			wantConnectivity: "ok",
			wantToolCalling:  "skipped",
			wantError:        false,
		},
		{
			name:     "successful ping with anthropic skips tool probe",
			provider: "anthropic",
			mock: &mockVerifyModel{
				generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
					return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
				},
			},
			wantConnectivity: "ok",
			wantToolCalling:  "skipped",
			wantError:        false,
		},
		{
			name:     "ping failure returns error",
			provider: "ollama",
			mock: &mockVerifyModel{
				generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
					return nil, fmt.Errorf("connection refused")
				},
			},
			wantConnectivity: "error",
			wantToolCalling:  "skipped",
			wantError:        true,
		},
		{
			name:     "unknown provider with tool support detected",
			provider: "ollama",
			mock: &mockVerifyModel{
				generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
					return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
				},
				toolResponse: &schema.Message{
					Role: schema.Assistant,
					ToolCalls: []schema.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: schema.FunctionCall{
								Name:      "calculator",
								Arguments: `{"expression":"2+2"}`,
							},
						},
					},
				},
			},
			wantConnectivity: "ok",
			wantToolCalling:  "supported",
			wantError:        false,
		},
		{
			name:     "unknown provider without tool support",
			provider: "ollama",
			mock: &mockVerifyModel{
				generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
					return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
				},
				toolResponse: &schema.Message{
					Role:    schema.Assistant,
					Content: "The answer is 4",
				},
			},
			wantConnectivity: "ok",
			wantToolCalling:  "not_detected",
			wantError:        false,
		},
		{
			name:     "WithTools fails returns error for tool_calling",
			provider: "ollama",
			mock: &mockVerifyModel{
				generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
					return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
				},
				withToolsErr: fmt.Errorf("tools not supported"),
			},
			wantConnectivity: "ok",
			wantToolCalling:  "error",
			wantError:        false,
		},
		{
			name:     "tool probe generate fails returns error for tool_calling",
			provider: "ollama",
			mock: &mockVerifyModel{
				generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
					return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
				},
				toolErr: fmt.Errorf("tool call failed"),
			},
			wantConnectivity: "ok",
			wantToolCalling:  "error",
			wantError:        false,
		},
		{
			name:     "google provider skips tool probe",
			provider: "google",
			mock: &mockVerifyModel{
				generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
					return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
				},
			},
			wantConnectivity: "ok",
			wantToolCalling:  "skipped",
			wantError:        false,
		},
		{
			name:     "mistral provider skips tool probe",
			provider: "mistral",
			mock: &mockVerifyModel{
				generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
					return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
				},
			},
			wantConnectivity: "ok",
			wantToolCalling:  "skipped",
			wantError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result := VerifyModel(ctx, tt.mock, "test-model", tt.provider)

			require.NotNil(t, result)
			assert.Equal(t, tt.wantConnectivity, result.Connectivity)
			assert.Equal(t, tt.wantToolCalling, result.ToolCalling)
			assert.Equal(t, "test-model", result.ModelName)
			assert.Equal(t, tt.provider, result.Provider)

			if tt.wantError {
				assert.NotNil(t, result.Error)
			} else {
				assert.Nil(t, result.Error)
			}

			if tt.wantConnectivity == "ok" {
				assert.GreaterOrEqual(t, result.ResponseTimeMs, int64(0))
			}
		})
	}
}

func TestVerifyModel_ResponseTimeMeasured(t *testing.T) {
	mock := &mockVerifyModel{
		generateFunc: func(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
			return &schema.Message{Role: schema.Assistant, Content: "Hi!"}, nil
		},
	}

	result := VerifyModel(context.Background(), mock, "model", "openai")
	assert.Equal(t, "ok", result.Connectivity)
	assert.GreaterOrEqual(t, result.ResponseTimeMs, int64(0))
}
