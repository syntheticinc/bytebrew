package developer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

func TestDeveloperKit_Name(t *testing.T) {
	kit := New()
	assert.Equal(t, "developer", kit.Name())
}

func TestDeveloperKit_OnSessionStart(t *testing.T) {
	kit := New()
	ctx := context.Background()
	session := domain.KitSession{
		SessionID:   "sess-1",
		ProjectRoot: "/tmp/project",
		ProjectKey:  "proj-1",
	}

	err := kit.OnSessionStart(ctx, session)
	require.NoError(t, err)
	assert.True(t, kit.HasSession("sess-1"))
}

func TestDeveloperKit_OnSessionEnd(t *testing.T) {
	kit := New()
	ctx := context.Background()
	session := domain.KitSession{
		SessionID:   "sess-1",
		ProjectRoot: "/tmp/project",
		ProjectKey:  "proj-1",
	}

	err := kit.OnSessionStart(ctx, session)
	require.NoError(t, err)
	assert.True(t, kit.HasSession("sess-1"))

	err = kit.OnSessionEnd(ctx, session)
	require.NoError(t, err)
	assert.False(t, kit.HasSession("sess-1"))
}

func TestDeveloperKit_OnSessionEnd_NotTracked(t *testing.T) {
	kit := New()
	ctx := context.Background()
	session := domain.KitSession{SessionID: "nonexistent"}

	err := kit.OnSessionEnd(ctx, session)
	require.NoError(t, err)
}

func TestDeveloperKit_Tools_ReturnsNil(t *testing.T) {
	kit := New()
	session := domain.KitSession{
		SessionID:   "sess-1",
		ProjectRoot: "/tmp/project",
	}

	tools := kit.Tools(session)
	assert.Nil(t, tools)
}

func TestDeveloperKit_PostToolCall_NonFileTool(t *testing.T) {
	kit := New()
	ctx := context.Background()
	session := domain.KitSession{SessionID: "sess-1"}

	enrichment := kit.PostToolCall(ctx, session, "read_file", "content")
	assert.Nil(t, enrichment)
}

func TestDeveloperKit_PostToolCall_EditFile(t *testing.T) {
	kit := New()
	ctx := context.Background()
	session := domain.KitSession{SessionID: "sess-1"}

	// Skeleton: returns nil even for edit_file (no LSP yet)
	enrichment := kit.PostToolCall(ctx, session, "edit_file", "ok")
	assert.Nil(t, enrichment)
}

func TestDeveloperKit_PostToolCall_WriteFile(t *testing.T) {
	kit := New()
	ctx := context.Background()
	session := domain.KitSession{SessionID: "sess-1"}

	// Skeleton: returns nil even for write_file (no LSP yet)
	enrichment := kit.PostToolCall(ctx, session, "write_file", "ok")
	assert.Nil(t, enrichment)
}

func TestDeveloperKit_MultipleSessions(t *testing.T) {
	kit := New()
	ctx := context.Background()

	s1 := domain.KitSession{SessionID: "sess-1", ProjectRoot: "/proj1"}
	s2 := domain.KitSession{SessionID: "sess-2", ProjectRoot: "/proj2"}

	require.NoError(t, kit.OnSessionStart(ctx, s1))
	require.NoError(t, kit.OnSessionStart(ctx, s2))
	assert.True(t, kit.HasSession("sess-1"))
	assert.True(t, kit.HasSession("sess-2"))

	require.NoError(t, kit.OnSessionEnd(ctx, s1))
	assert.False(t, kit.HasSession("sess-1"))
	assert.True(t, kit.HasSession("sess-2"))
}
