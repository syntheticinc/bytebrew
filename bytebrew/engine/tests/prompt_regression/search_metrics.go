//go:build prompt

package prompt_regression

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/agents"
)

// SearchMetrics contains metrics about search tool usage in a session
type SearchMetrics struct {
	TotalSteps        int                // Total REACT steps (max step number)
	TotalToolCalls    int                // Total tool calls across all steps
	ToolCallBreakdown map[string]int     // Tool name → count (e.g., "smart_search": 3, "read_file": 5)
	SmartSearchCalls  []SmartSearchCall  // Detailed info per smart_search call
	SearchToReadRatio float64            // smart_search calls / read_file calls
	TotalTokens       int                // From last snapshot
	UsedPercent       float64            // From last snapshot
	DuplicateQueries  int                // Number of similar/duplicate smart_search queries
}

// SmartSearchCall represents a single smart_search tool call
type SmartSearchCall struct {
	Step      int    // REACT step where this call was made
	Query     string // The search query
	ResultLen int    // Length of result content (chars)
}

// ExtractSearchMetrics analyzes snapshots and extracts search-related metrics.
// Each snapshot contains the FULL message history up to that step.
// We use the last snapshot for metrics, and compare snapshots to determine step attribution.
func ExtractSearchMetrics(snapshots []agents.ContextSnapshot) SearchMetrics {
	if len(snapshots) == 0 {
		return SearchMetrics{
			ToolCallBreakdown: make(map[string]int),
		}
	}

	metrics := SearchMetrics{
		TotalSteps:        snapshots[len(snapshots)-1].Step,
		ToolCallBreakdown: make(map[string]int),
		SmartSearchCalls:  make([]SmartSearchCall, 0),
	}

	// Build a map: toolCallID → step (by comparing consecutive snapshots)
	toolCallStep := buildToolCallStepMap(snapshots)

	// Analyze only the LAST snapshot (contains full conversation history)
	lastSnapshot := snapshots[len(snapshots)-1]

	// Track tool results by ToolCallID for result length
	toolResults := make(map[string]string)

	for _, msg := range lastSnapshot.Messages {
		// Collect tool results
		if msg.Role == "tool" && msg.ToolCallID != "" {
			toolResults[msg.ToolCallID] = msg.Content
		}

		// Count tool calls from assistant messages
		if msg.Role != "assistant" || len(msg.ToolCalls) == 0 {
			continue
		}

		for _, tc := range msg.ToolCalls {
			metrics.TotalToolCalls++
			metrics.ToolCallBreakdown[tc.Name]++

			if tc.Name != "smart_search" {
				continue
			}

			query := extractQueryFromArgs(tc.Arguments)
			step := toolCallStep[tc.ID]
			call := SmartSearchCall{
				Step:  step,
				Query: query,
			}

			// Get result length
			if result, ok := toolResults[tc.ID]; ok {
				call.ResultLen = len(result)
			}

			metrics.SmartSearchCalls = append(metrics.SmartSearchCalls, call)
		}
	}

	// Calculate search to read ratio
	smartSearchCount := metrics.ToolCallBreakdown["smart_search"]
	readFileCount := metrics.ToolCallBreakdown["read_file"]
	if readFileCount > 0 {
		metrics.SearchToReadRatio = float64(smartSearchCount) / float64(readFileCount)
	}

	// Detect duplicate queries
	metrics.DuplicateQueries = countDuplicateQueries(metrics.SmartSearchCalls)

	// Token info already in lastSnapshot (declared above)
	metrics.TotalTokens = lastSnapshot.TotalTokens
	metrics.UsedPercent = lastSnapshot.UsedPercent

	return metrics
}

// buildToolCallStepMap compares consecutive snapshots to determine which REACT step
// each tool call first appeared in. Returns toolCallID → step number.
func buildToolCallStepMap(snapshots []agents.ContextSnapshot) map[string]int {
	stepMap := make(map[string]int)
	prevToolCallIDs := make(map[string]bool)

	for _, snapshot := range snapshots {
		currentToolCallIDs := make(map[string]bool)

		for _, msg := range snapshot.Messages {
			if msg.Role != "assistant" || len(msg.ToolCalls) == 0 {
				continue
			}
			for _, tc := range msg.ToolCalls {
				currentToolCallIDs[tc.ID] = true
				// If this ID wasn't in the previous snapshot, it's new at this step
				if !prevToolCallIDs[tc.ID] {
					stepMap[tc.ID] = snapshot.Step
				}
			}
		}

		prevToolCallIDs = currentToolCallIDs
	}

	return stepMap
}

// extractQueryFromArgs parses tool call arguments JSON and extracts the query field
func extractQueryFromArgs(args string) string {
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(args), &parsed); err != nil {
		return ""
	}
	if query, ok := parsed["query"].(string); ok {
		return query
	}
	return ""
}

// countDuplicateQueries counts number of queries that have >70% word overlap
func countDuplicateQueries(calls []SmartSearchCall) int {
	duplicates := 0
	seen := make(map[int]bool)

	for i := 0; i < len(calls); i++ {
		if seen[i] {
			continue
		}
		for j := i + 1; j < len(calls); j++ {
			if seen[j] {
				continue
			}
			if querySimilarity(calls[i].Query, calls[j].Query) > 0.7 {
				duplicates++
				seen[j] = true
			}
		}
	}

	return duplicates
}

