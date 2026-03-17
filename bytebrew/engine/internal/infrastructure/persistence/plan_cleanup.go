package persistence

import (
	"context"
	"log/slog"
	"time"
)

// PlanCleanupWorker periodically removes old completed/abandoned plans
type PlanCleanupWorker struct {
	storage  *SQLitePlanStorage
	interval time.Duration // How often to run cleanup
	maxAge   time.Duration // Delete plans older than this
	stopCh   chan struct{}
	doneCh   chan struct{}
}

// NewPlanCleanupWorker creates a new cleanup worker
func NewPlanCleanupWorker(storage *SQLitePlanStorage, interval, maxAge time.Duration) *PlanCleanupWorker {
	return &PlanCleanupWorker{
		storage:  storage,
		interval: interval,
		maxAge:   maxAge,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start begins the cleanup worker goroutine
func (w *PlanCleanupWorker) Start() {
	slog.Info("starting plan cleanup worker",
		"interval", w.interval,
		"max_age", w.maxAge)

	go w.run()
}

// Stop gracefully stops the cleanup worker
func (w *PlanCleanupWorker) Stop() {
	slog.Info("stopping plan cleanup worker")
	close(w.stopCh)
	<-w.doneCh
	slog.Info("plan cleanup worker stopped")
}

// run is the main cleanup loop
func (w *PlanCleanupWorker) run() {
	defer close(w.doneCh)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run cleanup immediately on start
	w.cleanup()

	for {
		select {
		case <-ticker.C:
			w.cleanup()
		case <-w.stopCh:
			return
		}
	}
}

// cleanup performs the actual cleanup operation
func (w *PlanCleanupWorker) cleanup() {
	ctx := context.Background()

	deleted, err := w.storage.DeleteOldPlans(ctx, w.maxAge)
	if err != nil {
		slog.Error("failed to cleanup old plans", "error", err)
		return
	}

	if deleted > 0 {
		slog.Info("cleaned up old plans",
			"count", deleted,
			"max_age", w.maxAge)
	} else {
		slog.Debug("no old plans to cleanup")
	}
}
