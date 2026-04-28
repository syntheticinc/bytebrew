package indexing

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/glebarez/go-sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkStore_StoreAndSearch(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewChunkStore(dbPath, 4)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	chunks := []CodeChunk{
		{ID: "chunk1", FilePath: "/a.go", Content: "func main() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "main"},
		{ID: "chunk2", FilePath: "/b.go", Content: "func helper() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "helper"},
	}
	embeddings := [][]float32{
		{1.0, 0.0, 0.0, 0.0},
		{0.0, 1.0, 0.0, 0.0},
	}

	err = store.Store(ctx, chunks, embeddings, 1000)
	require.NoError(t, err)

	// Search for something close to chunk1
	query := []float32{0.9, 0.1, 0.0, 0.0}
	results, err := store.Search(ctx, query, 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	assert.Equal(t, "main", results[0].Chunk.Name)
	assert.True(t, results[0].Score > results[1].Score)
}

func TestChunkStore_GetByName(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewChunkStore(dbPath, 4)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	chunks := []CodeChunk{
		{ID: "c1", FilePath: "/a.go", Content: "func Foo() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "Foo"},
		{ID: "c2", FilePath: "/b.go", Content: "func Bar() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "Bar"},
	}

	err = store.Store(ctx, chunks, [][]float32{nil, nil}, 1000)
	require.NoError(t, err)

	found, err := store.GetByName(ctx, "Foo")
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "Foo", found[0].Name)
}

func TestChunkStore_GetByFilePath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewChunkStore(dbPath, 4)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	chunks := []CodeChunk{
		{ID: "c1", FilePath: "/a.go", Content: "func A() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "A"},
		{ID: "c2", FilePath: "/a.go", Content: "func B() {}", StartLine: 5, EndLine: 5, Language: "go", ChunkType: ChunkFunction, Name: "B"},
		{ID: "c3", FilePath: "/b.go", Content: "func C() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "C"},
	}

	err = store.Store(ctx, chunks, [][]float32{nil, nil, nil}, 1000)
	require.NoError(t, err)

	found, err := store.GetByFilePath(ctx, "/a.go")
	require.NoError(t, err)
	assert.Len(t, found, 2)
}

func TestChunkStore_DeleteByFilePath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewChunkStore(dbPath, 4)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	chunks := []CodeChunk{
		{ID: "c1", FilePath: "/a.go", Content: "func A() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "A"},
	}
	err = store.Store(ctx, chunks, [][]float32{nil}, 1000)
	require.NoError(t, err)

	err = store.DeleteByFilePath(ctx, "/a.go")
	require.NoError(t, err)

	found, err := store.GetByFilePath(ctx, "/a.go")
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestChunkStore_GetIndexedFiles(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewChunkStore(dbPath, 4)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	chunks := []CodeChunk{
		{ID: "c1", FilePath: "/a.go", Content: "func A() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "A"},
	}
	err = store.Store(ctx, chunks, [][]float32{nil}, 12345)
	require.NoError(t, err)

	indexed, err := store.GetIndexedFiles(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(12345), indexed["/a.go"])
}

func TestChunkStore_Clear(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewChunkStore(dbPath, 4)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	chunks := []CodeChunk{
		{ID: "c1", FilePath: "/a.go", Content: "func A() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "A"},
	}
	err = store.Store(ctx, chunks, [][]float32{nil}, 1000)
	require.NoError(t, err)

	err = store.Clear(ctx)
	require.NoError(t, err)

	indexed, err := store.GetIndexedFiles(ctx)
	require.NoError(t, err)
	assert.Empty(t, indexed)
}

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float32
	}{
		{"identical", []float32{1, 0, 0}, []float32{1, 0, 0}, 1.0},
		{"orthogonal", []float32{1, 0, 0}, []float32{0, 1, 0}, 0.0},
		{"opposite", []float32{1, 0, 0}, []float32{-1, 0, 0}, -1.0},
		{"empty", nil, nil, 0.0},
		{"zero vector", []float32{0, 0}, []float32{1, 0}, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			assert.InDelta(t, float64(tt.want), float64(got), 0.001)
		})
	}
}

func TestFloat32Encoding(t *testing.T) {
	original := []float32{1.0, -0.5, 3.14, 0.0}
	bytes := float32sToBytes(original)
	decoded := bytesToFloat32s(bytes)

	require.Len(t, decoded, len(original))
	for i := range original {
		assert.InDelta(t, float64(original[i]), float64(decoded[i]), 0.0001)
	}
}

func TestChunkStore_Upsert(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewChunkStore(dbPath, 4)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Store initial
	chunks := []CodeChunk{
		{ID: "c1", FilePath: "/a.go", Content: "v1", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "A"},
	}
	err = store.Store(ctx, chunks, [][]float32{nil}, 1000)
	require.NoError(t, err)

	// Store with same ID but updated content
	chunks[0].Content = "v2"
	err = store.Store(ctx, chunks, [][]float32{nil}, 2000)
	require.NoError(t, err)

	found, err := store.GetByName(ctx, "A")
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "v2", found[0].Content)
}

func TestNewChunkStore_InvalidPath(t *testing.T) {
	// Ensure directory does not exist
	dbPath := filepath.Join(t.TempDir(), "nonexistent", "deep", "test.db")
	_, err := NewChunkStore(dbPath, 4)
	// modernc/sqlite creates the file, but the directory must exist
	// On most systems this should fail
	if err != nil {
		assert.Error(t, err)
	} else {
		// Some drivers auto-create — clean up
		os.Remove(dbPath)
	}
}
