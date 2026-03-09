package tools

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

const maxFileSize = 1048576 // 1 MB

// alwaysIgnore contains entries that are always excluded from tree and glob results.
var alwaysIgnore = map[string]bool{
	".git":              true,
	".svn":              true,
	".hg":               true,
	"node_modules":      true,
	"vendor":            true,
	"__pycache__":       true,
	".idea":             true,
	".vscode":           true,
	".DS_Store":         true,
	"Thumbs.db":         true,
	"package-lock.json": true,
	"yarn.lock":         true,
	"pnpm-lock.yaml":   true,
	"bun.lockb":         true,
}

// defaultIgnore contains entries excluded by default (less critical than alwaysIgnore).
var defaultIgnore = map[string]bool{
	"dist":      true,
	"build":     true,
	"out":       true,
	"target":    true,
	"bin":       true,
	"obj":       true,
	".bytebrew": true,
	".next":     true,
	".nuxt":     true,
	"coverage":  true,
	".cache":    true,
	".venv":     true,
	"venv":      true,
	"env":       true,
}

// resolvePath resolves a file path relative to projectRoot.
// Absolute paths are returned as-is; relative paths are joined with projectRoot.
func (p *LocalClientOperationsProxy) resolvePath(filePath string) string {
	if filepath.IsAbs(filePath) {
		return filepath.Clean(filePath)
	}
	return filepath.Join(p.projectRoot, filePath)
}

// relativePath returns a forward-slash path relative to projectRoot.
func (p *LocalClientOperationsProxy) relativePath(absPath string) string {
	rel, err := filepath.Rel(p.projectRoot, absPath)
	if err != nil {
		return filepath.ToSlash(absPath)
	}
	return filepath.ToSlash(rel)
}

// ReadFile reads a file from the local filesystem.
func (p *LocalClientOperationsProxy) ReadFile(ctx context.Context, _, filePath string, startLine, endLine int32) (string, error) {
	resolved := p.resolvePath(filePath)

	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("stat file %s: %w", filePath, err)
	}

	if info.IsDir() {
		return fmt.Sprintf("[ERROR] Path is a directory, not a file: %s. This tool only reads files.", filePath), nil
	}

	if info.Size() > maxFileSize {
		return fmt.Sprintf("File too large: %d bytes (max %d)", info.Size(), maxFileSize), nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("read file %s: %w", filePath, err)
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	// No range requested — return full content
	if startLine == 0 && endLine == 0 {
		slog.InfoContext(ctx, "read file", "path", filePath, "lines", totalLines)
		return string(data), nil
	}

	// Clamp range
	start := int(startLine)
	if start < 1 {
		start = 1
	}
	start-- // convert to 0-based

	end := totalLines
	if endLine > 0 && int(endLine) < totalLines {
		end = int(endLine)
	}

	if start >= end {
		return fmt.Sprintf("[INFO] File: %s, Total lines: %d, Requested range: %d-%d (empty)", filePath, totalLines, startLine, endLine), nil
	}

	slog.InfoContext(ctx, "read file range", "path", filePath, "start", start+1, "end", end, "total", totalLines)
	return strings.Join(lines[start:end], "\n"), nil
}

// WriteFile writes content to a file on the local filesystem.
func (p *LocalClientOperationsProxy) WriteFile(ctx context.Context, _, filePath, content string) (string, error) {
	resolved := p.resolvePath(filePath)

	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return "", fmt.Errorf("create directories for %s: %w", filePath, err)
	}

	if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write file %s: %w", filePath, err)
	}

	lineCount := len(strings.Split(content, "\n"))
	relPath := p.relativePath(resolved)

	slog.InfoContext(ctx, "wrote file", "path", relPath, "lines", lineCount)
	return fmt.Sprintf("File written: %s (%d lines)", relPath, lineCount), nil
}

