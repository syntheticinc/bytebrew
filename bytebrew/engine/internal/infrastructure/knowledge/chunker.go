package knowledge

import (
	"path/filepath"
	"strings"
)

// Chunk represents a piece of a document.
type Chunk struct {
	Content string
	Order   int
}

// Chunker splits document content into chunks suitable for embedding.
type Chunker interface {
	Chunk(content string) []Chunk
}

const defaultMaxTokens = 1000

// charsPerToken is a rough approximation for token estimation.
const charsPerToken = 4

// MarkdownChunker splits by ## headers, respecting MaxTokens per chunk.
type MarkdownChunker struct {
	MaxTokens int
}

// PlainTextChunker splits by paragraph boundaries, merging small ones.
type PlainTextChunker struct {
	MaxTokens int
}

// ChunkerForFile returns an appropriate chunker based on file extension.
func ChunkerForFile(path string) Chunker {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".md" {
		return &MarkdownChunker{MaxTokens: defaultMaxTokens}
	}
	return &PlainTextChunker{MaxTokens: defaultMaxTokens}
}

func (c *MarkdownChunker) maxTokens() int {
	if c.MaxTokens <= 0 {
		return defaultMaxTokens
	}
	return c.MaxTokens
}

func (c *PlainTextChunker) maxTokens() int {
	if c.MaxTokens <= 0 {
		return defaultMaxTokens
	}
	return c.MaxTokens
}

func estimateTokens(s string) int {
	if len(s) == 0 {
		return 0
	}
	return (len(s) + charsPerToken - 1) / charsPerToken
}

// Chunk splits markdown content by level-2 headers.
// Sections exceeding MaxTokens are split further by paragraphs.
func (c *MarkdownChunker) Chunk(content string) []Chunk {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	sections := splitMarkdownSections(content)
	maxTok := c.maxTokens()

	var chunks []Chunk
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		if estimateTokens(section) <= maxTok {
			chunks = append(chunks, Chunk{Content: section, Order: len(chunks)})
			continue
		}
		// Split oversized section by paragraphs
		subChunks := splitByParagraphs(section, maxTok)
		for _, sc := range subChunks {
			chunks = append(chunks, Chunk{Content: sc, Order: len(chunks)})
		}
	}
	return chunks
}

// splitMarkdownSections splits content by lines starting with "## ".
func splitMarkdownSections(content string) []string {
	lines := strings.Split(content, "\n")
	var sections []string
	var current []string

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") && len(current) > 0 {
			sections = append(sections, strings.Join(current, "\n"))
			current = nil
		}
		current = append(current, line)
	}
	if len(current) > 0 {
		sections = append(sections, strings.Join(current, "\n"))
	}
	return sections
}

// Chunk splits plain text by double newlines, merging small paragraphs.
// Large paragraphs are split by sentences.
func (c *PlainTextChunker) Chunk(content string) []Chunk {
	if strings.TrimSpace(content) == "" {
		return nil
	}

	paragraphs := splitParagraphs(content)
	maxTok := c.maxTokens()

	var chunks []Chunk
	var buf strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		// Large paragraph: flush buffer, then split by sentences
		if estimateTokens(para) > maxTok {
			if buf.Len() > 0 {
				chunks = append(chunks, Chunk{Content: strings.TrimSpace(buf.String()), Order: len(chunks)})
				buf.Reset()
			}
			sentenceChunks := splitBySentences(para, maxTok)
			for _, sc := range sentenceChunks {
				chunks = append(chunks, Chunk{Content: sc, Order: len(chunks)})
			}
			continue
		}

		// Would adding this paragraph exceed the limit?
		combined := buf.String()
		if combined != "" {
			combined += "\n\n"
		}
		combined += para

		if estimateTokens(combined) > maxTok && buf.Len() > 0 {
			chunks = append(chunks, Chunk{Content: strings.TrimSpace(buf.String()), Order: len(chunks)})
			buf.Reset()
		}

		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(para)
	}

	if buf.Len() > 0 {
		chunks = append(chunks, Chunk{Content: strings.TrimSpace(buf.String()), Order: len(chunks)})
	}
	return chunks
}

// splitParagraphs splits text by double newlines.
func splitParagraphs(content string) []string {
	return strings.Split(content, "\n\n")
}

// splitByParagraphs splits a large text block into chunks by paragraph boundaries,
// respecting maxTokens. Never splits mid-paragraph.
func splitByParagraphs(text string, maxTokens int) []string {
	paragraphs := splitParagraphs(text)
	var result []string
	var buf strings.Builder

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}

		combined := buf.String()
		if combined != "" {
			combined += "\n\n"
		}
		combined += para

		if estimateTokens(combined) > maxTokens && buf.Len() > 0 {
			result = append(result, strings.TrimSpace(buf.String()))
			buf.Reset()
		}

		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(para)
	}

	if buf.Len() > 0 {
		result = append(result, strings.TrimSpace(buf.String()))
	}
	return result
}

// splitBySentences splits a large paragraph into chunks by ". " boundaries.
func splitBySentences(text string, maxTokens int) []string {
	sentences := splitSentences(text)
	var result []string
	var buf strings.Builder

	for _, sent := range sentences {
		sent = strings.TrimSpace(sent)
		if sent == "" {
			continue
		}

		combined := buf.String()
		if combined != "" {
			combined += " "
		}
		combined += sent

		if estimateTokens(combined) > maxTokens && buf.Len() > 0 {
			result = append(result, strings.TrimSpace(buf.String()))
			buf.Reset()
		}

		if buf.Len() > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(sent)
	}

	if buf.Len() > 0 {
		result = append(result, strings.TrimSpace(buf.String()))
	}
	return result
}

// splitSentences splits text by ". " keeping the period with the preceding sentence.
func splitSentences(text string) []string {
	var sentences []string
	remaining := text
	for {
		idx := strings.Index(remaining, ". ")
		if idx < 0 {
			break
		}
		sentences = append(sentences, remaining[:idx+1]) // include the period
		remaining = remaining[idx+2:]
	}
	if strings.TrimSpace(remaining) != "" {
		sentences = append(sentences, remaining)
	}
	return sentences
}
