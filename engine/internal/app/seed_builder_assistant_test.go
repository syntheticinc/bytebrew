package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// TestSeedBuilderAssistant_EmptyAPIKey_DoesNotBindEmptyModel is the regression
// guard for the 2026-04-23 prod bug where the engine's React agent chat node
// returned `401 Unauthorized, message: No cookie auth credentials found` to
// every chat turn on a fresh tenant.
//
// Chain of events reproduced here:
//   1. Engine starts with LLM_API_KEY env unset.
//   2. seedBuilderAssistant is called on a fresh database.
//   3. ensureDefaultModel creates a `default` model with api_key_encrypted=""
//      (from the empty env var).
//   4. builder-assistant is bound to that empty-key model.
//   5. The first chat turn hits OpenRouter without an Authorization header
//      → 401 → user sees a cryptic stack trace instead of a useful message.
//
// The correct behaviour: if we don't have a working API key, don't seed a
// broken model. Leave builder-assistant unbound so the onboarding wizard
// forces the user to add their own key first (happy path) — and so the
// chat handler can surface a clean "please configure a model" error
// instead of a provider 401.
func TestEnsureDefaultModel_EmptyAPIKey_DoesNotPersistBrokenModel(t *testing.T) {
	_ = models.LLMProviderModel{} // keep import live for future assertions
	t.Setenv("LLM_API_KEY", "")

	db := setupTestDB(t)
	ctx := context.Background()

	// Call the narrow helper — this is where the empty-key model creation
	// happens. Full seedBuilderAssistant also touches MCP servers and agents
	// which need more plumbing; isolating to ensureDefaultModel keeps the
	// regression guard laser-focused on the actual defect.
	returned := ensureDefaultModel(ctx, db)

	llmRepo := configrepo.NewGORMLLMProviderRepository(db)
	modelList, err := llmRepo.List(ctx)
	require.NoError(t, err)

	// Invariant: no persisted model may have an empty api_key_encrypted.
	// Such a model 401s on every chat turn — exactly the 2026-04-23
	// "No cookie auth credentials found" prod failure.
	for _, m := range modelList {
		assert.NotEmpty(t, m.APIKeyEncrypted,
			"ensureDefaultModel persisted model %q with empty api_key_encrypted — this is a guaranteed 401 on first chat (2026-04-23 prod bug)",
			m.Name,
		)
	}

	// Second invariant: when there is no usable key, ensureDefaultModel
	// must return "" (signalling the caller to leave the agent unbound)
	// rather than the name of a broken model. Returning the name of a
	// broken model is what binds builder-assistant to the empty-key
	// default and triggers the bug.
	if returned != "" {
		// If it did return a name, that model must have a non-empty key.
		var found *models.LLMProviderModel
		for i := range modelList {
			if modelList[i].Name == returned {
				found = &modelList[i]
				break
			}
		}
		require.NotNil(t, found, "ensureDefaultModel returned %q but no such row", returned)
		assert.NotEmpty(t, found.APIKeyEncrypted,
			"ensureDefaultModel returned model %q with empty api_key — bind this to builder-assistant and every chat 401s",
			returned,
		)
	}
}
