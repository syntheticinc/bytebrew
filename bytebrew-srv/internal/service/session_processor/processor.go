package session_processor

import (
	"context"
	"log/slog"
	"sync"

	pb "github.com/syntheticinc/bytebrew/bytebrew-srv/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/service/orchestrator"
)

// SessionRegistry provides session context and message channel (consumer-side interface).
type SessionRegistry interface {
	GetSessionContext(sessionID string) (projectRoot, platform, projectKey, userID string, ok bool)
	MessageChannel(sessionID string) <-chan string
	PublishEvent(sessionID string, event *pb.SessionEvent)
	ResetCancel(sessionID string)
	StoreTurnCancel(sessionID string, cancel context.CancelFunc)
	HasSession(sessionID string) bool
}

// TurnExecutorFactory creates a TurnExecutor for a given session (consumer-side interface).
type TurnExecutorFactory interface {
	CreateForSession(proxy tools.ClientOperationsProxy, sessionID, projectKey, projectRoot, platform string) orchestrator.TurnExecutor
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

	procCtx, cancel := context.WithCancel(ctx)
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
	projectRoot, platform, projectKey, _, ok := p.registry.GetSessionContext(sessionID)
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

	proxy := tools.NewLocalClientOperationsProxy(projectRoot)
	defer proxy.Dispose()

	turnExecutor := p.factory.CreateForSession(proxy, sessionID, projectKey, projectRoot, platform)

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
