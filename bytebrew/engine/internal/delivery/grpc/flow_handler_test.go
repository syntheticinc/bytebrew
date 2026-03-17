package grpc

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/agent"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/orchestrator"
	pkgerrors "github.com/syntheticinc/bytebrew/bytebrew/engine/pkg/errors"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Mock implementations for testing

type mockChatModel struct{}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "mock response",
	}, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](1)
	go func() {
		defer sw.Close()
		sw.Send(&schema.Message{
			Role:    schema.Assistant,
			Content: "mock stream response",
		}, nil)
	}()
	return sr, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

func (m *mockChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

func (m *mockChatModel) GetType() string {
	return "mock_chat_model"
}

func (m *mockChatModel) IsCallbacksEnabled() bool {
	return false
}

// newMockChatModel creates a mock chat model for testing
func newMockChatModel() *mockChatModel {
	return &mockChatModel{}
}

// mockTurnExecutorFactory implements TurnExecutorFactory for testing
type mockTurnExecutorFactory struct {
	executeTurnFunc func(ctx context.Context, sessionID, projectKey, question string, chunkCb func(string) error, eventCb func(*domain.AgentEvent) error) error
}

func (f *mockTurnExecutorFactory) CreateForSession(proxy tools.ClientOperationsProxy, sessionID, projectKey, projectRoot, platform string) orchestrator.TurnExecutor {
	return &mockTurnExecutor{
		executeTurnFunc: f.executeTurnFunc,
	}
}

// mockTurnExecutor implements orchestrator.TurnExecutor for testing
type mockTurnExecutor struct {
	executeTurnFunc func(ctx context.Context, sessionID, projectKey, question string, chunkCb func(string) error, eventCb func(*domain.AgentEvent) error) error
}

func (e *mockTurnExecutor) ExecuteTurn(ctx context.Context, sessionID, projectKey, question string,
	chunkCb func(string) error, eventCb func(*domain.AgentEvent) error) error {
	if e.executeTurnFunc != nil {
		return e.executeTurnFunc(ctx, sessionID, projectKey, question, chunkCb, eventCb)
	}
	// Default behavior: send some chunks
	chunks := []string{"chunk1", "chunk2", "chunk3"}
	for _, chunk := range chunks {
		if err := chunkCb(chunk); err != nil {
			return err
		}
	}
	return nil
}

// newMockTurnExecutorFactory creates a mock TurnExecutorFactory for testing
func newMockTurnExecutorFactory() *mockTurnExecutorFactory {
	return &mockTurnExecutorFactory{}
}

// mockFlowServiceStream implements pb.FlowService_ExecuteFlowServer for testing
type mockFlowServiceStream struct {
	recvFunc func() (*pb.FlowRequest, error)
	sendFunc func(*pb.FlowResponse) error
	ctx      context.Context
}

func (m *mockFlowServiceStream) Send(resp *pb.FlowResponse) error {
	if m.sendFunc != nil {
		return m.sendFunc(resp)
	}
	return nil
}

func (m *mockFlowServiceStream) Recv() (*pb.FlowRequest, error) {
	if m.recvFunc != nil {
		return m.recvFunc()
	}
	return &pb.FlowRequest{
		SessionId:  "session-1",
		ProjectKey: "project-1",
		Task:       "What is Go?",
		UserId:     "user-1",
	}, io.EOF
}

func (m *mockFlowServiceStream) Context() context.Context {
	if m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

func (m *mockFlowServiceStream) SetHeader(md metadata.MD) error {
	return nil
}

func (m *mockFlowServiceStream) SendHeader(md metadata.MD) error {
	return nil
}

func (m *mockFlowServiceStream) SetTrailer(md metadata.MD) {
}

func (m *mockFlowServiceStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockFlowServiceStream) RecvMsg(msg interface{}) error {
	return nil
}

// newMockAgentService creates a mock AgentService for testing
func newMockAgentService() *agent.Service {
	cfg := agent.Config{
		ChatModel: newMockChatModel(),
		MaxSteps:  10,
	}

	agentService, err := agent.New(cfg)
	if err != nil {
		panic("failed to create mock agent service: " + err.Error())
	}

	return agentService
}

func TestNewFlowHandler(t *testing.T) {
	factory := newMockTurnExecutorFactory()
	flowRegistry := flow_registry.NewInMemoryRegistry()
	handler, err := NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)

	if err != nil {
		t.Fatalf("NewFlowHandler() error = %v", err)
	}
	if handler == nil {
		t.Fatal("NewFlowHandler() returned nil")
	}

	if handler.turnExecutorFactory == nil {
		t.Error("NewFlowHandler() turnExecutorFactory is nil")
	}
}

func TestFlowHandler_ExecuteFlow_Success(t *testing.T) {
	receivedChunks := []string{}

	factory := &mockTurnExecutorFactory{
		executeTurnFunc: func(ctx context.Context, sessionID, projectKey, question string, chunkCb func(string) error, eventCb func(*domain.AgentEvent) error) error {
			if sessionID != "session-1" {
				t.Errorf("ExecuteTurn() SessionID = %v, want session-1", sessionID)
			}
			if projectKey != "project-1" {
				t.Errorf("ExecuteTurn() ProjectKey = %v, want project-1", projectKey)
			}
			if question != "What is Go?" {
				t.Errorf("ExecuteTurn() Question = %v, want 'What is Go?'", question)
			}

			chunks := []string{"Go ", "is ", "a ", "language"}
			for _, chunk := range chunks {
				if err := chunkCb(chunk); err != nil {
					return err
				}
			}
			return nil
		},
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	firstRecv := true
	stream := &mockFlowServiceStream{
		recvFunc: func() (*pb.FlowRequest, error) {
			if firstRecv {
				firstRecv = false
				return &pb.FlowRequest{
					SessionId:  "session-1",
					ProjectKey: "project-1",
					Task:       "What is Go?",
					UserId:     "user-1",
				}, nil
			}
			// Block until context is done (simulates client keeping stream open)
			<-ctx.Done()
			return nil, io.EOF
		},
		sendFunc: func(resp *pb.FlowResponse) error {
			if !resp.IsFinal {
				receivedChunks = append(receivedChunks, resp.Content)
			}
			// Cancel context after receiving completion response (IsFinal=true)
			if resp.IsFinal {
				ctxCancel()
			}
			return nil
		},
		ctx: ctx,
	}

	flowRegistry := flow_registry.NewInMemoryRegistry()
	handler, err := NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
	if err != nil {
		t.Fatalf("NewFlowHandler() error = %v", err)
	}
	err = handler.ExecuteFlow(stream)

	// Stream cancellation after completion is expected
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() != codes.Canceled {
			t.Errorf("ExecuteFlow() unexpected error: %v", err)
		}
	}

	if len(receivedChunks) != 4 {
		t.Errorf("ExecuteFlow() received %d chunks, want 4", len(receivedChunks))
	}
}

func TestFlowHandler_ExecuteFlow_RecvError(t *testing.T) {
	factory := newMockTurnExecutorFactory()

	stream := &mockFlowServiceStream{
		recvFunc: func() (*pb.FlowRequest, error) {
			return nil, errors.New("recv error")
		},
		ctx: context.Background(),
	}

	flowRegistry := flow_registry.NewInMemoryRegistry()
	handler, err := NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
	if err != nil {
		t.Fatalf("NewFlowHandler() error = %v", err)
	}
	err = handler.ExecuteFlow(stream)

	if err == nil {
		t.Error("ExecuteFlow() expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("ExecuteFlow() error is not a gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("ExecuteFlow() error code = %v, want %v", st.Code(), codes.InvalidArgument)
	}
}

func TestFlowHandler_ExecuteFlow_NilRequest(t *testing.T) {
	factory := newMockTurnExecutorFactory()

	stream := &mockFlowServiceStream{
		recvFunc: func() (*pb.FlowRequest, error) {
			return nil, nil
		},
		ctx: context.Background(),
	}

	flowRegistry := flow_registry.NewInMemoryRegistry()
	handler, err := NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
	if err != nil {
		t.Fatalf("NewFlowHandler() error = %v", err)
	}
	err = handler.ExecuteFlow(stream)

	if err == nil {
		t.Error("ExecuteFlow() expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("ExecuteFlow() error is not a gRPC status error")
	}

	if st.Code() != codes.InvalidArgument {
		t.Errorf("ExecuteFlow() error code = %v, want %v", st.Code(), codes.InvalidArgument)
	}
}

func TestFlowHandler_ExecuteFlow_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		request *pb.FlowRequest
		wantErr bool
	}{
		{
			name: "empty session_id",
			request: &pb.FlowRequest{
				SessionId:  "",
				ProjectKey: "project-1",
				Task:       "What is Go?",
				UserId:     "user-1",
			},
			wantErr: true,
		},
		{
			name: "empty project_key",
			request: &pb.FlowRequest{
				SessionId:  "session-1",
				ProjectKey: "",
				Task:       "What is Go?",
				UserId:     "user-1",
			},
			wantErr: true,
		},
		{
			name: "empty task - should wait for task message",
			request: &pb.FlowRequest{
				SessionId:  "session-1",
				ProjectKey: "project-1",
				Task:       "",
				UserId:     "user-1",
			},
			wantErr: false, // Task is now optional - server waits for task message
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := newMockTurnExecutorFactory()
			flowRegistry := flow_registry.NewInMemoryRegistry()

			// Use context with timeout for tests that wait for task message
			ctx := context.Background()
			if tt.name == "empty task - should wait for task message" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(context.Background(), 1*time.Second)
				defer cancel()
			}

			firstRecv := true
			stream := &mockFlowServiceStream{
				recvFunc: func() (*pb.FlowRequest, error) {
					if firstRecv {
						firstRecv = false
						return tt.request, nil
					}
					return nil, io.EOF
				},
				ctx: ctx,
			}

			handler, err := NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
			if err != nil {
				t.Fatalf("NewFlowHandler() error = %v", err)
			}
			err = handler.ExecuteFlow(stream)

			if tt.wantErr {
				if err == nil {
					t.Error("ExecuteFlow() expected error, got nil")
				}

				st, ok := status.FromError(err)
				if !ok {
					t.Fatal("ExecuteFlow() error is not a gRPC status error")
				}

				if st.Code() != codes.InvalidArgument {
					t.Errorf("ExecuteFlow() error code = %v, want %v", st.Code(), codes.InvalidArgument)
				}
			}
		})
	}
}

