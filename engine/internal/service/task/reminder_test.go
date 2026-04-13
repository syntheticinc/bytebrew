package task

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
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
	// Stable UUIDs for predictable assertions.
	deployID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	buildID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	taskAID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	taskBID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	taskCID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")

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
					{ID: deployID, Title: "Deploy", Status: domain.EngineTaskStatusInProgress},
				},
			},
			contains: []string{
				"## Current Tasks",
				fmt.Sprintf(`[in_progress] Task %s: "Deploy"`, deployID),
				"Progress: 0/1 top-level tasks completed.",
			},
		},
		{
			name:      "completed top-level task",
			sessionID: "sess-1",
			tasks: map[string][]domain.EngineTask{
				"sess-1": {
					{ID: deployID, Title: "Deploy", Status: domain.EngineTaskStatusCompleted},
				},
			},
			contains: []string{
				fmt.Sprintf(`[completed] Task %s: "Deploy"`, deployID),
				"Progress: 1/1 top-level tasks completed.",
			},
		},
		{
			name:      "with sub-task",
			sessionID: "sess-1",
			tasks: map[string][]domain.EngineTask{
				"sess-1": {
					{ID: deployID, Title: "Deploy", Status: domain.EngineTaskStatusInProgress},
					{ID: buildID, Title: "Build", Status: domain.EngineTaskStatusPending, ParentTaskID: &deployID},
				},
			},
			contains: []string{
				fmt.Sprintf(`[in_progress] Task %s: "Deploy"`, deployID),
				fmt.Sprintf(`[pending] Task %s: "Build" (sub-task of %s)`, buildID, deployID),
				"Progress: 0/1 top-level tasks completed.",
			},
		},
		{
			name:      "multiple top-level tasks progress",
			sessionID: "sess-1",
			tasks: map[string][]domain.EngineTask{
				"sess-1": {
					{ID: taskAID, Title: "Task A", Status: domain.EngineTaskStatusCompleted},
					{ID: taskBID, Title: "Task B", Status: domain.EngineTaskStatusInProgress},
					{ID: taskCID, Title: "Task C", Status: domain.EngineTaskStatusPending},
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
					{ID: deployID, Title: "Deploy", Status: domain.EngineTaskStatusPending},
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
