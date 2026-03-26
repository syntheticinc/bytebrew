package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/glebarez/go-sqlite" // SQLite driver for indexing.ChunkStore
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/indexing"
)

// setupTestStore creates a temporary ChunkStore with test data.
func setupTestStore(t *testing.T) (*indexing.ChunkStore, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test_chunks.db")

	store, err := indexing.NewChunkStore(dbPath, 4)
	if err != nil {
		t.Fatalf("create chunk store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	chunks := []indexing.CodeChunk{
		{
			ID: "id1", FilePath: "/project/cmd/server/main.go", Content: "func main() {\n\tfmt.Println(\"hello\")\n}",
			StartLine: 15, EndLine: 17, Language: "go",
			ChunkType: indexing.ChunkFunction, Name: "main", Signature: "func main()",
		},
		{
			ID: "id2", FilePath: "/project/internal/handler.go", Content: "func (s *Server) HandleRequest(ctx context.Context, req *Request) (*Response, error) {\n\treturn nil, nil\n}",
			StartLine: 78, EndLine: 80, Language: "go",
			ChunkType: indexing.ChunkMethod, Name: "HandleRequest",
			Signature: "func (s *Server) HandleRequest(ctx context.Context, req *Request) (*Response, error)",
		},
		{
			ID: "id3", FilePath: "/project/internal/handler.go", Content: "type Server struct {\n\tdb *sql.DB\n}",
			StartLine: 10, EndLine: 12, Language: "go",
			ChunkType: indexing.ChunkStruct, Name: "Server", Signature: "type Server struct",
		},
		{
			ID: "id4", FilePath: "/project/internal/repo.go", Content: "type Repository interface {\n\tGetByID(ctx context.Context, id string) (*User, error)\n}",
			StartLine: 5, EndLine: 7, Language: "go",
			ChunkType: indexing.ChunkInterface, Name: "Repository", Signature: "type Repository interface",
		},
		{
			ID: "id5", FilePath: "/project/internal/handler.go", Content: "func NewServer(db *sql.DB) *Server {\n\treturn &Server{db: db}\n}",
			StartLine: 14, EndLine: 16, Language: "go",
			ChunkType: indexing.ChunkFunction, Name: "NewServer", Signature: "func NewServer(db *sql.DB) *Server",
		},
	}

	// Store without embeddings (no semantic search in these tests)
	embeddings := make([][]float32, len(chunks))
	if err := store.Store(context.Background(), chunks, embeddings, 0); err != nil {
		t.Fatalf("store chunks: %v", err)
	}

	return store, dir
}

// TC-SY-01: SymbolSearch exact match from store.
func TestSymbolSearch_ExactMatch_TC_SY_01(t *testing.T) {
	store, _ := setupTestStore(t)
	proxy := NewLocalClientOperationsProxy("/project", WithChunkStore(store))

	result, err := proxy.SymbolSearch(context.Background(), "", "HandleRequest", 10, nil)
	require.NoError(t, err)

	assert.Contains(t, result, "HandleRequest")
	assert.Contains(t, result, "handler.go")
	assert.Contains(t, result, "[method]")
}

func TestSymbolSearch_NoStore(t *testing.T) {
	proxy := NewLocalClientOperationsProxy("/project")

	result, err := proxy.SymbolSearch(context.Background(), "", "anything", 10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Index not available") {
		t.Errorf("expected 'Index not available' message, got: %s", result)
	}
}

func TestSymbolSearch_FilterByType(t *testing.T) {
	store, _ := setupTestStore(t)
	proxy := NewLocalClientOperationsProxy("/project", WithChunkStore(store))

	// Search for "main" but filter to only struct type — should find nothing
	result, err := proxy.SymbolSearch(context.Background(), "", "main", 10, []string{"struct"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "[function]") {
		t.Errorf("should not contain function results when filtering for struct, got: %s", result)
	}
	if !strings.Contains(result, "No symbols found") {
		t.Errorf("expected 'No symbols found', got: %s", result)
	}

	// Search for "Server" with struct filter — should find it
	result, err = proxy.SymbolSearch(context.Background(), "", "Server", 10, []string{"struct"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Server") {
		t.Errorf("should contain Server struct, got: %s", result)
	}
	if !strings.Contains(result, "[struct]") {
		t.Errorf("should contain [struct] type, got: %s", result)
	}
}

func TestSymbolSearch_CamelCaseTokenization(t *testing.T) {
	store, _ := setupTestStore(t)
	proxy := NewLocalClientOperationsProxy("/project", WithChunkStore(store))

	// "NewServer" exists as exact match — should be found directly
	result, err := proxy.SymbolSearch(context.Background(), "", "NewServer", 10, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "NewServer") {
		t.Errorf("should find NewServer, got: %s", result)
	}
}

func TestTokenizeCamelCase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple camelCase", "myFunction", []string{"my", "function"}},
		{"PascalCase", "MyFunction", []string{"my", "function"}},
		{"acronym", "handleHTTPRequest", []string{"handle", "http", "request"}},
		{"single word", "main", []string{"main"}},
		{"all caps", "HTTP", []string{"http"}},
		{"underscore separated", "my_function", []string{"my", "function"}},
		{"empty string", "", nil},
		{"mixed separators", "get_HTTPClient", []string{"get", "http", "client"}},
		{"single char words", "ATest", []string{"a", "test"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenizeCamelCase(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("tokenizeCamelCase(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("tokenizeCamelCase(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSearchCode_NoStore(t *testing.T) {
	proxy := NewLocalClientOperationsProxy("/project")

	result, err := proxy.SearchCode(context.Background(), "", "test query", "", 5, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var msg map[string]string
	if err := json.Unmarshal(result, &msg); err != nil {
		t.Fatalf("expected JSON response, got: %s", string(result))
	}

	if !strings.Contains(msg["error"], "not available") {
		t.Errorf("expected 'not available' error message, got: %v", msg)
	}
}

// TC-SY-04: GetFunction exact match — returns code block with file path.
func TestGetFunctionTool_ExactMatch_TC_SY_04(t *testing.T) {
	store, _ := setupTestStore(t)
	tool := NewGetFunctionTool(store, nil)

	args, _ := json.Marshal(GetFunctionArgs{Name: "HandleRequest"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)

	assert.Contains(t, result, "HandleRequest")
	assert.Contains(t, result, "handler.go")
	assert.Contains(t, result, "```go")
}

func TestGetFunctionTool_NotFound(t *testing.T) {
	store, _ := setupTestStore(t)
	tool := NewGetFunctionTool(store, nil)

	args, _ := json.Marshal(GetFunctionArgs{Name: "nonExistentFunc"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No function or method") {
		t.Errorf("should indicate not found, got: %s", result)
	}
}

func TestGetFunctionTool_OnlyFunctionsAndMethods(t *testing.T) {
	store, _ := setupTestStore(t)
	tool := NewGetFunctionTool(store, nil)

	// "Server" is a struct, not a function — should not be found
	args, _ := json.Marshal(GetFunctionArgs{Name: "Server"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No function or method") {
		t.Errorf("should not return struct as function, got: %s", result)
	}
}

func TestGetClassTool_ExactMatch(t *testing.T) {
	store, _ := setupTestStore(t)
	tool := NewGetClassTool(store, nil)

	args, _ := json.Marshal(GetClassArgs{Name: "Server"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Server") {
		t.Errorf("should contain struct content, got: %s", result)
	}
	if !strings.Contains(result, "handler.go") {
		t.Errorf("should contain file path, got: %s", result)
	}
	if !strings.Contains(result, "```go") {
		t.Errorf("should contain code block, got: %s", result)
	}
}

func TestGetClassTool_FindsInterface(t *testing.T) {
	store, _ := setupTestStore(t)
	tool := NewGetClassTool(store, nil)

	args, _ := json.Marshal(GetClassArgs{Name: "Repository"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "Repository") {
		t.Errorf("should find interface, got: %s", result)
	}
	if !strings.Contains(result, "repo.go") {
		t.Errorf("should contain file path, got: %s", result)
	}
}

func TestGetClassTool_OnlyClassTypes(t *testing.T) {
	store, _ := setupTestStore(t)
	tool := NewGetClassTool(store, nil)

	// "main" is a function, not a class — should not be found
	args, _ := json.Marshal(GetClassArgs{Name: "main"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No class, struct, or interface") {
		t.Errorf("should not return function as class, got: %s", result)
	}
}

func TestGetFileStructureTool_Format(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "struct_test.db")

	store, err := indexing.NewChunkStore(dbPath, 4)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	projectRoot := dir
	handlerPath := filepath.Join(projectRoot, "internal", "handler.go")

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(handlerPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Use filepath.Clean to get OS-native path matching what the tool will resolve
	cleanHandlerPath := filepath.Clean(handlerPath)

	chunks := []indexing.CodeChunk{
		{
			ID: "s1", FilePath: cleanHandlerPath, Content: "type Server struct {\n\tdb *sql.DB\n}",
			StartLine: 10, EndLine: 12, Language: "go",
			ChunkType: indexing.ChunkStruct, Name: "Server", Signature: "type Server struct",
		},
		{
			ID: "s2", FilePath: cleanHandlerPath, Content: "func (s *Server) HandleRequest(ctx context.Context) error {\n\treturn nil\n}",
			StartLine: 20, EndLine: 22, Language: "go",
			ChunkType: indexing.ChunkMethod, Name: "HandleRequest",
			Signature: "func (s *Server) HandleRequest(ctx context.Context) error",
		},
		{
			ID: "s3", FilePath: cleanHandlerPath, Content: "func NewServer(db *sql.DB) *Server {\n\treturn &Server{db: db}\n}",
			StartLine: 14, EndLine: 16, Language: "go",
			ChunkType: indexing.ChunkFunction, Name: "NewServer", Signature: "func NewServer(db *sql.DB) *Server",
		},
	}

	if err := store.Store(context.Background(), chunks, make([][]float32, len(chunks)), 0); err != nil {
		t.Fatalf("store: %v", err)
	}

	tool := NewGetFileStructureTool(store, projectRoot)

	args, _ := json.Marshal(GetFileStructureArgs{FilePath: "internal/handler.go"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain the file name
	if !strings.Contains(result, "handler.go") {
		t.Errorf("should contain file name, got: %s", result)
	}

	// Should have sections for structs and methods/functions
	if !strings.Contains(result, "## Structs") {
		t.Errorf("should contain Structs section, got: %s", result)
	}
	if !strings.Contains(result, "## Methods") {
		t.Errorf("should contain Methods section, got: %s", result)
	}
	if !strings.Contains(result, "## Functions") {
		t.Errorf("should contain Functions section, got: %s", result)
	}

	// Should contain specific symbols
	if !strings.Contains(result, "Server") {
		t.Errorf("should contain Server struct, got: %s", result)
	}
	if !strings.Contains(result, "HandleRequest") {
		t.Errorf("should contain HandleRequest method, got: %s", result)
	}
}

func TestGetFileStructureTool_FileNotIndexed(t *testing.T) {
	store, _ := setupTestStore(t)
	tool := NewGetFileStructureTool(store, "/project")

	args, _ := json.Marshal(GetFileStructureArgs{FilePath: "nonexistent/file.go"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "No indexed symbols") {
		t.Errorf("should indicate no symbols found, got: %s", result)
	}
}

func TestGetFileStructureTool_EmptyPath(t *testing.T) {
	store, _ := setupTestStore(t)
	tool := NewGetFileStructureTool(store, "/project")

	args, _ := json.Marshal(GetFileStructureArgs{FilePath: ""})
	result, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("should return error for empty path, got: %s", result)
	}
}

// --- Helpers for semantic search tests ---

// newFakeEmbedServer creates an httptest server that returns embeddings via embeddingFn.
func newFakeEmbedServer(t *testing.T, dimension int, embeddingFn func(input []string) [][]float32) (*httptest.Server, *indexing.EmbeddingsClient) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embed" {
			http.NotFound(w, r)
			return
		}
		var req struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := struct {
			Embeddings [][]float32 `json:"embeddings"`
		}{Embeddings: embeddingFn(req.Input)}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	client := indexing.NewEmbeddingsClient(srv.URL, "test-model", dimension)
	return srv, client
}

// makeEmbedding creates a simple embedding: first element = val, rest = 0.
func makeEmbedding(dimension int, val float32) []float32 {
	emb := make([]float32, dimension)
	emb[0] = val
	return emb
}

// --- TC-SY-02: SymbolSearch fuzzy (semantic fallback) ---

func TestSymbolSearch_SemanticFallback(t *testing.T) {
	// TC-SY-02: GetByName returns empty, embedder + Search returns result with name similarity
	const dim = 4
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "sem.db")
	store, err := indexing.NewChunkStore(dbPath, dim)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	// Seed chunk with embedding; name "handleRequest" contains "handle"
	chunks := []indexing.CodeChunk{
		{
			ID: "sem-1", FilePath: "/project/handler.go", Content: "func handleRequest() {}",
			StartLine: 5, EndLine: 15, Language: "go",
			ChunkType: indexing.ChunkFunction, Name: "handleRequest",
			Signature: "func handleRequest()",
		},
	}
	chunkEmb := makeEmbedding(dim, 0.9)
	require.NoError(t, store.Store(context.Background(), chunks, [][]float32{chunkEmb}, 0))

	// Fake embedder returns vector similar to chunkEmb
	_, embedder := newFakeEmbedServer(t, dim, func(input []string) [][]float32 {
		results := make([][]float32, len(input))
		for i := range input {
			results[i] = makeEmbedding(dim, 0.85)
		}
		return results
	})

	proxy := NewLocalClientOperationsProxy("/project", WithChunkStore(store), WithEmbedder(embedder))

	// "handle" — no exact match, but semantic search finds "handleRequest" via name similarity
	result, err := proxy.SymbolSearch(context.Background(), "", "handle", 10, nil)
	require.NoError(t, err)

	assert.Contains(t, result, "handleRequest")
	assert.Contains(t, result, "handler.go")
}

// --- TC-SY-03 supplement: tokenizeCamelCase with XMLParser ---

func TestTokenizeCamelCase_XMLParser(t *testing.T) {
	// TC-SY-03: XMLParser -> [xml, parser]
	got := tokenizeCamelCase("XMLParser")
	assert.Equal(t, []string{"xml", "parser"}, got)
}

func TestTokenizeCamelCase_HandleHTTPRequest(t *testing.T) {
	// TC-SY-03: handleHTTPRequest -> [handle, http, request]
	got := tokenizeCamelCase("handleHTTPRequest")
	assert.Equal(t, []string{"handle", "http", "request"}, got)
}

func TestTokenizeCamelCase_MyFunction(t *testing.T) {
	// TC-SY-03: MyFunction -> [my, function]
	got := tokenizeCamelCase("MyFunction")
	assert.Equal(t, []string{"my", "function"}, got)
}

// --- TC-SY-05: GetFunction semantic fallback ---

func TestGetFunctionTool_SemanticFallback(t *testing.T) {
	// TC-SY-05: exact not found, semantic returns result
	const dim = 4
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "fn_sem.db")
	store, err := indexing.NewChunkStore(dbPath, dim)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })

	// Seed a function chunk with embedding; name "initializeApp" contains "initialize"
	chunks := []indexing.CodeChunk{
		{
			ID: "fn-sem-1", FilePath: "/project/service.go", Content: "func initializeApp() { ... }",
			StartLine: 1, EndLine: 10, Language: "go",
			ChunkType: indexing.ChunkFunction, Name: "initializeApp",
			Signature: "func initializeApp()",
		},
	}
	chunkEmb := makeEmbedding(dim, 0.9)
	require.NoError(t, store.Store(context.Background(), chunks, [][]float32{chunkEmb}, 0))

	_, embedder := newFakeEmbedServer(t, dim, func(input []string) [][]float32 {
		results := make([][]float32, len(input))
		for i := range input {
			results[i] = makeEmbedding(dim, 0.88)
		}
		return results
	})

	fnTool := NewGetFunctionTool(store, embedder)

	// "initialize" — not exact match, but semantic search finds "initializeApp"
	args, _ := json.Marshal(GetFunctionArgs{Name: "initialize"})
	result, err := fnTool.InvokableRun(context.Background(), string(args))
	require.NoError(t, err)

	assert.Contains(t, result, "initializeApp")
	assert.Contains(t, result, "service.go")
	assert.Contains(t, result, "```go")
}

// TestGetFileStructureTool_WindowsPath verifies that leading slashes are stripped.
func TestGetFileStructureTool_WindowsPath(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	store, err := indexing.NewChunkStore(dbPath, 4)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	projectRoot := dir
	absFilePath := filepath.Join(projectRoot, "src", "main.go")

	// Ensure directory exists for path resolution
	if err := os.MkdirAll(filepath.Dir(absFilePath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	chunks := []indexing.CodeChunk{
		{
			ID: "w1", FilePath: absFilePath, Content: "func main() {}",
			StartLine: 1, EndLine: 1, Language: "go",
			ChunkType: indexing.ChunkFunction, Name: "main", Signature: "func main()",
		},
	}
	if err := store.Store(context.Background(), chunks, make([][]float32, 1), 0); err != nil {
		t.Fatalf("store: %v", err)
	}

	tool := NewGetFileStructureTool(store, projectRoot)

	// Path with leading slash should be stripped
	args, _ := json.Marshal(GetFileStructureArgs{FilePath: "/src/main.go"})
	result, err := tool.InvokableRun(context.Background(), string(args))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "main") {
		t.Errorf("should find main function, got: %s", result)
	}
}
