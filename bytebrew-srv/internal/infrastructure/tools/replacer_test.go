package tools

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplace_SimpleExactMatch(t *testing.T) {
	content := "func main() {\n\tfmt.Println(\"hello\")\n}"
	oldStr := "fmt.Println(\"hello\")"
	newStr := "fmt.Println(\"world\")"

	result, err := Replace(content, oldStr, newStr, false)
	require.NoError(t, err)
	assert.Equal(t, "func main() {\n\tfmt.Println(\"world\")\n}", result)
}

func TestReplace_LineTrimmed(t *testing.T) {
	// Content has tabs, oldString has spaces — trimmed lines should match
	content := "func main() {\n\t\tfmt.Println(\"hello\")\n\t\tfmt.Println(\"world\")\n}"
	oldStr := "  fmt.Println(\"hello\")\n  fmt.Println(\"world\")"
	newStr := "\t\tfmt.Println(\"replaced\")"

	result, err := Replace(content, oldStr, newStr, false)
	require.NoError(t, err)
	assert.Contains(t, result, "fmt.Println(\"replaced\")")
	assert.NotContains(t, result, "fmt.Println(\"hello\")")
	assert.NotContains(t, result, "fmt.Println(\"world\")")
}

func TestReplace_LineTrimmed_TrailingEmptyLine(t *testing.T) {
	content := "line1\nline2\nline3"
	oldStr := "line1\nline2\n" // trailing newline — exact match includes the \n

	result, err := Replace(content, oldStr, "replaced\n", false)
	require.NoError(t, err)
	assert.Equal(t, "replaced\nline3", result)
}

func TestReplace_WhitespaceNormalized(t *testing.T) {
	// Content has tabs, oldString has spaces
	content := "func test() {\n\tif   a  ==   b {\n\t\treturn\n\t}\n}"
	oldStr := "if a == b {"
	newStr := "if a != b {"

	result, err := Replace(content, oldStr, newStr, false)
	require.NoError(t, err)
	assert.Contains(t, result, "if a != b {")
}

func TestReplace_WhitespaceNormalized_PartialLineMatch(t *testing.T) {
	content := "    result := doSomething(  arg1,   arg2  )"
	oldStr := "doSomething( arg1, arg2 )"
	newStr := "doOther(arg1, arg2)"

	result, err := Replace(content, oldStr, newStr, false)
	require.NoError(t, err)
	assert.Contains(t, result, "doOther(arg1, arg2)")
}

func TestReplace_IndentationFlexible(t *testing.T) {
	// Content indented with 2 tabs, oldString with no indent
	content := "func main() {\n\t\tline1\n\t\tline2\n\t\tline3\n}"
	oldStr := "line1\nline2\nline3"
	newStr := "replaced1\nreplaced2"

	result, err := Replace(content, oldStr, newStr, false)
	require.NoError(t, err)
	assert.Contains(t, result, "replaced1")
	assert.Contains(t, result, "replaced2")
	// The indented block "\t\tline1\n\t\tline2\n\t\tline3" should be fully replaced
	assert.NotContains(t, result, "\t\tline1")
}

func TestReplace_IndentationFlexible_EmptyLines(t *testing.T) {
	content := "    if true {\n\n        doSomething()\n    }"
	oldStr := "if true {\n\n    doSomething()\n}"

	result, err := Replace(content, oldStr, "// removed", false)
	require.NoError(t, err)
	assert.Contains(t, result, "// removed")
}