// GetProjectTree returns a directory tree listing.
func (p *LocalClientOperationsProxy) GetProjectTree(ctx context.Context, _, _, path string, maxDepth int) (string, error) {
	scanRoot := p.projectRoot
	if path != "" {
		scanRoot = filepath.Join(p.projectRoot, path)
	}

	info, err := os.Stat(scanRoot)
	if os.IsNotExist(err) {
		return fmt.Sprintf("[ERROR] Path not found: %s", path), nil
	}
	if err != nil {
		return "", fmt.Errorf("stat path %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Sprintf("[ERROR] Path is a file, not a directory: %s", path), nil
	}

	var lines []string
	if err := p.walkTree(scanRoot, 0, maxDepth, &lines); err != nil {
		return "", fmt.Errorf("walk tree %s: %w", path, err)
	}

	slog.InfoContext(ctx, "project tree", "path", path, "items", len(lines))
	return strings.Join(lines, "\n"), nil
}

// walkTree recursively builds a directory tree listing.
func (p *LocalClientOperationsProxy) walkTree(dir string, depth, maxDepth int, lines *[]string) error {
	if depth > maxDepth {
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}

	// Separate directories and files, filter ignored entries
	var dirs, files []os.DirEntry
	for _, entry := range entries {
		name := entry.Name()

		if shouldIgnoreEntry(name) {
			continue
		}

		if entry.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}

	// Sort directories alphabetically
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name()) < strings.ToLower(dirs[j].Name())
	})

	// Sort files alphabetically (case-insensitive)
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name()) < strings.ToLower(files[j].Name())
	})

	indent := strings.Repeat("  ", depth)

	// Directories first
	for _, d := range dirs {
		*lines = append(*lines, indent+d.Name()+"/")
		if err := p.walkTree(filepath.Join(dir, d.Name()), depth+1, maxDepth, lines); err != nil {
			return err
		}
	}

	// Then files
	for _, f := range files {
		*lines = append(*lines, indent+f.Name())
	}

	return nil
}

// shouldIgnoreEntry returns true if the entry name should be excluded from tree/glob results.
func shouldIgnoreEntry(name string) bool {
	if alwaysIgnore[name] {
		return true
	}
	if defaultIgnore[name] {
		return true
	}
	// Hide hidden files/directories (starting with '.') not already handled by alwaysIgnore
	if strings.HasPrefix(name, ".") {
		return true
	}
	return false
}

// GlobSearch finds files matching a glob pattern in the project directory.
func (p *LocalClientOperationsProxy) GlobSearch(ctx context.Context, _, pattern string, limit int32) (string, error) {
	fsys := os.DirFS(p.projectRoot)

	matches, err := doublestar.Glob(fsys, pattern)
	if err != nil {
		return "", fmt.Errorf("glob pattern %q: %w", pattern, err)
	}

	// Filter ignored entries
	var filtered []string
	for _, match := range matches {
		if isIgnoredPath(match) {
			continue
		}
		filtered = append(filtered, match)
	}

	if len(filtered) == 0 {
		return fmt.Sprintf("No files found matching pattern: %q", pattern), nil
	}

	// Sort by modification time (newest first)
	sortByMtimeDesc(p.projectRoot, filtered)

	// Apply limit
	truncated := false
	if limit > 0 && int(limit) < len(filtered) {
		filtered = filtered[:limit]
		truncated = true
	}

	// Convert to forward slashes
	for i, f := range filtered {
		filtered[i] = filepath.ToSlash(f)
	}

	result := strings.Join(filtered, "\n")
	if truncated {
		result += "\n(Results truncated. Consider using a more specific pattern or path.)"
	}

	slog.InfoContext(ctx, "glob search", "pattern", pattern, "matches", len(filtered), "truncated", truncated)
	return result, nil
}

// isIgnoredPath checks if any path segment matches the ignore lists.
func isIgnoredPath(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts {
		if shouldIgnoreEntry(part) {
			return true
		}
	}
	return false
}

// fileModTime returns the modification time for a file, or zero time on error.
func fileModTime(root, path string) time.Time {
	info, err := os.Stat(filepath.Join(root, path))
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// sortByMtimeDesc sorts file paths by modification time, newest first.
func sortByMtimeDesc(root string, paths []string) {
	// Pre-fetch mod times to avoid repeated stat calls during sort
	mtimes := make(map[string]time.Time, len(paths))
	for _, p := range paths {
		mtimes[p] = fileModTime(root, p)
	}

	sort.Slice(paths, func(i, j int) bool {
		return mtimes[paths[i]].After(mtimes[paths[j]])
	})
}