func TestFlowHandler_ExecuteFlow_UseCaseError(t *testing.T) {
	factory := &mockTurnExecutorFactory{
		executeTurnFunc: func(ctx context.Context, sessionID, projectKey, question string, chunkCb func(string) error, eventCb func(*domain.AgentEvent) error) error {
			return pkgerrors.New(pkgerrors.CodeInternal, "turn executor failed")
		},
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	errorResponseSent := false
	firstRecv := true
	stream := &mockFlowServiceStream{
		recvFunc: func() (*pb.FlowRequest, error) {
			if firstRecv {
				firstRecv = false
				return &pb.FlowRequest{
					SessionId:  "session-1",
					ProjectKey: "project-1",
					Task:       "What is Go?",
					UserId:     "user-1",
				}, nil
			}
			<-ctx.Done()
			return nil, io.EOF
		},
		sendFunc: func(resp *pb.FlowResponse) error {
			if resp.Type == pb.ResponseType_RESPONSE_TYPE_ERROR {
				errorResponseSent = true
				if resp.Error == nil {
					t.Error("ExecuteFlow() error response has nil Error field")
				}
				ctxCancel()
			}
			return nil
		},
		ctx: ctx,
	}

	flowRegistry := flow_registry.NewInMemoryRegistry()
	handler, err := NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
	if err != nil {
		t.Fatalf("NewFlowHandler() error = %v", err)
	}
	err = handler.ExecuteFlow(stream)

	// With blocking recv, error response is sent and handler continues
	// to main loop, then ctx is cancelled → handler returns nil
	if err != nil {
		// Accept both nil and Canceled
		st, ok := status.FromError(err)
		if ok && st.Code() != codes.Canceled {
			t.Errorf("ExecuteFlow() unexpected error: %v", err)
		}
	}

	if !errorResponseSent {
		t.Error("ExecuteFlow() did not send error response")
	}
}

func TestFlowHandler_ExecuteFlow_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	factory := &mockTurnExecutorFactory{
		executeTurnFunc: func(ctx context.Context, sessionID, projectKey, question string, chunkCb func(string) error, eventCb func(*domain.AgentEvent) error) error {
			// Simulate cancellation during execution
			cancel()
			return context.Canceled
		},
	}

	firstRecv := true
	stream := &mockFlowServiceStream{
		recvFunc: func() (*pb.FlowRequest, error) {
			if firstRecv {
				firstRecv = false
				return &pb.FlowRequest{
					SessionId:  "session-1",
					ProjectKey: "project-1",
					Task:       "What is Go?",
					UserId:     "user-1",
				}, nil
			}
			<-ctx.Done()
			return nil, io.EOF
		},
		sendFunc: func(resp *pb.FlowResponse) error {
			return nil
		},
		ctx: ctx,
	}

	flowRegistry := flow_registry.NewInMemoryRegistry()
	handler, err := NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
	if err != nil {
		t.Fatalf("NewFlowHandler() error = %v", err)
	}
	err = handler.ExecuteFlow(stream)

	if err == nil {
		t.Error("ExecuteFlow() expected error, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("ExecuteFlow() error is not a gRPC status error")
	}

	if st.Code() != codes.Canceled {
		t.Errorf("ExecuteFlow() error code = %v, want %v", st.Code(), codes.Canceled)
	}
}

func TestFlowHandler_ExecuteFlow_SendError(t *testing.T) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer ctxCancel()

	sendAttempted := false
	factory := &mockTurnExecutorFactory{
		executeTurnFunc: func(ctx context.Context, sessionID, projectKey, question string, chunkCb func(string) error, eventCb func(*domain.AgentEvent) error) error {
			return chunkCb("chunk")
		},
	}

	firstRecv := true
	stream := &mockFlowServiceStream{
		recvFunc: func() (*pb.FlowRequest, error) {
			if firstRecv {
				firstRecv = false
				return &pb.FlowRequest{
					SessionId:  "session-1",
					ProjectKey: "project-1",
					Task:       "What is Go?",
					UserId:     "user-1",
				}, nil
			}
			<-ctx.Done()
			return nil, io.EOF
		},
		sendFunc: func(resp *pb.FlowResponse) error {
			sendAttempted = true
			return errors.New("send error")
		},
		ctx: ctx,
	}

	flowRegistry := flow_registry.NewInMemoryRegistry()
	handler, err := NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
	if err != nil {
		t.Fatalf("NewFlowHandler() error = %v", err)
	}
	_ = handler.ExecuteFlow(stream)

	if !sendAttempted {
		t.Error("ExecuteFlow() did not attempt to send response")
	}
}

