package work

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// WorkContextReminder provides context reminders about active tasks/subtasks
// for injection into the LLM context. Implements ContextReminderProvider.
type WorkContextReminder struct {
	manager *Manager
}

// NewWorkContextReminder creates a new context reminder for work state
func NewWorkContextReminder(manager *Manager) *WorkContextReminder {
	return &WorkContextReminder{manager: manager}
}

// GetContextReminder returns a compact summary of active tasks and subtasks.
// Priority 90 — injected after plan context (80).
func (r *WorkContextReminder) GetContextReminder(ctx context.Context, sessionID string) (string, int, bool) {
	tasks, err := r.manager.GetTasks(ctx, sessionID)
	if err != nil || len(tasks) == 0 {
		return "", 0, false
	}

	var sb strings.Builder
	sb.WriteString("**ACTIVE WORK:**\n")

	hasActive := false
	for _, task := range tasks {
		if task.IsTerminal() {
			continue
		}

		hasActive = true
		subtasks, _ := r.manager.GetSubtasksByTask(ctx, task.ID)

		completed := 0
		total := len(subtasks)
		var runningAgents []string
		var readySubtasks []string

		for _, subtask := range subtasks {
			if subtask.IsCompleted() {
				completed++
			}
			if subtask.Status == "in_progress" && subtask.AssignedAgentID != "" {
				runningAgents = append(runningAgents, fmt.Sprintf("%s→%s", subtask.AssignedAgentID, subtask.Title))
			}
			if subtask.Status == "pending" && !subtask.IsBlocked() {
				readySubtasks = append(readySubtasks, fmt.Sprintf("[%s] %s", subtask.ID, subtask.Title))
			}
		}

		sb.WriteString(fmt.Sprintf("Task [%s] \"%s\" (%s, %d/%d subtasks)\n",
			task.ID, task.Title, task.Status, completed, total))

		if task.Status == domain.TaskStatusDraft {
			age := time.Since(task.CreatedAt)
			if age > 30*time.Minute {
				sb.WriteString(fmt.Sprintf("  ⚠ STALE: pending approval for %s — consider cancelling\n",
					age.Truncate(time.Minute)))
			} else {
				sb.WriteString("  ⏳ Awaiting user approval\n")
			}
		}

		if len(runningAgents) > 0 {
			sb.WriteString(fmt.Sprintf("  Running: %s\n", strings.Join(runningAgents, ", ")))
		}
		if len(readySubtasks) > 0 {
			sb.WriteString(fmt.Sprintf("  Ready to spawn: %s\n", strings.Join(readySubtasks, ", ")))
			sb.WriteString("  → ACTION REQUIRED: Call spawn_code_agent(action=spawn, subtask_id=<ID>) for each ready subtask.\n")
		}
		if total == 0 && task.Status == "in_progress" {
			sb.WriteString("  ⚠ No subtasks created yet. Next step: manage_subtasks(action=create, task_id=...) to create subtasks.\n")
		}
	}

	if !hasActive {
		return "", 0, false
	}

	return sb.String(), 90, true
}
