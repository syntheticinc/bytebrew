package indexing

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// TC-I-03: Scanner respects built-in ignore lists (node_modules, .git, etc.)
// ---------------------------------------------------------------------------

func TestFileScanner_IgnoresNodeModules(t *testing.T) {
	root := t.TempDir()

	// Create a file inside node_modules — should be ignored.
	nmDir := filepath.Join(root, "node_modules")
	require.NoError(t, os.MkdirAll(nmDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(nmDir, "dep.js"), []byte("console.log('dep')"), 0o644))

	// Create a valid source file outside node_modules.
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o644))

	scanner := NewFileScanner(root)
	results, err := scanner.Scan(context.Background())
	require.NoError(t, err)

	for _, r := range results {
		assert.NotContains(t, r.RelativePath, "node_modules",
			"files inside node_modules should be excluded")
	}
	require.Len(t, results, 1)
	assert.Equal(t, "main.go", results[0].RelativePath)
}

func TestFileScanner_IgnoresHiddenDirs(t *testing.T) {
	root := t.TempDir()

	// Hidden directory should be ignored.
	hiddenDir := filepath.Join(root, ".hidden")
	require.NoError(t, os.MkdirAll(hiddenDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenDir, "secret.go"), []byte("package secret"), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(root, "app.go"), []byte("package app"), 0o644))

	scanner := NewFileScanner(root)
	results, err := scanner.Scan(context.Background())
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, "app.go", results[0].RelativePath)
}

func TestFileScanner_IgnoresDefaultIgnoreDirs(t *testing.T) {
	root := t.TempDir()

	for _, dir := range []string{"dist", "build", "vendor", ".venv"} {
		dirPath := filepath.Join(root, dir)
		require.NoError(t, os.MkdirAll(dirPath, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dirPath, "file.go"), []byte("package x"), 0o644))
	}

	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o644))

	scanner := NewFileScanner(root)
	results, err := scanner.Scan(context.Background())
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, "main.go", results[0].RelativePath)
}

// ---------------------------------------------------------------------------
// TC-I-04: Scanner size limit — files > 1MB are skipped
// ---------------------------------------------------------------------------

func TestFileScanner_SkipsLargeFiles(t *testing.T) {
	root := t.TempDir()

	// Create a file slightly over MaxFileSize (1MB).
	bigContent := make([]byte, MaxFileSize+1)
	for i := range bigContent {
		bigContent[i] = 'x'
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "big.go"), bigContent, 0o644))

	// Small file should pass.
	require.NoError(t, os.WriteFile(filepath.Join(root, "small.go"), []byte("package small"), 0o644))

	scanner := NewFileScanner(root)
	results, err := scanner.Scan(context.Background())
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, "small.go", results[0].RelativePath)
}

func TestFileScanner_AcceptsMaxSizeFile(t *testing.T) {
	root := t.TempDir()

	// Exactly MaxFileSize — should be accepted.
	content := make([]byte, MaxFileSize)
	for i := range content {
		content[i] = 'a'
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "exact.go"), content, 0o644))

	scanner := NewFileScanner(root)
	results, err := scanner.Scan(context.Background())
	require.NoError(t, err)

	require.Len(t, results, 1)
}

// ---------------------------------------------------------------------------
// TC-I-04b: Scanner only picks up supported extensions
// ---------------------------------------------------------------------------

func TestFileScanner_SkipsUnsupportedExtensions(t *testing.T) {
	root := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(root, "readme.md"), []byte("# Hi"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "image.png"), []byte{0x89, 0x50}, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "data.json"), []byte("{}"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "main.go"), []byte("package main"), 0o644))

	scanner := NewFileScanner(root)
	results, err := scanner.Scan(context.Background())
	require.NoError(t, err)

	require.Len(t, results, 1)
	assert.Equal(t, "go", results[0].Language)
}

// ---------------------------------------------------------------------------
// TC-I-08: Chunker quality filter — content < minChunkBytes is filtered out
// ---------------------------------------------------------------------------

