package infrastructure

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/config"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	ollamaapi "github.com/eino-contrib/ollama/api"
)

// createChatModel creates a ToolCallingChatModel based on provider config.
func createChatModel(cfg config.Config) (model.ToolCallingChatModel, error) {
	ctx := context.Background()

	switch cfg.LLM.DefaultProvider {
	case "openrouter":
		return createOpenRouterModel(ctx, cfg.LLM.OpenRouter)
	case "ollama":
		return createOllamaModel(ctx, cfg.LLM.Ollama)
	case "anthropic":
		return createAnthropicModel(ctx, cfg.LLM.Anthropic)
	default:
		return nil, errors.New(errors.CodeInvalidInput, "unsupported LLM provider: "+cfg.LLM.DefaultProvider)
	}
}

func createOpenRouterModel(ctx context.Context, cfg config.OpenRouterConfig) (model.ToolCallingChatModel, error) {
	orCfg := &openai.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
		APIKey:  cfg.APIKey,
	}
	if len(cfg.Provider) > 0 {
		orCfg.ExtraFields = map[string]any{
			"provider": cfg.Provider,
		}
	}
	chatModel, err := openai.NewChatModel(ctx, orCfg)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to create openrouter chat model")
	}
	return chatModel, nil
}

func createOllamaModel(ctx context.Context, cfg config.OllamaConfig) (model.ToolCallingChatModel, error) {
	ollamaCfg := &ollama.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		Model:   cfg.Model,
	}
	if cfg.Thinking {
		thinking := ollamaapi.ThinkValue{Value: true}
		ollamaCfg.Thinking = &thinking
	}
	chatModel, err := ollama.NewChatModel(ctx, ollamaCfg)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to create ollama chat model")
	}
	return chatModel, nil
}

func createAnthropicModel(ctx context.Context, cfg config.AnthropicConfig) (model.ToolCallingChatModel, error) {
	baseURL := "https://api.anthropic.com/v1"
	if cfg.BaseURL != "" {
		baseURL = cfg.BaseURL
	}

	httpClient := &http.Client{Timeout: cfg.Timeout}
	httpClient.Transport = &anthropicTransport{
		base: http.DefaultTransport,
	}

	anthropicCfg := &openai.ChatModelConfig{
		BaseURL:    baseURL,
		Model:      cfg.Model,
		APIKey:     cfg.APIKey,
		HTTPClient: httpClient,
	}
	chatModel, err := openai.NewChatModel(ctx, anthropicCfg)
	if err != nil {
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to create anthropic model")
	}
	return chatModel, nil
}

// anthropicTransport adds the required anthropic-version header to all requests.
type anthropicTransport struct {
	base http.RoundTripper
}

func (t *anthropicTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("anthropic-version", "2023-06-01")
	return t.base.RoundTrip(req)
}

// wrapWithDebugModel wraps the chat model with debug logging if BYTEBREW_DEBUG_MODEL is set.
func wrapWithDebugModel(chatModel model.ToolCallingChatModel) model.ToolCallingChatModel {
	debugDir := os.Getenv("BYTEBREW_DEBUG_MODEL")
	if debugDir == "" {
		return chatModel
	}
	slog.Info("debug model wrapper enabled", "log_dir", debugDir)
	return llm.NewDebugChatModelWrapper(chatModel, debugDir, "global")
}

// createModelSelector creates a ModelSelector via ProviderResolver.
func createModelSelector(cfg config.Config, chatModel model.ToolCallingChatModel, modelName string) *llm.ModelSelector {
	return llm.ResolveModelSelector(llm.ProviderResolverConfig{
		Mode:          cfg.Provider.Mode,
		CloudAPIURL:   cfg.Provider.CloudAPIURL,
		AccessToken:   "", // TODO: populate from auth storage in a future phase
		BYOKModel:     chatModel,
		BYOKModelName: modelName,
	})
}

// getModelName returns model name based on LLM provider config.
func getModelName(cfg config.Config) string {
	switch cfg.LLM.DefaultProvider {
	case "openrouter":
		return cfg.LLM.OpenRouter.Model
	case "ollama":
		return cfg.LLM.Ollama.Model
	case "anthropic":
		return cfg.LLM.Anthropic.Model
	default:
		return cfg.LLM.Ollama.Model
	}
}