func TestReplace_MultipleMatches_Error(t *testing.T) {
	content := "a := 1\nb := 2\na := 1\nc := 3\na := 1"
	oldStr := "a := 1"

	_, err := Replace(content, oldStr, "x := 9", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Found multiple matches")
}

func TestReplace_ReplaceAll(t *testing.T) {
	content := "fmt.Println(\"a\")\nfmt.Println(\"b\")\nfmt.Println(\"a\")"
	oldStr := "fmt.Println(\"a\")"
	newStr := "fmt.Println(\"x\")"

	result, err := Replace(content, oldStr, newStr, true)
	require.NoError(t, err)
	assert.Equal(t, "fmt.Println(\"x\")\nfmt.Println(\"b\")\nfmt.Println(\"x\")", result)
}

func TestReplace_NotFound_NoHint(t *testing.T) {
	content := "line1\nline2\nline3"
	oldStr := "nonexistent line"

	_, err := Replace(content, oldStr, "replacement", false)
	require.Error(t, err)
	assert.Equal(t, "oldString not found in file content", err.Error())
}

func TestReplace_NotFound_WithHint(t *testing.T) {
	content := "func main() {\n\tfmt.Println(\"hello\")\n\tfmt.Println(\"world\")\n\treturn\n}"
	// Partial match: 3 out of 4 lines match
	oldStr := "func main() {\n\tfmt.Println(\"hello\")\n\tfmt.Println(\"DIFFERENT\")\n\treturn\n}"

	_, err := Replace(content, oldStr, "replacement", false)
	require.Error(t, err)
	errMsg := err.Error()
	assert.Contains(t, errMsg, "oldString not found in file content")
	assert.Contains(t, errMsg, "Closest match at line")
	assert.Contains(t, errMsg, "lines match")
	assert.Contains(t, errMsg, "Differing lines")
}

func TestReplace_CRLF_Normalization(t *testing.T) {
	// File with CRLF line endings
	content := "line1\r\nline2\r\nline3\r\n"
	// LLM sends LF
	oldStr := "line2"
	newStr := "replaced"

	result, err := Replace(content, oldStr, newStr, false)
	require.NoError(t, err)

	// Result should preserve CRLF
	assert.Contains(t, result, "\r\n")
	assert.Contains(t, result, "replaced")
	assert.Equal(t, "line1\r\nreplaced\r\nline3\r\n", result)
}

func TestReplace_CRLF_MultiLine(t *testing.T) {
	// File with CRLF, multi-line oldString with LF
	content := "func main() {\r\n\tfmt.Println(\"a\")\r\n\tfmt.Println(\"b\")\r\n}\r\n"
	oldStr := "fmt.Println(\"a\")\n\tfmt.Println(\"b\")"
	newStr := "fmt.Println(\"replaced\")"

	result, err := Replace(content, oldStr, newStr, false)
	require.NoError(t, err)
	assert.True(t, strings.Contains(result, "\r\n"), "should preserve CRLF")
	assert.Contains(t, result, "replaced")
}

func TestReplace_SameStrings_Error(t *testing.T) {
	_, err := Replace("content", "same", "same", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "oldString and newString must be different")
}

func TestReplace_EmptyOldString(t *testing.T) {
	_, err := Replace("hello", "", "x", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "oldString must not be empty")
}

// --- Unit tests for internal helpers ---

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"tabs to space", "a\t\tb", "a b"},
		{"multiple spaces", "a    b   c", "a b c"},
		{"trim", "  hello  ", "hello"},
		{"mixed", " \t a \t b \t ", "a b"},
		{"newlines", "a\n\nb", "a b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeWhitespace(tt.input))
		})
	}
}

func TestRemoveIndentation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			"uniform indent",
			"    line1\n    line2\n    line3",
			"line1\nline2\nline3",
		},
		{
			"mixed indent",
			"    line1\n        line2\n    line3",
			"line1\n    line2\nline3",
		},
		{
			"with empty lines",
			"    line1\n\n    line2",
			"line1\n\nline2",
		},
		{
			"no indent",
			"line1\nline2",
			"line1\nline2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, removeIndentation(tt.input))
		})
	}
}

func TestFindClosestMatch_NoMatch(t *testing.T) {
	result := findClosestMatch("aaa\nbbb\nccc", "xxx\nyyy\nzzz")
	assert.Empty(t, result)
}

func TestFindClosestMatch_SingleLine_NoHint(t *testing.T) {
	result := findClosestMatch("line1\nline2", "something")
	assert.Empty(t, result)
}

func TestFindClosestMatch_PartialMatch(t *testing.T) {
	content := "func main() {\n\tfmt.Println(\"hello\")\n\tfmt.Println(\"world\")\n\treturn\n}"
	find := "func main() {\n\tfmt.Println(\"hello\")\n\tfmt.Println(\"DIFFERENT\")\n\treturn\n}"

	result := findClosestMatch(content, find)
	assert.Contains(t, result, "Closest match at line 1")
	assert.Contains(t, result, "lines match")
	assert.Contains(t, result, "Differing lines")
	assert.Contains(t, result, "DIFFERENT")
}

func TestTruncateStr(t *testing.T) {
	assert.Equal(t, "hello", truncateStr("hello", 80))
	assert.Equal(t, "hel...", truncateStr("hello world", 3))
}
