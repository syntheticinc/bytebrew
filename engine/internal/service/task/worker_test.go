package task

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockExecutor struct {
	mu        sync.Mutex
	executed  []uuid.UUID
	execErr   error
	execDelay time.Duration
}

func (m *mockExecutor) Execute(_ context.Context, taskID uuid.UUID) error {
	if m.execDelay > 0 {
		time.Sleep(m.execDelay)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executed = append(m.executed, taskID)
	return m.execErr
}

func (m *mockExecutor) getExecuted() []uuid.UUID {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]uuid.UUID, len(m.executed))
	copy(result, m.executed)
	return result
}

func TestTaskWorker_SubmitAndProcess(t *testing.T) {
	exec := &mockExecutor{}
	w := NewTaskWorker(exec, 1)
	w.Start()

	id := uuid.New()
	ok := w.Submit(id)
	require.True(t, ok)

	// Wait for processing.
	require.Eventually(t, func() bool {
		return len(exec.getExecuted()) == 1
	}, 2*time.Second, 10*time.Millisecond)

	assert.Equal(t, []uuid.UUID{id}, exec.getExecuted())
	w.Stop()
}

func TestTaskWorker_GracefulStop(t *testing.T) {
	exec := &mockExecutor{}
	w := NewTaskWorker(exec, 2)
	w.Start()

	// Stop without submitting anything should not hang.
	done := make(chan struct{})
	go func() {
		w.Stop()
		close(done)
	}()

	select {
	case <-done:
		// success
	case <-time.After(3 * time.Second):
		t.Fatal("Stop() did not return in time")
	}
}

func TestTaskWorker_ConcurrentTasks(t *testing.T) {
	var counter atomic.Int32
	exec := &mockExecutor{execDelay: 50 * time.Millisecond}
	w := NewTaskWorker(exec, 4)
	w.Start()

	const numTasks = 8
	for i := 1; i <= numTasks; i++ {
		w.Submit(uuid.New())
		counter.Add(1)
	}

	require.Eventually(t, func() bool {
		return len(exec.getExecuted()) == numTasks
	}, 5*time.Second, 20*time.Millisecond)

	assert.Equal(t, numTasks, len(exec.getExecuted()))
	w.Stop()
}

func TestTaskWorker_ExecutionError(t *testing.T) {
	exec := &mockExecutor{execErr: fmt.Errorf("boom")}
	w := NewTaskWorker(exec, 1)
	w.Start()

	w.Submit(uuid.New())

	// Even on error, the worker continues processing.
	require.Eventually(t, func() bool {
		return len(exec.getExecuted()) == 1
	}, 2*time.Second, 10*time.Millisecond)

	// Submit another task — worker is still alive.
	w.Submit(uuid.New())
	require.Eventually(t, func() bool {
		return len(exec.getExecuted()) == 2
	}, 2*time.Second, 10*time.Millisecond)

	w.Stop()
}

func TestTaskWorker_FullQueue(t *testing.T) {
	exec := &mockExecutor{execDelay: 100 * time.Millisecond}
	w := NewTaskWorker(exec, 1)
	// Don't start workers — queue will fill up.

	// Fill the queue (capacity 100).
	for i := 0; i < 100; i++ {
		ok := w.Submit(uuid.New())
		require.True(t, ok)
	}

	// Next submit should return false (queue full).
	ok := w.Submit(uuid.New())
	assert.False(t, ok)

	w.Start()
	w.Stop()
}
