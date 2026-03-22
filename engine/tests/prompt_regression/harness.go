//go:build prompt

package prompt_regression

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agents"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

const defaultConfigPath = `C:\Users\busul\GolandProjects\usm-epicsmasher\bytebrew-srv\config.yaml`

// Harness provides test infrastructure for prompt regression tests
type Harness struct {
	chatModel model.ToolCallingChatModel
}

// NewHarness creates a new test harness
func NewHarness() (*Harness, error) {
	// Load config from default path
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// Only support openrouter provider for now
	if cfg.LLM.DefaultProvider != "openrouter" {
		return nil, fmt.Errorf("unsupported provider: %s (only openrouter supported)", cfg.LLM.DefaultProvider)
	}

	// Create ChatModel (similar to agent_service.go:67-79)
	ctx := context.Background()
	orCfg := &openai.ChatModelConfig{
		BaseURL: cfg.LLM.OpenRouter.BaseURL,
		Model:   cfg.LLM.OpenRouter.Model,
		APIKey:  cfg.LLM.OpenRouter.APIKey,
	}
	if len(cfg.LLM.OpenRouter.Provider) > 0 {
		orCfg.ExtraFields = map[string]any{
			"provider": cfg.LLM.OpenRouter.Provider,
		}
	}

	chatModel, err := openai.NewChatModel(ctx, orCfg)
	if err != nil {
		return nil, fmt.Errorf("create chat model: %w", err)
	}

	return &Harness{
		chatModel: chatModel,
	}, nil
}

// BindSupervisorTools binds supervisor tools to the chat model
func (h *Harness) BindSupervisorTools(ctx context.Context) error {
	schemas, err := getToolSchemas(ctx, supervisorToolNames)
	if err != nil {
		return fmt.Errorf("get tool schemas: %w", err)
	}

	// WithTools returns a new instance with tools bound
	modelWithTools, err := h.chatModel.WithTools(schemas)
	if err != nil {
		return fmt.Errorf("bind tools: %w", err)
	}

	h.chatModel = modelWithTools
	slog.InfoContext(ctx, "bound supervisor tools", "count", len(schemas))
	return nil
}

// ReconstructMessages reconstructs schema.Message array from context snapshot
func (h *Harness) ReconstructMessages(snapshot *agents.ContextSnapshot, systemPromptOverride string) []*schema.Message {
	messages := make([]*schema.Message, 0, len(snapshot.Messages))

	for i, msgInfo := range snapshot.Messages {
		msg := &schema.Message{
			Role:    schema.RoleType(msgInfo.Role),
			Content: msgInfo.Content,
		}

		// Override system prompt if provided
		if i == 0 && msg.Role == schema.System && systemPromptOverride != "" {
			msg.Content = systemPromptOverride
		}

		// Add tool-specific fields
		if msg.Role == schema.Tool {
			msg.Name = msgInfo.ToolName
			msg.ToolCallID = msgInfo.ToolCallID
		}

		// Reconstruct tool calls for assistant messages
		if msg.Role == schema.Assistant && len(msgInfo.ToolCalls) > 0 {
			msg.ToolCalls = make([]schema.ToolCall, 0, len(msgInfo.ToolCalls))
			for _, tcInfo := range msgInfo.ToolCalls {
				tc := schema.ToolCall{
					ID:    tcInfo.ID,
					Index: tcInfo.Index,
					Function: schema.FunctionCall{
						Name:      tcInfo.Name,
						Arguments: tcInfo.Arguments,
					},
				}
				msg.ToolCalls = append(msg.ToolCalls, tc)
			}
		}

		messages = append(messages, msg)
	}

	return messages
}

// Generate sends messages to LLM and returns response
func (h *Harness) Generate(ctx context.Context, msgs []*schema.Message) (*schema.Message, error) {
	response, err := h.chatModel.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	return response, nil
}

// GeneratePlain sends messages to LLM without tools bound (for judge evaluation)
func (h *Harness) GeneratePlain(ctx context.Context, msgs []*schema.Message) (*schema.Message, error) {
	plainHarness, err := NewHarness()
	if err != nil {
		return nil, fmt.Errorf("create plain harness: %w", err)
	}
	return plainHarness.chatModel.Generate(ctx, msgs)
}

// LoadCurrentSupervisorPrompt loads the current supervisor prompt from prompts.yaml
func LoadCurrentSupervisorPrompt() (string, error) {
	cfg, err := config.Load(defaultConfigPath)
	if err != nil {
		return "", fmt.Errorf("load config: %w", err)
	}
	if cfg.Agent.Prompts == nil || cfg.Agent.Prompts.SupervisorPrompt == "" {
		return "", fmt.Errorf("supervisor_prompt is empty in prompts.yaml")
	}
	return cfg.Agent.Prompts.SupervisorPrompt, nil
}
