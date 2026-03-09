package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
)

// rgMatch represents a single match from ripgrep JSON output.
type rgMatch struct {
	Type string `json:"type"`
	Data struct {
		Path struct {
			Text string `json:"text"`
		} `json:"path"`
		LineNumber int `json:"line_number"`
		Lines      struct {
			Text string `json:"text"`
		} `json:"lines"`
	} `json:"data"`
}

// grepResult holds a parsed match for formatting.
type grepResult struct {
	filePath   string
	lineNumber int
	content    string
}

// GrepSearch performs pattern-based search using ripgrep (rg).
func (p *LocalClientOperationsProxy) GrepSearch(ctx context.Context, _, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
	if pattern == "" {
		return "No matches found for pattern: \"\"", nil
	}

	if limit <= 0 {
		limit = 100
	}

	rgPath, err := exec.LookPath("rg")
	if err != nil {
		return p.grepFallback(ctx, pattern, limit, fileTypes, ignoreCase)
	}

	args := p.buildRgArgs(pattern, limit, fileTypes, ignoreCase)

	slog.InfoContext(ctx, "grep search", "pattern", pattern, "limit", limit, "file_types", fileTypes, "ignore_case", ignoreCase)

	cmd := exec.CommandContext(ctx, rgPath, args...)
	cmd.Dir = p.projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Exit code 1 = no matches (not an error)
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return fmt.Sprintf("No matches found for pattern: %q", pattern), nil
			}
			return "", fmt.Errorf("rg failed (exit %d): %s", exitErr.ExitCode(), stderr.String())
		}
		return "", fmt.Errorf("run rg: %w", runErr)
	}

	results, truncated := p.parseRgOutput(stdout.Bytes(), int(limit))

	if len(results) == 0 {
		return fmt.Sprintf("No matches found for pattern: %q", pattern), nil
	}

	return formatGrepResults(results, truncated), nil
}

// buildRgArgs constructs ripgrep command arguments.
func (p *LocalClientOperationsProxy) buildRgArgs(pattern string, limit int32, fileTypes []string, ignoreCase bool) []string {
	args := []string{
		"--json",
		"--line-number",
		fmt.Sprintf("--max-count=%d", limit+1),
	}

	if ignoreCase {
		args = append(args, "-i")
	}

	// File type filters
	for _, ft := range fileTypes {
		ft = strings.TrimSpace(ft)
		if ft == "" {
			continue
		}
		// If already has glob syntax, use as-is
		if strings.Contains(ft, "*") {
			args = append(args, fmt.Sprintf("--glob=%s", ft))
		} else {
			// Convert extension name to glob: go → *.go
			args = append(args, fmt.Sprintf("--glob=*.%s", ft))
		}
	}

	// Default exclusions
	excludes := []string{
		"node_modules", ".git", ".bytebrew",
		"dist", "build", "*.lock", "*.min.js", "*.min.css",
	}
	for _, ex := range excludes {
		args = append(args, fmt.Sprintf("--glob=!%s", ex))
	}

	args = append(args, "--", pattern, ".")

	return args
}

// parseRgOutput parses ripgrep JSON lines output and returns matches.
func (p *LocalClientOperationsProxy) parseRgOutput(output []byte, limit int) ([]grepResult, bool) {
	var results []grepResult
	truncated := false

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Bytes()

		var m rgMatch
		if err := json.Unmarshal(line, &m); err != nil {
			continue
		}

		if m.Type != "match" {
			continue
		}

		if len(results) >= limit {
			truncated = true
			break
		}

		filePath := filepath.ToSlash(m.Data.Path.Text)
		// Remove leading "./" if present
		filePath = strings.TrimPrefix(filePath, "./")

		results = append(results, grepResult{
			filePath:   filePath,
			lineNumber: m.Data.LineNumber,
			content:    strings.TrimRight(m.Data.Lines.Text, "\n\r"),
		})
	}

	return results, truncated
}

// formatGrepResults formats matches into a human-readable string.
func formatGrepResults(results []grepResult, truncated bool) string {
	var sb strings.Builder

	for i, r := range results {
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("%s:%d\n  %s", r.filePath, r.lineNumber, r.content))
	}

	// Summary
	summary := fmt.Sprintf("\n\n%d results", len(results))
	if truncated {
		summary += " (truncated)"
	}
	sb.WriteString(summary)

	return sb.String()
}

// grepFallback uses grep -rn as a fallback when rg is not installed.
func (p *LocalClientOperationsProxy) grepFallback(ctx context.Context, pattern string, limit int32, fileTypes []string, ignoreCase bool) (string, error) {
	grepPath, err := exec.LookPath("grep")
	if err != nil {
		return "", fmt.Errorf("neither rg (ripgrep) nor grep found in PATH; install ripgrep for best results")
	}

	slog.InfoContext(ctx, "grep search fallback to grep", "pattern", pattern)

	args := []string{"-rn"}
	if ignoreCase {
		args = append(args, "-i")
	}

	// File type includes
	for _, ft := range fileTypes {
		ft = strings.TrimSpace(ft)
		if ft == "" {
			continue
		}
		if strings.Contains(ft, "*") {
			args = append(args, "--include="+ft)
		} else {
			args = append(args, "--include=*."+ft)
		}
	}

	// Exclusions
	for _, ex := range []string{"node_modules", ".git", ".bytebrew", "dist", "build"} {
		args = append(args, "--exclude-dir="+ex)
	}

	args = append(args, "--", pattern, ".")

	cmd := exec.CommandContext(ctx, grepPath, args...)
	cmd.Dir = p.projectRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Exit code 1 = no matches
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return fmt.Sprintf("No matches found for pattern: %q", pattern), nil
		}
		return "", fmt.Errorf("grep failed: %s", stderr.String())
	}

	// Parse grep output: file:line:content
	var results []grepResult
	truncated := false

	scanner := bufio.NewScanner(bytes.NewReader(stdout.Bytes()))
	for scanner.Scan() {
		line := scanner.Text()

		if len(results) >= int(limit) {
			truncated = true
			break
		}

		// Parse file:line:content
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		filePath := filepath.ToSlash(strings.TrimPrefix(parts[0], "./"))
		lineNum := 0
		fmt.Sscanf(parts[1], "%d", &lineNum)

		results = append(results, grepResult{
			filePath:   filePath,
			lineNumber: lineNum,
			content:    strings.TrimRight(parts[2], "\n\r"),
		})
	}

	if len(results) == 0 {
		return fmt.Sprintf("No matches found for pattern: %q", pattern), nil
	}

	return formatGrepResults(results, truncated), nil
}