func TestChunker_FiltersSmallContent(t *testing.T) {
	chunker := NewChunker()

	// Content shorter than minChunkBytes (10 bytes).
	chunks := chunker.ChunkFile("/tiny.go", "x", "go")
	assert.Empty(t, chunks, "content shorter than minChunkBytes should produce no chunks")
}

func TestChunker_FiltersSmallContentUnknownLang(t *testing.T) {
	chunker := NewChunker()

	// Unknown language falls back to wholeFileChunk which also checks minChunkBytes.
	chunks := chunker.ChunkFile("/tiny.txt", "abc", "unknown")
	assert.Empty(t, chunks, "tiny file with unknown language should produce no chunks")
}

func TestChunker_AcceptsMinSizeContent(t *testing.T) {
	chunker := NewChunker()

	// Exactly minChunkBytes for unknown language (whole file chunk).
	content := "0123456789" // 10 bytes
	chunks := chunker.ChunkFile("/ok.txt", content, "unknown")
	require.Len(t, chunks, 1)
	assert.Equal(t, ChunkOther, chunks[0].ChunkType)
}

// ---------------------------------------------------------------------------
// TC-I-12: Vector search (brute-force cosine) — results sorted by score
// ---------------------------------------------------------------------------

func TestChunkStore_SearchSortedByScore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "search.db")
	store, err := NewChunkStore(dbPath, 4)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	chunks := []CodeChunk{
		{ID: "c1", FilePath: "/a.go", Content: "func Alpha() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "Alpha"},
		{ID: "c2", FilePath: "/b.go", Content: "func Beta() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "Beta"},
		{ID: "c3", FilePath: "/c.go", Content: "func Gamma() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "Gamma"},
	}
	embeddings := [][]float32{
		{1.0, 0.0, 0.0, 0.0}, // Alpha: x-axis
		{0.0, 1.0, 0.0, 0.0}, // Beta: y-axis
		{0.7, 0.7, 0.0, 0.0}, // Gamma: diagonal
	}

	require.NoError(t, store.Store(ctx, chunks, embeddings, 1000))

	// Query close to Alpha (x-axis direction).
	query := []float32{0.95, 0.05, 0.0, 0.0}
	results, err := store.Search(ctx, query, 10)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Alpha should be first (closest to x-axis), then Gamma (diagonal), then Beta.
	assert.Equal(t, "Alpha", results[0].Chunk.Name)
	assert.Equal(t, "Gamma", results[1].Chunk.Name)
	assert.Equal(t, "Beta", results[2].Chunk.Name)

	// Scores must be strictly descending.
	assert.True(t, results[0].Score > results[1].Score)
	assert.True(t, results[1].Score > results[2].Score)
}

func TestChunkStore_SearchWithLimit(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "limit.db")
	store, err := NewChunkStore(dbPath, 3)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	chunks := []CodeChunk{
		{ID: "c1", FilePath: "/a.go", Content: "func A() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "A"},
		{ID: "c2", FilePath: "/b.go", Content: "func B() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "B"},
		{ID: "c3", FilePath: "/c.go", Content: "func C() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "C"},
	}
	embeddings := [][]float32{
		{1.0, 0.0, 0.0},
		{0.0, 1.0, 0.0},
		{0.0, 0.0, 1.0},
	}

	require.NoError(t, store.Store(ctx, chunks, embeddings, 1000))

	results, err := store.Search(ctx, []float32{1.0, 0.0, 0.0}, 2)
	require.NoError(t, err)
	assert.Len(t, results, 2, "search should respect limit")
}

func TestChunkStore_SearchNoEmbeddings(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "noemb.db")
	store, err := NewChunkStore(dbPath, 3)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Store chunks without embeddings.
	chunks := []CodeChunk{
		{ID: "c1", FilePath: "/a.go", Content: "func A() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "A"},
	}
	require.NoError(t, store.Store(ctx, chunks, [][]float32{nil}, 1000))

	results, err := store.Search(ctx, []float32{1.0, 0.0, 0.0}, 10)
	require.NoError(t, err)
	assert.Empty(t, results, "chunks without embeddings should not appear in search")
}

// ---------------------------------------------------------------------------
// TC-I-13: Store clear + reindex — search works with new data after clear
// ---------------------------------------------------------------------------

func TestChunkStore_ClearAndReindex(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "reindex.db")
	store, err := NewChunkStore(dbPath, 3)
	require.NoError(t, err)
	defer store.Close()

	ctx := context.Background()

	// Phase 1: store old data.
	oldChunks := []CodeChunk{
		{ID: "old1", FilePath: "/old.go", Content: "func Old() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "Old"},
	}
	oldEmb := [][]float32{{1.0, 0.0, 0.0}}
	require.NoError(t, store.Store(ctx, oldChunks, oldEmb, 1000))

	// Phase 2: clear.
	require.NoError(t, store.Clear(ctx))

	// Verify old data is gone.
	results, err := store.Search(ctx, []float32{1.0, 0.0, 0.0}, 10)
	require.NoError(t, err)
	assert.Empty(t, results)

	indexed, err := store.GetIndexedFiles(ctx)
	require.NoError(t, err)
	assert.Empty(t, indexed)

	// Phase 3: store new data.
	newChunks := []CodeChunk{
		{ID: "new1", FilePath: "/new.go", Content: "func New() {}", StartLine: 1, EndLine: 1, Language: "go", ChunkType: ChunkFunction, Name: "New"},
	}
	newEmb := [][]float32{{0.0, 1.0, 0.0}}
	require.NoError(t, store.Store(ctx, newChunks, newEmb, 2000))

	// Phase 4: search should find only new data.
	results, err = store.Search(ctx, []float32{0.0, 1.0, 0.0}, 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "New", results[0].Chunk.Name)

	indexed, err = store.GetIndexedFiles(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2000), indexed["/new.go"])
}

// ---------------------------------------------------------------------------
// TC-I-11: Ollama unavailable — clear error from EmbeddingsClient
// ---------------------------------------------------------------------------

func TestEmbeddingsClient_UnavailableServer(t *testing.T) {
	// Create a test server that always returns 503.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("service unavailable"))
	}))
	defer srv.Close()

	client := NewEmbeddingsClient(srv.URL, "nomic-embed-text", 768)

	// Ping should fail (503 != 200).
	assert.False(t, client.Ping(context.Background()))

	// Embed should return an error (either from retries or empty result).
	_, err := client.Embed(context.Background(), "hello world")
	require.Error(t, err)
}

