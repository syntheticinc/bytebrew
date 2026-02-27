package grpc

import (
	"context"
	"log/slog"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	infragrpc "github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/grpc"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/orchestrator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AgentService defines interface for agent orchestration (used by delivery layer)
type AgentService interface {
	SetEnvironmentContext(projectRoot, platform string)
	SetTestingStrategy(yamlContent string)
}

// ActiveFlowRegistry defines interface for managing active flows
type ActiveFlowRegistry interface {
	Register(sessionID string, flow *domain.ActiveFlow) error
	Unregister(sessionID string) error
	Get(sessionID string) (*domain.ActiveFlow, bool)
	IsActive(sessionID string) bool
	BroadcastEvent(sessionID string, event *domain.AgentEvent) error
}

// AgentPoolProxy defines interface for updating proxy on agent pool (used by delivery layer)
type AgentPoolProxy interface {
	SetProxyForSession(sessionID string, proxy interface{})
	SetEventCallbackForSession(sessionID string, cb func(event *domain.AgentEvent) error)
	RemoveSession(sessionID string)
}

// WorkManagerForOrchestrator provides active work status for the Orchestrator (consumer-side)
type WorkManagerForOrchestrator interface {
	GetTasks(ctx context.Context, sessionID string) ([]*domain.Task, error)
}

// SessionStorage defines interface for session persistence (consumer-side)
type SessionStorage interface {
	GetByID(ctx context.Context, id string) (*domain.Session, error)
	Save(ctx context.Context, session *domain.Session) error
	Update(ctx context.Context, session *domain.Session) error
}

// TurnExecutorFactory creates a TurnExecutor for the given proxy/session.
// Consumer-side interface defined in FlowHandler.
type TurnExecutorFactory interface {
	CreateForSession(proxy tools.ClientOperationsProxy, sessionID, projectKey string) orchestrator.TurnExecutor
}

// FlowHandler handles FlowService gRPC requests
type FlowHandler struct {
	pb.UnimplementedFlowServiceServer
	agentService         AgentService
	agentPoolProxy       AgentPoolProxy              // For setting proxy/callback on agent pool
	agentPoolAdapter     tools.AgentPoolForTool      // Adapter for spawn_code_agent tool registration
	workManager          WorkManagerForOrchestrator  // For active work checking in Orchestrator
	sessionStorage       SessionStorage              // For session persistence (optional)
	turnExecutorFactory  TurnExecutorFactory         // Engine-based TurnExecutor factory (required)
	pingService          *infragrpc.PingService
	flowRegistry         ActiveFlowRegistry
}

// FlowHandlerConfig holds configuration for FlowHandler
type FlowHandlerConfig struct {
	AgentService        AgentService
	AgentPoolProxy      AgentPoolProxy              // Optional: for multi-agent mode
	AgentPoolAdapter    tools.AgentPoolForTool      // Optional: for spawn_code_agent tool
	WorkManager         WorkManagerForOrchestrator  // Optional: for Orchestrator active work checks
	SessionStorage      SessionStorage              // Optional: for session persistence
	TurnExecutorFactory TurnExecutorFactory         // Engine-based TurnExecutor factory (required)
	PingInterval        time.Duration
	FlowRegistry        ActiveFlowRegistry
}

// NewFlowHandler creates a new Flow handler
func NewFlowHandler(agentService AgentService, turnExecutorFactory TurnExecutorFactory, pingInterval time.Duration, flowRegistry ActiveFlowRegistry) (*FlowHandler, error) {
	return NewFlowHandlerWithConfig(FlowHandlerConfig{
		AgentService:        agentService,
		TurnExecutorFactory: turnExecutorFactory,
		PingInterval:        pingInterval,
		FlowRegistry:        flowRegistry,
	})
}

// NewFlowHandlerWithConfig creates a new Flow handler with full config
func NewFlowHandlerWithConfig(cfg FlowHandlerConfig) (*FlowHandler, error) {
	if cfg.AgentService == nil {
		return nil, status.Error(codes.InvalidArgument, "agent service is required")
	}

	if cfg.FlowRegistry == nil {
		return nil, status.Error(codes.InvalidArgument, "flow registry is required")
	}

	if cfg.TurnExecutorFactory == nil {
		return nil, status.Error(codes.InvalidArgument, "turn executor factory is required")
	}

	pingService, err := infragrpc.NewPingService(cfg.PingInterval)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create ping service: %v", err)
	}

	return &FlowHandler{
		agentService:        cfg.AgentService,
		agentPoolProxy:      cfg.AgentPoolProxy,
		agentPoolAdapter:    cfg.AgentPoolAdapter,
		workManager:         cfg.WorkManager,
		sessionStorage:      cfg.SessionStorage,
		turnExecutorFactory: cfg.TurnExecutorFactory,
		pingService:         pingService,
		flowRegistry:        cfg.FlowRegistry,
	}, nil
}

