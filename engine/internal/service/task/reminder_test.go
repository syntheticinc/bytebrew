package task

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

type mockTaskLister struct {
	tasks map[string][]domain.EngineTask
	err   error
}

func (m *mockTaskLister) GetBySession(_ context.Context, sessionID string) ([]domain.EngineTask, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tasks[sessionID], nil
}

func TestTaskReminderProvider_GetReminder(t *testing.T) {
	parentID := "1"

	tests := []struct {
		name      string
		sessionID string
		tasks     map[string][]domain.EngineTask
		listerErr error
		wantEmpty bool
		contains  []string
	}{
		{
			name:      "empty when no tasks",
			sessionID: "sess-1",
			tasks:     map[string][]domain.EngineTask{},
			wantEmpty: true,
		},
		{
			name:      "empty on error",
			sessionID: "sess-1",
			listerErr: fmt.Errorf("db error"),
			wantEmpty: true,
		},
		{
			name:      "single top-level task",
			sessionID: "sess-1",
			tasks: map[string][]domain.EngineTask{
				"sess-1": {
					{ID: "1", Title: "Deploy", Status: domain.EngineTaskStatusInProgress},
				},
			},
			contains: []string{
				"## Current Tasks",
				`[in_progress] Task 1: "Deploy"`,
				"Progress: 0/1 top-level tasks completed.",
			},
		},
		{
			name:      "completed top-level task",
			sessionID: "sess-1",
			tasks: map[string][]domain.EngineTask{
				"sess-1": {
					{ID: "1", Title: "Deploy", Status: domain.EngineTaskStatusCompleted},
				},
			},
			contains: []string{
				`[completed] Task 1: "Deploy"`,
				"Progress: 1/1 top-level tasks completed.",
			},
		},
		{
			name:      "with sub-task",
			sessionID: "sess-1",
			tasks: map[string][]domain.EngineTask{
				"sess-1": {
					{ID: "1", Title: "Deploy", Status: domain.EngineTaskStatusInProgress},
					{ID: "2", Title: "Build", Status: domain.EngineTaskStatusPending, ParentTaskID: &parentID},
				},
			},
			contains: []string{
				`[in_progress] Task 1: "Deploy"`,
				`[pending] Task 2: "Build" (sub-task of 1)`,
				"Progress: 0/1 top-level tasks completed.",
			},
		},
		{
			name:      "multiple top-level tasks progress",
			sessionID: "sess-1",
			tasks: map[string][]domain.EngineTask{
				"sess-1": {
					{ID: "1", Title: "Task A", Status: domain.EngineTaskStatusCompleted},
					{ID: "2", Title: "Task B", Status: domain.EngineTaskStatusInProgress},
					{ID: "3", Title: "Task C", Status: domain.EngineTaskStatusPending},
				},
			},
			contains: []string{
				"Progress: 1/3 top-level tasks completed.",
			},
		},
		{
			name:      "different session returns empty",
			sessionID: "sess-other",
			tasks: map[string][]domain.EngineTask{
				"sess-1": {
					{ID: "1", Title: "Deploy", Status: domain.EngineTaskStatusPending},
				},
			},
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lister := &mockTaskLister{tasks: tt.tasks, err: tt.listerErr}
			provider := NewTaskReminderProvider(lister)

			result := provider.GetReminder(context.Background(), tt.sessionID)

			if tt.wantEmpty {
				assert.Empty(t, result)
				return
			}

			require.NotEmpty(t, result)
			for _, s := range tt.contains {
				assert.Contains(t, result, s)
			}
		})
	}
}
