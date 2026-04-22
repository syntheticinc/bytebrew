package task

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
)

// TaskExecutor executes a single task by its UUID.
type TaskExecutor interface {
	Execute(ctx context.Context, taskID uuid.UUID) error
}

// TaskWorker processes background tasks from a queue using a goroutine pool.
type TaskWorker struct {
	executor    TaskExecutor
	queue       chan uuid.UUID
	concurrency int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	stopped     atomic.Bool
}

// NewTaskWorker creates a new TaskWorker with the given concurrency level.
func NewTaskWorker(executor TaskExecutor, concurrency int) *TaskWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskWorker{
		executor:    executor,
		queue:       make(chan uuid.UUID, 100),
		concurrency: concurrency,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start launches worker goroutines that process tasks from the queue.
func (w *TaskWorker) Start() {
	for i := 0; i < w.concurrency; i++ {
		w.wg.Add(1)
		go w.worker(i)
	}
	slog.InfoContext(context.Background(), "task worker started", "concurrency", w.concurrency)
}

func (w *TaskWorker) worker(id int) {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		case taskID, ok := <-w.queue:
			if !ok {
				return
			}
			if err := w.executor.Execute(w.ctx, taskID); err != nil {
				slog.ErrorContext(context.Background(), "task execution failed", "task_id", taskID, "worker", id, "error", err)
			}
		}
	}
}

// Submit adds a task ID to the queue for background execution.
// Returns false if the queue is full, the worker is stopped, or the task was dropped.
func (w *TaskWorker) Submit(taskID uuid.UUID) bool {
	if w.stopped.Load() {
		slog.DebugContext(context.Background(), "task worker stopped, rejecting submit", "task_id", taskID)
		return false
	}
	select {
	case w.queue <- taskID:
		return true
	default:
		slog.WarnContext(context.Background(), "task queue full, dropping task", "task_id", taskID)
		return false
	}
}

// Stop gracefully shuts down the worker pool.
// After Stop returns, Submit is a no-op.
func (w *TaskWorker) Stop() {
	w.stopped.Store(true)
	w.cancel()
	close(w.queue)
	w.wg.Wait()
	slog.InfoContext(context.Background(), "task worker stopped")
}
