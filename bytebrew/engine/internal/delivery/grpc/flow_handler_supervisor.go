package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/orchestrator"
)

// MessageRouter defines the minimal interface for anti-duplicate message routing.
// Consumer-side: only the methods needed for routing user messages.
type MessageRouter interface {
	HasBlockingWait(sessionID string) bool
	NotifyUserMessage(sessionID, message string)
}

// routeUserMessage routes a user message to either the interrupt mechanism or EventBus.
// Anti-duplicate: message goes to exactly ONE path, never both.
// Returns true if routed via interrupt, false if via EventBus.
func routeUserMessage(sessionID, message string, router MessageRouter, eventBus *orchestrator.SessionEventBus) bool {
	if router != nil && router.HasBlockingWait(sessionID) {
		router.NotifyUserMessage(sessionID, message)
		return true
	}
	if err := eventBus.PublishInterrupt(orchestrator.OrchestratorEvent{
		Type:    orchestrator.EventUserMessage,
		Content: message,
	}); err != nil {
		slog.Warn("[routeUserMessage] failed to publish user message", "error", err, "session_id", sessionID)
	}
	return false
}

// eventBusSink adapts SessionEventBus to flow_registry.UserMessageSink.
type eventBusSink struct {
	sessionID string
	router    MessageRouter
	eventBus  *orchestrator.SessionEventBus
}

func (s *eventBusSink) PublishUserMessage(message string) error {
	routeUserMessage(s.sessionID, message, s.router, s.eventBus)
	return nil
}

// touchSessionActivity updates session's last_activity_at timestamp.
func touchSessionActivity(ctx context.Context, storage SessionStorage, sessionID string) {
	if storage == nil {
		return
	}
	session, err := storage.GetByID(ctx, sessionID)
	if err != nil || session == nil {
		return
	}
	session.TouchActivity()
	_ = storage.Update(ctx, session)
}

// runSupervisorMode runs event-driven Orchestrator for multi-agent sessions.
func (h *FlowHandler) runSupervisorMode(
	ctx context.Context,
	req *pb.FlowRequest,
	stream pb.FlowService_ExecuteFlowServer,
	proxy *StreamBasedClientOperationsProxy,
	streamWriter *StreamWriter,
	agentEventStream *GrpcAgentEventStream,
	cancel context.CancelFunc,
	projectRoot, platform string,
) error {
	slog.InfoContext(ctx, "[Supervisor] starting event-driven mode", "session_id", req.SessionId)

	// 0. Session management (create or resume)
	if h.sessionStorage != nil {
		session, err := h.sessionStorage.GetByID(ctx, req.SessionId)
		if err != nil {
			slog.ErrorContext(ctx, "failed to get session", "error", err)
		} else if session != nil {
			// Resume existing session
			session.Activate()
			if err := h.sessionStorage.Update(ctx, session); err != nil {
				slog.ErrorContext(ctx, "failed to update session", "error", err)
			} else {
				slog.InfoContext(ctx, "resumed session", "session_id", session.ID, "project_key", session.ProjectKey)
			}
		} else {
			// Create new session
			newSession, err := domain.NewSession(req.SessionId, req.ProjectKey)
			if err != nil {
				slog.ErrorContext(ctx, "failed to create session entity", "error", err)
			} else if err := h.sessionStorage.Save(ctx, newSession); err != nil {
				slog.ErrorContext(ctx, "failed to save session", "error", err)
			} else {
				slog.InfoContext(ctx, "created new session", "session_id", newSession.ID, "project_key", newSession.ProjectKey)
			}
		}

		// Suspend session on disconnect (deferred)
		defer func() {
			session, err := h.sessionStorage.GetByID(context.Background(), req.SessionId)
			if err != nil {
				slog.Error("failed to get session for suspend", "error", err)
				return
			}
			if session != nil {
				session.Suspend()
				if err := h.sessionStorage.Update(context.Background(), session); err != nil {
					slog.Error("failed to suspend session", "error", err)
				} else {
					slog.Info("session suspended", "session_id", session.ID)
				}
			}
		}()
	}

	// 1. Create EventBus
	eventBus := orchestrator.NewSessionEventBus(64)
	defer eventBus.Close()

	// 2. Connect EventBus to AgentPool
	if pool, ok := h.agentPoolProxy.(interface{ SetEventBus(*orchestrator.SessionEventBus) }); ok {
		pool.SetEventBus(eventBus)
	}

	// 2b. Wire message sink so reconnecting clients can forward messages to this flow's EventBus
	h.flowRegistry.SetMessageSink(req.SessionId, &eventBusSink{
		sessionID: req.SessionId,
		router:    h.agentPoolAdapter,
		eventBus:  eventBus,
	})

	// 3. Background goroutine: receive from client stream
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				slog.InfoContext(ctx, "client stream ended", "error", err)
				eventBus.Close()
				cancel()
				return
			}
			if msg == nil {
				continue
			}
			if msg.Cancel {
				slog.InfoContext(ctx, "received cancel request from client", "session_id", req.SessionId)
				eventBus.Close()
				cancel()
				return
			}
			if msg.Ping != nil {
				slog.DebugContext(ctx, "received ping from client")
			}
			if msg.ToolResult != nil {
				handled := proxy.HandleToolResult(msg.ToolResult)
				if !handled && eventBus != nil {
					proxy.ClearWaitingForUser()
					_ = eventBus.Publish(orchestrator.OrchestratorEvent{
						Type:    orchestrator.EventUserResponded,
						Content: msg.ToolResult.Result,
					})
				}
			}
			if msg.Task != "" {
				slog.InfoContext(ctx, "received new task message", "task", msg.Task)
				routeUserMessage(req.SessionId, msg.Task, h.agentPoolAdapter, eventBus)
				touchSessionActivity(ctx, h.sessionStorage, req.SessionId)
			}
		}
	}()

	// 4. Create WorkChecker
	var workChecker orchestrator.ActiveWorkChecker
	if h.workManager != nil {
		workChecker = &workCheckerAdapter{manager: h.workManager, sessionID: req.SessionId, proxy: proxy}
	}

	// 5. Create chunk + event callbacks
	chunkCallback := createChunkCallback(ctx, streamWriter, req.SessionId)
	eventCallback := h.createEventCallback(ctx, agentEventStream, req.SessionId, workChecker)

	// 6. Create TurnExecutor (Engine-based, always available)
	turnExecutor := h.turnExecutorFactory.CreateForSession(proxy, req.SessionId, req.ProjectKey, projectRoot, platform)
	slog.InfoContext(ctx, "[Supervisor] using Engine-based TurnExecutor", "session_id", req.SessionId)

	// 7. Register active flow (cancel func stored in registry, not in domain entity)
	activeFlow, err := h.registerActiveFlow(req.SessionId, req.ProjectKey, req.UserId, req.Task, projectRoot, platform, cancel)
	if err != nil {
		slog.ErrorContext(ctx, "failed to register active flow", "error", err)
		return err
	}
	defer h.cleanupFlowResources(req.SessionId, activeFlow)

	// 8. Publish initial user message
	if req.Task != "" {
		_ = eventBus.Publish(orchestrator.OrchestratorEvent{
			Type:    orchestrator.EventUserMessage,
			Content: req.Task,
		})
	}

	// 9. Create and run Orchestrator (blocks until session ends)
	orch := orchestrator.New(orchestrator.Config{
		SessionID:        req.SessionId,
		ProjectKey:       req.ProjectKey,
		EventBus:         eventBus,
		TurnExecutor:     turnExecutor,
		WorkChecker:      workChecker,
		ChunkCallback:    chunkCallback,
		EventCallback:    eventCallback,
		ReminderInterval: 30 * time.Second,
	})

	orchErr := orch.Run(ctx)

	if orchErr != nil && ctx.Err() == nil {
		activeFlow.MarkFailed()
		slog.ErrorContext(ctx, "[Supervisor] orchestrator failed", "error", orchErr)
		sendErrorResponse(streamWriter, req.SessionId, orchErr)
	} else {
		activeFlow.MarkComplete()
		// Send final signal so the client knows the session is truly done
		if ctx.Err() == nil {
			_ = streamWriter.Send(&pb.FlowResponse{
				SessionId: req.SessionId,
				Type:      pb.ResponseType_RESPONSE_TYPE_ANSWER,
				IsFinal:   true,
				AgentId:   "supervisor",
			})
		}
	}

	slog.InfoContext(ctx, "[Supervisor] session ended", "session_id", req.SessionId)
	return nil
}

