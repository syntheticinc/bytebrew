package session_processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/engine/internal/service/orchestrator"
)

// SessionRegistry provides session context and message channel (consumer-side interface).
type SessionRegistry interface {
	GetSessionContext(sessionID string) (projectRoot, platform, projectKey, userID, agentName string, ok bool)
	MessageChannel(sessionID string) <-chan string
	PublishEvent(sessionID string, event *pb.SessionEvent)
	ResetCancel(sessionID string)
	StoreTurnCancel(sessionID string, cancel context.CancelFunc)
	HasSession(sessionID string) bool
	RegisterAskUser(sessionID, callID string) <-chan string
	UnregisterAskUser(sessionID, callID string)
}

// TurnExecutorFactory creates a TurnExecutor for a given session (consumer-side interface).
type TurnExecutorFactory interface {
	CreateForSession(proxy tools.ClientOperationsProxy, sessionID, projectKey, projectRoot, platform, agentName string) orchestrator.TurnExecutor
}

// AgentPoolRegistrar registers per-session resources on the AgentPool (consumer-side interface).
// Used to deliver lifecycle events and provide proxy for code agent tool execution.
type AgentPoolRegistrar interface {
	SetEventCallbackForSession(sessionID string, cb func(event *domain.AgentEvent) error)
	SetProxyForSession(sessionID string, proxy interface{})
	RemoveSession(sessionID string)
}

// Processor runs background message-processing loops for server-streaming sessions.
// It is shared between gRPC SubscribeSession and bridge MobileRequestHandler.
type Processor struct {
	registry           SessionRegistry
	factory            TurnExecutorFactory
	agentPoolRegistrar AgentPoolRegistrar // optional, nil-safe
	eventStore         EventStore         // persists events for reliable replay

	mu          sync.Mutex
	active      map[string]context.CancelFunc
	turnsActive map[string]bool // sessions with an actively executing turn
}

// New creates a new Processor.
func New(registry SessionRegistry, factory TurnExecutorFactory, eventStore EventStore) *Processor {
	return &Processor{
		registry:    registry,
		factory:     factory,
		eventStore:  eventStore,
		active:      make(map[string]context.CancelFunc),
		turnsActive: make(map[string]bool),
	}
}

// SetAgentPoolRegistrar sets the registrar for agent pool resources.
// When set, processMessage will register event callbacks and proxy on the AgentPool
// so that lifecycle events reach WS/mobile clients and code agents can execute tools.
func (p *Processor) SetAgentPoolRegistrar(registrar AgentPoolRegistrar) {
	p.agentPoolRegistrar = registrar
}

// StartProcessing launches the message processing loop for a session.
// Idempotent: if already running for this session, does nothing.
func (p *Processor) StartProcessing(ctx context.Context, sessionID string) {
	p.mu.Lock()
	if _, exists := p.active[sessionID]; exists {
		p.mu.Unlock()
		return
	}

	// Use context.Background() — processing must NOT be tied to the HTTP
	// request context. The HTTP handler may return (e.g., after SSE flush)
	// while the LLM is still generating. If we used ctx here, the request
	// cancellation would kill the LLM turn ("turn cancelled by user").
	// Values from the original context (RequestContext for MCP headers) are
	// copied via context.WithoutCancel if available, otherwise Background.
	procCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	p.active[sessionID] = cancel
	p.mu.Unlock()

	go p.processMessages(procCtx, sessionID)
}

// StopProcessing stops the message processing loop for a session.
func (p *Processor) StopProcessing(sessionID string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if cancel, exists := p.active[sessionID]; exists {
		cancel()
		delete(p.active, sessionID)
	}
}

// IsProcessing returns true if a processing loop is active for the session.
func (p *Processor) IsProcessing(sessionID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, exists := p.active[sessionID]
	return exists
}

// IsTurnActive returns true if a turn (message processing) is currently executing.
// Unlike IsProcessing which tracks the background loop, this tracks the actual
// turn execution between ProcessingStarted and ProcessingStopped.
func (p *Processor) IsTurnActive(sessionID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.turnsActive[sessionID]
}

func (p *Processor) processMessages(ctx context.Context, sessionID string) {
	defer func() {
		p.mu.Lock()
		delete(p.active, sessionID)
		p.mu.Unlock()
	}()

	msgCh := p.registry.MessageChannel(sessionID)

	for {
		select {
		case <-ctx.Done():
			return
		case message, ok := <-msgCh:
			if !ok {
				return
			}
			p.processMessage(ctx, sessionID, message)
		}
	}
}

