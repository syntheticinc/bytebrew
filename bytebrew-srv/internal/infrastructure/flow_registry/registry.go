package flow_registry

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// FlowSubscriber подписчик на события flow
type FlowSubscriber interface {
	// ID возвращает уникальный ID подписчика
	ID() string

	// OnEvent вызывается при получении события
	OnEvent(event *domain.AgentEvent) error

	// OnComplete вызывается при завершении flow
	OnComplete() error

	// OnError вызывается при ошибке
	OnError(err error) error
}

// flowEntry holds a flow and its associated cancel function.
// The cancel func is stored here (not in domain) to keep ActiveFlow pure.
type flowEntry struct {
	flow   *domain.ActiveFlow
	cancel context.CancelFunc
}

// InMemoryRegistry in-memory реализация flow registry
type InMemoryRegistry struct {
	mu    sync.RWMutex
	flows map[string]*flowEntry
	subs  map[string]map[string]FlowSubscriber // sessionID -> subscriberID -> subscriber
}

// NewInMemoryRegistry создает новый InMemoryRegistry
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		flows: make(map[string]*flowEntry),
		subs:  make(map[string]map[string]FlowSubscriber),
	}
}

// Register регистрирует активный flow с его cancel function.
// If a flow already exists for this session, its cancel func is called and the flow is replaced.
// cancel may be nil if cancellation is not needed.
func (r *InMemoryRegistry) Register(sessionID string, flow *domain.ActiveFlow, cancel context.CancelFunc) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, exists := r.flows[sessionID]; exists {
		if existing.cancel != nil {
			existing.cancel()
		}
		slog.Info("replacing existing flow", "session_id", sessionID)
	}

	r.flows[sessionID] = &flowEntry{flow: flow, cancel: cancel}
	if _, exists := r.subs[sessionID]; !exists {
		r.subs[sessionID] = make(map[string]FlowSubscriber)
	}

	return nil
}

// Unregister удаляет flow из реестра (idempotent — no error if not found)
func (r *InMemoryRegistry) Unregister(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.flows, sessionID)
	delete(r.subs, sessionID)

	return nil
}

// UnregisterIfCurrent atomically unregisters the flow only if the currently
// registered flow matches the expected one (pointer equality).
// This prevents a stale defer from removing a replacement flow.
func (r *InMemoryRegistry) UnregisterIfCurrent(sessionID string, expected *domain.ActiveFlow) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.flows[sessionID]
	if !exists {
		return false
	}
	if entry.flow != expected {
		return false
	}

	delete(r.flows, sessionID)
	delete(r.subs, sessionID)
	return true
}

// Get возвращает активный flow по session_id
func (r *InMemoryRegistry) Get(sessionID string) (*domain.ActiveFlow, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.flows[sessionID]
	if !exists {
		return nil, false
	}
	return entry.flow, true
}

// IsActive проверяет, есть ли активный flow для сессии
func (r *InMemoryRegistry) IsActive(sessionID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.flows[sessionID]
	if !exists {
		return false
	}

	return entry.flow.IsRunning()
}

// Subscribe подписывает клиента на события flow
func (r *InMemoryRegistry) Subscribe(sessionID string, subscriber FlowSubscriber) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.flows[sessionID]; !exists {
		return fmt.Errorf("flow not found for session: %s", sessionID)
	}

	r.subs[sessionID][subscriber.ID()] = subscriber
	return nil
}

// Unsubscribe отписывает клиента от событий flow
func (r *InMemoryRegistry) Unsubscribe(sessionID string, subscriberID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.subs[sessionID]; !exists {
		return fmt.Errorf("subscribers not found for session: %s", sessionID)
	}

	delete(r.subs[sessionID], subscriberID)
	return nil
}

// ListActiveFlows returns all currently registered flows
func (r *InMemoryRegistry) ListActiveFlows() []*domain.ActiveFlow {
	r.mu.RLock()
	defer r.mu.RUnlock()

	flows := make([]*domain.ActiveFlow, 0, len(r.flows))
	for _, entry := range r.flows {
		flows = append(flows, entry.flow)
	}
	return flows
}

// CancelFlow cancels the flow for the given session ID
func (r *InMemoryRegistry) CancelFlow(sessionID string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.flows[sessionID]
	if !exists {
		return fmt.Errorf("flow not found for session: %s", sessionID)
	}

	if entry.cancel != nil {
		entry.cancel()
	}
	return nil
}

// BroadcastEvent отправляет событие всем подписчикам
func (r *InMemoryRegistry) BroadcastEvent(sessionID string, event *domain.AgentEvent) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, exists := r.flows[sessionID]; !exists {
		return fmt.Errorf("flow not found for session: %s", sessionID)
	}

	for _, subscriber := range r.subs[sessionID] {
		if err := subscriber.OnEvent(event); err != nil {
			return fmt.Errorf("subscriber %s failed to handle event: %w", subscriber.ID(), err)
		}
	}

	return nil
}
