package agent

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// EnvironmentContextReminder provides environment metadata (platform, project root)
// to the LLM context as a reminder. Implements ContextReminderProvider.
type EnvironmentContextReminder struct {
	projectRoot string
	platform    string // "win32" | "darwin" | "linux"
}

// NewEnvironmentContextReminder creates a new EnvironmentContextReminder
func NewEnvironmentContextReminder(projectRoot, platform string) *EnvironmentContextReminder {
	return &EnvironmentContextReminder{
		projectRoot: projectRoot,
		platform:    platform,
	}
}

func (r *EnvironmentContextReminder) GetContextReminder(_ context.Context, _ string) (string, int, bool) {
	if r.projectRoot == "" && r.platform == "" {
		return "", 0, false
	}

	var sb strings.Builder
	sb.WriteString("**ENVIRONMENT:**\n")

	// Current date and time (server-local, same timezone as user)
	now := time.Now()
	sb.WriteString(fmt.Sprintf("- Current date/time: %s\n", now.Format("2006-01-02 15:04 MST")))

	if r.platform != "" {
		sb.WriteString(fmt.Sprintf("- Platform: %s (%s)\n", r.platform, platformDescription(r.platform)))
	}
	if r.projectRoot != "" {
		sb.WriteString(fmt.Sprintf("- Project root: %s\n", r.projectRoot))
		sb.WriteString("- All file paths in read_file/write_file/edit_file are resolved relative to project root.\n")
	}
	sb.WriteString("- ALWAYS use platform-independent tools (get_project_tree, search_code, read_file) instead of shell commands (find, ls, grep, dir, cat, type, more, head, tail).\n")
	if r.platform != "" {
		sb.WriteString(fmt.Sprintf("- When you MUST use execute_command, use %s-compatible syntax.\n", shellHint(r.platform)))
	}

	return sb.String(), 95, true // priority 95 — after work context (90)
}

// platformDescription returns a human-readable OS name
func platformDescription(platform string) string {
	switch platform {
	case "win32":
		return "Windows"
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	default:
		return platform
	}
}

// shellHint returns shell syntax hint for the platform
func shellHint(platform string) string {
	switch platform {
	case "win32":
		return "PowerShell/cmd"
	case "darwin", "linux":
		return "bash/sh"
	default:
		return "shell"
	}
}
