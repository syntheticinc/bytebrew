// Replace logic adapted from OpenCode (MIT License)
// https://github.com/anthropics/opencode

package tools

import (
	"fmt"
	"math"
	"regexp"
	"strings"
)

// Replace applies a replacement with 5-tier fuzzy matching.
// Returns the new content or an error with a helpful message for the LLM.
func Replace(content, oldString, newString string, replaceAll bool) (string, error) {
	if oldString == newString {
		return "", fmt.Errorf("oldString and newString must be different")
	}

	if oldString == "" {
		return "", fmt.Errorf("oldString must not be empty")
	}

	// Normalize CRLF to LF for consistent matching (Windows files have \r\n, LLM sends \n)
	hasCRLF := strings.Contains(content, "\r\n")
	if hasCRLF {
		content = strings.ReplaceAll(content, "\r\n", "\n")
		oldString = strings.ReplaceAll(oldString, "\r\n", "\n")
		newString = strings.ReplaceAll(newString, "\r\n", "\n")
	}

	notFound := true

	replacers := []func(string, string) []string{
		simpleReplacer,
		lineTrimmedReplacer,
		whitespaceNormalizedReplacer,
		indentationFlexibleReplacer,
		multiOccurrenceReplacer,
	}

	for _, replacer := range replacers {
		candidates := replacer(content, oldString)
		for _, candidate := range candidates {
			idx := strings.Index(content, candidate)
			if idx == -1 {
				continue
			}
			notFound = false

			if replaceAll {
				result := strings.ReplaceAll(content, candidate, newString)
				return restoreCRLF(result, hasCRLF), nil
			}

			// Single replacement: require unique match
			lastIdx := strings.LastIndex(content, candidate)
			if idx != lastIdx {
				continue // multiple matches, try next replacer
			}

			result := content[:idx] + newString + content[idx+len(candidate):]
			return restoreCRLF(result, hasCRLF), nil
		}
	}

	if notFound {
		hint := findClosestMatch(content, oldString)
		if hint != "" {
			return "", fmt.Errorf("oldString not found in file content. %s", hint)
		}
		return "", fmt.Errorf("oldString not found in file content")
	}

	return "", fmt.Errorf("Found multiple matches for oldString. Provide more surrounding lines for context to uniquely identify the match.")
}

// restoreCRLF converts LF back to CRLF if the original content had CRLF line endings.
func restoreCRLF(s string, hasCRLF bool) string {
	if !hasCRLF {
		return s
	}
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// simpleReplacer yields the exact find string if it exists in content.
func simpleReplacer(content, find string) []string {
	if strings.Contains(content, find) {
		return []string{find}
	}
	return nil
}

// lineTrimmedReplacer matches lines ignoring leading/trailing whitespace on each line.
// Yields the original substring from content (preserving original indentation).
func lineTrimmedReplacer(content, find string) []string {
	originalLines := strings.Split(content, "\n")
	searchLines := strings.Split(find, "\n")

	// Check if find had a trailing empty line (trailing \n)
	hadTrailingNewline := len(searchLines) > 0 && searchLines[len(searchLines)-1] == ""
	if hadTrailingNewline {
		searchLines = searchLines[:len(searchLines)-1]
	}

	if len(searchLines) == 0 {
		return nil
	}

	var results []string
	for i := 0; i <= len(originalLines)-len(searchLines); i++ {
		if !linesMatchTrimmed(originalLines, searchLines, i) {
			continue
		}

		// Calculate the original substring
		matchStart := calcLineOffset(originalLines, i)
		matchEnd := calcBlockEnd(originalLines, i, len(searchLines))
		// If the original find had a trailing newline, include it in the match
		// so that replacement preserves the line boundary
		if hadTrailingNewline && matchEnd < len(content) && content[matchEnd] == '\n' {
			matchEnd++
		}
		results = append(results, content[matchStart:matchEnd])
	}
	return results
}

// linesMatchTrimmed checks if lines at position i in original match search lines when trimmed.
func linesMatchTrimmed(original, search []string, start int) bool {
	for j := 0; j < len(search); j++ {
		if strings.TrimSpace(original[start+j]) != strings.TrimSpace(search[j]) {
			return false
		}
	}
	return true
}

// calcLineOffset returns the byte offset of the start of line at index i.
func calcLineOffset(lines []string, i int) int {
	offset := 0
	for k := 0; k < i; k++ {
		offset += len(lines[k]) + 1 // +1 for \n
	}
	return offset
}

// calcBlockEnd returns the byte offset of the end of a block starting at line i with count lines.
func calcBlockEnd(lines []string, start, count int) int {
	end := calcLineOffset(lines, start)
	for k := 0; k < count; k++ {
		end += len(lines[start+k])
		if k < count-1 {
			end++ // \n between lines, but not after last
		}
	}
	return end
}

// whitespaceNormalizedReplacer normalizes all whitespace sequences to single space before matching.
func whitespaceNormalizedReplacer(content, find string) []string {
	normalizedFind := normalizeWhitespace(find)
	lines := strings.Split(content, "\n")

	var results []string

	// Single-line matches
	for _, line := range lines {
		if normalizeWhitespace(line) == normalizedFind {
			results = append(results, line)
			continue
		}

		normalizedLine := normalizeWhitespace(line)
		if !strings.Contains(normalizedLine, normalizedFind) {
			continue
		}

		// Try regex match for partial line match
		match := matchPartialLine(line, find)
		if match != "" {
			results = append(results, match)
		}
	}

	// Multi-line matches
	findLines := strings.Split(find, "\n")
	if len(findLines) <= 1 {
		return results
	}

	for i := 0; i <= len(lines)-len(findLines); i++ {
		block := strings.Join(lines[i:i+len(findLines)], "\n")
		if normalizeWhitespace(block) == normalizedFind {
			results = append(results, block)
		}
	}

	return results
}

var wsRegex = regexp.MustCompile(`\s+`)

// normalizeWhitespace replaces all whitespace sequences with a single space and trims.
func normalizeWhitespace(text string) string {
	return strings.TrimSpace(wsRegex.ReplaceAllString(text, " "))
}

// matchPartialLine tries to find a partial match within a line using a regex built from find words.
func matchPartialLine(line, find string) string {
	words := strings.Fields(strings.TrimSpace(find))
	if len(words) == 0 {
		return ""
	}

	// Build regex: each word escaped, joined with \s+
	escaped := make([]string, len(words))
	for i, w := range words {
		escaped[i] = regexp.QuoteMeta(w)
	}
	pattern := strings.Join(escaped, `\s+`)

	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}

	match := re.FindString(line)
	return match
}

