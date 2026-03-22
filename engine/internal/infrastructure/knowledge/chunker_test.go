package knowledge

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownChunker_SplitByHeaders(t *testing.T) {
	content := `# Title

Intro paragraph.

## Section One

Content of section one.

## Section Two

Content of section two.

## Section Three

Content of section three.
`
	chunker := &MarkdownChunker{MaxTokens: 1000}
	chunks := chunker.Chunk(content)

	require.Len(t, chunks, 4)
	assert.Contains(t, chunks[0].Content, "# Title")
	assert.Contains(t, chunks[0].Content, "Intro paragraph")
	assert.Contains(t, chunks[1].Content, "## Section One")
	assert.Contains(t, chunks[2].Content, "## Section Two")
	assert.Contains(t, chunks[3].Content, "## Section Three")

	// Verify order is sequential
	for i, c := range chunks {
		assert.Equal(t, i, c.Order)
	}
}

func TestMarkdownChunker_LargeSection(t *testing.T) {
	// Create a section that exceeds MaxTokens (100 tokens = 400 chars)
	largeParagraph1 := strings.Repeat("Word ", 60) // ~300 chars
	largeParagraph2 := strings.Repeat("Text ", 60) // ~300 chars

	content := "## Big Section\n\n" + largeParagraph1 + "\n\n" + largeParagraph2
	chunker := &MarkdownChunker{MaxTokens: 100}
	chunks := chunker.Chunk(content)

	require.True(t, len(chunks) >= 2, "large section should be split into multiple chunks, got %d", len(chunks))
	for _, c := range chunks {
		assert.True(t, estimateTokens(c.Content) <= 100+20, // some tolerance for paragraph boundaries
			"chunk should not greatly exceed MaxTokens: %d tokens", estimateTokens(c.Content))
	}
}

func TestMarkdownChunker_EmptyContent(t *testing.T) {
	chunker := &MarkdownChunker{MaxTokens: 1000}

	assert.Nil(t, chunker.Chunk(""))
	assert.Nil(t, chunker.Chunk("   "))
	assert.Nil(t, chunker.Chunk("\n\n"))
}

func TestMarkdownChunker_NoHeaders(t *testing.T) {
	content := "Just a plain paragraph without any headers."
	chunker := &MarkdownChunker{MaxTokens: 1000}
	chunks := chunker.Chunk(content)

	require.Len(t, chunks, 1)
	assert.Equal(t, content, chunks[0].Content)
	assert.Equal(t, 0, chunks[0].Order)
}

func TestPlainTextChunker_SplitByParagraphs(t *testing.T) {
	// Create paragraphs that individually fit but together exceed MaxTokens
	para1 := strings.Repeat("Alpha ", 40)  // ~240 chars = ~60 tokens
	para2 := strings.Repeat("Bravo ", 40)  // ~240 chars = ~60 tokens
	para3 := strings.Repeat("Charlie ", 40) // ~320 chars = ~80 tokens

	content := para1 + "\n\n" + para2 + "\n\n" + para3
	chunker := &PlainTextChunker{MaxTokens: 100}
	chunks := chunker.Chunk(content)

	require.True(t, len(chunks) >= 2, "should split into multiple chunks, got %d", len(chunks))
	for i, c := range chunks {
		assert.Equal(t, i, c.Order)
	}
}

func TestPlainTextChunker_MergeSmall(t *testing.T) {
	content := "Small one.\n\nSmall two.\n\nSmall three."
	chunker := &PlainTextChunker{MaxTokens: 1000}
	chunks := chunker.Chunk(content)

	// All small paragraphs should be merged into one chunk
	require.Len(t, chunks, 1)
	assert.Contains(t, chunks[0].Content, "Small one.")
	assert.Contains(t, chunks[0].Content, "Small two.")
	assert.Contains(t, chunks[0].Content, "Small three.")
}

func TestPlainTextChunker_LargeParagraph(t *testing.T) {
	// A single paragraph with many sentences that exceeds MaxTokens
	sentences := make([]string, 20)
	for i := range sentences {
		sentences[i] = strings.Repeat("word ", 10) + "end."
	}
	content := strings.Join(sentences, " ")

	chunker := &PlainTextChunker{MaxTokens: 50}
	chunks := chunker.Chunk(content)

	require.True(t, len(chunks) >= 2, "large paragraph should be split by sentences, got %d", len(chunks))
}

func TestPlainTextChunker_EmptyContent(t *testing.T) {
	chunker := &PlainTextChunker{MaxTokens: 1000}

	assert.Nil(t, chunker.Chunk(""))
	assert.Nil(t, chunker.Chunk("   "))
}

func TestChunkerForFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantType string
	}{
		{"markdown", "docs/readme.md", "*knowledge.MarkdownChunker"},
		{"markdown uppercase", "docs/README.MD", "*knowledge.MarkdownChunker"},
		{"plain text", "notes.txt", "*knowledge.PlainTextChunker"},
		{"unknown extension", "data.csv", "*knowledge.PlainTextChunker"},
		{"no extension", "Makefile", "*knowledge.PlainTextChunker"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ChunkerForFile(tt.path)
			require.NotNil(t, c)

			switch tt.wantType {
			case "*knowledge.MarkdownChunker":
				_, ok := c.(*MarkdownChunker)
				assert.True(t, ok, "expected MarkdownChunker for %s", tt.path)
			case "*knowledge.PlainTextChunker":
				_, ok := c.(*PlainTextChunker)
				assert.True(t, ok, "expected PlainTextChunker for %s", tt.path)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	assert.Equal(t, 0, estimateTokens(""))
	assert.Equal(t, 1, estimateTokens("Hi"))     // 2 chars / 4 = ceil to 1
	assert.Equal(t, 3, estimateTokens("Hello World")) // 11 chars / 4 = ceil to 3
	assert.Equal(t, 25, estimateTokens(strings.Repeat("a", 100)))
}

func TestSplitSentences(t *testing.T) {
	text := "First sentence. Second sentence. Third sentence."
	sentences := splitSentences(text)

	require.Len(t, sentences, 3)
	assert.Equal(t, "First sentence.", sentences[0])
	assert.Equal(t, "Second sentence.", sentences[1])
	assert.Equal(t, "Third sentence.", sentences[2])
}

func TestSplitSentences_NoPeriodSpace(t *testing.T) {
	text := "No sentence boundary here"
	sentences := splitSentences(text)

	require.Len(t, sentences, 1)
	assert.Equal(t, text, sentences[0])
}
