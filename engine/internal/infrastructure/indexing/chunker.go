package indexing

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// ChunkType identifies the kind of code chunk.
type ChunkType string

const (
	ChunkFunction  ChunkType = "function"
	ChunkMethod    ChunkType = "method"
	ChunkClass     ChunkType = "class"
	ChunkStruct    ChunkType = "struct"
	ChunkInterface ChunkType = "interface"
	ChunkOther     ChunkType = "other"
)

const minChunkBytes = 10

// CodeChunk represents a parsed code block within a file.
type CodeChunk struct {
	ID         string    // first 16 hex chars of SHA256(filePath:startLine:name)
	FilePath   string    // absolute path
	Content    string    // chunk source code
	StartLine  int       // 1-indexed
	EndLine    int       // 1-indexed, inclusive
	Language   string    // language identifier
	ChunkType  ChunkType // function, method, class, etc.
	Name       string    // symbol name
	ParentName string    // enclosing symbol name, if nested
	Signature  string    // definition line(s) up to body start
}

// chunkPattern defines a regex pattern and its corresponding chunk type.
type chunkPattern struct {
	re        *regexp.Regexp
	chunkType ChunkType
}

// Chunker splits source files into semantic code chunks using regex patterns.
type Chunker struct {
	patterns map[string][]chunkPattern
}

// NewChunker creates a chunker with pre-compiled language patterns.
func NewChunker() *Chunker {
	c := &Chunker{
		patterns: make(map[string][]chunkPattern),
	}
	c.initPatterns()
	return c
}

// ChunkFile splits a file's content into code chunks based on its language.
func (c *Chunker) ChunkFile(filePath, content, language string) []CodeChunk {
	patterns, ok := c.patterns[language]
	if !ok {
		return c.wholeFileChunk(filePath, content, language)
	}

	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return nil
	}

	if language == "python" {
		return c.chunkPython(filePath, content, language, lines, patterns)
	}

	return c.chunkBraceLanguage(filePath, content, language, lines, patterns)
}

// chunkBraceLanguage handles languages that use {} for block delimiting.
func (c *Chunker) chunkBraceLanguage(filePath, content, language string, lines []string, patterns []chunkPattern) []CodeChunk {
	type match struct {
		lineIdx   int
		name      string
		chunkType ChunkType
	}

	var matches []match
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, p := range patterns {
			m := p.re.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			name := m[1]
			matches = append(matches, match{lineIdx: i, name: name, chunkType: p.chunkType})
			break
		}
	}

	if len(matches) == 0 {
		return c.wholeFileChunk(filePath, content, language)
	}

	var chunks []CodeChunk
	for idx, m := range matches {
		startLine := m.lineIdx
		endLine := c.findBraceEnd(lines, startLine)

		// Clamp endLine to not overlap with next match
		if idx+1 < len(matches) && endLine >= matches[idx+1].lineIdx {
			endLine = matches[idx+1].lineIdx - 1
		}
		if endLine < startLine {
			endLine = startLine
		}

		chunkContent := strings.Join(lines[startLine:endLine+1], "\n")
		if len(chunkContent) < minChunkBytes {
			continue
		}

		sig := extractSignature(lines, startLine)

		chunk := CodeChunk{
			ID:        generateChunkID(filePath, startLine+1, m.name),
			FilePath:  filePath,
			Content:   chunkContent,
			StartLine: startLine + 1,
			EndLine:   endLine + 1,
			Language:  language,
			ChunkType: m.chunkType,
			Name:      m.name,
			Signature: sig,
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		return c.wholeFileChunk(filePath, content, language)
	}
	return chunks
}