func TestFlowHandler_ExecuteFlow_CompletionResponse(t *testing.T) {
	completionSent := false

	factory := &mockTurnExecutorFactory{
		executeTurnFunc: func(ctx context.Context, sessionID, projectKey, question string, chunkCb func(string) error, eventCb func(*domain.AgentEvent) error) error {
			return chunkCb("chunk")
		},
	}

	ctx, ctxCancel := context.WithCancel(context.Background())
	firstRecv := true
	stream := &mockFlowServiceStream{
		recvFunc: func() (*pb.FlowRequest, error) {
			if firstRecv {
				firstRecv = false
				return &pb.FlowRequest{
					SessionId:  "session-1",
					ProjectKey: "project-1",
					Task:       "What is Go?",
					UserId:     "user-1",
				}, nil
			}
			<-ctx.Done()
			return nil, io.EOF
		},
		sendFunc: func(resp *pb.FlowResponse) error {
			if resp.IsFinal && resp.Type == pb.ResponseType_RESPONSE_TYPE_ANSWER {
				completionSent = true
				ctxCancel()
			}
			return nil
		},
		ctx: ctx,
	}

	flowRegistry := flow_registry.NewInMemoryRegistry()
	handler, err := NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
	if err != nil {
		t.Fatalf("NewFlowHandler() error = %v", err)
	}
	err = handler.ExecuteFlow(stream)

	// Stream cancellation after completion is expected
	if err != nil {
		st, ok := status.FromError(err)
		if ok && st.Code() != codes.Canceled {
			t.Errorf("ExecuteFlow() unexpected error: %v", err)
		}
	}

	if !completionSent {
		t.Error("ExecuteFlow() did not send completion response")
	}
}