// indentationFlexibleReplacer strips common indentation before comparing.
func indentationFlexibleReplacer(content, find string) []string {
	normalizedFind := removeIndentation(find)
	contentLines := strings.Split(content, "\n")
	findLines := strings.Split(find, "\n")

	if len(findLines) == 0 {
		return nil
	}

	var results []string
	for i := 0; i <= len(contentLines)-len(findLines); i++ {
		block := strings.Join(contentLines[i:i+len(findLines)], "\n")
		if removeIndentation(block) == normalizedFind {
			results = append(results, block)
		}
	}
	return results
}

// removeIndentation finds the minimum indentation of non-empty lines and removes it.
func removeIndentation(text string) string {
	lines := strings.Split(text, "\n")

	minIndent := math.MaxInt
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent < minIndent {
			minIndent = indent
		}
	}

	if minIndent == math.MaxInt || minIndent == 0 {
		return text
	}

	var result []string
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			result = append(result, line)
			continue
		}
		if len(line) >= minIndent {
			result = append(result, line[minIndent:])
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

// multiOccurrenceReplacer yields all exact occurrences for replaceAll support.
func multiOccurrenceReplacer(content, find string) []string {
	var results []string
	startIdx := 0
	for {
		idx := strings.Index(content[startIdx:], find)
		if idx == -1 {
			break
		}
		results = append(results, find)
		startIdx += idx + len(find)
	}
	return results
}

// findClosestMatch finds the closest matching block in content for better error messages.
func findClosestMatch(content, oldString string) string {
	contentLines := strings.Split(content, "\n")
	findLines := strings.Split(oldString, "\n")

	// Remove trailing empty line from find
	if len(findLines) > 1 && strings.TrimSpace(findLines[len(findLines)-1]) == "" {
		findLines = findLines[:len(findLines)-1]
	}

	if len(findLines) == 0 || len(contentLines) == 0 {
		return ""
	}

	windowSize := len(findLines)
	if windowSize > len(contentLines) {
		return ""
	}

	// For single-line: no hint (handled by replacers already)
	if windowSize == 1 {
		return ""
	}

	bestStartLine := -1
	bestMatchCount := 0

	for i := 0; i <= len(contentLines)-windowSize; i++ {
		matchCount := 0
		for j := 0; j < windowSize; j++ {
			if strings.TrimSpace(contentLines[i+j]) == strings.TrimSpace(findLines[j]) {
				matchCount++
			}
		}
		if matchCount > bestMatchCount {
			bestMatchCount = matchCount
			bestStartLine = i
		}
	}

	if bestStartLine == -1 || bestMatchCount == 0 {
		return ""
	}

	ratio := float64(bestMatchCount) / float64(windowSize)

	// Need at least 30% matching lines
	if ratio < 0.3 {
		return ""
	}

	// Build differing lines list (max 3)
	var diffs []string
	for j := 0; j < windowSize && len(diffs) < 3; j++ {
		actual := strings.TrimSpace(contentLines[bestStartLine+j])
		expected := strings.TrimSpace(findLines[j])
		if actual == expected {
			continue
		}
		lineNum := bestStartLine + j + 1
		diffs = append(diffs, fmt.Sprintf("  line %d: file has \"%s\" but old_string has \"%s\"",
			lineNum, truncateStr(actual, 80), truncateStr(expected, 80)))
	}

	hint := fmt.Sprintf("Closest match at line %d (%d%% lines match).",
		bestStartLine+1, int(math.Round(ratio*100)))

	if len(diffs) > 0 {
		hint += "\nDiffering lines:\n" + strings.Join(diffs, "\n")
	}

	return hint
}

// truncateStr truncates a string to maxLen characters, appending "..." if truncated.
func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
