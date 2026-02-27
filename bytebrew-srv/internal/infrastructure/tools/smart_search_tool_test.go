package tools

import (
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain/search"
)

func TestMergeResults(t *testing.T) {
	tests := []struct {
		name     string
		vector   []*search.Citation
		grep     []*search.Citation
		symbol   []*search.Citation
		limit    int
		expected int
	}{
		{
			name:     "empty results",
			vector:   nil,
			grep:     nil,
			symbol:   nil,
			limit:    10,
			expected: 0,
		},
		{
			name: "single source",
			vector: []*search.Citation{
				{FilePath: "file1.go", StartLine: 10, Score: 0.9, Source: search.SourceVector},
				{FilePath: "file2.go", StartLine: 20, Score: 0.8, Source: search.SourceVector},
			},
			limit:    10,
			expected: 2,
		},
		{
			name: "deduplicate same file:line",
			vector: []*search.Citation{
				{FilePath: "file1.go", StartLine: 10, Score: 0.7, Source: search.SourceVector},
			},
			grep: []*search.Citation{
				{FilePath: "file1.go", StartLine: 10, Score: 0.9, Source: search.SourceGrep},
			},
			limit:    10,
			expected: 1,
		},
		{
			name: "keep higher score on dedupe",
			vector: []*search.Citation{
				{FilePath: "file1.go", StartLine: 10, Score: 0.5, Source: search.SourceVector},
			},
			grep: []*search.Citation{
				{FilePath: "file1.go", StartLine: 10, Score: 0.9, Source: search.SourceGrep},
			},
			limit:    10,
			expected: 1,
		},
		{
			name: "all three sources",
			vector: []*search.Citation{
				{FilePath: "vector.go", StartLine: 10, Score: 0.9, Source: search.SourceVector},
			},
			grep: []*search.Citation{
				{FilePath: "grep.go", StartLine: 20, Score: 0.8, Source: search.SourceGrep},
			},
			symbol: []*search.Citation{
				{FilePath: "symbol.go", StartLine: 30, Score: 0.95, Source: search.SourceSymbol},
			},
			limit:    10,
			expected: 3,
		},
		{
			name: "respect limit",
			vector: []*search.Citation{
				{FilePath: "file1.go", StartLine: 10, Score: 0.9, Source: search.SourceVector},
				{FilePath: "file2.go", StartLine: 20, Score: 0.8, Source: search.SourceVector},
				{FilePath: "file3.go", StartLine: 30, Score: 0.7, Source: search.SourceVector},
			},
			limit:    2,
			expected: 2,
		},
		{
			name: "sorted by score descending",
			vector: []*search.Citation{
				{FilePath: "low.go", StartLine: 10, Score: 0.3, Source: search.SourceVector},
			},
			grep: []*search.Citation{
				{FilePath: "high.go", StartLine: 20, Score: 0.9, Source: search.SourceGrep},
			},
			symbol: []*search.Citation{
				{FilePath: "mid.go", StartLine: 30, Score: 0.6, Source: search.SourceSymbol},
			},
			limit:    10,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeResults(tt.vector, tt.grep, tt.symbol, tt.limit)

			if len(result) != tt.expected {
				t.Errorf("expected %d results, got %d", tt.expected, len(result))
			}

			// Note: Results are now interleaved for diversity, not strictly sorted by score
			// Verify no duplicates
			seen := make(map[string]bool)
			for _, r := range result {
				key := fmt.Sprintf("%s:%d", r.FilePath, r.StartLine)
				if seen[key] {
					t.Errorf("duplicate result: %s", key)
				}
				seen[key] = true
			}
		})
	}
}

func TestMergeResultsKeepsHigherScore(t *testing.T) {
	vector := []*search.Citation{
		{FilePath: "file.go", StartLine: 10, Score: 0.5, Source: search.SourceVector},
	}
	grep := []*search.Citation{
		{FilePath: "file.go", StartLine: 10, Score: 0.9, Source: search.SourceGrep},
	}

	result := mergeResults(vector, grep, nil, 10)

	if len(result) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result))
	}

	if result[0].Score != 0.9 {
		t.Errorf("expected score 0.9 (higher), got %f", result[0].Score)
	}

	if result[0].Source != search.SourceGrep {
		t.Errorf("expected source grep (higher score), got %s", result[0].Source)
	}
}

