//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/delivery/grpc"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/flow_registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

// mockStream implements pb.FlowService_ExecuteFlowServer for testing
type mockStream struct {
	ctx       context.Context
	requests  []*pb.FlowRequest
	responses []*pb.FlowResponse
	sendErr   error
	recvErr   error
	recvIdx   int
}

func (m *mockStream) Send(resp *pb.FlowResponse) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	m.responses = append(m.responses, resp)
	return nil
}

func (m *mockStream) Recv() (*pb.FlowRequest, error) {
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	if m.recvIdx >= len(m.requests) {
		return nil, nil
	}
	req := m.requests[m.recvIdx]
	m.recvIdx++
	return req, nil
}

func (m *mockStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockStream) SetTrailer(md metadata.MD) {
}

func (m *mockStream) Context() context.Context {
	return m.ctx
}

func (m *mockStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockStream) RecvMsg(msg interface{}) error {
	return nil
}

// mockTaskRepository implements TaskRepository for testing (using domain.Subtask)
type mockTaskRepository struct {
	subtasks map[string]*domain.Subtask
}

func newMockTaskRepository() *mockTaskRepository {
	return &mockTaskRepository{
		subtasks: make(map[string]*domain.Subtask),
	}
}

func (r *mockTaskRepository) Create(ctx context.Context, subtask *domain.Subtask) error {
	r.subtasks[subtask.ID] = subtask
	return nil
}

func (r *mockTaskRepository) GetByID(ctx context.Context, id string) (*domain.Subtask, error) {
	subtask, ok := r.subtasks[id]
	if !ok {
		return nil, nil
	}
	return subtask, nil
}

func (r *mockTaskRepository) Update(ctx context.Context, subtask *domain.Subtask) error {
	r.subtasks[subtask.ID] = subtask
	return nil
}

// mockMessageRepository implements MessageRepository for testing
type mockMessageRepository struct {
	messages []*domain.Message
}

func newMockMessageRepository() *mockMessageRepository {
	return &mockMessageRepository{
		messages: make([]*domain.Message, 0),
	}
}

func (r *mockMessageRepository) Create(ctx context.Context, message *domain.Message) error {
	r.messages = append(r.messages, message)
	return nil
}

func (r *mockMessageRepository) GetBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*domain.Message, error) {
	result := make([]*domain.Message, 0)
	for _, msg := range r.messages {
		if msg.SessionID == sessionID {
			result = append(result, msg)
		}
	}
	return result, nil
}

// TestFlowHandler_ExecuteFlow_ValidationErrors tests input validation
func TestFlowHandler_ExecuteFlow_ValidationErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	tests := []struct {
		name    string
		request *pb.FlowRequest
		wantErr bool
	}{
		{
			name: "missing session_id",
			request: &pb.FlowRequest{
				SessionId:  "",
				ProjectKey: "test-project",
				Task:       "test question",
				UserId:     "test-user",
			},
			wantErr: true,
		},
		{
			name: "missing project_key",
			request: &pb.FlowRequest{
				SessionId:  "test-session",
				ProjectKey: "",
				Task:       "test question",
				UserId:     "test-user",
			},
			wantErr: true,
		},
		{
			name: "missing task",
			request: &pb.FlowRequest{
				SessionId:  "test-session",
				ProjectKey: "test-project",
				Task:       "",
				UserId:     "test-user",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup dependencies
			agentService := newMockAgentService()
			factory := newMockTurnExecutorFactory()

			flowRegistry := flow_registry.NewInMemoryRegistry()
			handler, err := grpc.NewFlowHandler(agentService, factory, 20*time.Second, flowRegistry)
			require.NoError(t, err)

			// Create mock stream
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stream := &mockStream{
				ctx:       ctx,
				requests:  []*pb.FlowRequest{tt.request},
				responses: make([]*pb.FlowResponse, 0),
			}

			// Execute
			err = handler.ExecuteFlow(stream)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFlowHandler_ExecuteFlow_Success tests successful flow execution
// NOTE: This test is skipped because it requires full Eino Agent integration
// For full e2e testing, use manual tests with real server
func TestFlowHandler_ExecuteFlow_Success(t *testing.T) {
	t.Skip("Skipping full e2e test - requires real Eino Agent integration")
}

// TestRepositories_Integration tests repository integration
func TestRepositories_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	t.Run("TaskRepository", func(t *testing.T) {
		repo := newMockTaskRepository()

		// Create subtask
		subtask, err := domain.NewSubtask("session-1", "Test subtask")
		require.NoError(t, err)

		err = repo.Create(ctx, subtask)
		require.NoError(t, err)

		// Retrieve subtask
		retrieved, err := repo.GetByID(ctx, subtask.ID)
		require.NoError(t, err)
		assert.Equal(t, subtask.ID, retrieved.ID)
		assert.Equal(t, subtask.Description, retrieved.Description)

		// Update subtask
		err = subtask.Start()
		require.NoError(t, err)
		err = repo.Update(ctx, subtask)
		require.NoError(t, err)

		// Verify update
		updated, err := repo.GetByID(ctx, subtask.ID)
		require.NoError(t, err)
		assert.Equal(t, domain.SubtaskStatusInProgress, updated.Status)
	})

	t.Run("MessageRepository", func(t *testing.T) {
		repo := newMockMessageRepository()

		// Create messages
		msg1, err := domain.NewMessage("session-1", domain.MessageTypeUser, "user", "Hello")
		require.NoError(t, err)

		msg2, err := domain.NewMessage("session-1", domain.MessageTypeAgent, "agent", "Hi there")
		require.NoError(t, err)

		err = repo.Create(ctx, msg1)
		require.NoError(t, err)

		err = repo.Create(ctx, msg2)
		require.NoError(t, err)

		// Retrieve messages by session
		messages, err := repo.GetBySessionID(ctx, "session-1", 0, 0)
		require.NoError(t, err)
		assert.Equal(t, 2, len(messages))
	})
}
