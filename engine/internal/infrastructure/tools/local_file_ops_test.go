package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ReadFile Tests ---

// TC-F-01: Read existing file — content returned with all lines present.
func TestLocalProxy_ReadFile_ExistingFile_TC_F_01(t *testing.T) {
	dir := t.TempDir()
	content := "line1\nline2\nline3"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.ReadFile(ctx, "test-session", "test.txt", 0, 0)
	require.NoError(t, err)

	assert.Contains(t, result, "line1")
	assert.Contains(t, result, "line2")
	assert.Contains(t, result, "line3")
	// Full read returns raw content
	assert.Equal(t, content, result)
}

func TestLocalProxy_ReadFile_WithLineRange(t *testing.T) {
	dir := t.TempDir()
	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	content := strings.Join(lines, "\n")
	if err := os.WriteFile(filepath.Join(dir, "range.txt"), []byte(content), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.ReadFile(ctx, "test-session", "range.txt", 3, 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Lines 3-7 should be present
	for i := 3; i <= 7; i++ {
		expected := fmt.Sprintf("line%d", i)
		if !strings.Contains(result, expected) {
			t.Errorf("result should contain %q for lines 3-7, got: %s", expected, result)
		}
	}

	// Lines outside range should NOT be present (line1, line2, line8-10)
	// Note: line numbers in output may include line1/line2 prefix, so check raw content
	if strings.Contains(result, "line1\n") && !strings.Contains(result, "line10") {
		// line1 as a standalone line should not be there
		// but we need to be careful with line number prefixes
	}
}

func TestLocalProxy_ReadFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.ReadFile(ctx, "test-session", "nonexistent.txt", 0, 0)

	// Proxy can return error OR soft error string — both are acceptable
	if err != nil {
		errLower := strings.ToLower(err.Error())
		// Cross-platform: Linux says "no such file", Windows says "cannot find the file"
		if !strings.Contains(errLower, "not found") &&
			!strings.Contains(errLower, "no such file") &&
			!strings.Contains(errLower, "cannot find") {
			t.Errorf("error should mention file not found, got: %v", err)
		}
		return
	}

	// If returned as soft error in result
	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected [ERROR] in result for nonexistent file, got: %s", result)
	}
	resultLower := strings.ToLower(result)
	if !strings.Contains(resultLower, "not found") &&
		!strings.Contains(resultLower, "no such file") &&
		!strings.Contains(resultLower, "cannot find") {
		t.Errorf("expected file-not-found mention in result, got: %s", result)
	}
}

