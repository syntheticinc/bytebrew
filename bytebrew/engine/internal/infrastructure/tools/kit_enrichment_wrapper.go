package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/kit"
)

// kitEnrichmentWrapper wraps a tool to append kit enrichment after execution.
// Used for tools like edit_file, write_file where the kit needs to inject
// additional context (e.g., LSP diagnostics after code changes).
type kitEnrichmentWrapper struct {
	inner   tool.InvokableTool
	kit     kit.Kit
	session domain.KitSession
}

// NewKitEnrichmentWrapper creates a wrapper that calls kit.PostToolCall after
// the inner tool succeeds, appending any enrichment to the result.
func NewKitEnrichmentWrapper(inner tool.InvokableTool, k kit.Kit, session domain.KitSession) tool.InvokableTool {
	return &kitEnrichmentWrapper{inner: inner, kit: k, session: session}
}

func (w *kitEnrichmentWrapper) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return w.inner.Info(ctx)
}

func (w *kitEnrichmentWrapper) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	result, err := w.inner.InvokableRun(ctx, argumentsInJSON, opts...)
	if err != nil {
		return result, err
	}

	info, infoErr := w.inner.Info(ctx)
	if infoErr != nil {
		return result, nil
	}

	enrichment := w.kit.PostToolCall(ctx, w.session, info.Name, result)
	if enrichment != nil && enrichment.Content != "" {
		result = result + "\n\n" + enrichment.Content
	}

	return result, nil
}
