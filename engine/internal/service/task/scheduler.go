package task

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"
)

// TaskCreator creates tasks from trigger events (cron, webhook).
type TaskCreator interface {
	CreateFromTrigger(ctx context.Context, params TriggerTaskParams) (string, error)
}

// TriggerTaskParams holds parameters for creating a task from a trigger.
type TriggerTaskParams struct {
	Title       string
	Description string
	AgentName   string
	Source      string // "cron" or "webhook"
	SourceID    string
}

// CronScheduler manages cron-based triggers that create tasks on schedule.
type CronScheduler struct {
	cron    *cron.Cron
	creator TaskCreator
}

// NewCronScheduler creates a new CronScheduler.
func NewCronScheduler(creator TaskCreator) *CronScheduler {
	return &CronScheduler{
		cron:    cron.New(),
		creator: creator,
	}
}

// AddTrigger registers a cron trigger that creates a task on the given schedule.
func (s *CronScheduler) AddTrigger(schedule, title, description, agentName, sourceID string) error {
	_, err := s.cron.AddFunc(schedule, func() {
		_, createErr := s.creator.CreateFromTrigger(context.Background(), TriggerTaskParams{
			Title:       title,
			Description: description,
			AgentName:   agentName,
			Source:      "cron",
			SourceID:    sourceID,
		})
		if createErr != nil {
			slog.Error("cron trigger failed to create task", "error", createErr, "trigger", sourceID)
		}
	})
	if err != nil {
		return err
	}
	return nil
}

// Start begins running scheduled triggers.
func (s *CronScheduler) Start() { s.cron.Start() }

// Stop halts the scheduler.
func (s *CronScheduler) Stop() { s.cron.Stop() }
