package tools_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/kit"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
)

// --- mock kit ---

type enrichmentMockKit struct {
	name       string
	enrichment *domain.Enrichment
}

func (k *enrichmentMockKit) Name() string { return k.name }
func (k *enrichmentMockKit) OnSessionStart(_ context.Context, _ domain.KitSession) error {
	return nil
}
func (k *enrichmentMockKit) OnSessionEnd(_ context.Context, _ domain.KitSession) error {
	return nil
}
func (k *enrichmentMockKit) Tools(_ domain.KitSession) []tool.InvokableTool { return nil }
func (k *enrichmentMockKit) PostToolCall(_ context.Context, _ domain.KitSession, _ string, _ string) *domain.Enrichment {
	return k.enrichment
}

// --- mock inner tool ---

type mockInnerTool struct {
	result string
	err    error
}

func (t *mockInnerTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "edit_file", Desc: "Edit a file"}, nil
}

func (t *mockInnerTool) InvokableRun(_ context.Context, _ string, _ ...tool.Option) (string, error) {
	return t.result, t.err
}

// --- tests ---

func TestKitEnrichmentWrapper_AppendsEnrichment(t *testing.T) {
	inner := &mockInnerTool{result: "file saved"}
	kit := &enrichmentMockKit{
		name:       "developer",
		enrichment: &domain.Enrichment{Content: "2 diagnostics found"},
	}
	session := domain.KitSession{SessionID: "s1", ProjectRoot: "/tmp"}

	wrapped := tools.NewKitEnrichmentWrapper(inner, kit, session)

	result, err := wrapped.InvokableRun(context.Background(), `{}`)
	require.NoError(t, err)
	assert.Equal(t, "file saved\n\n2 diagnostics found", result)
}

func TestKitEnrichmentWrapper_NoEnrichment(t *testing.T) {
	inner := &mockInnerTool{result: "file saved"}
	kit := &enrichmentMockKit{name: "developer", enrichment: nil}
	session := domain.KitSession{SessionID: "s1"}

	wrapped := tools.NewKitEnrichmentWrapper(inner, kit, session)

	result, err := wrapped.InvokableRun(context.Background(), `{}`)
	require.NoError(t, err)
	assert.Equal(t, "file saved", result)
}

func TestKitEnrichmentWrapper_EmptyEnrichmentContent(t *testing.T) {
	inner := &mockInnerTool{result: "file saved"}
	kit := &enrichmentMockKit{
		name:       "developer",
		enrichment: &domain.Enrichment{Content: ""},
	}
	session := domain.KitSession{SessionID: "s1"}

	wrapped := tools.NewKitEnrichmentWrapper(inner, kit, session)

	result, err := wrapped.InvokableRun(context.Background(), `{}`)
	require.NoError(t, err)
	assert.Equal(t, "file saved", result)
}

func TestKitEnrichmentWrapper_InnerToolError(t *testing.T) {
	inner := &mockInnerTool{result: "partial", err: fmt.Errorf("write failed")}
	kit := &enrichmentMockKit{
		name:       "developer",
		enrichment: &domain.Enrichment{Content: "should not appear"},
	}
	session := domain.KitSession{SessionID: "s1"}

	wrapped := tools.NewKitEnrichmentWrapper(inner, kit, session)

	result, err := wrapped.InvokableRun(context.Background(), `{}`)
	require.Error(t, err)
	assert.Equal(t, "partial", result)
}

func TestKitEnrichmentWrapper_InfoDelegatesToInner(t *testing.T) {
	inner := &mockInnerTool{result: "ok"}
	kit := &enrichmentMockKit{name: "developer"}
	session := domain.KitSession{SessionID: "s1"}

	wrapped := tools.NewKitEnrichmentWrapper(inner, kit, session)

	info, err := wrapped.Info(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "edit_file", info.Name)
}

// Verify the wrapper satisfies the interface at compile time.
var _ tool.InvokableTool = (*mockInnerTool)(nil)
var _ kit.Kit = (*enrichmentMockKit)(nil)
