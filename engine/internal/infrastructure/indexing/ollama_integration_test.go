//go:build ollama

package indexing

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipIfOllamaUnavailable skips the test if the local Ollama server is not reachable.
func skipIfOllamaUnavailable(t *testing.T) {
	t.Helper()
	resp, err := http.Get(DefaultOllamaURL + "/api/tags")
	if err != nil {
		t.Skip("Ollama not available")
	}
	resp.Body.Close()
}

// TC-I-09: Single chunk embedding via Ollama produces a 768-dimensional vector.
func TestOllamaEmbed_SingleChunk(t *testing.T) {
	skipIfOllamaUnavailable(t)

	client := NewEmbeddingsClient(DefaultOllamaURL, DefaultEmbedModel, DefaultDimension)
	ctx := context.Background()

	vec, err := client.Embed(ctx, `func main() { fmt.Println("hello") }`)
	require.NoError(t, err)
	require.NotNil(t, vec)
	assert.Len(t, vec, DefaultDimension, "embedding dimension should be %d", DefaultDimension)

	// At least some values must be non-zero.
	hasNonZero := false
	for _, v := range vec {
		if v != 0 {
			hasNonZero = true
			break
		}
	}
	assert.True(t, hasNonZero, "embedding vector should contain non-zero values")
}

// TC-I-10: Batch embedding returns one vector per input text.
func TestOllamaEmbed_Batch(t *testing.T) {
	skipIfOllamaUnavailable(t)

	client := NewEmbeddingsClient(DefaultOllamaURL, DefaultEmbedModel, DefaultDimension)
	ctx := context.Background()

	texts := []string{
		"func Alpha() {}",
		"func Beta() {}",
		"type Repository interface {}",
		"import fmt",
		"package main",
	}

	results, err := client.EmbedBatch(ctx, texts)
	require.NoError(t, err)
	require.Len(t, results, len(texts), "expected one embedding per input text")

	for i, vec := range results {
		require.NotNil(t, vec, "embedding %d should not be nil", i)
		assert.Len(t, vec, DefaultDimension, "embedding %d dimension should be %d", i, DefaultDimension)

		hasNonZero := false
		for _, v := range vec {
			if v != 0 {
				hasNonZero = true
				break
			}
		}
		assert.True(t, hasNonZero, "embedding %d should contain non-zero values", i)
	}
}
