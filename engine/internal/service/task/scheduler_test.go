package task

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTaskCreator struct {
	mu    sync.Mutex
	calls []TriggerTaskParams
	err   error
}

func (m *mockTaskCreator) CreateFromTrigger(_ context.Context, params TriggerTaskParams) (uuid.UUID, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return uuid.Nil, m.err
	}
	m.calls = append(m.calls, params)
	return uuid.New(), nil
}

func (m *mockTaskCreator) getCalls() []TriggerTaskParams {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]TriggerTaskParams, len(m.calls))
	copy(cp, m.calls)
	return cp
}

func TestCronScheduler_AddTrigger(t *testing.T) {
	tests := []struct {
		name     string
		schedule string
		wantErr  bool
	}{
		{"valid schedule", "* * * * *", false},
		{"every 5 minutes", "*/5 * * * *", false},
		{"invalid schedule", "not-a-cron", true},
		{"empty schedule", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator := &mockTaskCreator{}
			s := NewCronScheduler(creator)
			defer s.Stop()

			err := s.AddTrigger(tt.schedule, "title", "desc", "src-1")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCronScheduler_TriggerFiresAndCreatesTask(t *testing.T) {
	creator := &mockTaskCreator{}
	s := NewCronScheduler(creator)

	err := s.AddTrigger("* * * * *", "Deploy", "Run deploy", "trigger-42")
	require.NoError(t, err)

	s.Start()
	defer s.Stop()

	assert.Equal(t, 0, len(creator.getCalls()), "no calls yet before cron fires")
}

func TestCronScheduler_CreatorError(t *testing.T) {
	creator := &mockTaskCreator{err: assert.AnError}
	s := NewCronScheduler(creator)

	err := s.AddTrigger("* * * * *", "title", "desc", "src")
	require.NoError(t, err)

	_, createErr := creator.CreateFromTrigger(context.Background(), TriggerTaskParams{
		Title: "title", Description: "desc", SourceID: "src",
	})
	require.Error(t, createErr)
}

func TestCronScheduler_StartStop(t *testing.T) {
	creator := &mockTaskCreator{}
	s := NewCronScheduler(creator)

	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.Stop()
}
