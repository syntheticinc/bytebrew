package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"unicode"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/indexing"
)

// SymbolSearch finds code symbols by name using the chunk store.
// Falls back to CamelCase tokenization when exact match fails and no embedder is available.
func (p *LocalClientOperationsProxy) SymbolSearch(ctx context.Context, _, symbolName string, limit int32, symbolTypes []string) (string, error) {
	if p.chunkStore == nil {
		return "Index not available. Run indexing first.", nil
	}

	if symbolName == "" {
		return "[ERROR] symbol name is required", nil
	}

	if limit <= 0 {
		limit = 10
	}

	typeFilter := buildTypeFilter(symbolTypes)

	// Step 1: Exact match
	chunks, err := p.chunkStore.GetByName(ctx, symbolName)
	if err != nil {
		return "", fmt.Errorf("symbol search by name: %w", err)
	}

	filtered := filterByType(chunks, typeFilter)
	if len(filtered) > 0 {
		slog.InfoContext(ctx, "symbol search exact match", "symbol", symbolName, "results", len(filtered))
		return formatSymbolResults(filtered, int(limit)), nil
	}

	// Step 2: Semantic search via embedder if available
	if p.embedder != nil {
		results, err := p.semanticSymbolSearch(ctx, symbolName, int(limit), typeFilter)
		if err != nil {
			slog.WarnContext(ctx, "semantic symbol search failed, trying camelCase", "error", err)
		} else if len(results) > 0 {
			slog.InfoContext(ctx, "symbol search semantic match", "symbol", symbolName, "results", len(results))
			return formatSymbolResults(results, int(limit)), nil
		}
	}

	// Step 3: CamelCase tokenization fallback
	tokens := tokenizeCamelCase(symbolName)
	if len(tokens) <= 1 {
		return fmt.Sprintf("No symbols found matching %q", symbolName), nil
	}

	var allChunks []indexing.CodeChunk
	seen := make(map[string]bool)

	for _, token := range tokens {
		tokenChunks, err := p.chunkStore.GetByName(ctx, token)
		if err != nil {
			continue
		}
		for _, c := range tokenChunks {
			if seen[c.ID] {
				continue
			}
			seen[c.ID] = true
			allChunks = append(allChunks, c)
		}
	}

	filtered = filterByType(allChunks, typeFilter)
	if len(filtered) == 0 {
		return fmt.Sprintf("No symbols found matching %q", symbolName), nil
	}

	slog.InfoContext(ctx, "symbol search camelCase match", "symbol", symbolName, "tokens", tokens, "results", len(filtered))
	return formatSymbolResults(filtered, int(limit)), nil
}

// SearchCode performs vector-based semantic code search.
func (p *LocalClientOperationsProxy) SearchCode(ctx context.Context, _, query, _ string, limit int32, minScore float32) ([]byte, error) {
	if p.chunkStore == nil || p.embedder == nil {
		msg := map[string]string{"error": "Index or embeddings not available. Run indexing first."}
		b, _ := json.Marshal(msg)
		return b, nil
	}

	if limit <= 0 {
		limit = 5
	}

	embedding, err := p.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	results, err := p.chunkStore.Search(ctx, embedding, int(limit)*2) // fetch extra for filtering
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}

	// Filter by minScore
	var filtered []indexing.SearchResult
	for _, r := range results {
		if minScore > 0 && r.Score < minScore {
			continue
		}
		filtered = append(filtered, r)
	}

	if int32(len(filtered)) > limit {
		filtered = filtered[:limit]
	}

	type resultEntry struct {
		FilePath  string  `json:"file_path"`
		Name      string  `json:"name"`
		ChunkType string  `json:"chunk_type"`
		Content   string  `json:"content"`
		StartLine int     `json:"start_line"`
		EndLine   int     `json:"end_line"`
		Score     float32 `json:"score"`
	}

	entries := make([]resultEntry, 0, len(filtered))
	for _, r := range filtered {
		entries = append(entries, resultEntry{
			FilePath:  r.Chunk.FilePath,
			Name:      r.Chunk.Name,
			ChunkType: string(r.Chunk.ChunkType),
			Content:   r.Chunk.Content,
			StartLine: r.Chunk.StartLine,
			EndLine:   r.Chunk.EndLine,
			Score:     r.Score,
		})
	}

	slog.InfoContext(ctx, "search code", "query", query, "results", len(entries))
	return json.Marshal(entries)
}

