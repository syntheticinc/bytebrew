package grpc

import (
	"context"
	"log/slog"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// runSingleAgentMode runs tasks directly via Engine TurnExecutor (no Orchestrator).
func (h *FlowHandler) runSingleAgentMode(
	ctx context.Context,
	req *pb.FlowRequest,
	stream pb.FlowService_ExecuteFlowServer,
	proxy *StreamBasedClientOperationsProxy,
	streamWriter *StreamWriter,
	agentEventStream *GrpcAgentEventStream,
	cancel context.CancelFunc,
	projectRoot, platform string,
) error {
	taskChan := make(chan string, 10)

	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				slog.InfoContext(ctx, "client stream ended", "error", err)
				close(taskChan)
				cancel()
				return
			}
			if msg == nil {
				continue
			}
			if msg.Cancel {
				slog.InfoContext(ctx, "received cancel request from client")
				close(taskChan)
				cancel()
				return
			}
			if msg.Ping != nil {
				slog.DebugContext(ctx, "received ping from client")
			}
			if msg.ToolResult != nil {
				_ = proxy.HandleToolResult(msg.ToolResult)
			}
			if msg.Task != "" {
				slog.InfoContext(ctx, "received new task message", "task", msg.Task)
				select {
				case taskChan <- msg.Task:
				default:
					slog.WarnContext(ctx, "task channel full, dropping task")
				}
			}
		}
	}()

	executeTask := func(task string) error {
		activeFlow, err := h.registerActiveFlow(req.SessionId, req.ProjectKey, req.UserId, task, projectRoot, platform, cancel)
		if err != nil {
			return err
		}
		defer h.cleanupFlowResources(req.SessionId, activeFlow)

		// Create TurnExecutor via Engine (same path as supervisor mode)
		turnExecutor := h.turnExecutorFactory.CreateForSession(proxy, req.SessionId, req.ProjectKey, projectRoot, platform, "supervisor")

		err = turnExecutor.ExecuteTurn(ctx, req.SessionId, req.ProjectKey, task,
			createChunkCallback(ctx, streamWriter, req.SessionId),
			h.createEventCallback(ctx, agentEventStream, req.SessionId, nil),
		)

		if err != nil {
			slog.ErrorContext(ctx, "turn execution failed", "error", err)
			activeFlow.MarkFailed()
			if ctx.Err() != nil {
				return ctx.Err()
			}
			sendErrorResponse(streamWriter, req.SessionId, err)
			return nil
		}

		activeFlow.MarkComplete()
		return streamWriter.Send(&pb.FlowResponse{
			SessionId: req.SessionId,
			Type:      pb.ResponseType_RESPONSE_TYPE_ANSWER,
			IsFinal:   true,
		})
	}

	if req.Task != "" {
		if err := executeTask(req.Task); err != nil {
			if ctx.Err() != nil {
				return status.Error(codes.Canceled, "stream cancelled")
			}
			return status.Errorf(codes.Internal, "task execution failed: %v", err)
		}
	}

	slog.InfoContext(ctx, "waiting for new task messages from client")
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "client disconnected")
			return nil
		case task, ok := <-taskChan:
			if !ok {
				slog.InfoContext(ctx, "task channel closed")
				return nil
			}
			slog.InfoContext(ctx, "processing new task", "task", task)
			if err := executeTask(task); err != nil {
				if ctx.Err() != nil {
					return status.Error(codes.Canceled, "stream cancelled")
				}
				slog.ErrorContext(ctx, "task execution error", "error", err)
			}
		}
	}
}

