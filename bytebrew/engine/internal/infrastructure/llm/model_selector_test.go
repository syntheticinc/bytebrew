package llm

import (
	"context"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
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

func TestModelSelector_NamedModels(t *testing.T) {
	defaultModel := &mockChatModel{id: "default"}

	t.Run("register and resolve named model", func(t *testing.T) {
		selector := NewModelSelector(defaultModel, "default-model")
		namedModel := &mockChatModel{id: "llama-4"}

		selector.RegisterNamedModel("llama-4", namedModel)

		got, err := selector.ResolveByName("llama-4")
		require.NoError(t, err)
		require.NotNil(t, got)

		resp, err := got.Generate(context.Background(), nil)
		require.NoError(t, err)
		assert.Equal(t, "llama-4", resp.Content)
	})

	t.Run("resolve unknown name returns error", func(t *testing.T) {
		selector := NewModelSelector(defaultModel, "default-model")

		got, err := selector.ResolveByName("nonexistent")
		require.Error(t, err)
		assert.Nil(t, got)
		assert.Contains(t, err.Error(), "nonexistent")
	})

	t.Run("multiple named models", func(t *testing.T) {
		selector := NewModelSelector(defaultModel, "default-model")
		model1 := &mockChatModel{id: "model-a"}
		model2 := &mockChatModel{id: "model-b"}

		selector.RegisterNamedModel("model-a", model1)
		selector.RegisterNamedModel("model-b", model2)

		assert.Equal(t, 2, selector.NamedModelCount())

		gotA, err := selector.ResolveByName("model-a")
		require.NoError(t, err)
		respA, _ := gotA.Generate(context.Background(), nil)
		assert.Equal(t, "model-a", respA.Content)

		gotB, err := selector.ResolveByName("model-b")
		require.NoError(t, err)
		respB, _ := gotB.Generate(context.Background(), nil)
		assert.Equal(t, "model-b", respB.Content)
	})

	t.Run("overwrite named model", func(t *testing.T) {
		selector := NewModelSelector(defaultModel, "default-model")
		original := &mockChatModel{id: "v1"}
		replacement := &mockChatModel{id: "v2"}

		selector.RegisterNamedModel("my-model", original)
		selector.RegisterNamedModel("my-model", replacement)

		got, err := selector.ResolveByName("my-model")
		require.NoError(t, err)
		resp, _ := got.Generate(context.Background(), nil)
		assert.Equal(t, "v2", resp.Content)
	})

	t.Run("named model count initially zero", func(t *testing.T) {
		selector := NewModelSelector(defaultModel, "default-model")
		assert.Equal(t, 0, selector.NamedModelCount())
	})
}
