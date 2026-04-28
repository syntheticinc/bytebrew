package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsKnowledgeTypeSupported(t *testing.T) {
	// AC-KB-FMT-01..04: supported formats
	assert.True(t, IsKnowledgeTypeSupported("pdf"))
	assert.True(t, IsKnowledgeTypeSupported("docx"))
	assert.True(t, IsKnowledgeTypeSupported("doc"))
	assert.True(t, IsKnowledgeTypeSupported("txt"))
	assert.True(t, IsKnowledgeTypeSupported("md"))
	assert.True(t, IsKnowledgeTypeSupported("csv"))

	// AC-KB-FMT-05: unsupported format
	assert.False(t, IsKnowledgeTypeSupported("exe"))
	assert.False(t, IsKnowledgeTypeSupported("zip"))
	assert.False(t, IsKnowledgeTypeSupported("png"))
}

func TestSupportedKnowledgeTypes(t *testing.T) {
	types := SupportedKnowledgeTypes()
	assert.Len(t, types, 6)
}

func TestDefaultKnowledgeConfig(t *testing.T) {
	cfg := DefaultKnowledgeConfig()
	// AC-KB-PARAM-01: default top_k = 5
	assert.Equal(t, 5, cfg.TopK)
	// AC-KB-PARAM-02: default similarity_threshold = 0.75
	assert.Equal(t, 0.75, cfg.SimilarityThreshold)
}

func TestKnowledgeConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  KnowledgeConfig
		wantErr bool
	}{
		{"valid defaults", DefaultKnowledgeConfig(), false},
		{"valid custom", KnowledgeConfig{TopK: 10, SimilarityThreshold: 0.5}, false},
		{"top_k too low", KnowledgeConfig{TopK: 0, SimilarityThreshold: 0.5}, true},
		{"top_k too high", KnowledgeConfig{TopK: 100, SimilarityThreshold: 0.5}, true},
		{"threshold too low", KnowledgeConfig{TopK: 5, SimilarityThreshold: -0.1}, true},
		{"threshold too high", KnowledgeConfig{TopK: 5, SimilarityThreshold: 1.5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
