package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListModels_NoFilters(t *testing.T) {
	reg := New()
	models := reg.ListModels(ModelFilters{})

	assert.NotEmpty(t, models)
	assert.Len(t, models, len(builtinModels()))
}

func TestListModels_FilterByProvider(t *testing.T) {
	reg := New()
	models := reg.ListModels(ModelFilters{Provider: "anthropic"})

	require.NotEmpty(t, models)
	for _, m := range models {
		assert.Equal(t, "anthropic", m.Provider)
	}

	// Verify we got the expected anthropic models.
	ids := modelIDs(models)
	assert.Contains(t, ids, "claude-opus-4-6")
	assert.Contains(t, ids, "claude-sonnet-4-6")
	assert.Contains(t, ids, "claude-haiku-4-5")
}

func TestListModels_FilterByTier(t *testing.T) {
	reg := New()

	tier1 := Tier1
	models := reg.ListModels(ModelFilters{Tier: &tier1})

	require.NotEmpty(t, models)
	for _, m := range models {
		assert.Equal(t, Tier1, m.Tier)
	}

	// All tier 1 models should support tools.
	for _, m := range models {
		assert.True(t, m.SupportsTools, "tier 1 model %s should support tools", m.ID)
	}
}

func TestListModels_FilterByTier3(t *testing.T) {
	reg := New()

	tier3 := Tier3
	models := reg.ListModels(ModelFilters{Tier: &tier3})

	require.Len(t, models, 1)
	assert.Equal(t, "gpt-5.4-nano", models[0].ID)
}

func TestListModels_FilterBySupportsTools(t *testing.T) {
	reg := New()

	toolsTrue := true
	withTools := reg.ListModels(ModelFilters{SupportsTools: &toolsTrue})

	for _, m := range withTools {
		assert.True(t, m.SupportsTools, "model %s should support tools", m.ID)
	}

	toolsFalse := false
	withoutTools := reg.ListModels(ModelFilters{SupportsTools: &toolsFalse})

	for _, m := range withoutTools {
		assert.False(t, m.SupportsTools, "model %s should not support tools", m.ID)
	}

	// The total should equal all models.
	assert.Equal(t, len(builtinModels()), len(withTools)+len(withoutTools))
}

func TestListModels_CombinedFilters(t *testing.T) {
	reg := New()

	tier2 := Tier2
	models := reg.ListModels(ModelFilters{
		Provider: "google",
		Tier:     &tier2,
	})

	require.Len(t, models, 1)
	assert.Equal(t, "gemini-2.5-flash", models[0].ID)
}

func TestListModels_NoMatches(t *testing.T) {
	reg := New()

	tier3 := Tier3
	models := reg.ListModels(ModelFilters{
		Provider: "anthropic",
		Tier:     &tier3,
	})

	assert.Empty(t, models)
}

func TestGetModel_Found(t *testing.T) {
	reg := New()

	m := reg.GetModel("claude-opus-4-6")
	require.NotNil(t, m)
	assert.Equal(t, "Claude Opus 4.6", m.DisplayName)
	assert.Equal(t, "anthropic", m.Provider)
	assert.Equal(t, Tier1, m.Tier)
	assert.Equal(t, 1_000_000, m.ContextWindow)
	assert.True(t, m.SupportsTools)
	assert.True(t, m.SupportsVision)
	assert.Equal(t, 5.0, m.PricingInput)
	assert.Equal(t, 25.0, m.PricingOutput)
}

func TestGetModel_NotFound(t *testing.T) {
	reg := New()

	m := reg.GetModel("nonexistent-model")
	assert.Nil(t, m)
}

func TestListProviders(t *testing.T) {
	reg := New()
	providers := reg.ListProviders()

	assert.Len(t, providers, len(builtinProviders()))

	providerIDs := make(map[string]bool)
	for _, p := range providers {
		providerIDs[p.ID] = true
	}

	expectedIDs := []string{
		"openai", "anthropic", "google", "ollama",
		"openai_compatible", "openrouter", "azure_openai",
		"deepseek", "mistral", "xai", "zai",
	}
	for _, id := range expectedIDs {
		assert.True(t, providerIDs[id], "provider %s should be present", id)
	}
}

func TestGetProvider_Found(t *testing.T) {
	reg := New()

	p := reg.GetProvider("openrouter")
	require.NotNil(t, p)
	assert.Equal(t, "OpenRouter", p.DisplayName)
	assert.Equal(t, "api_key", p.AuthType)
	assert.Equal(t, "openrouter.ai", p.Website)
}

func TestGetProvider_NotFound(t *testing.T) {
	reg := New()

	p := reg.GetProvider("nonexistent-provider")
	assert.Nil(t, p)
}

func modelIDs(models []ModelInfo) []string {
	ids := make([]string, len(models))
	for i, m := range models {
		ids[i] = m.ID
	}
	return ids
}
