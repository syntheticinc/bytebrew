package kit

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// Kit is a domain-specific module that provides session-level state,
// additional tools, and automatic post-tool-call hooks.
type Kit interface {
	Name() string
	OnSessionStart(ctx context.Context, session domain.KitSession) error
	OnSessionEnd(ctx context.Context, session domain.KitSession) error
	Tools(session domain.KitSession) []tool.InvokableTool
	PostToolCall(ctx context.Context, session domain.KitSession, toolName string, result string) *domain.Enrichment
}