// chunkPython handles Python files using indentation-based block detection.
func (c *Chunker) chunkPython(filePath, content, language string, lines []string, patterns []chunkPattern) []CodeChunk {
	type match struct {
		lineIdx   int
		name      string
		chunkType ChunkType
		indent    int
	}

	var matches []match
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		for _, p := range patterns {
			m := p.re.FindStringSubmatch(trimmed)
			if m == nil {
				continue
			}
			indent := countLeadingSpaces(line)
			matches = append(matches, match{lineIdx: i, name: m[1], chunkType: p.chunkType, indent: indent})
			break
		}
	}

	if len(matches) == 0 {
		return c.wholeFileChunk(filePath, content, language)
	}

	var chunks []CodeChunk
	for idx, m := range matches {
		endLine := c.findIndentEnd(lines, m.lineIdx, m.indent)

		// Clamp to next match
		if idx+1 < len(matches) && endLine >= matches[idx+1].lineIdx {
			endLine = matches[idx+1].lineIdx - 1
		}
		if endLine < m.lineIdx {
			endLine = m.lineIdx
		}

		chunkContent := strings.Join(lines[m.lineIdx:endLine+1], "\n")
		if len(chunkContent) < minChunkBytes {
			continue
		}

		sig := strings.TrimSpace(lines[m.lineIdx])

		chunk := CodeChunk{
			ID:        generateChunkID(filePath, m.lineIdx+1, m.name),
			FilePath:  filePath,
			Content:   chunkContent,
			StartLine: m.lineIdx + 1,
			EndLine:   endLine + 1,
			Language:  language,
			ChunkType: m.chunkType,
			Name:      m.name,
			Signature: sig,
		}
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		return c.wholeFileChunk(filePath, content, language)
	}
	return chunks
}

// findBraceEnd finds the closing brace for a block starting at startLine.
func (c *Chunker) findBraceEnd(lines []string, startLine int) int {
	depth := 0
	started := false

	for i := startLine; i < len(lines); i++ {
		for _, ch := range lines[i] {
			if ch == '{' {
				depth++
				started = true
			}
			if ch == '}' {
				depth--
			}
		}
		if started && depth <= 0 {
			return i
		}
	}

	// No matching brace found — return last line of file
	return len(lines) - 1
}

// findIndentEnd finds the end of a Python block by indentation level.
func (c *Chunker) findIndentEnd(lines []string, startLine, defIndent int) int {
	lastContentLine := startLine
	for i := startLine + 1; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Skip blank lines
		if trimmed == "" {
			continue
		}

		indent := countLeadingSpaces(line)
		if indent <= defIndent {
			break
		}
		lastContentLine = i
	}
	return lastContentLine
}

// wholeFileChunk returns the entire file as a single "other" chunk.
func (c *Chunker) wholeFileChunk(filePath, content, language string) []CodeChunk {
	if len(content) < minChunkBytes {
		return nil
	}
	lineCount := strings.Count(content, "\n") + 1
	return []CodeChunk{{
		ID:        generateChunkID(filePath, 1, "file"),
		FilePath:  filePath,
		Content:   content,
		StartLine: 1,
		EndLine:   lineCount,
		Language:  language,
		ChunkType: ChunkOther,
		Name:      "file",
	}}
}

// generateChunkID creates a deterministic ID from file path, line, and name.
func generateChunkID(filePath string, startLine int, name string) string {
	data := fmt.Sprintf("%s:%d:%s", filePath, startLine, name)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash[:8]) // 16 hex chars
}

// extractSignature extracts the function/class signature from the definition line(s).
func extractSignature(lines []string, startLine int) string {
	if startLine >= len(lines) {
		return ""
	}

	sig := strings.TrimSpace(lines[startLine])
	// If the line ends with '{', strip it for a cleaner signature
	sig = strings.TrimRight(sig, " {")
	return sig
}

// countLeadingSpaces returns the number of leading space characters.
// Tabs are counted as 4 spaces.
func countLeadingSpaces(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}

