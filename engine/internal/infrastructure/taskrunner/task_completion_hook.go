package taskrunner

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/service/task"
)

// webhookShutdownTimeout caps how long Stop() waits for in-flight webhooks.
// Matches the per-webhook timeout so at worst we wait for one full round-trip
// even if the server is shutting down at the wrong moment.
const webhookShutdownTimeout = 45 * time.Second

// TaskCompletionHook reacts to a task finishing (completed/failed/cancelled) and
// fires the on-complete webhook registered on the originating trigger, if any.
//
// It's wired into the EngineTaskManagerAdapter so every completion path
// (agent tool, REST API, spawned code-agent) emits the notification.
//
// Concurrency: fire() runs in a detached goroutine so a cancelled caller context
// does not abort the webhook. In-flight deliveries are tracked by wg, and Stop()
// blocks shutdown up to webhookShutdownTimeout so customers do not silently lose
// completion notifications during restarts.
type TaskCompletionHook struct {
	taskRepo    *configrepo.GORMTaskRepository
	triggerRepo *configrepo.GORMTriggerRepository
	notifier    *task.CompletionNotifier

	wg      sync.WaitGroup
	stopped atomic.Bool
}

// NewTaskCompletionHook creates a completion hook.
// triggerRepo may be nil; in that case the hook is a no-op.
func NewTaskCompletionHook(
	taskRepo *configrepo.GORMTaskRepository,
	triggerRepo *configrepo.GORMTriggerRepository,
	notifier *task.CompletionNotifier,
) *TaskCompletionHook {
	return &TaskCompletionHook{
		taskRepo:    taskRepo,
		triggerRepo: triggerRepo,
		notifier:    notifier,
	}
}

// OnCompleted is invoked after a task's status was moved to completed/failed/cancelled.
// It runs asynchronously to avoid blocking the caller; errors are logged but not returned.
// After Stop() has been called, new notifications are dropped (logged) so the hook
// cannot leak goroutines past shutdown.
//
// Accepts uuid.UUID directly — callers are always the task manager adapter which
// already carries the parsed id.
func (h *TaskCompletionHook) OnCompleted(ctx context.Context, taskID uuid.UUID) {
	if h == nil || h.triggerRepo == nil || h.notifier == nil {
		return
	}
	if h.stopped.Load() {
		slog.Debug("completion hook stopped, dropping notification", "task_id", taskID)
		return
	}
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.fire(taskID)
	}()
}

// Stop blocks until all in-flight webhooks complete or webhookShutdownTimeout elapses.
// After Stop returns, OnCompleted is a no-op. Safe to call multiple times.
func (h *TaskCompletionHook) Stop() {
	if h == nil {
		return
	}
	// Mark stopped before waiting so newly-arriving OnCompleted calls are dropped.
	h.stopped.Store(true)
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		slog.Info("completion hook stopped cleanly")
	case <-time.After(webhookShutdownTimeout):
		slog.Warn("completion hook stop timed out; some webhooks may be mid-flight", "timeout", webhookShutdownTimeout)
	}
}

func (h *TaskCompletionHook) fire(taskID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), webhookShutdownTimeout)
	defer cancel()

	t, err := h.taskRepo.GetByID(ctx, taskID)
	if err != nil || t == nil {
		slog.Debug("completion hook: task not found", "task_id", taskID, "err", err)
		return
	}
	if t.SourceID == "" {
		// Task was not created from a trigger (agent, dashboard, API) — nothing to notify.
		return
	}
	if t.Source != domain.TaskSourceCron && t.Source != domain.TaskSourceWebhook {
		return
	}

	trigger, err := h.triggerRepo.GetByID(ctx, t.SourceID)
	if err != nil || trigger == nil {
		slog.Debug("completion hook: trigger not found", "trigger_id", t.SourceID, "err", err)
		return
	}
	if trigger.OnCompleteURL == "" {
		return
	}

	// Validate URL scheme — only http and https are allowed.
	parsed, err := url.Parse(trigger.OnCompleteURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		slog.Error("completion hook: invalid webhook URL scheme, skipping", "url", trigger.OnCompleteURL, "trigger_id", trigger.ID)
		return
	}

	headers := map[string]string{}
	if trigger.OnCompleteHeaders != "" {
		if err := json.Unmarshal([]byte(trigger.OnCompleteHeaders), &headers); err != nil {
			slog.Warn("completion hook: unmarshal on_complete_headers failed, using empty headers", "error", err, "trigger_id", trigger.ID)
		}
	}

	durationMs := int64(0)
	if t.StartedAt != nil && t.CompletedAt != nil {
		durationMs = t.CompletedAt.Sub(*t.StartedAt).Milliseconds()
	}

	payload := task.CompletionPayload{
		TaskID:     t.ID.String(),
		Status:     string(t.Status),
		Result:     t.Result,
		DurationMs: durationMs,
		TriggerID:  trigger.ID,
		AgentName:  t.AgentName,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}

	if err := h.notifier.Notify(ctx, trigger.OnCompleteURL, headers, payload); err != nil {
		slog.Warn("completion webhook failed", "task_id", t.ID, "trigger_id", trigger.ID, "url", trigger.OnCompleteURL, "error", err)
		return
	}
	slog.Info("completion webhook delivered", "task_id", t.ID, "trigger_id", trigger.ID)
}