// semanticSymbolSearch embeds the symbol name and searches for similar chunks.
func (p *LocalClientOperationsProxy) semanticSymbolSearch(ctx context.Context, symbolName string, limit int, typeFilter map[indexing.ChunkType]bool) ([]indexing.CodeChunk, error) {
	embedding, err := p.embedder.Embed(ctx, symbolName)
	if err != nil {
		return nil, fmt.Errorf("embed symbol name: %w", err)
	}

	results, err := p.chunkStore.Search(ctx, embedding, limit*3)
	if err != nil {
		return nil, fmt.Errorf("search by embedding: %w", err)
	}

	var chunks []indexing.CodeChunk
	nameLower := strings.ToLower(symbolName)

	for _, r := range results {
		if r.Score < 0.3 {
			continue
		}
		chunkNameLower := strings.ToLower(r.Chunk.Name)
		if !strings.Contains(chunkNameLower, nameLower) && !strings.Contains(nameLower, chunkNameLower) {
			continue
		}
		if len(typeFilter) > 0 && !typeFilter[r.Chunk.ChunkType] {
			continue
		}
		chunks = append(chunks, r.Chunk)
	}

	return chunks, nil
}

// tokenizeCamelCase splits a CamelCase or mixedCase string into lowercase tokens.
// Example: "MyFunction" -> ["my", "function"], "handleHTTPRequest" -> ["handle", "http", "request"]
func tokenizeCamelCase(s string) []string {
	if s == "" {
		return nil
	}

	var tokens []string
	var current strings.Builder

	runes := []rune(s)
	for i, r := range runes {
		if i == 0 {
			current.WriteRune(unicode.ToLower(r))
			continue
		}

		if unicode.IsUpper(r) {
			// Check if this is part of an acronym (consecutive uppercase)
			prevUpper := unicode.IsUpper(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])

			if prevUpper && nextLower {
				// End of acronym: "HTTPRequest" -> last 'P' starts new word "Request"
				// Flush current (without last char would be wrong, so flush all)
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				current.WriteRune(unicode.ToLower(r))
			} else if !prevUpper {
				// New word: "myFunction" -> 'F' starts "function"
				if current.Len() > 0 {
					tokens = append(tokens, current.String())
					current.Reset()
				}
				current.WriteRune(unicode.ToLower(r))
			} else {
				// Continuing acronym: "HTTP"
				current.WriteRune(unicode.ToLower(r))
			}
		} else if r == '_' || r == '-' {
			// Separator
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		} else {
			current.WriteRune(unicode.ToLower(r))
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

// buildTypeFilter creates a set of allowed ChunkTypes from string slice.
func buildTypeFilter(symbolTypes []string) map[indexing.ChunkType]bool {
	if len(symbolTypes) == 0 {
		return nil
	}
	filter := make(map[indexing.ChunkType]bool, len(symbolTypes))
	for _, st := range symbolTypes {
		filter[indexing.ChunkType(st)] = true
	}
	return filter
}

// filterByType filters chunks by type if typeFilter is non-empty.
func filterByType(chunks []indexing.CodeChunk, typeFilter map[indexing.ChunkType]bool) []indexing.CodeChunk {
	if len(typeFilter) == 0 {
		return chunks
	}
	var filtered []indexing.CodeChunk
	for _, c := range chunks {
		if typeFilter[c.ChunkType] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// formatSymbolResults formats chunks into a readable string.
func formatSymbolResults(chunks []indexing.CodeChunk, limit int) string {
	if len(chunks) > limit {
		chunks = chunks[:limit]
	}

	// Sort by file path, then start line
	sort.Slice(chunks, func(i, j int) bool {
		if chunks[i].FilePath != chunks[j].FilePath {
			return chunks[i].FilePath < chunks[j].FilePath
		}
		return chunks[i].StartLine < chunks[j].StartLine
	})

	var sb strings.Builder
	for i, c := range chunks {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sig := c.Signature
		if sig == "" {
			sig = c.Name
		}
		sb.WriteString(fmt.Sprintf("[%s] %s\n  %s:%d-%d", c.ChunkType, sig, c.FilePath, c.StartLine, c.EndLine))
	}

	return sb.String()
}
