package task

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// TaskCreator creates tasks from trigger events (cron, webhook).
type TaskCreator interface {
	CreateFromTrigger(ctx context.Context, params TriggerTaskParams) (uuid.UUID, error)
}

// TriggerTaskParams holds parameters for creating a task from a trigger.
// Q.5: AgentName and Source dropped — no longer persisted on tasks.
type TriggerTaskParams struct {
	Title       string
	Description string
	SourceID    string // trigger id, used for MarkFired
}

// CronScheduler manages cron-based triggers that create tasks on schedule.
type CronScheduler struct {
	cron    *cron.Cron
	creator TaskCreator
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewCronScheduler creates a new CronScheduler.
func NewCronScheduler(creator TaskCreator) *CronScheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &CronScheduler{
		cron:    cron.New(),
		creator: creator,
		ctx:     ctx,
		cancel:  cancel,
	}
}

// AddTrigger registers a cron trigger that creates a task on the given schedule.
func (s *CronScheduler) AddTrigger(schedule, title, description, sourceID string) error {
	_, err := s.cron.AddFunc(schedule, func() {
		_, createErr := s.creator.CreateFromTrigger(s.ctx, TriggerTaskParams{
			Title:       title,
			Description: description,
			SourceID:    sourceID,
		})
		if createErr != nil {
			slog.ErrorContext(context.Background(), "cron trigger failed to create task", "error", createErr, "trigger", sourceID)
		}
	})
	if err != nil {
		return err
	}
	return nil
}

// Start begins running scheduled triggers.
func (s *CronScheduler) Start() { s.cron.Start() }

// Stop halts the scheduler and cancels any in-flight trigger callbacks.
func (s *CronScheduler) Stop() {
	s.cancel()
	s.cron.Stop()
}
