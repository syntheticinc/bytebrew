package mobile_command

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// --- Mocks ---

type mockFlowReader struct {
	flows map[string]*domain.ActiveFlow
}

func newMockFlowReader() *mockFlowReader {
	return &mockFlowReader{flows: make(map[string]*domain.ActiveFlow)}
}

func (m *mockFlowReader) Get(sessionID string) (*domain.ActiveFlow, bool) {
	flow, ok := m.flows[sessionID]
	return flow, ok
}

type mockFlowCanceller struct {
	cancelled map[string]bool
	err       error
}

func newMockFlowCanceller() *mockFlowCanceller {
	return &mockFlowCanceller{cancelled: make(map[string]bool)}
}

func (m *mockFlowCanceller) CancelFlow(sessionID string) error {
	if m.err != nil {
		return m.err
	}
	m.cancelled[sessionID] = true
	return nil
}

type mockMessageInjector struct {
	messages      map[string]string
	askUserReplies map[string]string
	err           error
}

func newMockMessageInjector() *mockMessageInjector {
	return &mockMessageInjector{
		messages:      make(map[string]string),
		askUserReplies: make(map[string]string),
	}
}

func (m *mockMessageInjector) InjectMessage(sessionID, task string) error {
	if m.err != nil {
		return m.err
	}
	m.messages[sessionID] = task
	return nil
}

func (m *mockMessageInjector) InjectAskUserReply(sessionID, question, answer string) error {
	if m.err != nil {
		return m.err
	}
	m.askUserReplies[sessionID] = answer
	return nil
}

// --- Helper ---

func newTestUsecase(t *testing.T) (*Usecase, *mockFlowReader, *mockFlowCanceller, *mockMessageInjector) {
	t.Helper()
	reader := newMockFlowReader()
	canceller := newMockFlowCanceller()
	injector := newMockMessageInjector()

	uc, err := New(reader, canceller, injector)
	require.NoError(t, err)

	return uc, reader, canceller, injector
}

// --- Constructor Tests ---

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		flowReader FlowReader
		canceller  FlowCanceller
		injector   MessageInjector
		wantErr    bool
	}{
		{
			name:       "valid inputs",
			flowReader: newMockFlowReader(),
			canceller:  newMockFlowCanceller(),
			injector:   newMockMessageInjector(),
			wantErr:    false,
		},
		{
			name:       "nil flow reader",
			flowReader: nil,
			canceller:  newMockFlowCanceller(),
			injector:   newMockMessageInjector(),
			wantErr:    true,
		},
		{
			name:       "nil canceller",
			flowReader: newMockFlowReader(),
			canceller:  nil,
			injector:   newMockMessageInjector(),
			wantErr:    true,
		},
		{
			name:       "nil injector",
			flowReader: newMockFlowReader(),
			canceller:  newMockFlowCanceller(),
			injector:   nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, err := New(tt.flowReader, tt.canceller, tt.injector)
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

// --- SendNewTask Tests ---

func TestSendNewTask(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		uc, reader, _, injector := newTestUsecase(t)
		reader.flows["session-1"] = &domain.ActiveFlow{SessionID: "session-1", Status: domain.FlowStatusRunning}

		err := uc.SendNewTask(ctx, "session-1", "implement feature X")
		require.NoError(t, err)
		assert.Equal(t, "implement feature X", injector.messages["session-1"])
	})

	t.Run("empty session id", func(t *testing.T) {
		uc, _, _, _ := newTestUsecase(t)

		err := uc.SendNewTask(ctx, "", "task")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})

	t.Run("empty task", func(t *testing.T) {
		uc, _, _, _ := newTestUsecase(t)

		err := uc.SendNewTask(ctx, "session-1", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})

	t.Run("session not found", func(t *testing.T) {
		uc, _, _, _ := newTestUsecase(t)

		err := uc.SendNewTask(ctx, "nonexistent", "task")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeNotFound))
	})

	t.Run("injector error", func(t *testing.T) {
		uc, reader, _, injector := newTestUsecase(t)
		reader.flows["session-1"] = &domain.ActiveFlow{SessionID: "session-1"}
		injector.err = assert.AnError

		err := uc.SendNewTask(ctx, "session-1", "task")
		require.Error(t, err)
	})
}

// --- SendAskUserReply Tests ---

func TestSendAskUserReply(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		uc, reader, _, injector := newTestUsecase(t)
		reader.flows["session-1"] = &domain.ActiveFlow{SessionID: "session-1", Status: domain.FlowStatusRunning}

		err := uc.SendAskUserReply(ctx, "session-1", "Continue?", "yes")
		require.NoError(t, err)
		assert.Equal(t, "yes", injector.askUserReplies["session-1"])
	})

	t.Run("empty session id", func(t *testing.T) {
		uc, _, _, _ := newTestUsecase(t)

		err := uc.SendAskUserReply(ctx, "", "q", "a")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})

	t.Run("empty question", func(t *testing.T) {
		uc, _, _, _ := newTestUsecase(t)

		err := uc.SendAskUserReply(ctx, "session-1", "", "answer")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})

	t.Run("empty answer", func(t *testing.T) {
		uc, _, _, _ := newTestUsecase(t)

		err := uc.SendAskUserReply(ctx, "session-1", "question", "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})

	t.Run("session not found", func(t *testing.T) {
		uc, _, _, _ := newTestUsecase(t)

		err := uc.SendAskUserReply(ctx, "nonexistent", "q", "a")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeNotFound))
	})

	t.Run("injector error", func(t *testing.T) {
		uc, reader, _, injector := newTestUsecase(t)
		reader.flows["session-1"] = &domain.ActiveFlow{SessionID: "session-1"}
		injector.err = assert.AnError

		err := uc.SendAskUserReply(ctx, "session-1", "q", "a")
		require.Error(t, err)
	})
}

// --- CancelSession Tests ---

func TestCancelSession(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		uc, reader, canceller, _ := newTestUsecase(t)
		reader.flows["session-1"] = &domain.ActiveFlow{SessionID: "session-1", Status: domain.FlowStatusRunning}

		err := uc.CancelSession(ctx, "session-1")
		require.NoError(t, err)
		assert.True(t, canceller.cancelled["session-1"])
	})

	t.Run("empty session id", func(t *testing.T) {
		uc, _, _, _ := newTestUsecase(t)

		err := uc.CancelSession(ctx, "")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeInvalidInput))
	})

	t.Run("session not found", func(t *testing.T) {
		uc, _, _, _ := newTestUsecase(t)

		err := uc.CancelSession(ctx, "nonexistent")
		require.Error(t, err)
		assert.True(t, errors.Is(err, errors.CodeNotFound))
	})

	t.Run("canceller error", func(t *testing.T) {
		uc, reader, canceller, _ := newTestUsecase(t)
		reader.flows["session-1"] = &domain.ActiveFlow{SessionID: "session-1"}
		canceller.err = assert.AnError

		err := uc.CancelSession(ctx, "session-1")
		require.Error(t, err)
	})
}
