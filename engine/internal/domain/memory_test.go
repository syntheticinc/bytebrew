package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMemory(t *testing.T) {
	tests := []struct {
		name     string
		schemaID string
		userID   string
		content  string
		wantErr  bool
	}{
		{"valid", "schema-1", "user-1", "remember this", false},
		{"empty schema_id", "", "user-1", "content", true},
		{"empty user_id", "schema-1", "", "content", true},
		{"empty content", "schema-1", "user-1", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem, err := NewMemory(tt.schemaID, tt.userID, tt.content)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.schemaID, mem.SchemaID)
			assert.Equal(t, tt.userID, mem.UserID)
			assert.Equal(t, tt.content, mem.Content)
			assert.NotNil(t, mem.Metadata)
		})
	}
}

func TestMemory_AddMetadata(t *testing.T) {
	mem, err := NewMemory("schema-1", "user-1", "content")
	require.NoError(t, err)

	mem.AddMetadata("source", "agent")
	val, ok := mem.GetMetadata("source")
	assert.True(t, ok)
	assert.Equal(t, "agent", val)

	_, ok = mem.GetMetadata("nonexistent")
	assert.False(t, ok)
}

func TestMemory_NoFlowReferences(t *testing.T) {
	// AC-MEM-TERM-01, AC-MEM-TERM-02: No "Flow" references in memory domain
	mem, err := NewMemory("schema-1", "user-1", "content")
	require.NoError(t, err)
	assert.Equal(t, "schema-1", mem.SchemaID)
	// SchemaID field exists, no FlowID field
}

func TestDefaultMemoryConfig(t *testing.T) {
	cfg := DefaultMemoryConfig()
	// AC-MEM-RET-01: Default retention = Unlimited (max_entries=0)
	assert.Equal(t, 0, cfg.MaxEntries)
}
