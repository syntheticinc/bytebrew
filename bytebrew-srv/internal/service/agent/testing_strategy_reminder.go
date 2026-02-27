package agent

import (
	"context"
	"fmt"
	"strings"
)

// TestingStrategyReminder provides project testing strategy to the LLM context.
// Implements ContextReminderProvider.
type TestingStrategyReminder struct {
	strategy *TestingStrategy
}

// NewTestingStrategyReminder creates a new TestingStrategyReminder
func NewTestingStrategyReminder(strategy *TestingStrategy) *TestingStrategyReminder {
	return &TestingStrategyReminder{strategy: strategy}
}

func (r *TestingStrategyReminder) GetContextReminder(_ context.Context, _ string) (string, int, bool) {
	cfg := r.strategy.Testing
	if cfg.Build == nil && cfg.Unit == nil && cfg.Integration == nil && cfg.Lint == nil && cfg.Notes == "" {
		return "", 0, false
	}

	var sb strings.Builder
	sb.WriteString("**PROJECT TESTING STRATEGY:**\n")

	formatCommandEntry(&sb, "Build", cfg.Build)
	formatUnitEntry(&sb, cfg.Unit)
	formatCommandEntry(&sb, "Integration", cfg.Integration)
	formatCommandEntry(&sb, "Lint", cfg.Lint)

	if cfg.Notes != "" {
		fmt.Fprintf(&sb, "- Notes:\n%s\n", cfg.Notes)
	}

	sb.WriteString("\nWhen creating subtask acceptance criteria, use these commands. Do not invent test commands.\n")

	// priority 85 — between plan context (~80) and work context (90)
	return sb.String(), 85, true
}

// formatCommandEntry writes a labeled command entry to the builder
func formatCommandEntry(sb *strings.Builder, label string, entry *CommandEntry) {
	if entry == nil {
		return
	}
	fmt.Fprintf(sb, "- %s: %s", label, entry.Command)
	if entry.Description != "" {
		fmt.Fprintf(sb, " (%s)", entry.Description)
	}
	sb.WriteString("\n")
}

// formatUnitEntry writes the unit test entry with extras (pattern, framework)
func formatUnitEntry(sb *strings.Builder, entry *CommandEntry) {
	if entry == nil {
		return
	}
	fmt.Fprintf(sb, "- Unit tests: %s", entry.Command)

	var extras []string
	if entry.Pattern != "" {
		extras = append(extras, fmt.Sprintf("pattern: %s", entry.Pattern))
	}
	if entry.Framework != "" {
		extras = append(extras, fmt.Sprintf("framework: %s", entry.Framework))
	}
	if len(extras) > 0 {
		fmt.Fprintf(sb, " (%s)", strings.Join(extras, ", "))
	}
	if entry.Description != "" {
		fmt.Fprintf(sb, " — %s", entry.Description)
	}
	sb.WriteString("\n")
}
