package taskrunner

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/engine/internal/service/task"
)

// triggerMarker narrows GORMTriggerRepository to just MarkFired for the
// scheduler closure, so the adapter's other methods are not transitively
// pinned into the cron fan-out.
type triggerMarker interface {
	MarkFired(ctx context.Context, id string) error
}

// taskSubmitter hands a created task id to a background worker for autonomous
// execution. Implemented by *task.TaskWorker; defined here (consumer-side) so
// triggerTaskCreator can be nil-safe when no worker is wired.
type taskSubmitter interface {
	Submit(taskID uuid.UUID) bool
}

// triggerTaskCreator is the bridge from CronScheduler's TaskCreator interface to
// the unified EngineTaskManagerAdapter. It records the trigger id as SourceID so
// the run is traceable back to the originating trigger, and hands the new task
// id to the background worker so the agent actually runs (otherwise the row
// just sits pending in the DB forever).
//
// V2 (§4.1): every cron tick also stamps the trigger's last_fired_at via
// MarkFired so admin UIs can show the most recent fire for each trigger.
type triggerTaskCreator struct {
	manager     *EngineTaskManagerAdapter
	worker      taskSubmitter // optional — nil means cron only records the task without auto-executing
	triggerRepo triggerMarker // optional — nil means fire timestamps are not tracked (tests)
}

// CreateFromTrigger implements task.TaskCreator.
func (c *triggerTaskCreator) CreateFromTrigger(ctx context.Context, params task.TriggerTaskParams) (uuid.UUID, error) {
	taskID, err := c.manager.CreateTask(ctx, tools.CreateEngineTaskParams{
		Title:       params.Title,
		Description: params.Description,
		AgentName:   params.AgentName,
		Source:      params.Source,
		SourceID:    params.SourceID,
	})
	if err != nil {
		return uuid.Nil, err
	}
	if c.triggerRepo != nil && params.SourceID != "" {
		if markErr := c.triggerRepo.MarkFired(ctx, params.SourceID); markErr != nil {
			// Non-fatal: the task is already persisted; a missing last_fired_at
			// is cosmetic for cron. Log and move on.
			slog.Warn("mark trigger fired failed", "trigger_id", params.SourceID, "error", markErr)
		}
	}
	if c.worker != nil {
		if !c.worker.Submit(taskID) {
			// Queue is full. The task row is persisted as `pending`, so it is not
			// silently lost from the DB — operators see it in the admin UI and the
			// next cron tick (or a manual requeue) will pick it up. We still log at
			// Error level because this means the engine is over-saturated.
			slog.Error("background worker queue full — task will not auto-run now",
				"task_id", taskID, "source_id", params.SourceID,
				"hint", "task row is still in DB as pending; it will be picked up on next tick or can be re-triggered manually")
		}
	}
	return taskID, nil
}

// NewTriggerTaskCreator returns a task.TaskCreator backed by the unified task manager.
// If worker is non-nil, every task produced by a trigger is submitted for autonomous
// execution on the background worker pool. If triggerRepo is non-nil, each fire stamps
// the originating trigger's last_fired_at (§4.1).
func NewTriggerTaskCreator(manager *EngineTaskManagerAdapter, worker taskSubmitter, triggerRepo triggerMarker) task.TaskCreator {
	return &triggerTaskCreator{manager: manager, worker: worker, triggerRepo: triggerRepo}
}

// StartCronScheduler loads all enabled cron triggers from the DB and registers them
// with a newly-created CronScheduler, then starts it. Returns the started scheduler
// so the caller can Stop() it on shutdown.
//
// If no cron triggers exist (or the DB is empty), an empty scheduler is returned
// and Start/Stop are safe no-ops.
//
// The optional worker is handed to the trigger-task creator so that each cron-fired
// task is submitted for autonomous execution. Passing nil disables auto-run (tasks
// are still created in the DB but no agent will pick them up).
func StartCronScheduler(
	ctx context.Context,
	triggerRepo *configrepo.GORMTriggerRepository,
	manager *EngineTaskManagerAdapter,
	worker taskSubmitter,
) (*task.CronScheduler, error) {
	scheduler := task.NewCronScheduler(NewTriggerTaskCreator(manager, worker, triggerRepo))

	triggers, err := triggerRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list triggers for cron bootstrap: %w", err)
	}

	registered, considered, skipped := 0, 0, 0
	for _, t := range triggers {
		if !t.Enabled || t.Type != "cron" || t.Config.Schedule == "" {
			continue
		}
		considered++
		title, agentName, description := triggerTaskMetadata(&t)
		if agentName == "" {
			slog.Warn("cron trigger has no agent, skipping", "trigger_id", t.ID, "title", t.Title)
			skipped++
			continue
		}
		if err := scheduler.AddTrigger(t.Config.Schedule, title, description, agentName, t.ID); err != nil {
			slog.Warn("invalid cron schedule, skipping trigger", "trigger_id", t.ID, "schedule", t.Config.Schedule, "error", err)
			skipped++
			continue
		}
		registered++
	}

	scheduler.Start()
	switch {
	case considered == 0:
		slog.Info("cron scheduler started with no cron triggers configured (expected on fresh install)", "total_triggers_in_db", len(triggers))
	case registered == 0:
		slog.Warn("cron scheduler started but every considered cron trigger was skipped — check logs for per-trigger errors", "considered", considered, "skipped", skipped)
	case skipped > 0:
		slog.Warn("cron scheduler started with some triggers skipped", "registered", registered, "skipped", skipped, "considered", considered)
	default:
		slog.Info("cron scheduler started", "triggers_registered", registered)
	}
	return scheduler, nil
}

// triggerTaskMetadata extracts task fields from a trigger model.
// Falls back to the trigger title/description/agent name as sensible defaults.
// Schema scope is resolved downstream from the agent (see AgentSchemaResolver),
// so it's not returned here.
func triggerTaskMetadata(t *models.TriggerModel) (title, agentName, description string) {
	title = t.Title
	if title == "" {
		title = "Cron task"
	}
	description = t.Description
	agentName = t.Agent.Name
	return
}

// StartBackgroundWorker creates a TaskWorker with the given executor + concurrency.
// Returns the started worker so the caller can Stop() it on shutdown.
// Passing a nil executor yields nil — callers must guard against this before Submit.
func StartBackgroundWorker(executor task.TaskExecutor, concurrency int) *task.TaskWorker {
	if executor == nil {
		slog.Info("background task worker not started (no executor provided)")
		return nil
	}
	if concurrency <= 0 {
		concurrency = 4
	}
	worker := task.NewTaskWorker(executor, concurrency)
	worker.Start()
	return worker
}
