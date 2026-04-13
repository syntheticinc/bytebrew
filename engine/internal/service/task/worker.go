package task

import (
	"context"
	"log/slog"
	"sync"
)

// TaskExecutor executes a single task by ID.
type TaskExecutor interface {
	Execute(ctx context.Context, taskID uint) error
}

// TaskWorker processes background tasks from a queue using a goroutine pool.
type TaskWorker struct {
	executor    TaskExecutor
	queue       chan uint
	concurrency int
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewTaskWorker creates a new TaskWorker with the given concurrency level.
func NewTaskWorker(executor TaskExecutor, concurrency int) *TaskWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &TaskWorker{
		executor:    executor,
		queue:       make(chan uint, 100),
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
	slog.Info("task worker started", "concurrency", w.concurrency)
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
				slog.Error("task execution failed", "task_id", taskID, "worker", id, "error", err)
			}
		}
	}
}

// Submit adds a task ID to the queue for background execution.
// Returns false if the queue is full and the task was dropped.
func (w *TaskWorker) Submit(taskID uint) bool {
	select {
	case w.queue <- taskID:
		return true
	default:
		slog.Warn("task queue full, dropping task", "task_id", taskID)
		return false
	}
}

// Stop gracefully shuts down the worker pool.
func (w *TaskWorker) Stop() {
	w.cancel()
	close(w.queue)
	w.wg.Wait()
	slog.Info("task worker stopped")
}
