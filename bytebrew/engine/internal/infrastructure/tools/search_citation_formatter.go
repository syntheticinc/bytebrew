package tools

import (
	"fmt"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain/search"
)

// formatCitations formats citations as compact output for the agent
func formatCitations(citations []*search.Citation) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Found %d results:\n\n", len(citations)))

	for i, c := range citations {
		// Normalize path: use forward slashes, remove leading slashes
		filePath := strings.ReplaceAll(c.FilePath, "\\", "/")
		filePath = strings.TrimPrefix(filePath, "/")

		// Format: path:startLine-endLine [source] symbol
		location := filePath
		if c.StartLine > 0 {
			if c.EndLine > 0 && c.EndLine != c.StartLine {
				location = fmt.Sprintf("%s:%d-%d", filePath, c.StartLine, c.EndLine)
			} else {
				location = fmt.Sprintf("%s:%d", filePath, c.StartLine)
			}
		}

		sb.WriteString(fmt.Sprintf("%d. %s", i+1, location))

		// Add source tag
		sb.WriteString(fmt.Sprintf(" [%s]", c.Source))

		// Add symbol/type info if available
		if c.Symbol != "" {
			if c.ChunkType != "" {
				sb.WriteString(fmt.Sprintf(" (%s) %s", c.ChunkType, c.Symbol))
			} else {
				sb.WriteString(fmt.Sprintf(" %s", c.Symbol))
			}
		}

		// Add signature if available
		if c.Signature != "" {
			sb.WriteString(fmt.Sprintf(": %s", c.Signature))
		}

		sb.WriteString("\n")

		// Add preview if available (truncated)
		if c.Preview != "" {
			preview := c.Preview
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			sb.WriteString(fmt.Sprintf("   %s\n", preview))
		}
	}

	sb.WriteString("\nUse read_file with the paths above to view full content.")

	return sb.String()
}