// ExecuteFlow handles bidirectional streaming for agent flow execution
func (h *FlowHandler) ExecuteFlow(stream pb.FlowService_ExecuteFlowServer) error {
	ctx := stream.Context()
	slog.InfoContext(ctx, "ExecuteFlow stream started")

	// Monitor context cancellation in background
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		slog.InfoContext(ctx, "stream context done", "error", ctx.Err())
	}()

	// Receive first request with task details
	req, err := stream.Recv()
	if err != nil {
		slog.ErrorContext(ctx, "failed to receive first request", "error", err)
		return status.Error(codes.InvalidArgument, "failed to receive request")
	}

	if req == nil {
		return status.Error(codes.InvalidArgument, "request is required")
	}

	// Validate request (session_id, project_key, and user_id required for stream initialization)
	if req.SessionId == "" {
		return status.Error(codes.InvalidArgument, "session_id is required")
	}

	if req.ProjectKey == "" {
		return status.Error(codes.InvalidArgument, "project_key is required")
	}

	if req.UserId == "" {
		return status.Error(codes.InvalidArgument, "user_id is required")
	}

	// Task is optional in first message - can be sent later
	// This allows bidirectional streaming without requiring task upfront

	slog.InfoContext(ctx, "processing flow request",
		"session_id", req.SessionId,
		"project_key", req.ProjectKey,
		"task", req.Task)

	// Extract environment context from client (project root, platform)
	if len(req.Context) > 0 {
		projectRoot := req.Context["project_root"]
		platform := req.Context["platform"]
		if projectRoot != "" || platform != "" {
			h.agentService.SetEnvironmentContext(projectRoot, platform)
			slog.InfoContext(ctx, "environment context set",
				"project_root", projectRoot,
				"platform", platform)
		}

		// Extract testing strategy from client
		if ts := req.Context["testing_strategy"]; ts != "" {
			h.agentService.SetTestingStrategy(ts)
			slog.InfoContext(ctx, "testing strategy set", "size", len(ts))
		}
	}

	// Check if there's an active flow for this session
	if h.flowRegistry.IsActive(req.SessionId) {
		slog.InfoContext(ctx, "active flow found, client will receive events via broadcast", "session_id", req.SessionId)

		// Start ping service
		err = h.pingService.Start(ctx, req.SessionId, func(pong *pb.PongResponse) error {
			return stream.Send(&pb.FlowResponse{
				SessionId: req.SessionId,
				Pong:      pong,
			})
		})
		if err != nil {
			slog.ErrorContext(ctx, "failed to start ping service", "error", err)
			return status.Errorf(codes.Internal, "failed to start ping service: %v", err)
		}
		defer h.pingService.Stop(req.SessionId)

		// Wait for context cancellation
		<-ctx.Done()
		slog.InfoContext(ctx, "client disconnected from active flow")
		return nil
	}

	// No active flow - wait for task from client
	slog.InfoContext(ctx, "no active flow, waiting for task", "session_id", req.SessionId)

	// Create thread-safe StreamWriter for all stream.Send() operations
	streamWriter := NewStreamWriter(stream)
	defer streamWriter.Close()

	// Create stream-based client operations proxy (uses StreamWriter for thread-safe writes)
	proxy := NewStreamBasedClientOperationsProxy(stream, req.SessionId, req.ProjectKey, streamWriter)
	defer proxy.CleanupPendingCalls()

	// Set proxy on AgentPool if available (for Code Agent tool calls)
	if h.agentPoolProxy != nil {
		h.agentPoolProxy.SetProxyForSession(req.SessionId, proxy)
		defer h.agentPoolProxy.RemoveSession(req.SessionId)
	}

	// Create agent event stream for sending events to client (uses StreamWriter)
	toolClassifier := tools.NewToolClassifier()
	agentEventStream := NewGrpcAgentEventStream(stream, req.SessionId, toolClassifier, streamWriter)

	// Set event callback on AgentPool (Code Agent events go through stream)
	if h.agentPoolProxy != nil {
		h.agentPoolProxy.SetEventCallbackForSession(req.SessionId, func(event *domain.AgentEvent) error {
			return agentEventStream.Send(event)
		})
	}

	// Log mode (tools are resolved per-session by TurnExecutorFactory)
	if h.agentPoolAdapter != nil {
		slog.InfoContext(ctx, "Supervisor mode enabled with agent pool")
	} else {
		slog.InfoContext(ctx, "Single-agent mode")
	}

	// Start ping service for keep-alive (uses StreamWriter for thread-safe writes)
	err = h.pingService.Start(ctx, req.SessionId, func(pong *pb.PongResponse) error {
		return streamWriter.Send(&pb.FlowResponse{
			SessionId: req.SessionId,
			Pong:      pong,
		})
	})
	if err != nil {
		slog.ErrorContext(ctx, "failed to start ping service", "error", err)
		return status.Errorf(codes.Internal, "failed to start ping service: %v", err)
	}
	defer h.pingService.Stop(req.SessionId)

	// Supervisor mode uses event-driven Orchestrator
	if h.agentPoolAdapter != nil {
		return h.runSupervisorMode(ctx, req, stream, proxy, streamWriter, agentEventStream, cancel)
	}

	// Single-agent mode uses direct task execution
	return h.runSingleAgentMode(ctx, req, stream, proxy, streamWriter, agentEventStream, cancel)
}

