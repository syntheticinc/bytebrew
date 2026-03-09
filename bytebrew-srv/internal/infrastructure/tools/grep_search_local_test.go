package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func requireRg(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("rg"); err != nil {
		t.Skip("rg (ripgrep) not found in PATH, skipping test")
	}
}

// setupGrepProject creates a temp directory with sample files for grep testing.
func setupGrepProject(t *testing.T) (string, *LocalClientOperationsProxy) {
	t.Helper()
	dir := t.TempDir()

	// Create Go files
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cmd"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmd", "main.go"), []byte(`package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`), 0644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "utils.go"), []byte(`package utils

func Add(a, b int) int {
	return a + b
}

func Subtract(a, b int) int {
	return a - b
}
`), 0644))

	// Create a TypeScript file
	require.NoError(t, os.WriteFile(filepath.Join(dir, "app.ts"), []byte(`export function greet(name: string): string {
	return "Hello " + name;
}
`), 0644))

	// Create a file in node_modules (should be excluded)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "index.js"), []byte(`function main() { console.log("from node_modules"); }
`), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	return dir, proxy
}

// TC-S-01: Grep pattern match — results with file:line format.
func TestLocalProxy_GrepSearch_BasicPattern_TC_S_01(t *testing.T) {
	requireRg(t)
	_, proxy := setupGrepProject(t)
	ctx := context.Background()

	result, err := proxy.GrepSearch(ctx, "", "func main", 100, nil, false)
	require.NoError(t, err)

	assert.Contains(t, result, "main.go")
	assert.Contains(t, result, "func main")
	assert.NotContains(t, result, "No matches found")

	// Verify file:line format (e.g., "cmd/main.go:5")
	assert.Regexp(t, `main\.go:\d+`, result, "result should contain file:line format")
	// Verify indented content line follows
	assert.Contains(t, result, "  func main", "content should be indented with 2 spaces")
}

// TC-S-02: Grep file_types filter — create .go and .ts files, grep with file_types=["go"], verify only .go results
func TestLocalProxy_GrepSearch_FileTypes(t *testing.T) {
	requireRg(t)
	_, proxy := setupGrepProject(t)
	ctx := context.Background()

	// Search only in .go files — should find Go funcs but not TS
	result, err := proxy.GrepSearch(ctx, "", "func", 100, []string{"go"}, false)
	require.NoError(t, err)

	assert.Contains(t, result, ".go")
	assert.NotContains(t, result, "app.ts")
}

func TestLocalProxy_GrepSearch_FileTypesGlob(t *testing.T) {
	requireRg(t)
	_, proxy := setupGrepProject(t)
	ctx := context.Background()

	// Use glob-style file type
	result, err := proxy.GrepSearch(ctx, "", "function", 100, []string{"*.ts"}, false)
	require.NoError(t, err)

	assert.Contains(t, result, "app.ts")
	assert.NotContains(t, result, ".go")
}

// TC-S-03: Grep ignore_case — create file with "Error", grep "error" with ignore_case=true
func TestLocalProxy_GrepSearch_IgnoreCase(t *testing.T) {
	requireRg(t)
	_, proxy := setupGrepProject(t)
	ctx := context.Background()

	// "hello" should match "Hello" with ignore_case
	result, err := proxy.GrepSearch(ctx, "", "hello", 100, nil, true)
	require.NoError(t, err)

	assert.Contains(t, result, "Hello")
	assert.NotContains(t, result, "No matches found")
}

func TestLocalProxy_GrepSearch_CaseSensitive(t *testing.T) {
	requireRg(t)
	_, proxy := setupGrepProject(t)
	ctx := context.Background()

	// "hello" should NOT match "Hello" without ignore_case
	result, err := proxy.GrepSearch(ctx, "", "hello", 100, nil, false)
	require.NoError(t, err)

	assert.Contains(t, result, "No matches found")
}

// TC-S-04: Grep no results — empty result with "No matches found" message.
func TestLocalProxy_GrepSearch_NoResults_TC_S_04(t *testing.T) {
	requireRg(t)
	_, proxy := setupGrepProject(t)
	ctx := context.Background()

	result, err := proxy.GrepSearch(ctx, "", "nonexistent_pattern_xyz_12345", 100, nil, false)
	require.NoError(t, err)

	assert.Contains(t, result, "No matches found")
	assert.Contains(t, result, "nonexistent_pattern_xyz_12345")
	// Should not contain any file:line entries
	assert.NotRegexp(t, `\w+\.\w+:\d+`, result, "no results should not contain file:line entries")
}

// TC-S-05: Grep truncation — create file with many matches, limit=3, verify truncation message
func TestLocalProxy_GrepSearch_Truncation(t *testing.T) {
	requireRg(t)
	dir := t.TempDir()

	// Create a file with many matching lines
	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "match_target line content here")
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "many.txt"), []byte(strings.Join(lines, "\n")), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GrepSearch(ctx, "", "match_target", 2, nil, false)
	require.NoError(t, err)

	assert.Contains(t, result, "(truncated)")
	// Should have exactly 2 file:line entries
	count := strings.Count(result, "many.txt:")
	assert.Equal(t, 2, count, "should have exactly 2 matches")
}

func TestLocalProxy_GrepSearch_OutputFormat(t *testing.T) {
	requireRg(t)
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "format.go"), []byte("first\nsecond target line\nthird\n"), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GrepSearch(ctx, "", "target", 100, nil, false)
	require.NoError(t, err)

	// Verify format: file:line\n  content
	assert.Contains(t, result, "format.go:2")
	assert.Contains(t, result, "  second target line")
	assert.Contains(t, result, "1 results")
}

func TestLocalProxy_GrepSearch_ExcludesNodeModules(t *testing.T) {
	requireRg(t)
	_, proxy := setupGrepProject(t)
	ctx := context.Background()

	// "console" exists in node_modules/pkg/index.js but should be excluded
	result, err := proxy.GrepSearch(ctx, "", "from node_modules", 100, nil, false)
	require.NoError(t, err)

	assert.Contains(t, result, "No matches found")
}

func TestLocalProxy_GrepSearch_EmptyPattern(t *testing.T) {
	requireRg(t)
	_, proxy := setupGrepProject(t)
	ctx := context.Background()

	result, err := proxy.GrepSearch(ctx, "", "", 100, nil, false)
	require.NoError(t, err)

	assert.Contains(t, result, "No matches found")
}

func TestLocalProxy_GrepSearch_RegexPattern(t *testing.T) {
	requireRg(t)
	_, proxy := setupGrepProject(t)
	ctx := context.Background()

	// Use regex to find function definitions
	result, err := proxy.GrepSearch(ctx, "", "func \\w+\\(", 100, []string{"go"}, false)
	require.NoError(t, err)

	assert.Contains(t, result, "func main(")
	assert.Contains(t, result, "func Add(")
}
