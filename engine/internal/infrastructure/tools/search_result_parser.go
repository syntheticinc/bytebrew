package tools

import (
	"fmt"
	"strings"

	"github.com/syntheticinc/bytebrew/engine/internal/domain/search"
)

// isNoResultsMessage returns true if the result string is a "no results" message
func isNoResultsMessage(s string) bool {
	t := strings.TrimSpace(s)
	return strings.HasPrefix(t, "No results found") ||
		strings.HasPrefix(t, "No matches found") ||
		strings.HasPrefix(t, "No symbols found")
}

// parseVectorResults parses vector search results into citations
func parseVectorResults(results []byte) ([]*search.Citation, error) {
	if len(results) == 0 || isNoResultsMessage(string(results)) {
		return nil, nil
	}

	// Parse the vector search output - it's formatted text, not JSON
	var citations []*search.Citation
	lines := strings.Split(string(results), "\n")

	var current *search.Citation
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse "## type: name" header
		if strings.HasPrefix(line, "## ") {
			if current != nil {
				citations = append(citations, current)
			}
			current = &search.Citation{Source: search.SourceVector}
			// Extract type and name
			header := strings.TrimPrefix(line, "## ")
			if colonIdx := strings.Index(header, ": "); colonIdx > 0 {
				current.ChunkType = header[:colonIdx]
				current.Symbol = header[colonIdx+2:]
			}
		} else if strings.HasPrefix(line, "File: ") && current != nil {
			// Parse "File: path:startLine-endLine"
			// Handle Windows paths with drive letter (e.g., C:\Users\...)
			fileInfo := strings.TrimPrefix(line, "File: ")

			// Find the last colon that precedes line numbers (digits and dash)
			lastColonIdx := strings.LastIndex(fileInfo, ":")
			if lastColonIdx > 0 {
				lineRange := fileInfo[lastColonIdx+1:]
				// Verify it looks like line range (digits and optional dash)
				if len(lineRange) > 0 && (lineRange[0] >= '0' && lineRange[0] <= '9') {
					current.FilePath = fileInfo[:lastColonIdx]
					if dashIdx := strings.Index(lineRange, "-"); dashIdx > 0 {
						fmt.Sscanf(lineRange[:dashIdx], "%d", &current.StartLine)
						fmt.Sscanf(lineRange[dashIdx+1:], "%d", &current.EndLine)
					} else {
						fmt.Sscanf(lineRange, "%d", &current.StartLine)
						current.EndLine = current.StartLine
					}
				} else {
					// No line range, just path
					current.FilePath = fileInfo
				}
			} else {
				current.FilePath = fileInfo
			}
		} else if strings.HasPrefix(line, "Score: ") && current != nil {
			// Parse "Score: 0.xxx"
			scoreStr := strings.TrimPrefix(line, "Score: ")
			fmt.Sscanf(scoreStr, "%f", &current.Score)
		}
	}

	if current != nil {
		citations = append(citations, current)
	}

	return citations, nil
}

// parseGrepResults parses grep search results into citations
func parseGrepResults(results string) ([]*search.Citation, error) {
	if results == "" || isNoResultsMessage(results) {
		return nil, nil
	}

	var citations []*search.Citation
	entries := strings.Split(results, "\n\n")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		lines := strings.Split(entry, "\n")
		if len(lines) < 2 {
			continue
		}

		// First line: file:line (handle Windows paths with drive letter)
		firstLine := strings.TrimSpace(lines[0])

		// Find last colon that precedes line number
		lastColonIdx := strings.LastIndex(firstLine, ":")
		if lastColonIdx <= 0 {
			continue
		}

		lineNumStr := firstLine[lastColonIdx+1:]
		if len(lineNumStr) == 0 || lineNumStr[0] < '0' || lineNumStr[0] > '9' {
			continue
		}

		citation := &search.Citation{
			FilePath: firstLine[:lastColonIdx],
			Source:   search.SourceGrep,
			Score:    0.7, // Default score for grep (lower than vector to allow diversity)
		}

		fmt.Sscanf(lineNumStr, "%d", &citation.StartLine)
		citation.EndLine = citation.StartLine

		// Second line: content preview
		if len(lines) >= 2 {
			citation.Preview = strings.TrimSpace(lines[1])
		}

		citations = append(citations, citation)
	}

	return citations, nil
}

// parseSymbolResults parses symbol search results into citations
func parseSymbolResults(results string) ([]*search.Citation, error) {
	if results == "" || isNoResultsMessage(results) {
		return nil, nil
	}

	var citations []*search.Citation
	entries := strings.Split(results, "\n\n")

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		lines := strings.Split(entry, "\n")
		if len(lines) < 2 {
			continue
		}

		// First line: [type] name - signature
		firstLine := strings.TrimSpace(lines[0])
		citation := &search.Citation{
			Source: search.SourceSymbol,
			Score:  0.9, // Higher score for symbol matches
		}

		// Parse [type] name
		if strings.HasPrefix(firstLine, "[") {
			closeIdx := strings.Index(firstLine, "]")
			if closeIdx > 0 {
				citation.ChunkType = firstLine[1:closeIdx]
				rest := strings.TrimSpace(firstLine[closeIdx+1:])
				if dashIdx := strings.Index(rest, " - "); dashIdx > 0 {
					citation.Symbol = rest[:dashIdx]
					citation.Signature = rest[dashIdx+3:]
				} else {
					citation.Symbol = rest
				}
			}
		}

		// Second line: file:startLine-endLine (handle Windows paths)
		if len(lines) >= 2 {
			secondLine := strings.TrimSpace(lines[1])
			lastColonIdx := strings.LastIndex(secondLine, ":")
			if lastColonIdx > 0 {
				lineRange := secondLine[lastColonIdx+1:]
				if len(lineRange) > 0 && lineRange[0] >= '0' && lineRange[0] <= '9' {
					citation.FilePath = secondLine[:lastColonIdx]
					if dashIdx := strings.Index(lineRange, "-"); dashIdx > 0 {
						fmt.Sscanf(lineRange[:dashIdx], "%d", &citation.StartLine)
						fmt.Sscanf(lineRange[dashIdx+1:], "%d", &citation.EndLine)
					} else {
						fmt.Sscanf(lineRange, "%d", &citation.StartLine)
						citation.EndLine = citation.StartLine
					}
				}
			}
		}

		if citation.FilePath != "" {
			citations = append(citations, citation)
		}
	}

	return citations, nil
}