func TestEmbeddingsClient_ConnectionRefused(t *testing.T) {
	// Use an address that is guaranteed to refuse connections.
	client := NewEmbeddingsClient("http://127.0.0.1:1", "nomic-embed-text", 768)

	assert.False(t, client.Ping(context.Background()))

	_, err := client.Embed(context.Background(), "hello")
	require.Error(t, err)
}

func TestEmbeddingsClient_EmptyBatch(t *testing.T) {
	client := NewEmbeddingsClient("http://127.0.0.1:1", "nomic-embed-text", 768)

	results, err := client.EmbedBatch(context.Background(), []string{})
	require.NoError(t, err)
	assert.Nil(t, results)
}

// ---------------------------------------------------------------------------
// Helper: fake Ollama server for indexer tests
// ---------------------------------------------------------------------------

// newFakeOllama creates an httptest server that responds to /api/tags (ping)
// and /api/embed (embeddings). Returns dim-dimensional zero vectors.
func newFakeOllama(t *testing.T, dim int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"models":[]}`))
		case "/api/embed":
			var req struct {
				Input []string `json:"input"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			embeddings := make([][]float32, len(req.Input))
			for i := range req.Input {
				emb := make([]float32, dim)
				// Simple deterministic embedding: set first element to float of index
				emb[0] = float32(i+1) * 0.1
				embeddings[i] = emb
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"embeddings": embeddings})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// ---------------------------------------------------------------------------
// TC-I-01: Full index (scan -> parse -> chunk -> store)
// ---------------------------------------------------------------------------

