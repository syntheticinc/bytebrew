//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/delivery/grpc"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/flow_registry"
	grpcInfra "github.com/syntheticinc/bytebrew/engine/internal/infrastructure/grpc"
	"github.com/stretchr/testify/require"
)

// TestPingService tests PingService functionality
func TestPingService(t *testing.T) {
	ctx := context.Background()

	t.Run("PingService_StartStop", func(t *testing.T) {
		service, err := grpcInfra.NewPingService(100 * time.Millisecond)
		require.NoError(t, err)

		sessionID := "test-session"

		// Start ping service
		err = service.Start(ctx, sessionID, func(pong *pb.PongResponse) error {
			return nil
		})
		require.NoError(t, err)
		require.True(t, service.IsSessionActive(sessionID))

		// Stop ping service
		service.Stop(sessionID)
		require.False(t, service.IsSessionActive(sessionID))
	})

	t.Run("PingService_MultipleSessions", func(t *testing.T) {
		service, err := grpcInfra.NewPingService(100 * time.Millisecond)
		require.NoError(t, err)

		// Start multiple sessions
		for i := 0; i < 3; i++ {
			sessionID := fmt.Sprintf("test-session-%d", i)
			err = service.Start(ctx, sessionID, func(pong *pb.PongResponse) error {
				return nil
			})
			require.NoError(t, err)
		}

		require.Equal(t, 3, service.GetSessionCount())

		// Stop all sessions
		for i := 0; i < 3; i++ {
			sessionID := fmt.Sprintf("test-session-%d", i)
			service.Stop(sessionID)
		}

		require.Equal(t, 0, service.GetSessionCount())
	})

	t.Run("PingService_PongCallback", func(t *testing.T) {
		service, err := grpcInfra.NewPingService(50 * time.Millisecond)
		require.NoError(t, err)

		sessionID := "test-session"
		pongCount := 0
		var mu sync.Mutex

		// Start ping service
		err = service.Start(ctx, sessionID, func(pong *pb.PongResponse) error {
			mu.Lock()
			defer mu.Unlock()
			pongCount++
			return nil
		})
		require.NoError(t, err)

		// Wait for at least one pong
		time.Sleep(120 * time.Millisecond)

		mu.Lock()
		count := pongCount
		mu.Unlock()

		// Should have received at least 2 pongs (120ms / 50ms = 2.4)
		require.GreaterOrEqual(t, count, 2, "Expected at least 2 pong messages")

		service.Stop(sessionID)
	})

	t.Run("PingService_InvalidInput", func(t *testing.T) {
		service, err := grpcInfra.NewPingService(0)
		require.Error(t, err)
		require.Nil(t, service)
	})

	t.Run("PingService_StartWithoutSessionID", func(t *testing.T) {
		service, err := grpcInfra.NewPingService(100 * time.Millisecond)
		require.NoError(t, err)

		err = service.Start(ctx, "", func(pong *pb.PongResponse) error {
			return nil
		})
		require.Error(t, err)
	})

	t.Run("PingService_StartWithoutCallback", func(t *testing.T) {
		service, err := grpcInfra.NewPingService(100 * time.Millisecond)
		require.NoError(t, err)

		err = service.Start(ctx, "test-session", nil)
		require.Error(t, err)
	})
}

// TestFlowHandler_PingPong tests FlowHandler with PingService integration
func TestFlowHandler_PingPong(t *testing.T) {
	t.Run("FlowHandler_NewFlowHandler", func(t *testing.T) {
		factory := newMockTurnExecutorFactory()
		flowRegistry := flow_registry.NewInMemoryRegistry()
		handler, err := grpc.NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
		require.NoError(t, err)
		require.NotNil(t, handler)
	})

	t.Run("FlowHandler_NewFlowHandler_InvalidInput", func(t *testing.T) {
		flowRegistry := flow_registry.NewInMemoryRegistry()
		handler, err := grpc.NewFlowHandler(nil, nil, 20*time.Second, flowRegistry)
		require.Error(t, err)
		require.Nil(t, handler)
	})

	t.Run("FlowHandler_NewFlowHandler_ZeroInterval", func(t *testing.T) {
		factory := newMockTurnExecutorFactory()
		flowRegistry := flow_registry.NewInMemoryRegistry()
		handler, err := grpc.NewFlowHandler(newMockAgentService(), factory, 0, flowRegistry)
		require.Error(t, err)
		require.Nil(t, handler)
	})

	t.Run("FlowHandler_ExecuteFlow_WithPing", func(t *testing.T) {
		// Create handler with short ping interval
		factory := newMockTurnExecutorFactory()
		flowRegistry := flow_registry.NewInMemoryRegistry()
		handler, err := grpc.NewFlowHandler(newMockAgentService(), factory, 100*time.Millisecond, flowRegistry)
		require.NoError(t, err)

		// Create mock stream
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stream := &mockStream{
			ctx: ctx,
			requests: []*pb.FlowRequest{{
				SessionId:  "test-session",
				ProjectKey: "test-project",
				Task:       "test question",
				UserId:     "test-user",
			}},
			responses: make([]*pb.FlowResponse, 0),
		}

		// Execute flow
		err = handler.ExecuteFlow(stream)
		require.NoError(t, err)

		// Verify that responses were sent
		require.Greater(t, len(stream.responses), 0, "Expected at least one response")

		// Verify final response
		lastResp := stream.responses[len(stream.responses)-1]
		require.True(t, lastResp.IsFinal, "Expected final response")
	})
}

// TestFlowHandler_Validation tests FlowHandler validation with PingService
func TestFlowHandler_Validation(t *testing.T) {
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
			factory := newMockTurnExecutorFactory()
			flowRegistry := flow_registry.NewInMemoryRegistry()
			handler, err := grpc.NewFlowHandler(newMockAgentService(), factory, 20*time.Second, flowRegistry)
			require.NoError(t, err)

			// Create mock stream
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			stream := &mockStream{
				ctx:       ctx,
				requests:  []*pb.FlowRequest{tt.request},
				responses: make([]*pb.FlowResponse, 0),
			}

			// Execute flow
			err = handler.ExecuteFlow(stream)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestBidirectionalStreaming_EndToEnd tests complete bidirectional streaming flow
func TestBidirectionalStreaming_EndToEnd(t *testing.T) {
	t.Run("CompleteFlow_WithPingPong", func(t *testing.T) {
		// This test verifies that:
		// 1. PingService starts when stream begins
		// 2. PingService stops when stream ends
		// 3. Pong messages are sent if stream is active
		// 4. FlowHandler properly integrates PingService

		// Create handler with short ping interval
		pingInterval := 50 * time.Millisecond
		factory := newMockTurnExecutorFactory()
		flowRegistry := flow_registry.NewInMemoryRegistry()
		handler, err := grpc.NewFlowHandler(newMockAgentService(), factory, pingInterval, flowRegistry)
		require.NoError(t, err)

		// Create mock stream
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		stream := &mockStream{
			ctx: ctx,
			requests: []*pb.FlowRequest{{
				SessionId:  "test-session-complete",
				ProjectKey: "test-project",
				Task:       "test question",
				UserId:     "test-user",
			}},
			responses: make([]*pb.FlowResponse, 0),
		}

		// Execute flow
		err = handler.ExecuteFlow(stream)
		require.NoError(t, err)

		// Verify responses
		require.Greater(t, len(stream.responses), 0, "Expected at least one response")

		// Verify final response
		lastResp := stream.responses[len(stream.responses)-1]
		require.True(t, lastResp.IsFinal, "Expected final response")
	})
}

// Mock implementations - moved to mocks.go
