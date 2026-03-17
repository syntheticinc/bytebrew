package grpc

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/google/uuid"
	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SessionRegistryForHandler defines the interface for server-streaming session management.
// Consumer-side: only methods needed by FlowHandler (ISP).
type SessionRegistryForHandler interface {
	CreateSession(sessionID, projectKey, userID, projectRoot, platform, agentName string)
	GetSessionContext(sessionID string) (projectRoot, platform, projectKey, userID, agentName string, ok bool)
	Subscribe(sessionID string) (ch <-chan *pb.SessionEvent, cleanup func())
	PublishEvent(sessionID string, event *pb.SessionEvent)
	ReplayEvents(sessionID string, lastEventID int64) []*pb.SessionEvent
	EnqueueMessage(sessionID, content string) error
	DrainMessages(sessionID string)
	SendAskUserReply(sessionID, callID, reply string)
	Cancel(sessionID string) bool
	IsCancelled(sessionID string) bool
	HasSession(sessionID string) bool
	RemoveSession(sessionID string)
}

// CreateSession creates a new server-streaming session.
func (h *FlowHandler) CreateSession(ctx context.Context, req *pb.CreateSessionRequest) (*pb.CreateSessionResponse, error) {
	if h.sessionRegistry == nil {
		return nil, status.Error(codes.Unimplemented, "server-streaming API not enabled")
	}

	if req.ProjectKey == "" {
		return nil, status.Error(codes.InvalidArgument, "project_key is required")
	}
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	sessionID := uuid.New().String()

	var projectRoot, platform string
	if req.Context != nil {
		projectRoot = req.Context["project_root"]
		platform = req.Context["platform"]

		if ts := req.Context["testing_strategy"]; ts != "" {
			h.agentService.SetTestingStrategy(ts)
		}
	}

	// Set environment context on agent service
	if projectRoot != "" || platform != "" {
		h.agentService.SetEnvironmentContext(projectRoot, platform)
	}

	h.sessionRegistry.CreateSession(sessionID, req.ProjectKey, req.UserId, projectRoot, platform, "")

	slog.InfoContext(ctx, "[Streaming] session created",
		"session_id", sessionID,
		"project_key", req.ProjectKey,
		"project_root", projectRoot,
		"platform", platform)

	return &pb.CreateSessionResponse{SessionId: sessionID}, nil
}

// SendMessage sends a user message or ask_user reply to an active session.
func (h *FlowHandler) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	if h.sessionRegistry == nil {
		return nil, status.Error(codes.Unimplemented, "server-streaming API not enabled")
	}

	if req.SessionId == "" {
		return &pb.SendMessageResponse{Error: "session_id is required"}, nil
	}

	if !h.sessionRegistry.HasSession(req.SessionId) {
		return &pb.SendMessageResponse{Error: "session not found"}, nil
	}

	// Reply to ask_user question
	if req.ReplyTo != "" {
		h.sessionRegistry.SendAskUserReply(req.SessionId, req.ReplyTo, req.Content)
		slog.InfoContext(ctx, "[Streaming] ask_user reply sent",
			"session_id", req.SessionId,
			"reply_to", req.ReplyTo)
		return &pb.SendMessageResponse{Accepted: true}, nil
	}

	// New user message
	if req.Content == "" {
		return &pb.SendMessageResponse{Error: "content is required"}, nil
	}

	if err := h.sessionRegistry.EnqueueMessage(req.SessionId, req.Content); err != nil {
		return &pb.SendMessageResponse{Error: err.Error()}, nil
	}

	slog.InfoContext(ctx, "[Streaming] message enqueued",
		"session_id", req.SessionId,
		"content_len", len(req.Content))

	return &pb.SendMessageResponse{Accepted: true}, nil
}

// SubscribeSession streams events for a session to the client.
func (h *FlowHandler) SubscribeSession(req *pb.SubscribeSessionRequest, stream pb.FlowService_SubscribeSessionServer) error {
	if h.sessionRegistry == nil {
		return status.Error(codes.Unimplemented, "server-streaming API not enabled")
	}

	sessionID := req.SessionId
	if sessionID == "" {
		return status.Error(codes.InvalidArgument, "session_id is required")
	}

	if !h.sessionRegistry.HasSession(sessionID) {
		return status.Error(codes.NotFound, "session not found")
	}

	ctx := stream.Context()

	slog.InfoContext(ctx, "[Streaming] client subscribed",
		"session_id", sessionID,
		"last_event_id", req.LastEventId)

	// Subscribe FIRST, then replay (subscribe-first-replay-second pattern).
	eventCh, cleanup := h.sessionRegistry.Subscribe(sessionID)
	defer cleanup()

	// Start message processing loop via shared SessionProcessor
	if h.sessionProcessor != nil {
		h.sessionProcessor.StartProcessing(ctx, sessionID)
	}

	// Replay missed events on reconnect
	lastEventID, _ := strconv.ParseInt(req.LastEventId, 10, 64)
	missed := h.sessionRegistry.ReplayEvents(sessionID, lastEventID)
	for _, ev := range missed {
		if err := stream.Send(ev); err != nil {
			return err
		}
	}
	if len(missed) > 0 {
		slog.InfoContext(ctx, "[Streaming] replayed events",
			"session_id", sessionID,
			"count", len(missed))
	}

	// Main event loop — stream events to client
	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "[Streaming] client disconnected", "session_id", sessionID)
			return nil
		case event, ok := <-eventCh:
			if !ok {
				slog.InfoContext(ctx, "[Streaming] event channel closed", "session_id", sessionID)
				return nil
			}
			if err := stream.Send(event); err != nil {
				slog.ErrorContext(ctx, "[Streaming] failed to send event",
					"session_id", sessionID, "error", err)
				return err
			}
		}
	}
}

// CancelSession cancels an active session.
func (h *FlowHandler) CancelSession(ctx context.Context, req *pb.CancelSessionRequest) (*pb.CancelSessionResponse, error) {
	if h.sessionRegistry == nil {
		return nil, status.Error(codes.Unimplemented, "server-streaming API not enabled")
	}

	if req.SessionId == "" {
		return nil, status.Error(codes.InvalidArgument, "session_id is required")
	}

	cancelled := h.sessionRegistry.Cancel(req.SessionId)
	if cancelled {
		// Drain stale messages from queue immediately so they don't get processed after cancel
		h.sessionRegistry.DrainMessages(req.SessionId)
	}

	slog.InfoContext(ctx, "[Streaming] session cancel requested",
		"session_id", req.SessionId,
		"cancelled", cancelled)

	return &pb.CancelSessionResponse{Cancelled: cancelled}, nil
}

