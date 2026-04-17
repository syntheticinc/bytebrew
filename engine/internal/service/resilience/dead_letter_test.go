package resilience

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeadLetterQueue_TrackAndComplete(t *testing.T) {
	q := NewDeadLetterQueue(DefaultDeadLetterConfig(), nil)

	q.Track("task-1", "agent-1")
	assert.Equal(t, 1, q.RunningCount())

	q.Complete("task-1")
	assert.Equal(t, 0, q.RunningCount())
	assert.Len(t, q.DeadLetters(), 0)
}

func TestDeadLetterQueue_TrackAndFail(t *testing.T) {
	q := NewDeadLetterQueue(DefaultDeadLetterConfig(), nil)

	q.Track("task-1", "agent-1")
	q.Fail("task-1")
	assert.Equal(t, 0, q.RunningCount())
	assert.Len(t, q.DeadLetters(), 0) // failed != dead letter
}

func TestDeadLetterQueue_Timeout(t *testing.T) {
	// AC-RESIL-07: task timeout → status=timeout, parent gets event
	var mu sync.Mutex
	var timeoutCalls []string

	callback := func(task TrackedTask, elapsed time.Duration) {
		mu.Lock()
		timeoutCalls = append(timeoutCalls, task.TaskID)
		mu.Unlock()
	}

	q := NewDeadLetterQueue(DeadLetterConfig{TaskTimeout: 50 * time.Millisecond}, callback)
	q.Track("task-1", "agent-1")

	// Not yet timed out
	timedOut := q.CheckTimeouts()
	assert.Len(t, timedOut, 0)

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	timedOut = q.CheckTimeouts()
	require.Len(t, timedOut, 1)
	assert.Equal(t, "task-1", timedOut[0].TaskID)
	assert.Equal(t, TaskStatusTimeout, timedOut[0].Status)

	mu.Lock()
	assert.Len(t, timeoutCalls, 1)
	mu.Unlock()
}

func TestDeadLetterQueue_DeadLettersVisible(t *testing.T) {
	// AC-RESIL-08: dead letter tasks visible in Inspect
	q := NewDeadLetterQueue(DeadLetterConfig{TaskTimeout: 10 * time.Millisecond}, nil)
	q.Track("task-1", "agent-1")
	q.Track("task-2", "agent-2")

	time.Sleep(20 * time.Millisecond)
	q.CheckTimeouts()

	dead := q.DeadLetters()
	assert.Len(t, dead, 2)
}

func TestDeadLetterQueue_CompletedNotTimeout(t *testing.T) {
	q := NewDeadLetterQueue(DeadLetterConfig{TaskTimeout: 30 * time.Millisecond}, nil)
	q.Track("task-1", "agent-1")

	// Complete before timeout
	time.Sleep(10 * time.Millisecond)
	q.Complete("task-1")

	time.Sleep(30 * time.Millisecond)
	timedOut := q.CheckTimeouts()
	assert.Len(t, timedOut, 0) // completed, not timed out
}

func TestDeadLetterQueue_CustomTimeout(t *testing.T) {
	q := NewDeadLetterQueue(DefaultDeadLetterConfig(), nil)
	q.TrackWithTimeout("task-1", "agent-1", 30*time.Millisecond)

	time.Sleep(40 * time.Millisecond)
	timedOut := q.CheckTimeouts()
	assert.Len(t, timedOut, 1)
}

func TestDeadLetterQueue_Remove(t *testing.T) {
	q := NewDeadLetterQueue(DefaultDeadLetterConfig(), nil)
	q.Track("task-1", "agent-1")
	assert.Equal(t, 1, q.RunningCount())

	q.Remove("task-1")
	assert.Equal(t, 0, q.RunningCount())
}

func TestDeadLetterQueue_TimeoutAnnotatesReasonAndMovedAt(t *testing.T) {
	q := NewDeadLetterQueue(DeadLetterConfig{TaskTimeout: 20 * time.Millisecond}, nil)
	q.TrackWithName("task-1", "agent-1", "Support Agent")

	time.Sleep(30 * time.Millisecond)
	q.CheckTimeouts()

	dead := q.DeadLetters()
	require.Len(t, dead, 1)
	assert.Equal(t, "task_timeout", dead[0].Reason)
	assert.Equal(t, "Support Agent", dead[0].AgentName)
	assert.False(t, dead[0].MovedAt.IsZero(), "MovedAt should be set after timeout")
}

func TestDeadLetterQueue_RecordError(t *testing.T) {
	q := NewDeadLetterQueue(DeadLetterConfig{TaskTimeout: 10 * time.Millisecond}, nil)
	q.Track("task-1", "agent-1")
	q.RecordError("task-1", "context deadline exceeded")

	time.Sleep(20 * time.Millisecond)
	q.CheckTimeouts()

	dead := q.DeadLetters()
	require.Len(t, dead, 1)
	assert.Equal(t, "context deadline exceeded", dead[0].LastError)
}

