package list_mobile_sessions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// --- Mocks ---

type mockFlowReader struct {
	flows []*domain.ActiveFlow
}

func (m *mockFlowReader) ListActiveFlows() []*domain.ActiveFlow {
	return m.flows
}

// --- Constructor Tests ---

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		flowReader FlowReader
		wantErr    bool
	}{
		{
			name:       "valid flow reader",
			flowReader: &mockFlowReader{},
			wantErr:    false,
		},
		{
			name:       "nil flow reader",
			flowReader: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, err := New(tt.flowReader)
			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, uc)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, uc)
		})
	}
}

// --- Execute Tests ---

func TestExecute(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("no active flows", func(t *testing.T) {
		uc, err := New(&mockFlowReader{flows: nil})
		require.NoError(t, err)

		sessions, err := uc.Execute(ctx)
		require.NoError(t, err)
		assert.Empty(t, sessions)
	})

	t.Run("single active flow", func(t *testing.T) {
		flows := []*domain.ActiveFlow{
			{
				SessionID:  "session-1",
				ProjectKey: "my-project",
				UserID:     "user-1",
				Task:       "Add health check endpoint",
				Status:     domain.FlowStatusRunning,
				StartedAt:  now,
			},
		}

		uc, err := New(&mockFlowReader{flows: flows})
		require.NoError(t, err)

		sessions, err := uc.Execute(ctx)
		require.NoError(t, err)
		require.Len(t, sessions, 1)

		s := sessions[0]
		assert.Equal(t, "session-1", s.SessionID)
		assert.Equal(t, "my-project", s.ProjectKey)
		assert.Equal(t, domain.FlowStatusRunning, s.Status)
		assert.Equal(t, "Add health check endpoint", s.CurrentTask)
		assert.Equal(t, now, s.StartedAt)
		assert.Equal(t, now, s.LastActivityAt)
		assert.Empty(t, s.ProjectRoot)
		assert.Empty(t, s.Platform)
		assert.False(t, s.HasAskUser)
	})

	t.Run("multiple active flows", func(t *testing.T) {
		flows := []*domain.ActiveFlow{
			{
				SessionID:  "session-1",
				ProjectKey: "project-a",
				Status:     domain.FlowStatusRunning,
				Task:       "Task A",
				StartedAt:  now,
			},
			{
				SessionID:  "session-2",
				ProjectKey: "project-b",
				Status:     domain.FlowStatusCompleted,
				Task:       "Task B",
				StartedAt:  now.Add(-time.Hour),
			},
			{
				SessionID:  "session-3",
				ProjectKey: "project-c",
				Status:     domain.FlowStatusFailed,
				Task:       "Task C",
				StartedAt:  now.Add(-2 * time.Hour),
			},
		}

		uc, err := New(&mockFlowReader{flows: flows})
		require.NoError(t, err)

		sessions, err := uc.Execute(ctx)
		require.NoError(t, err)
		require.Len(t, sessions, 3)

		assert.Equal(t, "session-1", sessions[0].SessionID)
		assert.Equal(t, domain.FlowStatusRunning, sessions[0].Status)

		assert.Equal(t, "session-2", sessions[1].SessionID)
		assert.Equal(t, domain.FlowStatusCompleted, sessions[1].Status)

		assert.Equal(t, "session-3", sessions[2].SessionID)
		assert.Equal(t, domain.FlowStatusFailed, sessions[2].Status)
	})
}
