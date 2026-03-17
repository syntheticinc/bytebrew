package llm

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Compile-time interface check.
var _ model.ToolCallingChatModel = (*AutoChatModel)(nil)

// AutoChatModel wraps a proxy and a BYOK model. It tries the proxy first and
// falls back to the BYOK model when the proxy returns HTTP 402 (quota
// exhausted) or 429 (rate limited).
type AutoChatModel struct {
	proxy model.ToolCallingChatModel
	byok  model.ToolCallingChatModel
}

// NewAutoChatModel creates an AutoChatModel that delegates to proxy first,
// falling back to byok on quota/rate-limit errors.
func NewAutoChatModel(proxy, byok model.ToolCallingChatModel) *AutoChatModel {
	return &AutoChatModel{
		proxy: proxy,
		byok:  byok,
	}
}

// Generate tries proxy.Generate; on quota/rate-limit errors it falls back to byok.
func (a *AutoChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	msg, err := a.proxy.Generate(ctx, input, opts...)
	if err == nil {
		return msg, nil
	}

	if !isProxyFallbackError(err) {
		return nil, fmt.Errorf("proxy generate: %w", err)
	}

	slog.InfoContext(ctx, "proxy fallback to BYOK", "reason", err)
	return a.byok.Generate(ctx, input, opts...)
}

// Stream tries proxy.Stream; on quota/rate-limit errors it falls back to byok.
func (a *AutoChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	reader, err := a.proxy.Stream(ctx, input, opts...)
	if err == nil {
		return reader, nil
	}

	if !isProxyFallbackError(err) {
		return nil, fmt.Errorf("proxy stream: %w", err)
	}

	slog.InfoContext(ctx, "proxy stream fallback to BYOK", "reason", err)
	return a.byok.Stream(ctx, input, opts...)
}

// WithTools creates a new AutoChatModel where both proxy and byok have the
// given tools bound.
func (a *AutoChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	newProxy, err := a.proxy.WithTools(tools)
	if err != nil {
		return nil, fmt.Errorf("auto proxy with tools: %w", err)
	}

	newByok, err := a.byok.WithTools(tools)
	if err != nil {
		return nil, fmt.Errorf("auto byok with tools: %w", err)
	}

	return &AutoChatModel{
		proxy: newProxy,
		byok:  newByok,
	}, nil
}

// isProxyFallbackError returns true for errors that should trigger a fallback
// to the BYOK model.
func isProxyFallbackError(err error) bool {
	return errors.Is(err, ErrQuotaExhausted) || errors.Is(err, ErrRateLimited)
}
