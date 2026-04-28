// Package search contains domain entities for code search functionality
package search

// Citation represents a compact search result with location and preview
type Citation struct {
	FilePath  string  // Path to the file
	StartLine int     // Starting line number
	EndLine   int     // Ending line number
	Symbol    string  // Symbol name (function, class, etc.)
	Signature string  // Symbol signature if available
	ChunkType string  // Type of code chunk (function, class, method, etc.)
	Preview   string  // Short preview of the content
	Score     float32 // Relevance score (0-1)
	Source    string  // Search source: "vector", "grep", or "symbol"
}

// CitationSource constants for search result sources
const (
	SourceVector = "vector"
	SourceGrep   = "grep"
	SourceSymbol = "symbol"
)
