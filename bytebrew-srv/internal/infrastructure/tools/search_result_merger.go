package tools

import (
	"fmt"
	"sort"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain/search"
)

// mergeResults combines results from all search strategies with diversity.
// First deduplicates by file:line keeping highest score, then selects
// with interleaved strategy to ensure representation from each source.
func mergeResults(vector, grep, symbol []*search.Citation, limit int) []*search.Citation {
	// Phase 1: Deduplicate all results by file:line, keeping highest score
	seen := make(map[string]*search.Citation)

	addToSeen := func(citations []*search.Citation) {
		for _, c := range citations {
			if c == nil || c.FilePath == "" {
				continue
			}
			key := fmt.Sprintf("%s:%d", c.FilePath, c.StartLine)
			existing, exists := seen[key]
			if !exists || c.Score > existing.Score {
				seen[key] = c
			}
		}
	}

	addToSeen(symbol)
	addToSeen(grep)
	addToSeen(vector)

	// Phase 2: Separate back into source groups (deduplicated)
	var dedupSymbol, dedupGrep, dedupVector []*search.Citation
	for _, c := range seen {
		switch c.Source {
		case search.SourceSymbol:
			dedupSymbol = append(dedupSymbol, c)
		case search.SourceGrep:
			dedupGrep = append(dedupGrep, c)
		case search.SourceVector:
			dedupVector = append(dedupVector, c)
		}
	}

	// Sort each group by score
	sortByScore(dedupSymbol)
	sortByScore(dedupGrep)
	sortByScore(dedupVector)

	// Phase 3: Interleaved selection for diversity
	result := make([]*search.Citation, 0, limit)
	si, gi, vi := 0, 0, 0

	for len(result) < limit && (si < len(dedupSymbol) || gi < len(dedupGrep) || vi < len(dedupVector)) {
		// Symbol first (exact name matches - highest priority)
		if si < len(dedupSymbol) && len(result) < limit {
			result = append(result, dedupSymbol[si])
			si++
		}

		// Grep second (exact pattern matches)
		if gi < len(dedupGrep) && len(result) < limit {
			result = append(result, dedupGrep[gi])
			gi++
		}

		// Vector third (semantic similarity)
		if vi < len(dedupVector) && len(result) < limit {
			result = append(result, dedupVector[vi])
			vi++
		}
	}

	return result
}

// sortByScore sorts citations by score descending
func sortByScore(citations []*search.Citation) {
	sort.Slice(citations, func(i, j int) bool {
		if citations[i] == nil {
			return false
		}
		if citations[j] == nil {
			return true
		}
		return citations[i].Score > citations[j].Score
	})
}