func (p *Processor) processMessage(ctx context.Context, sessionID, message string) {
	projectRoot, platform, projectKey, _, agentName, ok := p.registry.GetSessionContext(sessionID)
	if !ok {
		slog.ErrorContext(ctx, "[SessionProcessor] session context not found", "session_id", sessionID)
		return
	}

	p.mu.Lock()
	p.turnsActive[sessionID] = true
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.turnsActive, sessionID)
		p.mu.Unlock()
	}()

	eventStream := NewEventStream(sessionID, p.registry, p.eventStore)

	// Broadcast user message so it appears in backfill history on reconnect.
	eventStream.PublishUserMessage(message)

	eventStream.PublishProcessingStarted()

	// Create proxy with blocking ask_user handler: publishes event to client,
	// registers reply channel, and blocks until client sends ask_user_reply.
	askUserHandler := func(ctx context.Context, sid, questionsJSON string) (string, error) {
		callID := fmt.Sprintf("ask-%d", time.Now().UnixNano())
		replyCh := p.registry.RegisterAskUser(sid, callID)
		defer p.registry.UnregisterAskUser(sid, callID)

		// Try to extract tool_name from envelope format {"questions": [...], "tool_name": "..."}
		// sent by AskUserTool when confirm_before is active.
		toolName := ""
		content := questionsJSON
		var envelope struct {
			Questions json.RawMessage `json:"questions"`
			ToolName  string          `json:"tool_name"`
		}
		if json.Unmarshal([]byte(questionsJSON), &envelope) == nil && len(envelope.Questions) > 0 {
			toolName = envelope.ToolName
			content = string(envelope.Questions)
		}

		// Publish AskUserRequested event so the client sees the questions
		eventStream.Send(&domain.AgentEvent{
			Type:    domain.EventTypeUserQuestion,
			Content: content,
			Metadata: map[string]interface{}{
				"call_id":   callID,
				"tool_name": toolName,
			},
		})

		// Dedicated timeout prevents indefinite hang when client never responds
		// (e.g., wrong call_id, client disconnect). This is BUG-001 defensive fix.
		askTimeout := 60 * time.Second
		select {
		case reply := <-replyCh:
			return reply, nil
		case <-time.After(askTimeout):
			slog.WarnContext(ctx, "[askUserHandler] timed out waiting for user response",
				"session_id", sid, "call_id", callID, "timeout", askTimeout)
			return "[TIMEOUT] User did not respond within 60 seconds. Inform the user that the operation was not completed due to timeout.", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	proxy := tools.NewLocalClientOperationsProxy(projectRoot, tools.WithAskUserHandler(askUserHandler))
	defer proxy.Dispose()

	turnExecutor := p.factory.CreateForSession(proxy, sessionID, projectKey, projectRoot, platform, agentName)
	if turnExecutor == nil {
		slog.ErrorContext(ctx, "[SessionProcessor] failed to create turn executor — check model configuration in Admin Dashboard",
			"session_id", sessionID, "agent", agentName)
		eventStream.PublishError(fmt.Errorf("no model available for agent %q — configure a model via Admin Dashboard", agentName))
		eventStream.PublishProcessingStopped()
		return
	}

	chunkCallback := func(chunk string) error {
		eventStream.PublishAnswerChunk(chunk)
		return nil
	}

	eventCallback := func(event *domain.AgentEvent) error {
		return eventStream.Send(event)
	}

	// Register proxy and lifecycle callback on AgentPool so code agents can
	// execute tools and lifecycle events reach WS/mobile clients.
	if p.agentPoolRegistrar != nil {
		p.agentPoolRegistrar.SetProxyForSession(sessionID, proxy)
		p.agentPoolRegistrar.SetEventCallbackForSession(sessionID, eventCallback)
		defer p.agentPoolRegistrar.RemoveSession(sessionID)
	}

	turnCtx, turnCancel := context.WithCancel(ctx)
	defer turnCancel()

	p.registry.StoreTurnCancel(sessionID, turnCancel)
	defer p.registry.StoreTurnCancel(sessionID, nil)

	err := turnExecutor.ExecuteTurn(turnCtx, sessionID, projectKey, message, chunkCallback, eventCallback)

	p.registry.ResetCancel(sessionID)

	if err != nil {
		if turnCtx.Err() != nil {
			slog.InfoContext(ctx, "[SessionProcessor] turn cancelled by user", "session_id", sessionID)
		} else {
			slog.ErrorContext(ctx, "[SessionProcessor] turn execution failed", "session_id", sessionID, "error", err)
			eventStream.PublishError(err)
		}
	}

	eventStream.PublishProcessingStopped()
}