func TestIndexer_FullIndex_TC_I_01(t *testing.T) {
	const dim = 4
	root := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "index.db")

	// Create source files in project root
	require.NoError(t, os.MkdirAll(filepath.Join(root, "cmd"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "cmd", "main.go"), []byte(`package main

func main() {
	println("hello")
}
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(root, "utils.go"), []byte(`package utils

func Add(a, b int) int {
	return a + b
}
`), 0o644))

	// Fake Ollama server
	ollamaSrv := newFakeOllama(t, dim)

	// Build Indexer with fake embedder (same package, direct struct construction)
	store, err := NewChunkStore(dbPath, dim)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	idx := &Indexer{
		scanner:    NewFileScanner(root),
		chunker:    NewChunker(),
		embeddings: NewEmbeddingsClient(ollamaSrv.URL, "test-model", dim),
		store:      store,
		rootPath:   root,
	}

	// Run full index
	var phases []string
	err = idx.Index(context.Background(), true, func(p IndexProgress) {
		if len(phases) == 0 || phases[len(phases)-1] != p.Phase {
			phases = append(phases, p.Phase)
		}
	})
	require.NoError(t, err)

	// Verify all phases executed
	assert.Contains(t, phases, "scanning")
	assert.Contains(t, phases, "parsing")
	assert.Contains(t, phases, "embedding")
	assert.Contains(t, phases, "storing")
	assert.Contains(t, phases, "complete")

	// Verify chunks were stored — search by name
	ctx := context.Background()
	chunks, err := store.GetByName(ctx, "main")
	require.NoError(t, err)
	assert.NotEmpty(t, chunks, "should find 'main' function chunk")

	chunks, err = store.GetByName(ctx, "Add")
	require.NoError(t, err)
	assert.NotEmpty(t, chunks, "should find 'Add' function chunk")

	// Verify indexed files are tracked
	indexed, err := store.GetIndexedFiles(ctx)
	require.NoError(t, err)
	assert.Len(t, indexed, 2, "two source files should be indexed")
}

// ---------------------------------------------------------------------------
// TC-I-02: Incremental re-index (only changed files)
// ---------------------------------------------------------------------------

func TestIndexer_IncrementalReindex_TC_I_02(t *testing.T) {
	const dim = 4
	root := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "incr.db")

	// Create initial file
	mainPath := filepath.Join(root, "main.go")
	require.NoError(t, os.WriteFile(mainPath, []byte(`package main

func main() {
	println("v1")
}
`), 0o644))

	ollamaSrv := newFakeOllama(t, dim)

	store, err := NewChunkStore(dbPath, dim)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	idx := &Indexer{
		scanner:    NewFileScanner(root),
		chunker:    NewChunker(),
		embeddings: NewEmbeddingsClient(ollamaSrv.URL, "test-model", dim),
		store:      store,
		rootPath:   root,
	}

	ctx := context.Background()

	// Full index first
	require.NoError(t, idx.Index(ctx, true, nil))

	indexed, err := store.GetIndexedFiles(ctx)
	require.NoError(t, err)
	require.Len(t, indexed, 1, "one file indexed initially")

	// Incremental re-index without changes — nothing should be processed
	var parsingCalled bool
	err = idx.Index(ctx, false, func(p IndexProgress) {
		if p.Phase == "parsing" {
			parsingCalled = true
		}
	})
	require.NoError(t, err)
	assert.False(t, parsingCalled, "no parsing should occur when no files changed")

	// Now modify the file (touch mtime)
	time.Sleep(50 * time.Millisecond) // ensure mtime differs
	require.NoError(t, os.WriteFile(mainPath, []byte(`package main

func main() {
	println("v2")
}

func NewHelper() {}
`), 0o644))

	// Incremental re-index — should process the changed file
	parsingCalled = false
	err = idx.Index(ctx, false, func(p IndexProgress) {
		if p.Phase == "parsing" {
			parsingCalled = true
		}
	})
	require.NoError(t, err)
	assert.True(t, parsingCalled, "parsing should occur for changed file")

	// Verify new function is findable
	chunks, err := store.GetByName(ctx, "NewHelper")
	require.NoError(t, err)
	assert.NotEmpty(t, chunks, "should find newly added 'NewHelper' function")
}
