package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Compile-time interface check.
var _ model.ToolCallingChatModel = (*AzureOpenAIChatModel)(nil)

const defaultAzureAPIVersion = "2024-10-21"

// AzureOpenAIChatModel wraps the Eino OpenAI library configured for Azure OpenAI Service.
//
// Azure OpenAI differs from standard OpenAI in URL structure and authentication:
//   - Base URL: https://{resource}.openai.azure.com/openai/deployments/{deployment}
//   - Auth: api-key header instead of Bearer token
//   - API Version: required query parameter (e.g. 2024-10-21)
type AzureOpenAIChatModel struct {
	inner      model.ToolCallingChatModel
	baseURL    string
	modelName  string
	apiVersion string
}

// AzureOpenAIOption configures an AzureOpenAIChatModel.
type AzureOpenAIOption func(*azureOpenAIConfig)

type azureOpenAIConfig struct {
	httpClient *http.Client
}

// WithAzureHTTPClient sets a custom HTTP client for the Azure OpenAI client.
func WithAzureHTTPClient(client *http.Client) AzureOpenAIOption {
	return func(c *azureOpenAIConfig) {
		c.httpClient = client
	}
}

// NewAzureOpenAIChatModel creates a new Azure OpenAI chat model.
//
// Parameters:
//   - baseURL: Azure OpenAI endpoint (e.g. "https://myresource.openai.azure.com")
//   - apiKey: Azure API key
//   - modelName: deployment name (e.g. "gpt-4o")
//   - apiVersion: API version (empty defaults to "2024-10-21")
func NewAzureOpenAIChatModel(baseURL, apiKey, modelName, apiVersion string, opts ...AzureOpenAIOption) (*AzureOpenAIChatModel, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("azure openai: base_url is required")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("azure openai: api_key is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("azure openai: model_name (deployment name) is required")
	}

	if apiVersion == "" {
		apiVersion = defaultAzureAPIVersion
	}

	cfg := &azureOpenAIConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	einoCfg := &openai.ChatModelConfig{
		ByAzure:    true,
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Model:      modelName,
		APIKey:     apiKey,
		APIVersion: apiVersion,
	}
	if cfg.httpClient != nil {
		einoCfg.HTTPClient = cfg.httpClient
	}

	inner, err := openai.NewChatModel(context.Background(), einoCfg)
	if err != nil {
		return nil, fmt.Errorf("azure openai: create client: %w", err)
	}

	return &AzureOpenAIChatModel{
		inner:      inner,
		baseURL:    baseURL,
		modelName:  modelName,
		apiVersion: apiVersion,
	}, nil
}

// Generate sends a non-streaming request to Azure OpenAI.
func (a *AzureOpenAIChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return a.inner.Generate(ctx, input, opts...)
}

// Stream sends a streaming request to Azure OpenAI.
func (a *AzureOpenAIChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return a.inner.Stream(ctx, input, opts...)
}

// WithTools returns a copy of AzureOpenAIChatModel with the given tools attached.
func (a *AzureOpenAIChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	if len(tools) == 0 {
		return &AzureOpenAIChatModel{
			inner:      a.inner,
			baseURL:    a.baseURL,
			modelName:  a.modelName,
			apiVersion: a.apiVersion,
		}, nil
	}
	newInner, err := a.inner.WithTools(tools)
	if err != nil {
		return nil, fmt.Errorf("azure openai: bind tools: %w", err)
	}
	return &AzureOpenAIChatModel{
		inner:      newInner,
		baseURL:    a.baseURL,
		modelName:  a.modelName,
		apiVersion: a.apiVersion,
	}, nil
}

// Ping verifies connectivity by sending a minimal chat request.
func (a *AzureOpenAIChatModel) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := a.inner.Generate(ctx, []*schema.Message{
		{Role: schema.User, Content: "Hi"},
	})
	if err != nil {
		return fmt.Errorf("azure openai ping: %w", err)
	}
	return nil
}

// BaseURL returns the configured Azure endpoint.
func (a *AzureOpenAIChatModel) BaseURL() string {
	return a.baseURL
}

// ModelName returns the deployment name.
func (a *AzureOpenAIChatModel) ModelName() string {
	return a.modelName
}

// APIVersion returns the configured API version.
func (a *AzureOpenAIChatModel) APIVersion() string {
	return a.apiVersion
}