func TestFormatCitations(t *testing.T) {
	citations := []*search.Citation{
		{
			FilePath:  "src/main.go",
			StartLine: 10,
			EndLine:   20,
			Symbol:    "handleAuth",
			ChunkType: "function",
			Source:    search.SourceVector,
			Score:     0.95,
		},
		{
			FilePath:  "src/utils.go",
			StartLine: 5,
			EndLine:   5,
			Preview:   "if err != nil { return err }",
			Source:    search.SourceGrep,
			Score:     0.8,
		},
	}

	output := formatCitations(citations)

	// Check key elements
	if output == "" {
		t.Fatal("output should not be empty")
	}

	// Check file paths present
	if !contains(output, "src/main.go") {
		t.Error("output should contain file path src/main.go")
	}
	if !contains(output, "src/utils.go") {
		t.Error("output should contain file path src/utils.go")
	}

	// Check source tags
	if !contains(output, "[vector]") {
		t.Error("output should contain [vector] source tag")
	}
	if !contains(output, "[grep]") {
		t.Error("output should contain [grep] source tag")
	}

	// Check symbol info
	if !contains(output, "handleAuth") {
		t.Error("output should contain symbol name")
	}

	// Check line numbers
	if !contains(output, ":10-20") || !contains(output, ":5") {
		t.Error("output should contain line numbers")
	}
}

func TestParseVectorResults(t *testing.T) {
	input := `## function: handleAuth
File: src/auth/handler.go:45-78
Score: 0.95

## class: UserService
File: src/services/user.go:10-150
Score: 0.87`

	citations, err := parseVectorResults([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(citations))
	}

	// Check first citation
	c1 := citations[0]
	if c1.ChunkType != "function" {
		t.Errorf("expected type 'function', got '%s'", c1.ChunkType)
	}
	if c1.Symbol != "handleAuth" {
		t.Errorf("expected symbol 'handleAuth', got '%s'", c1.Symbol)
	}
	if c1.FilePath != "src/auth/handler.go" {
		t.Errorf("expected path 'src/auth/handler.go', got '%s'", c1.FilePath)
	}
	if c1.Source != search.SourceVector {
		t.Errorf("expected source 'vector', got '%s'", c1.Source)
	}
}

func TestParseGrepResults(t *testing.T) {
	input := `src/auth/handler.go:45
  if err := validateToken(token); err != nil {

src/middleware/auth.go:112
  return handleAuthError(ctx, err)`

	citations, err := parseGrepResults(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(citations))
	}

	// Check first citation
	c1 := citations[0]
	if c1.FilePath != "src/auth/handler.go" {
		t.Errorf("expected path 'src/auth/handler.go', got '%s'", c1.FilePath)
	}
	if c1.StartLine != 45 {
		t.Errorf("expected line 45, got %d", c1.StartLine)
	}
	if c1.Source != search.SourceGrep {
		t.Errorf("expected source 'grep', got '%s'", c1.Source)
	}
}

func TestParseSymbolResults(t *testing.T) {
	input := `[function] handleAuthError - func(ctx context.Context, err error) error
  src/auth/handler.go:45-78

[class] UserService
  src/services/user.go:10-150`

	citations, err := parseSymbolResults(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(citations))
	}

	// Check first citation
	c1 := citations[0]
	if c1.ChunkType != "function" {
		t.Errorf("expected type 'function', got '%s'", c1.ChunkType)
	}
	if c1.Symbol != "handleAuthError" {
		t.Errorf("expected symbol 'handleAuthError', got '%s'", c1.Symbol)
	}
	if c1.Signature != "func(ctx context.Context, err error) error" {
		t.Errorf("expected signature, got '%s'", c1.Signature)
	}
	if c1.Source != search.SourceSymbol {
		t.Errorf("expected source 'symbol', got '%s'", c1.Source)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
