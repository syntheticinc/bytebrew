package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineTask_IsTopLevel(t *testing.T) {
	someParent := uuid.New()
	tests := []struct {
		name         string
		parentTaskID *uuid.UUID
		want         bool
	}{
		{"nil parent is top level", nil, true},
		{"non-nil parent is not top level", &someParent, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &EngineTask{ParentTaskID: tt.parentTaskID}
			assert.Equal(t, tt.want, task.IsTopLevel())
		})
	}
}

func TestEngineTask_IsTerminal(t *testing.T) {
	tests := []struct {
		name   string
		status EngineTaskStatus
		want   bool
	}{
		{"completed is terminal", EngineTaskStatusCompleted, true},
		{"failed is terminal", EngineTaskStatusFailed, true},
		{"cancelled is terminal", EngineTaskStatusCancelled, true},
		{"pending is not terminal", EngineTaskStatusPending, false},
		{"in_progress is not terminal", EngineTaskStatusInProgress, false},
		{"needs_input is not terminal", EngineTaskStatusNeedsInput, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &EngineTask{Status: tt.status}
			assert.Equal(t, tt.want, task.IsTerminal())
		})
	}
}

func TestEngineTask_CanTransitionTo_Valid(t *testing.T) {
	tests := []struct {
		name   string
		from   EngineTaskStatus
		to     EngineTaskStatus
	}{
		{"pending -> in_progress", EngineTaskStatusPending, EngineTaskStatusInProgress},
		{"pending -> cancelled", EngineTaskStatusPending, EngineTaskStatusCancelled},
		{"in_progress -> completed", EngineTaskStatusInProgress, EngineTaskStatusCompleted},
		{"in_progress -> failed", EngineTaskStatusInProgress, EngineTaskStatusFailed},
		{"in_progress -> needs_input", EngineTaskStatusInProgress, EngineTaskStatusNeedsInput},
		{"in_progress -> cancelled", EngineTaskStatusInProgress, EngineTaskStatusCancelled},
		{"needs_input -> in_progress", EngineTaskStatusNeedsInput, EngineTaskStatusInProgress},
		{"needs_input -> cancelled", EngineTaskStatusNeedsInput, EngineTaskStatusCancelled},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &EngineTask{Status: tt.from}
			assert.True(t, task.CanTransitionTo(tt.to))
		})
	}
}

func TestEngineTask_CanTransitionTo_Invalid(t *testing.T) {
	tests := []struct {
		name string
		from EngineTaskStatus
		to   EngineTaskStatus
	}{
		{"completed -> in_progress", EngineTaskStatusCompleted, EngineTaskStatusInProgress},
		{"completed -> failed", EngineTaskStatusCompleted, EngineTaskStatusFailed},
		{"failed -> completed", EngineTaskStatusFailed, EngineTaskStatusCompleted},
		{"failed -> in_progress", EngineTaskStatusFailed, EngineTaskStatusInProgress},
		{"cancelled -> pending", EngineTaskStatusCancelled, EngineTaskStatusPending},
		{"cancelled -> in_progress", EngineTaskStatusCancelled, EngineTaskStatusInProgress},
		{"pending -> completed", EngineTaskStatusPending, EngineTaskStatusCompleted},
		{"pending -> failed", EngineTaskStatusPending, EngineTaskStatusFailed},
		{"unknown status", EngineTaskStatus("unknown"), EngineTaskStatusInProgress},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &EngineTask{Status: tt.from}
			assert.False(t, task.CanTransitionTo(tt.to))
		})
	}
}

func TestEngineTask_Transition_UpdatesTimestamps(t *testing.T) {
	t.Run("pending to in_progress sets StartedAt", func(t *testing.T) {
		task := &EngineTask{Status: EngineTaskStatusPending}
		require.Nil(t, task.StartedAt)

		err := task.Transition(EngineTaskStatusInProgress)
		require.NoError(t, err)
		assert.Equal(t, EngineTaskStatusInProgress, task.Status)
		assert.NotNil(t, task.StartedAt)
	})

	t.Run("in_progress to completed sets CompletedAt", func(t *testing.T) {
		task := &EngineTask{Status: EngineTaskStatusInProgress}
		require.Nil(t, task.CompletedAt)

		err := task.Transition(EngineTaskStatusCompleted)
		require.NoError(t, err)
		assert.Equal(t, EngineTaskStatusCompleted, task.Status)
		assert.NotNil(t, task.CompletedAt)
	})

	t.Run("in_progress to failed sets CompletedAt", func(t *testing.T) {
		task := &EngineTask{Status: EngineTaskStatusInProgress}

		err := task.Transition(EngineTaskStatusFailed)
		require.NoError(t, err)
		assert.NotNil(t, task.CompletedAt)
	})

	t.Run("in_progress to cancelled sets CompletedAt", func(t *testing.T) {
		task := &EngineTask{Status: EngineTaskStatusInProgress}

		err := task.Transition(EngineTaskStatusCancelled)
		require.NoError(t, err)
		assert.NotNil(t, task.CompletedAt)
	})

	t.Run("re-entering in_progress does not overwrite StartedAt", func(t *testing.T) {
		task := &EngineTask{Status: EngineTaskStatusPending}

		err := task.Transition(EngineTaskStatusInProgress)
		require.NoError(t, err)
		firstStartedAt := *task.StartedAt

		// Transition to needs_input, then back to in_progress
		err = task.Transition(EngineTaskStatusNeedsInput)
		require.NoError(t, err)

		err = task.Transition(EngineTaskStatusInProgress)
		require.NoError(t, err)
		assert.Equal(t, firstStartedAt, *task.StartedAt, "StartedAt should not be overwritten")
	})
}

func TestEngineTask_Transition_InvalidReturnsError(t *testing.T) {
	task := &EngineTask{Status: EngineTaskStatusCompleted}

	err := task.Transition(EngineTaskStatusInProgress)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidTransition)
	assert.Equal(t, EngineTaskStatusCompleted, task.Status, "status should not change on invalid transition")
}

