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
	Register(sessionID string, flow *domain.ActiveFlow, cancel context.CancelFunc) error
	Unregister(sessionID string) error
	UnregisterIfCurrent(sessionID string, expected *domain.ActiveFlow) bool
	Get(sessionID string) (*domain.ActiveFlow, bool)
	IsActive(sessionID string) bool
	// SetMessageSink attaches a message sink so reconnecting clients can forward messages.
	SetMessageSink(sessionID string, sink interface{ PublishUserMessage(string) error })
	// PublishUserMessage delivers a user message to the active flow's EventBus.
	PublishUserMessage(sessionID, message string) bool
}

// AgentPoolProxy defines interface for updating proxy on agent pool (used by delivery layer)
type AgentPoolProxy interface {
	SetProxyForSession(sessionID string, proxy interface{})
	SetEventCallbackForSession(sessionID string, cb func(event *domain.AgentEvent) error)
	RemoveSession(sessionID string)
}

// ToolCallHistoryCleaner defines interface for clearing tool call history per session (consumer-side)
type ToolCallHistoryCleaner interface {
	ClearSession(sessionID string)
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
	CreateForSession(proxy tools.ClientOperationsProxy, sessionID, projectKey, projectRoot, platform string) orchestrator.TurnExecutor
}

// FlowHandler handles FlowService gRPC requests
type FlowHandler struct {
	pb.UnimplementedFlowServiceServer
	agentService           AgentService
	agentPoolProxy         AgentPoolProxy              // For setting proxy/callback on agent pool
	agentPoolAdapter       tools.AgentPoolForTool      // Adapter for spawn_code_agent tool registration
	workManager            WorkManagerForOrchestrator  // For active work checking in Orchestrator
	sessionStorage         SessionStorage              // For session persistence (optional)
	turnExecutorFactory    TurnExecutorFactory         // Engine-based TurnExecutor factory (required)
	toolCallHistoryCleaner ToolCallHistoryCleaner      // For clearing tool call history on cleanup (optional)
	pingService            *infragrpc.PingService
	flowRegistry           ActiveFlowRegistry
	sessionRegistry        SessionRegistryForHandler   // For server-streaming API (optional)
}

// FlowHandlerConfig holds configuration for FlowHandler
type FlowHandlerConfig struct {
	AgentService           AgentService
	AgentPoolProxy         AgentPoolProxy              // Optional: for multi-agent mode
	AgentPoolAdapter       tools.AgentPoolForTool      // Optional: for spawn_code_agent tool
	WorkManager            WorkManagerForOrchestrator  // Optional: for Orchestrator active work checks
	SessionStorage         SessionStorage              // Optional: for session persistence
	TurnExecutorFactory    TurnExecutorFactory         // Engine-based TurnExecutor factory (required)
	ToolCallHistoryCleaner ToolCallHistoryCleaner      // Optional: for clearing tool call history on cleanup
	PingInterval           time.Duration
	FlowRegistry           ActiveFlowRegistry
	SessionRegistry        SessionRegistryForHandler   // Optional: for server-streaming API
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
		agentService:           cfg.AgentService,
		agentPoolProxy:         cfg.AgentPoolProxy,
		agentPoolAdapter:       cfg.AgentPoolAdapter,
		workManager:            cfg.WorkManager,
		sessionStorage:         cfg.SessionStorage,
		turnExecutorFactory:    cfg.TurnExecutorFactory,
		toolCallHistoryCleaner: cfg.ToolCallHistoryCleaner,
		pingService:            pingService,
		flowRegistry:           cfg.FlowRegistry,
		sessionRegistry:        cfg.SessionRegistry,
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
	var projectRoot, platform string
	if len(req.Context) > 0 {
		projectRoot = req.Context["project_root"]
		platform = req.Context["platform"]
		if projectRoot != "" || platform != "" {
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
		if req.Task == "" {
			// Reconnect mode: subscribe to events and forward incoming messages
			slog.InfoContext(ctx, "active flow found, reconnecting client to existing flow", "session_id", req.SessionId)

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

			// Read from client stream and forward user messages to the active flow
			go func() {
				for {
					msg, recvErr := stream.Recv()
					if recvErr != nil {
						slog.InfoContext(ctx, "reconnected client stream ended", "error", recvErr)
						cancel()
						return
					}
					if msg == nil {
						continue
					}
					if msg.Task != "" {
						slog.InfoContext(ctx, "forwarding user message to active flow",
							"session_id", req.SessionId, "task_len", len(msg.Task))
						if !h.flowRegistry.PublishUserMessage(req.SessionId, msg.Task) {
							slog.WarnContext(ctx, "failed to forward message, flow may have ended",
								"session_id", req.SessionId)
						}
					}
				}
			}()

			<-ctx.Done()
			slog.InfoContext(ctx, "reconnected client disconnected from active flow")
			return nil
		}

		// New task received while flow is active — old flow will be replaced during registration
		slog.InfoContext(ctx, "active flow exists but new task received, replacing",
			"session_id", req.SessionId, "task", req.Task)
	}

	// Proceed with flow setup (either no prior flow, or replacing an existing one)
	slog.InfoContext(ctx, "proceeding with flow setup", "session_id", req.SessionId)

	// Create thread-safe StreamWriter for all stream.Send() operations
	streamWriter := NewStreamWriter(stream)
	defer streamWriter.Close()

	// Create stream-based client operations proxy (uses StreamWriter for thread-safe writes)
	proxy := NewStreamBasedClientOperationsProxy(stream, req.SessionId, req.ProjectKey, streamWriter)
	defer proxy.CleanupPendingCalls()

	// Set proxy on AgentPool if available (for Code Agent tool calls).
	// NOTE: RemoveSession is NOT deferred here — it's called from cleanupFlowResources
	// to avoid stale defers from replaced flows cleaning up the new flow's resources.
	if h.agentPoolProxy != nil {
		h.agentPoolProxy.SetProxyForSession(req.SessionId, proxy)
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
		return h.runSupervisorMode(ctx, req, stream, proxy, streamWriter, agentEventStream, cancel, projectRoot, platform)
	}

	// Single-agent mode uses direct task execution
	return h.runSingleAgentMode(ctx, req, stream, proxy, streamWriter, agentEventStream, cancel, projectRoot, platform)
}

// registerActiveFlow creates and registers an active flow with its cancel function.
// The cancel func is stored in the registry (not in the domain entity) to keep ActiveFlow pure.
func (h *FlowHandler) registerActiveFlow(sessionID, projectKey, userID, task, projectRoot, platform string, cancel context.CancelFunc) (*domain.ActiveFlow, error) {
	activeFlow, err := domain.NewActiveFlow(sessionID, projectKey, userID, task)
	if err != nil {
		return nil, err
	}
	activeFlow.ProjectRoot = projectRoot
	activeFlow.Platform = platform

	if err := h.flowRegistry.Register(sessionID, activeFlow, cancel); err != nil {
		return nil, err
	}

	return activeFlow, nil
}

// cleanupFlowResources cleans up resources only if this flow is still the current one.
// Prevents stale defer from cleaning up a replacement flow's resources.
func (h *FlowHandler) cleanupFlowResources(sessionID string, activeFlow *domain.ActiveFlow) {
	if !h.flowRegistry.UnregisterIfCurrent(sessionID, activeFlow) {
		return
	}
	if h.agentPoolProxy != nil {
		h.agentPoolProxy.RemoveSession(sessionID)
	}
	if h.toolCallHistoryCleaner != nil {
		h.toolCallHistoryCleaner.ClearSession(sessionID)
	}
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
