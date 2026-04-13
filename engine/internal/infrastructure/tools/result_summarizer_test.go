package tools

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSummarizeToolResult_SmartSearch(t *testing.T) {
	tests := []struct {
		name     string
		result   string
		expected string
	}{
		{
			name: "multiple citations (real format)",
			result: `Found 3 results:

1. src/auth/AuthService.ts:45 [vector] (function) validateToken
   preview of the code...

2. src/auth/TokenValidator.ts:12 [grep] validate
   another preview...

3. src/middleware/auth.ts:8 [symbol] (class) AuthMiddleware`,
			expected: "3 citations",
		},
		{
			name: "single citation",
			result: `Found 1 results:

1. single.go:15 [vector] (function) myFunc
   Some content`,
			expected: "1 citation",
		},
		{
			name:     "no citations (Found 0)",
			result:   "Found 0 results:\n\nUse read_file with the paths above to view full content.",
			expected: "0 citations",
		},
		{
			name: "fallback - numbered lines without Found header",
			result: `1. file1.go:10 [vector]
2. file2.go:20 [grep]`,
			expected: "2 citations",
		},
		{
			name:     "empty result",
			result:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeToolResult("smart_search", tt.result)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSummarizeToolResult_WebSearch(t *testing.T) {
	tests := []struct {
		name     string
		result   string
		expected string
	}{
		{
			name: "multiple results",
			result: `1. First result - some title
2. Second result - another title
3. Third result - third title`,
			expected: "3 results",
		},
		{
			name: "single result",
			result: `1. Only result - some title`,
			expected: "1 result",
		},
		{
			name:     "no results",
			result:   "No results found for your query",
			expected: "0 results",
		},
		{
			name:     "empty result",
			result:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeToolResult("web_search", tt.result)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSummarizeToolResult_WebFetch(t *testing.T) {
	tests := []struct {
		name     string
		result   string
		expected string
	}{
		{
			name:     "small content (bytes)",
			result:   "Hello, world!",
			expected: "fetched (13 bytes)",
		},
		{
			name:     "medium content (KB)",
			result:   string(make([]byte, 2048)), // 2 KB
			expected: "fetched (2.0 KB)",
		},
		{
			name:     "large content (MB)",
			result:   string(make([]byte, 1024*1024*2)), // 2 MB
			expected: "fetched (2.0 MB)",
		},
		{
			name:     "empty result",
			result:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeToolResult("web_fetch", tt.result)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestSummarizeToolResult_UnknownTool(t *testing.T) {
	result := "Some random result"
	got := SummarizeToolResult("unknown_tool", result)
	assert.Equal(t, "", got, "Unknown tool should return empty summary")
}

func TestSummarizeToolResult_EmptyResult(t *testing.T) {
	got := SummarizeToolResult("web_search", "")
	assert.Equal(t, "", got, "Empty result should return empty summary")
}

// Helper function tests
func TestFirstLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line",
			input:    "First line",
			expected: "First line",
		},
		{
			name:     "multiple lines",
			input:    "First line\nSecond line\nThird line",
			expected: "First line",
		},
		{
			name:     "long line (truncate)",
			input:    "This is a very long line that exceeds sixty characters and should be truncated at sixty",
			expected: "This is a very long line that exceeds sixty characters and s...",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "cyrillic truncation (UTF-8 safe)",
			input:    "Привет мир! Это очень длинная строка на русском языке которая должна быть обрезана корректно по рунам",
			expected: "Привет мир! Это очень длинная строка на русском языке котора...",
		},
		{
			name:     "emoji truncation (UTF-8 safe)",
			input:    strings.Repeat("😀", 70),
			expected: strings.Repeat("😀", 60) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstLine(tt.input)
			assert.Equal(t, tt.expected, got)
			assert.True(t, utf8.ValidString(got), "Output must be valid UTF-8")
		})
	}
}

func TestHumanizeBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int
		expected string
	}{
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 bytes",
		},
		{
			name:     "kilobytes",
			bytes:    2048,
			expected: "2.0 KB",
		},
		{
			name:     "megabytes",
			bytes:    1024 * 1024 * 3,
			expected: "3.0 MB",
		},
		{
			name:     "fractional KB",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := humanizeBytes(tt.bytes)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestCountOccurrences(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		substr   string
		expected int
	}{
		{
			name:     "multiple occurrences",
			text:     "### Source: file1\n### Source: file2\n### Source: file3",
			substr:   "### Source:",
			expected: 3,
		},
		{
			name:     "single occurrence",
			text:     "### Source: file1",
			substr:   "### Source:",
			expected: 1,
		},
		{
			name:     "no occurrences",
			text:     "No sources here",
			substr:   "### Source:",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countOccurrences(tt.text, tt.substr)
			assert.Equal(t, tt.expected, got)
		})
	}
}