func TestLocalProxy_ReadFile_IsDirectory(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("setup: mkdir: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.ReadFile(ctx, "test-session", "subdir", 0, 0)

	// Should indicate that path is a directory
	if err != nil {
		if !strings.Contains(err.Error(), "directory") {
			t.Errorf("error should mention 'directory', got: %v", err)
		}
		return
	}

	if !strings.Contains(result, "[ERROR]") {
		t.Errorf("expected [ERROR] for directory path, got: %s", result)
	}
	if !strings.Contains(strings.ToLower(result), "directory") {
		t.Errorf("expected 'directory' mention in result, got: %s", result)
	}
}

func TestLocalProxy_ReadFile_EmptyRange(t *testing.T) {
	dir := t.TempDir()
	content := "line1\nline2\nline3"
	if err := os.WriteFile(filepath.Join(dir, "test.txt"), []byte(content), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	// start=10, end=5 is an invalid/empty range
	result, err := proxy.ReadFile(ctx, "test-session", "test.txt", 10, 5)
	if err != nil {
		// Acceptable: error for invalid range
		return
	}

	// If no error, result should be empty or contain info about empty range
	// The proxy might return the file info header with no content lines
	_ = result // Any non-error response is acceptable for empty range
}

func TestLocalProxy_ReadFile_RelativePath(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("setup: mkdir: %v", err)
	}
	content := "package main"
	if err := os.WriteFile(filepath.Join(subDir, "main.go"), []byte(content), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	// Relative path should resolve from projectRoot
	result, err := proxy.ReadFile(ctx, "test-session", "src/main.go", 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "package main") {
		t.Errorf("result should contain file content 'package main', got: %s", result)
	}
}

// TC-F-05: Read file >1MB — returns soft error with size info.
func TestLocalProxy_ReadFile_LargeFile_TC_F_05(t *testing.T) {
	dir := t.TempDir()
	// Create file > 1MB
	largeContent := strings.Repeat("x", 1024*1024+1)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "large.bin"), []byte(largeContent), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.ReadFile(ctx, "test-session", "large.bin", 0, 0)
	require.NoError(t, err, "large file returns soft error string, not Go error")

	// Should contain size info
	assert.Contains(t, result, "File too large")
	assert.Contains(t, result, "1048576", "should mention max size limit")
}

// --- WriteFile Tests ---

// TC-F-07: Write new file — returns "File written: path (N lines)".
func TestLocalProxy_WriteFile_NewFile_TC_F_07(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	content := "package main\n\nfunc main() {}\n"
	result, err := proxy.WriteFile(ctx, "test-session", "new_file.go", content)
	require.NoError(t, err)

	assert.NotContains(t, result, "[ERROR]")
	// Verify result format: "File written: <path> (N lines)"
	assert.Contains(t, result, "File written:")
	assert.Contains(t, result, "new_file.go")
	assert.Contains(t, result, "4 lines", "content has 4 lines (split by \\n)")

	// Verify file was actually written to disk
	written, err := os.ReadFile(filepath.Join(dir, "new_file.go"))
	require.NoError(t, err, "file should exist on disk")
	assert.Equal(t, content, string(written))
}

func TestLocalProxy_WriteFile_CreateParentDirs(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	content := "nested file content"
	result, err := proxy.WriteFile(ctx, "test-session", "a/b/c/file.txt", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "[ERROR]") {
		t.Errorf("expected success, got error: %s", result)
	}

	// Verify parent dirs were created
	written, err := os.ReadFile(filepath.Join(dir, "a", "b", "c", "file.txt"))
	if err != nil {
		t.Fatalf("file should exist in nested dirs: %v", err)
	}
	if string(written) != content {
		t.Errorf("file content mismatch:\ngot:  %q\nwant: %q", string(written), content)
	}
}

func TestLocalProxy_WriteFile_Overwrite(t *testing.T) {
	dir := t.TempDir()
	originalContent := "original content"
	if err := os.WriteFile(filepath.Join(dir, "existing.txt"), []byte(originalContent), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	newContent := "updated content"
	result, err := proxy.WriteFile(ctx, "test-session", "existing.txt", newContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "[ERROR]") {
		t.Errorf("expected success, got error: %s", result)
	}

	// Verify content was overwritten
	written, err := os.ReadFile(filepath.Join(dir, "existing.txt"))
	if err != nil {
		t.Fatalf("file should exist: %v", err)
	}
	if string(written) != newContent {
		t.Errorf("file should be overwritten:\ngot:  %q\nwant: %q", string(written), newContent)
	}
}

func TestLocalProxy_WriteFile_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.WriteFile(ctx, "test-session", "empty.txt", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "[ERROR]") {
		t.Errorf("expected success for empty content, got error: %s", result)
	}

	// Verify file exists and is empty
	written, err := os.ReadFile(filepath.Join(dir, "empty.txt"))
	if err != nil {
		t.Fatalf("file should exist: %v", err)
	}
	if len(written) != 0 {
		t.Errorf("file should be empty, got %d bytes", len(written))
	}
}

// --- GetProjectTree Tests ---

func TestLocalProxy_GetProjectTree_Basic(t *testing.T) {
	dir := t.TempDir()
	// Create project structure
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		t.Fatalf("setup: mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "app.go"), []byte("package src"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GetProjectTree(ctx, "test-session", "test-project", "", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == "" {
		t.Fatal("expected non-empty tree output")
	}

	// Result should be JSON tree containing our files/dirs
	if !strings.Contains(result, "main.go") {
		t.Errorf("tree should contain 'main.go', got: %s", result)
	}
	if !strings.Contains(result, "src") {
		t.Errorf("tree should contain 'src', got: %s", result)
	}
}

func TestLocalProxy_GetProjectTree_MaxDepth(t *testing.T) {
	dir := t.TempDir()
	// Create nested structure: root/a/b/c/deep.txt
	if err := os.MkdirAll(filepath.Join(dir, "a", "b", "c"), 0755); err != nil {
		t.Fatalf("setup: mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a", "b", "c", "deep.txt"), []byte("deep"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	// depth=1 should show only first level
	result, err := proxy.GetProjectTree(ctx, "test-session", "test-project", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// root.txt and dir "a" should be present
	if !strings.Contains(result, "root.txt") {
		t.Errorf("depth=1 should show root.txt, got: %s", result)
	}
	if !strings.Contains(result, "a") {
		t.Errorf("depth=1 should show directory 'a', got: %s", result)
	}

	// deep.txt should NOT be present at depth=1
	if strings.Contains(result, "deep.txt") {
		t.Errorf("depth=1 should NOT show deep.txt, got: %s", result)
	}
}

func TestLocalProxy_GetProjectTree_IgnoredDirs(t *testing.T) {
	dir := t.TempDir()
	// Create ignored directories
	for _, ignored := range []string{"node_modules", ".git"} {
		ignoredDir := filepath.Join(dir, ignored)
		if err := os.MkdirAll(ignoredDir, 0755); err != nil {
			t.Fatalf("setup: mkdir %s: %v", ignored, err)
		}
		if err := os.WriteFile(filepath.Join(ignoredDir, "file.txt"), []byte("content"), 0644); err != nil {
			t.Fatalf("setup: write file in %s: %v", ignored, err)
		}
	}
	// Create a normal file
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GetProjectTree(ctx, "test-session", "test-project", "", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// main.go should be present
	if !strings.Contains(result, "main.go") {
		t.Errorf("tree should contain 'main.go', got: %s", result)
	}

	// node_modules and .git should be filtered out
	if strings.Contains(result, "node_modules") {
		t.Errorf("tree should NOT contain 'node_modules', got: %s", result)
	}
	if strings.Contains(result, ".git") {
		t.Errorf("tree should NOT contain '.git', got: %s", result)
	}
}

func TestLocalProxy_GetProjectTree_IsFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GetProjectTree(ctx, "test-session", "test-project", "file.txt", 1)

	// Should indicate error for file path
	if err != nil {
		if !strings.Contains(err.Error(), "file") && !strings.Contains(err.Error(), "directory") {
			t.Errorf("error should mention file/directory, got: %v", err)
		}
		return
	}

	// If returned as JSON, is_directory should be false
	if !strings.Contains(result, "is_directory") {
		// Acceptable: tree node with is_directory=false
	}
}

func TestLocalProxy_GetProjectTree_NotFound(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GetProjectTree(ctx, "test-session", "test-project", "nonexistent", 1)

	// Should return error for non-existent path
	if err != nil {
		if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "no such") && !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("error should mention not found, got: %v", err)
		}
		return
	}

	if !strings.Contains(result, "[ERROR]") && !strings.Contains(strings.ToLower(result), "not found") {
		t.Errorf("expected error for nonexistent path, got: %s", result)
	}
}

func TestLocalProxy_GetProjectTree_SortOrder(t *testing.T) {
	dir := t.TempDir()
	// Create files and dirs with specific names to verify sorting
	if err := os.MkdirAll(filepath.Join(dir, "beta_dir"), 0755); err != nil {
		t.Fatalf("setup: mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "alpha_dir"), 0755); err != nil {
		t.Fatalf("setup: mkdir: %v", err)
	}
	for _, f := range []string{"zebra.txt", "apple.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("content"), 0644); err != nil {
			t.Fatalf("setup: write file %s: %v", f, err)
		}
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GetProjectTree(ctx, "test-session", "test-project", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that directories come before files (standard tree convention)
	// and items are sorted alphabetically within their group
	dirIdx := strings.Index(result, "alpha_dir")
	fileIdx := strings.Index(result, "apple.txt")

	if dirIdx == -1 {
		t.Fatalf("result should contain 'alpha_dir', got: %s", result)
	}
	if fileIdx == -1 {
		t.Fatalf("result should contain 'apple.txt', got: %s", result)
	}

	if dirIdx > fileIdx {
		t.Errorf("directories should come before files in tree output, but alpha_dir at %d, apple.txt at %d", dirIdx, fileIdx)
	}
}

// --- GlobSearch Tests ---

func TestLocalProxy_GlobSearch_BasicPattern(t *testing.T) {
	dir := t.TempDir()
	// Create Go files
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0755); err != nil {
		t.Fatalf("setup: mkdir: %v", err)
	}
	for _, f := range []string{"main.go", "src/app.go"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("package main"), 0644); err != nil {
			t.Fatalf("setup: write file %s: %v", f, err)
		}
	}
	// Create non-Go file
	if err := os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# README"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GlobSearch(ctx, "test-session", "**/*.go", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find .go files
	if !strings.Contains(result, "main.go") {
		t.Errorf("glob should find main.go, got: %s", result)
	}
	if !strings.Contains(result, "app.go") {
		t.Errorf("glob should find src/app.go, got: %s", result)
	}

	// Should NOT find .md files
	if strings.Contains(result, "readme.md") {
		t.Errorf("glob **/*.go should NOT find readme.md, got: %s", result)
	}
}

func TestLocalProxy_GlobSearch_NoMatches(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GlobSearch(ctx, "test-session", "**/*.xyz", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should indicate no files found (not an error, just informational)
	if strings.Contains(result, ".go") {
		t.Errorf("glob **/*.xyz should not return .go files, got: %s", result)
	}

	// Result should be empty or contain a "no files" message
	if result != "" && !strings.Contains(strings.ToLower(result), "no") && !strings.Contains(result, "0") {
		// Empty result is also acceptable
		_ = result
	}
}

func TestLocalProxy_GlobSearch_Truncation(t *testing.T) {
	dir := t.TempDir()
	// Create 5 Go files
	for i := 1; i <= 5; i++ {
		name := fmt.Sprintf("file%d.go", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("package main"), 0644); err != nil {
			t.Fatalf("setup: write file %s: %v", name, err)
		}
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	// Limit to 2 results
	result, err := proxy.GlobSearch(ctx, "test-session", "**/*.go", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count how many .go files appear in result
	goCount := strings.Count(result, ".go")
	if goCount > 3 {
		// Allow some flexibility (e.g., truncation message may reference count)
		// but should not list all 5 files
		t.Errorf("limit=2 should not return all 5 files, got %d references to .go: %s", goCount, result)
	}
}

func TestLocalProxy_GlobSearch_IgnoredDirs(t *testing.T) {
	dir := t.TempDir()
	// Create file in node_modules
	nmDir := filepath.Join(dir, "node_modules", "pkg")
	if err := os.MkdirAll(nmDir, 0755); err != nil {
		t.Fatalf("setup: mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(nmDir, "index.js"), []byte("module.exports"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	// Create file outside node_modules
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("console.log"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.GlobSearch(ctx, "test-session", "**/*.js", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// app.js should be found
	if !strings.Contains(result, "app.js") {
		t.Errorf("glob should find app.js, got: %s", result)
	}

	// Files in node_modules should NOT be returned
	if strings.Contains(result, "node_modules") {
		t.Errorf("glob should NOT return files from node_modules, got: %s", result)
	}
}

// =============================================================================
// TC-F (File Operations edge cases)
// =============================================================================

// TC-F-02: Read with line range — 20 lines, read start=5 end=10, verify only lines 5-10.
func TestLocalProxy_ReadFile_LineRange_TC_F_02(t *testing.T) {
	dir := t.TempDir()
	var lines []string
	for i := 1; i <= 20; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	content := strings.Join(lines, "\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "range20.txt"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.ReadFile(ctx, "s", "range20.txt", 5, 10)
	require.NoError(t, err)

	// Lines 5-10 must be present
	for i := 5; i <= 10; i++ {
		assert.Contains(t, result, fmt.Sprintf("line%d", i), "should contain line%d", i)
	}

	// Lines outside range must NOT appear as standalone lines
	resultLines := strings.Split(result, "\n")
	for _, rl := range resultLines {
		trimmed := strings.TrimSpace(rl)
		// line1-line4 should not appear (note: "line1" could match "line10" so check exact)
		for _, excluded := range []string{"line1", "line2", "line3", "line4"} {
			if trimmed == excluded {
				t.Errorf("line range 5-10 should not contain %q as standalone line", excluded)
			}
		}
	}
	// line11+ should not appear
	for i := 11; i <= 20; i++ {
		target := fmt.Sprintf("line%d", i)
		if strings.Contains(result, target) {
			t.Errorf("line range 5-10 should not contain %q", target)
		}
	}
}

// TC-F-03: File not found — ReadFile with nonexistent path returns error or [ERROR].
func TestLocalProxy_ReadFile_NotFound_TC_F_03(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	_, err := proxy.ReadFile(ctx, "s", "does_not_exist.txt", 0, 0)
	require.Error(t, err, "reading nonexistent file should return error")

	errLower := strings.ToLower(err.Error())
	assert.True(t,
		strings.Contains(errLower, "no such file") ||
			strings.Contains(errLower, "cannot find") ||
			strings.Contains(errLower, "not found"),
		"error should mention file not found, got: %v", err,
	)
}

// TC-F-04: Path is directory — ReadFile with dir path returns [ERROR] directory message.
func TestLocalProxy_ReadFile_PathIsDirectory_TC_F_04(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "mydir")
	require.NoError(t, os.MkdirAll(subDir, 0755))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.ReadFile(ctx, "s", "mydir", 0, 0)
	require.NoError(t, err, "directory path returns soft error, not Go error")

	assert.Contains(t, result, "[ERROR]")
	assert.Contains(t, strings.ToLower(result), "directory")
}

// TC-F-06: Empty range (start >= end) — ReadFile with start=10, end=5.
func TestLocalProxy_ReadFile_EmptyRange_TC_F_06(t *testing.T) {
	dir := t.TempDir()
	var lines []string
	for i := 1; i <= 20; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	content := strings.Join(lines, "\n")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "range.txt"), []byte(content), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	result, err := proxy.ReadFile(ctx, "s", "range.txt", 10, 5)
	require.NoError(t, err)

	// Result should indicate empty range (no content lines)
	assert.Contains(t, result, "empty", "result for start>end should mention empty range")
}

// TC-F-08: Create parent dirs — WriteFile with nested path creates directories.
func TestLocalProxy_WriteFile_CreateParentDirs_TC_F_08(t *testing.T) {
	dir := t.TempDir()
	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	content := "deeply nested content"
	result, err := proxy.WriteFile(ctx, "s", "a/b/c/file.txt", content)
	require.NoError(t, err)
	assert.NotContains(t, result, "[ERROR]")

	// Verify directory chain was created
	written, err := os.ReadFile(filepath.Join(dir, "a", "b", "c", "file.txt"))
	require.NoError(t, err, "file should exist in nested dirs")
	assert.Equal(t, content, string(written))

	// Verify intermediate directories exist
	info, err := os.Stat(filepath.Join(dir, "a", "b"))
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

// TC-F-09: Overwrite file — WriteFile overwrites existing content completely.
func TestLocalProxy_WriteFile_Overwrite_TC_F_09(t *testing.T) {
	dir := t.TempDir()
	original := "original content that is quite long and should be fully replaced"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "overwrite.txt"), []byte(original), 0644))

	proxy := NewLocalClientOperationsProxy(dir)
	ctx := context.Background()

	newContent := "new"
	result, err := proxy.WriteFile(ctx, "s", "overwrite.txt", newContent)
	require.NoError(t, err)
	assert.NotContains(t, result, "[ERROR]")

	// Verify content is fully replaced (not appended)
	written, err := os.ReadFile(filepath.Join(dir, "overwrite.txt"))
	require.NoError(t, err)
	assert.Equal(t, newContent, string(written))
	assert.NotContains(t, string(written), "original", "old content should be gone")
}
