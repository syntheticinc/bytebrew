package developer

import (
	"context"
	"log/slog"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

// DeveloperKit provides LSP integration and code indexing for coding agents.
// Implements infrastructure.Kit interface.
type DeveloperKit struct {
	sessions map[string]*devSession
	mu       sync.RWMutex
}

type devSession struct {
	kitSession domain.KitSession
	// In future: lspService, indexer, watcher, chunkStore, embedder
	// For now: tools are resolved from existing infrastructure
}

// New creates a new DeveloperKit.
func New() *DeveloperKit {
	return &DeveloperKit{
		sessions: make(map[string]*devSession),
	}
}

// Name returns the kit identifier.
func (k *DeveloperKit) Name() string { return "developer" }

// OnSessionStart initializes session-level state for the developer kit.
func (k *DeveloperKit) OnSessionStart(ctx context.Context, session domain.KitSession) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.sessions[session.SessionID] = &devSession{
		kitSession: session,
	}
	slog.InfoContext(ctx, "developer kit: session started",
		"session_id", session.SessionID,
		"project_root", session.ProjectRoot,
	)
	return nil
}

// OnSessionEnd cleans up session-level state for the developer kit.
func (k *DeveloperKit) OnSessionEnd(ctx context.Context, session domain.KitSession) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	delete(k.sessions, session.SessionID)
	slog.InfoContext(ctx, "developer kit: session ended",
		"session_id", session.SessionID,
	)
	return nil
}

// Tools returns kit-specific tools for the given session.
// Phase 3 skeleton: developer-specific tools (LSP, indexing) are still
// resolved through builtin_factories.go. When code is physically moved
// to kits/developer/ (tasks 3.5-3.7), these tools will be returned here.
func (k *DeveloperKit) Tools(_ domain.KitSession) []tool.InvokableTool {
	return nil
}

// PostToolCall is called after a tool execution to provide enrichment.
// Phase 3 skeleton: LSP diagnostics enrichment will be implemented
// when LSP code is moved to kits/developer/.
func (k *DeveloperKit) PostToolCall(_ context.Context, _ domain.KitSession, toolName string, _ string) *domain.Enrichment {
	// Only enrich file-modifying tools
	if toolName != "edit_file" && toolName != "write_file" {
		return nil
	}

	// Skeleton: no enrichment yet (existing behavior preserved)
	return nil
}

// HasSession reports whether a session is tracked by this kit.
func (k *DeveloperKit) HasSession(sessionID string) bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	_, ok := k.sessions[sessionID]
	return ok
}
