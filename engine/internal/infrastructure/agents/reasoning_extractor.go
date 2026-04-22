package agents

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// ReasoningExtractor extracts reasoning content from model responses.
// Does not check model name — if reasoning content is present, it extracts it.
type ReasoningExtractor struct{}

// NewReasoningExtractor creates a new ReasoningExtractor
func NewReasoningExtractor() *ReasoningExtractor {
	return &ReasoningExtractor{}
}

// ExtractReasoning extracts reasoning content from message.
// Returns reasoning content and whether it was found.
func (r *ReasoningExtractor) ExtractReasoning(msg *schema.Message) (string, bool) {
	if msg == nil || msg.ReasoningContent == "" {
		return "", false
	}

	content := cleanReasoningContent(msg.ReasoningContent)

	slog.DebugContext(context.Background(), "reasoning extracted", "length", len(content))
	return content, true
}

// cleanReasoningContent fixes garbled content from OpenRouter streaming.
// When eino-ext reads "reasoning" from ExtraFields, it doesn't JSON-decode
// the value, so each streamed chunk includes JSON quotes.
// This results in content like: "П""ользователь"" просит""
// We need to remove these spurious quotes.
func cleanReasoningContent(content string) string {
	// Check if content looks garbled (contains "" pattern which indicates
	// concatenated JSON strings)
	if !strings.Contains(content, "\"\"") && !strings.HasPrefix(content, "\"") {
		return content
	}

	// Try to decode as JSON string first (handles proper escaping)
	if strings.HasPrefix(content, "\"") {
		var decoded string
		if err := json.Unmarshal([]byte(content), &decoded); err == nil {
			return decoded
		}
	}

	// Fallback: manual cleanup for garbled concatenated JSON strings
	// Pattern: "chunk1""chunk2""chunk3"
	// Remove leading quote
	if strings.HasPrefix(content, "\"") {
		content = content[1:]
	}
	// Remove trailing quote
	if strings.HasSuffix(content, "\"") {
		content = content[:len(content)-1]
	}
	// Replace double quotes between chunks with empty string
	content = strings.ReplaceAll(content, "\"\"", "")

	return content
}