// initPatterns initializes regex patterns for all supported languages.
func (c *Chunker) initPatterns() {
	// Go
	c.patterns["go"] = []chunkPattern{
		{regexp.MustCompile(`^func\s+\([^)]+\)\s+(\w+)\s*\(`), ChunkMethod},
		{regexp.MustCompile(`^func\s+(\w+)\s*\(`), ChunkFunction},
		{regexp.MustCompile(`^type\s+(\w+)\s+struct\s*\{`), ChunkStruct},
		{regexp.MustCompile(`^type\s+(\w+)\s+interface\s*\{`), ChunkInterface},
	}

	// TypeScript / JavaScript
	tsPatterns := []chunkPattern{
		{regexp.MustCompile(`^(?:export\s+)?class\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^(?:export\s+)?interface\s+(\w+)`), ChunkInterface},
		{regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+(\w+)`), ChunkFunction},
		{regexp.MustCompile(`^(?:export\s+)?const\s+(\w+)\s*=\s*(?:async\s+)?\(`), ChunkFunction},
	}
	c.patterns["typescript"] = tsPatterns
	c.patterns["javascript"] = tsPatterns

	// Python
	c.patterns["python"] = []chunkPattern{
		{regexp.MustCompile(`^class\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^(?:async\s+)?def\s+(\w+)\s*\(`), ChunkFunction},
	}

	// Rust
	c.patterns["rust"] = []chunkPattern{
		{regexp.MustCompile(`^(?:pub\s+)?(?:async\s+)?fn\s+(\w+)`), ChunkFunction},
		{regexp.MustCompile(`^(?:pub\s+)?struct\s+(\w+)`), ChunkStruct},
		{regexp.MustCompile(`^(?:pub\s+)?trait\s+(\w+)`), ChunkInterface},
		{regexp.MustCompile(`^(?:pub\s+)?enum\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^impl\s+(?:<[^>]+>\s+)?(\w+)`), ChunkClass},
	}

	// Java
	javaPatterns := []chunkPattern{
		{regexp.MustCompile(`^(?:public|private|protected|internal)?\s*(?:abstract|sealed)?\s*class\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^(?:public|private|protected|internal)?\s*interface\s+(\w+)`), ChunkInterface},
		{regexp.MustCompile(`^(?:public|private|protected)?\s*(?:static\s+)?[\w<>\[\]]+\s+(\w+)\s*\(`), ChunkMethod},
	}
	c.patterns["java"] = javaPatterns
	c.patterns["kotlin"] = javaPatterns
	c.patterns["csharp"] = javaPatterns

	// C/C++
	cPatterns := []chunkPattern{
		{regexp.MustCompile(`^(?:class|struct)\s+(\w+)`), ChunkStruct},
		{regexp.MustCompile(`^[\w*&:<>]+\s+(\w+)\s*\(`), ChunkFunction},
	}
	c.patterns["c"] = cPatterns
	c.patterns["cpp"] = cPatterns

	// Swift
	c.patterns["swift"] = []chunkPattern{
		{regexp.MustCompile(`^(?:public|private|internal|open)?\s*class\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^(?:public|private|internal|open)?\s*struct\s+(\w+)`), ChunkStruct},
		{regexp.MustCompile(`^(?:public|private|internal|open)?\s*protocol\s+(\w+)`), ChunkInterface},
		{regexp.MustCompile(`^(?:public|private|internal|open)?\s*func\s+(\w+)`), ChunkFunction},
	}

	// Dart
	c.patterns["dart"] = []chunkPattern{
		{regexp.MustCompile(`^(?:abstract\s+)?class\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^[\w<>]+\s+(\w+)\s*\(`), ChunkFunction},
	}

	// Ruby
	c.patterns["ruby"] = []chunkPattern{
		{regexp.MustCompile(`^class\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^module\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^def\s+(\w+)`), ChunkFunction},
	}

	// PHP
	c.patterns["php"] = []chunkPattern{
		{regexp.MustCompile(`^(?:abstract\s+)?class\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^interface\s+(\w+)`), ChunkInterface},
		{regexp.MustCompile(`^(?:public|private|protected)?\s*(?:static\s+)?function\s+(\w+)`), ChunkFunction},
	}

	// Scala
	c.patterns["scala"] = []chunkPattern{
		{regexp.MustCompile(`^(?:case\s+)?class\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^object\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^trait\s+(\w+)`), ChunkInterface},
		{regexp.MustCompile(`^def\s+(\w+)`), ChunkFunction},
	}

	// Elixir
	c.patterns["elixir"] = []chunkPattern{
		{regexp.MustCompile(`^defmodule\s+(\w+)`), ChunkClass},
		{regexp.MustCompile(`^def\s+(\w+)`), ChunkFunction},
		{regexp.MustCompile(`^defp\s+(\w+)`), ChunkFunction},
	}

	// Lua
	c.patterns["lua"] = []chunkPattern{
		{regexp.MustCompile(`^(?:local\s+)?function\s+(\w+)`), ChunkFunction},
	}
}
