package llm

import (
	"context"

	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	openai "github.com/cloudwego/eino-ext/components/model/openai"
)

// ModelParams holds per-agent LLM parameter overrides.
type ModelParams struct {
	Temperature *float64
	TopP        *float64
	MaxTokens   *int
	Stop        []string
}

// IsEmpty returns true if no overrides are set.
func (p ModelParams) IsEmpty() bool {
	return p.Temperature == nil && p.TopP == nil && p.MaxTokens == nil && len(p.Stop) == 0
}

// modelParamsWrapper wraps a ToolCallingChatModel and injects per-agent
// model parameters via WithExtraFields on every Generate/Stream call.
type modelParamsWrapper struct {
	inner  model.ToolCallingChatModel
	params ModelParams
}

// WrapWithModelParams wraps a ChatModel with per-agent parameter overrides.
// Returns the original model unchanged if params are empty.
func WrapWithModelParams(m model.ToolCallingChatModel, params ModelParams) model.ToolCallingChatModel {
	if params.IsEmpty() {
		return m
	}
	return &modelParamsWrapper{inner: m, params: params}
}

func (w *modelParamsWrapper) extraFieldsOption() model.Option {
	fields := make(map[string]any)
	if w.params.Temperature != nil {
		fields["temperature"] = *w.params.Temperature
	}
	if w.params.TopP != nil {
		fields["top_p"] = *w.params.TopP
	}
	if w.params.MaxTokens != nil {
		fields["max_tokens"] = *w.params.MaxTokens
	}
	if len(w.params.Stop) > 0 {
		fields["stop"] = w.params.Stop
	}
	return openai.WithExtraFields(fields)
}

func (w *modelParamsWrapper) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return w.inner.Generate(ctx, input, append(opts, w.extraFieldsOption())...)
}

func (w *modelParamsWrapper) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return w.inner.Stream(ctx, input, append(opts, w.extraFieldsOption())...)
}

func (w *modelParamsWrapper) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	inner, err := w.inner.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &modelParamsWrapper{inner: inner, params: w.params}, nil
}

// IsCallbacksEnabled forwards the inner model's callback aspect status.
// Without this, eino's components.IsCallbacksEnabled type-assertion fails on the
// wrapper and the framework auto-injects an aspect on top of the inner model's
// own manual callbacks.OnEndWithStreamOutput dispatch — producing every chunk
// twice on the SSE wire.
func (w *modelParamsWrapper) IsCallbacksEnabled() bool {
	return components.IsCallbacksEnabled(w.inner)
}
