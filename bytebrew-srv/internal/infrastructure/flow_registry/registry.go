package flow_registry

import (
	"fmt"
	"sync"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// ActiveFlowRegistry отслеживает активные flow сессии
type ActiveFlowRegistry interface {
	// Register регистрирует активный flow
	Register(sessionID string, flow *domain.ActiveFlow) error

	// Unregister удаляет flow из реестра
	Unregister(sessionID string) error

	// Get возвращает активный flow по session_id
	Get(sessionID string) (*domain.ActiveFlow, bool)

	// IsActive проверяет, есть ли активный flow для сессии
	IsActive(sessionID string) bool

	// Subscribe подписывает клиента на события flow
	Subscribe(sessionID string, subscriber FlowSubscriber) error

	// Unsubscribe отписывает клиента от событий flow
	Unsubscribe(sessionID string, subscriberID string) error

	// BroadcastEvent отправляет событие всем подписчикам
	BroadcastEvent(sessionID string, event *domain.AgentEvent) error
}

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

// InMemoryRegistry in-memory реализация ActiveFlowRegistry
type InMemoryRegistry struct {
	mu    sync.RWMutex
	flows map[string]*domain.ActiveFlow
	subs  map[string]map[string]FlowSubscriber // sessionID -> subscriberID -> subscriber
}

// NewInMemoryRegistry создает новый InMemoryRegistry
func NewInMemoryRegistry() *InMemoryRegistry {
	return &InMemoryRegistry{
		flows: make(map[string]*domain.ActiveFlow),
		subs:  make(map[string]map[string]FlowSubscriber),
	}
}

// Register регистрирует активный flow
func (r *InMemoryRegistry) Register(sessionID string, flow *domain.ActiveFlow) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.flows[sessionID]; exists {
		return fmt.Errorf("flow already exists for session: %s", sessionID)
	}

	r.flows[sessionID] = flow
	r.subs[sessionID] = make(map[string]FlowSubscriber)

	return nil
}

// Unregister удаляет flow из реестра
func (r *InMemoryRegistry) Unregister(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.flows[sessionID]; !exists {
		return fmt.Errorf("flow not found for session: %s", sessionID)
	}

	delete(r.flows, sessionID)
	delete(r.subs, sessionID)

	return nil
}

// Get возвращает активный flow по session_id
func (r *InMemoryRegistry) Get(sessionID string) (*domain.ActiveFlow, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	flow, exists := r.flows[sessionID]
	return flow, exists
}

// IsActive проверяет, есть ли активный flow для сессии
func (r *InMemoryRegistry) IsActive(sessionID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	flow, exists := r.flows[sessionID]
	if !exists {
		return false
	}

	return flow.IsRunning()
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