// workCheckerAdapter adapts WorkManagerForOrchestrator to orchestrator.ActiveWorkChecker.
type workCheckerAdapter struct {
	manager   WorkManagerForOrchestrator
	sessionID string
	proxy     *StreamBasedClientOperationsProxy
}

func (a *workCheckerAdapter) HasActiveWork(ctx context.Context) bool {
	tasks, err := a.manager.GetTasks(ctx, a.sessionID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check active work", "error", err)
		return true // fail-safe
	}
	for _, t := range tasks {
		if !t.IsTerminal() {
			return true
		}
	}
	return false
}

func (a *workCheckerAdapter) IsWaitingForUser(_ context.Context) bool {
	if a.proxy == nil {
		return false
	}
	return a.proxy.IsWaitingForUser()
}

func (a *workCheckerAdapter) ActiveWorkSummary(ctx context.Context) string {
	tasks, err := a.manager.GetTasks(ctx, a.sessionID)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	var parts []string
	for _, t := range tasks {
		if t != nil && !t.IsTerminal() {
			parts = append(parts, fmt.Sprintf("[%s] %q (%s)", t.ID, t.Title, t.Status))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// shouldSuppressIsFinal checks if IsFinal should be suppressed for an answer event.
// Returns true when there's active work (tasks/agents) or pending user interaction.
// In supervisor mode, suppress IsComplete on turn answers ONLY when there is
// still active work (tasks in progress, agents running) or pending user response.
// This prevents premature spinner-off while the Orchestrator has more turns to run.
// When no active work → allow IsFinal through so the client knows processing is done.
func shouldSuppressIsFinal(event *domain.AgentEvent, workChecker orchestrator.ActiveWorkChecker, ctx context.Context) bool {
	if event.Type != domain.EventTypeAnswer || !event.IsComplete {
		return false
	}
	if workChecker == nil {
		return false
	}
	return workChecker.HasActiveWork(ctx) || workChecker.IsWaitingForUser(ctx)
}