// querySimilarity calculates Jaccard similarity between two queries
func querySimilarity(q1, q2 string) float64 {
	words1 := splitWords(q1)
	words2 := splitWords(q2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0
	}

	// Build sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, w := range words1 {
		set1[w] = true
	}
	for _, w := range words2 {
		set2[w] = true
	}

	// Calculate intersection
	intersection := 0
	for w := range set1 {
		if set2[w] {
			intersection++
		}
	}

	// Calculate union
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 0
	}

	return float64(intersection) / float64(union)
}

// splitWords splits a string into normalized words
func splitWords(s string) []string {
	// Normalize: lowercase, remove punctuation
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^\w\s]+`)
	s = re.ReplaceAllString(s, " ")

	// Split and filter empty
	words := strings.Fields(s)
	return words
}

// LoadSnapshots loads all supervisor_step_*_context.json files from a session directory
func LoadSnapshots(sessionDir string) ([]agents.ContextSnapshot, error) {
	pattern := filepath.Join(sessionDir, "supervisor_step_*_context.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("glob snapshots: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no snapshot files found in %s", sessionDir)
	}

	// Parse step numbers and sort
	type fileWithStep struct {
		path string
		step int
	}

	filesWithSteps := make([]fileWithStep, 0, len(files))
	stepPattern := regexp.MustCompile(`supervisor_step_(\d+)_context\.json`)

	for _, f := range files {
		basename := filepath.Base(f)
		matches := stepPattern.FindStringSubmatch(basename)
		if len(matches) < 2 {
			continue
		}
		step, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		filesWithSteps = append(filesWithSteps, fileWithStep{path: f, step: step})
	}

	// Sort by step number
	sort.Slice(filesWithSteps, func(i, j int) bool {
		return filesWithSteps[i].step < filesWithSteps[j].step
	})

	// Load snapshots
	snapshots := make([]agents.ContextSnapshot, 0, len(filesWithSteps))
	for _, fws := range filesWithSteps {
		data, err := os.ReadFile(fws.path)
		if err != nil {
			return nil, fmt.Errorf("read snapshot %s: %w", fws.path, err)
		}

		var snapshot agents.ContextSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return nil, fmt.Errorf("parse snapshot %s: %w", fws.path, err)
		}

		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// FormatMetricsReport formats metrics as a human-readable text report
func FormatMetricsReport(m SearchMetrics) string {
	var sb strings.Builder

	sb.WriteString("=== Search Metrics Report ===\n")
	sb.WriteString(fmt.Sprintf("Total Steps: %d\n", m.TotalSteps))
	sb.WriteString(fmt.Sprintf("Total Tool Calls: %d\n", m.TotalToolCalls))
	sb.WriteString("\n")

	sb.WriteString("Tool Call Breakdown:\n")
	// Sort tool names for consistent output
	toolNames := make([]string, 0, len(m.ToolCallBreakdown))
	for name := range m.ToolCallBreakdown {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)

	for _, name := range toolNames {
		count := m.ToolCallBreakdown[name]
		sb.WriteString(fmt.Sprintf("  %-20s %d calls\n", name+":", count))
	}
	sb.WriteString("\n")

	if len(m.SmartSearchCalls) > 0 {
		sb.WriteString("Smart Search Queries:\n")
		// Track seen queries for duplicate detection
		seenQueries := make(map[string]int) // query → first step
		for _, call := range m.SmartSearchCalls {
			suffix := ""
			if firstStep, seen := seenQueries[call.Query]; seen {
				suffix = fmt.Sprintf(" ⚠️ SIMILAR to step %d", firstStep)
			} else {
				// Check for similar queries
				for otherQuery, otherStep := range seenQueries {
					if querySimilarity(call.Query, otherQuery) > 0.7 {
						suffix = fmt.Sprintf(" ⚠️ SIMILAR to step %d", otherStep)
						break
					}
				}
			}
			seenQueries[call.Query] = call.Step

			resultInfo := ""
			if call.ResultLen > 0 {
				resultInfo = fmt.Sprintf(" (result: %s chars)", formatNumber(call.ResultLen))
			}

			sb.WriteString(fmt.Sprintf("  Step %d: \"%s\"%s%s\n", call.Step, call.Query, resultInfo, suffix))
		}
		sb.WriteString("\n")
	}

	if m.DuplicateQueries > 0 {
		sb.WriteString(fmt.Sprintf("Duplicate Queries: %d\n", m.DuplicateQueries))
	}

	if m.ToolCallBreakdown["smart_search"] > 0 {
		sb.WriteString(fmt.Sprintf("Search → Read Ratio: %.2f (%d searches → %d reads)\n",
			m.SearchToReadRatio,
			m.ToolCallBreakdown["smart_search"],
			m.ToolCallBreakdown["read_file"]))
	}

	sb.WriteString(fmt.Sprintf("Token Usage: %s / %s (%.1f%%)\n",
		formatNumber(m.TotalTokens),
		formatNumber(calculateMaxTokens(m.TotalTokens, m.UsedPercent)),
		m.UsedPercent))

	return sb.String()
}

// formatNumber formats a number with thousand separators
func formatNumber(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	for i, digit := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(digit)
	}
	return result.String()
}

// calculateMaxTokens calculates max tokens from total and used percent
func calculateMaxTokens(total int, usedPercent float64) int {
	if usedPercent <= 0 {
		return 0
	}
	return int(float64(total) / (usedPercent / 100.0))
}
