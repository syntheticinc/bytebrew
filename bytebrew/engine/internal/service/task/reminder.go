package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
)

// TaskLister retrieves tasks for a given session.
type TaskLister interface {
	GetBySession(ctx context.Context, sessionID string) ([]domain.EngineTask, error)
}

// TaskReminderProvider generates context reminders about current tasks for agent prompts.
type TaskReminderProvider struct {
	tasks TaskLister
}

// NewTaskReminderProvider creates a new TaskReminderProvider.
func NewTaskReminderProvider(tasks TaskLister) *TaskReminderProvider {
	return &TaskReminderProvider{tasks: tasks}
}

// GetReminder generates a markdown summary of tasks for the agent's context.
// Returns empty string if there are no tasks or an error occurs.
func (p *TaskReminderProvider) GetReminder(ctx context.Context, sessionID string) string {
	tasks, err := p.tasks.GetBySession(ctx, sessionID)
	if err != nil || len(tasks) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("## Current Tasks\n\n")

	for _, t := range tasks {
		sb.WriteString(fmt.Sprintf("[%s] Task %d: %q", string(t.Status), t.ID, t.Title))
		if t.ParentTaskID != nil {
			sb.WriteString(fmt.Sprintf(" (sub-task of %d)", *t.ParentTaskID))
		}
		sb.WriteString("\n")
	}

	completed := 0
	total := 0
	for _, t := range tasks {
		if t.IsTopLevel() {
			total++
			if t.Status == domain.EngineTaskStatusCompleted {
				completed++
			}
		}
	}
	if total > 0 {
		sb.WriteString(fmt.Sprintf("\nProgress: %d/%d top-level tasks completed.\n", completed, total))
	}

	return sb.String()
}
