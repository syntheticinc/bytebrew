package llm

import (
	"context"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockChatModel implements model.ToolCallingChatModel for testing
type mockChatModel struct {
	id string // used to distinguish between models in tests
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return &schema.Message{Role: schema.Assistant, Content: m.id}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](1)
	sw.Close()
	return sr, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func (m *mockChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func TestModelSelector_Select(t *testing.T) {
	defaultModel := &mockChatModel{id: "default"}
	coderModel := &mockChatModel{id: "coder"}

	tests := []struct {
		name     string
		flowType domain.FlowType
		setup    func(s *ModelSelector)
		wantID   string
	}{
		{
			name:     "default returned when no override",
			flowType: domain.FlowType("supervisor"),
			setup:    func(s *ModelSelector) {},
			wantID:   "default",
		},
		{
			name:     "default returned for unregistered flow type",
			flowType: domain.FlowType("reviewer"),
			setup: func(s *ModelSelector) {
				s.SetModel(domain.FlowType("coder"), coderModel, "coder-model")
			},
			wantID: "default",
		},
		{
			name:     "override returned when set",
			flowType: domain.FlowType("coder"),
			setup: func(s *ModelSelector) {
				s.SetModel(domain.FlowType("coder"), coderModel, "coder-model")
			},
			wantID: "coder",
		},
		{
			name:     "each flow type gets its own model",
			flowType: domain.FlowType("supervisor"),
			setup: func(s *ModelSelector) {
				supervisorModel := &mockChatModel{id: "supervisor"}
				s.SetModel(domain.FlowType("supervisor"), supervisorModel, "supervisor-model")
				s.SetModel(domain.FlowType("coder"), coderModel, "coder-model")
			},
			wantID: "supervisor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewModelSelector(defaultModel, "default-model")
			tt.setup(selector)

			got := selector.Select(tt.flowType)
			require.NotNil(t, got)

			// Verify by generating a response (mockChatModel returns its id as content)
			resp, err := got.Generate(context.Background(), nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantID, resp.Content)
		})
	}
}

func TestModelSelector_ModelName(t *testing.T) {
	defaultModel := &mockChatModel{id: "default"}
	coderModel := &mockChatModel{id: "coder"}

	tests := []struct {
		name     string
		flowType domain.FlowType
		setup    func(s *ModelSelector)
		want     string
	}{
		{
			name:     "default name returned when no override",
			flowType: domain.FlowType("supervisor"),
			setup:    func(s *ModelSelector) {},
			want:     "default-model",
		},
		{
			name:     "default name returned for unregistered flow type",
			flowType: domain.FlowType("reviewer"),
			setup: func(s *ModelSelector) {
				s.SetModel(domain.FlowType("coder"), coderModel, "coder-model")
			},
			want: "default-model",
		},
		{
			name:     "override name returned when set",
			flowType: domain.FlowType("coder"),
			setup: func(s *ModelSelector) {
				s.SetModel(domain.FlowType("coder"), coderModel, "coder-model")
			},
			want: "coder-model",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			selector := NewModelSelector(defaultModel, "default-model")
			tt.setup(selector)

			got := selector.ModelName(tt.flowType)
			assert.Equal(t, tt.want, got)
		})
	}
}
