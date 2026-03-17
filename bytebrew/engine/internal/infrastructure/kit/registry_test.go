package kit_test

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/kit"
)

type mockKit struct {
	name string
}

func (k *mockKit) Name() string { return k.name }
func (k *mockKit) OnSessionStart(_ context.Context, _ domain.KitSession) error {
	return nil
}
func (k *mockKit) OnSessionEnd(_ context.Context, _ domain.KitSession) error {
	return nil
}
func (k *mockKit) Tools(_ domain.KitSession) []tool.InvokableTool { return nil }
func (k *mockKit) PostToolCall(_ context.Context, _ domain.KitSession, _ string, _ string) *domain.Enrichment {
	return nil
}

var _ kit.Kit = (*mockKit)(nil)

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := kit.NewRegistry()
	k := &mockKit{name: "developer"}

	reg.Register(k)

	got, err := reg.Get("developer")
	require.NoError(t, err)
	assert.Equal(t, k, got)
}

func TestRegistry_GetUnknown(t *testing.T) {
	reg := kit.NewRegistry()

	_, err := reg.Get("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestRegistry_List(t *testing.T) {
	reg := kit.NewRegistry()
	reg.Register(&mockKit{name: "alpha"})
	reg.Register(&mockKit{name: "beta"})

	names := reg.List()
	assert.Len(t, names, 2)
	assert.ElementsMatch(t, []string{"alpha", "beta"}, names)
}

func TestRegistry_Has(t *testing.T) {
	reg := kit.NewRegistry()
	reg.Register(&mockKit{name: "developer"})

	assert.True(t, reg.Has("developer"))
	assert.False(t, reg.Has("unknown"))
}