// registerActiveFlow creates and registers an active flow
func (h *FlowHandler) registerActiveFlow(sessionID, projectKey, userID, task string) (*domain.ActiveFlow, error) {
	activeFlow, err := domain.NewActiveFlow(sessionID, projectKey, userID, task)
	if err != nil {
		return nil, err
	}

	if err := h.flowRegistry.Register(sessionID, activeFlow); err != nil {
		return nil, err
	}

	return activeFlow, nil
}

// createChunkCallback creates a callback for sending answer chunks to the client.
// Used by both supervisor and single-agent modes.
func createChunkCallback(ctx context.Context, streamWriter *StreamWriter, sessionID string) func(string) error {
	return func(chunk string) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return streamWriter.Send(&pb.FlowResponse{
			SessionId: sessionID,
			Type:      pb.ResponseType_RESPONSE_TYPE_ANSWER_CHUNK,
			Content:   chunk,
			IsFinal:   false,
		})
	}
}

// createEventCallback creates a callback for sending agent events to the client.
// If workChecker is provided (supervisor mode), suppresses IsFinal when active work exists.
// If workChecker is nil (single-agent mode), events are sent as-is.
func (h *FlowHandler) createEventCallback(
	ctx context.Context,
	agentEventStream *GrpcAgentEventStream,
	sessionID string,
	workChecker orchestrator.ActiveWorkChecker,
) func(*domain.AgentEvent) error {
	return func(event *domain.AgentEvent) error {
		_ = h.flowRegistry.BroadcastEvent(sessionID, event)
		if workChecker != nil && shouldSuppressIsFinal(event, workChecker, ctx) {
			slog.DebugContext(ctx, "[Supervisor] suppressing IsFinal — active work pending")
			eventCopy := *event
			eventCopy.IsComplete = false
			return agentEventStream.Send(&eventCopy)
		}
		return agentEventStream.Send(event)
	}
}

// sendErrorResponse sends an error response to the client.
// Used by both supervisor and single-agent modes when task execution fails.
func sendErrorResponse(streamWriter *StreamWriter, sessionID string, err error) {
	errResp := &pb.FlowResponse{
		SessionId: sessionID,
		Type:      pb.ResponseType_RESPONSE_TYPE_ERROR,
		Error:     mapDomainErrorToProto(err),
		IsFinal:   true,
	}
	_ = streamWriter.Send(errResp)
}
