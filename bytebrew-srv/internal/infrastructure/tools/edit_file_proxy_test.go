package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalProxy_EditFile_ExactMatch(t *testing.T) {
	dir := t.TempDir()
	content := "func main() {\n\tfmt.Println(\"hello\")\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.EditFile(ctx, "test-session", "main.go", "fmt.Println(\"hello\")", "fmt.Println(\"world\")", false)
	require.NoError(t, err)
	assert.Contains(t, result, "Edit applied")
	assert.NotContains(t, result, "[ERROR]")

	// Verify file content on disk
	written, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(written), "fmt.Println(\"world\")")
	assert.NotContains(t, string(written), "fmt.Println(\"hello\")")
}

func TestLocalProxy_EditFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.EditFile(ctx, "test-session", "nonexistent.go", "old", "new", false)
	require.NoError(t, err) // soft error returned in result, not Go error
	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, result, "not found")
}

func TestLocalProxy_EditFile_NoMatch(t *testing.T) {
	dir := t.TempDir()
	content := "func main() {\n\tfmt.Println(\"hello\")\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.EditFile(ctx, "test-session", "main.go", "this does not exist in file", "replacement", false)
	require.NoError(t, err)
	// Replace returns error as string (soft error for LLM)
	assert.Contains(t, strings.ToLower(result), "not found")
}

func TestLocalProxy_EditFile_MultipleMatches(t *testing.T) {
	dir := t.TempDir()
	content := "a := 1\nb := 2\na := 1\nc := 3\na := 1\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.EditFile(ctx, "test-session", "main.go", "a := 1", "x := 9", false)
	require.NoError(t, err)
	// Should report multiple matches error
	assert.Contains(t, result, "multiple matches")

	// File should remain unchanged
	written, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	assert.Equal(t, content, string(written))
}

func TestLocalProxy_EditFile_ReplaceAll(t *testing.T) {
	dir := t.TempDir()
	content := "a := 1\nb := 2\na := 1\nc := 3\na := 1\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.EditFile(ctx, "test-session", "main.go", "a := 1", "x := 9", true)
	require.NoError(t, err)
	assert.Contains(t, result, "Edit applied")

	written, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	assert.Equal(t, 3, strings.Count(string(written), "x := 9"))
	assert.Equal(t, 0, strings.Count(string(written), "a := 1"))
}

func TestLocalProxy_EditFile_FuzzyMatch_LineTrimmed(t *testing.T) {
	dir := t.TempDir()
	content := "func main() {\n\tfmt.Println(\"hello\")\n\tfmt.Println(\"world\")\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	// old_string with spaces instead of tabs — lineTrimmedReplacer should match
	oldStr := "  fmt.Println(\"hello\")\n  fmt.Println(\"world\")"
	result, err := proxy.EditFile(ctx, "test-session", "main.go", oldStr, "fmt.Println(\"replaced\")", false)
	require.NoError(t, err)
	assert.Contains(t, result, "Edit applied")

	written, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(written), "fmt.Println(\"replaced\")")
	assert.NotContains(t, string(written), "fmt.Println(\"hello\")")
}

func TestLocalProxy_EditFile_FuzzyMatch_Indentation(t *testing.T) {
	dir := t.TempDir()
	content := "func main() {\n\t\tline1\n\t\tline2\n\t\tline3\n}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	// old_string without indentation — indentationFlexibleReplacer should match
	result, err := proxy.EditFile(ctx, "test-session", "main.go", "line1\nline2\nline3", "replaced", false)
	require.NoError(t, err)
	assert.Contains(t, result, "Edit applied")

	written, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(written), "replaced")
	assert.NotContains(t, string(written), "line1")
}

func TestLocalProxy_EditFile_EmptyOldString(t *testing.T) {
	dir := t.TempDir()
	content := "some content\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.EditFile(ctx, "test-session", "main.go", "", "new", false)
	require.NoError(t, err)
	// Replace returns error for empty oldString
	assert.Contains(t, strings.ToLower(result), "empty")
}

func TestLocalProxy_EditFile_ReturnsDiff(t *testing.T) {
	dir := t.TempDir()
	content := "line1\nline2\nline3\nline4\nline5\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "file.txt"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.EditFile(ctx, "test-session", "file.txt", "line2\nline3", "replaced_line", false)
	require.NoError(t, err)
	// Result should contain diff-like info about the edit
	assert.Contains(t, result, "Edit applied")
	// Line count change: was 5 lines (\n count), now fewer
	assert.Contains(t, result, "lines")
}
