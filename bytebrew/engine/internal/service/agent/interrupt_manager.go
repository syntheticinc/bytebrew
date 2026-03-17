package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// InterruptManager manages interrupt contexts for blocking spawns.
// Each session can have an interrupt context that is cancelled when a user
// sends a message while agents are running. Only one goroutine per session
// can claim the "interrupt responder" role.
type InterruptManager struct {
	mu              sync.Mutex
	interruptCtx    map[string]context.Context
	interruptFn     map[string]context.CancelCauseFunc
	interruptClaimed map[string]bool
}

// NewInterruptManager creates a new InterruptManager.
func NewInterruptManager() *InterruptManager {
	return &InterruptManager{
		interruptCtx:     make(map[string]context.Context),
		interruptFn:      make(map[string]context.CancelCauseFunc),
		interruptClaimed: make(map[string]bool),
	}
}

// NotifyUserMessage broadcasts interrupt to ALL blocking spawns for session.
// All parallel WaitForAllSessionAgents calls wake simultaneously.
// Only the first to call ClaimInterruptResponder() gets the full message.
func (m *InterruptManager) NotifyUserMessage(sessionID, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	fn, ok := m.interruptFn[sessionID]
	if !ok {
		return
	}

	// Reset responder claim for this interrupt
	m.interruptClaimed[sessionID] = false
	// Cancel with cause containing user message
	fn(fmt.Errorf("user_message:%s", message))
	// Re-create for next wait cycle
	ctx, cancel := context.WithCancelCause(context.Background())
	m.interruptCtx[sessionID] = ctx
	m.interruptFn[sessionID] = cancel
}

// HasBlockingWait returns true if any blocking spawn is active for session.
func (m *InterruptManager) HasBlockingWait(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.interruptCtx[sessionID]
	return ok
}

// ClaimInterruptResponder atomically claims the interrupt responder role.
// Only the first caller for a given session returns true.
func (m *InterruptManager) ClaimInterruptResponder(sessionID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.interruptClaimed[sessionID] {
		m.interruptClaimed[sessionID] = true
		return true
	}
	return false
}

// GetOrCreateInterruptCtx gets or creates interrupt context for session.
func (m *InterruptManager) GetOrCreateInterruptCtx(sessionID string) context.Context {
	m.mu.Lock()
	defer m.mu.Unlock()
	if ctx, ok := m.interruptCtx[sessionID]; ok {
		return ctx
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	m.interruptCtx[sessionID] = ctx
	m.interruptFn[sessionID] = cancel
	m.interruptClaimed[sessionID] = false
	return ctx
}

// CleanupIfNoRunning removes interrupt context when no running agents exist.
// The hasRunning parameter should be determined by the caller (AgentPool)
// which holds the agents map.
func (m *InterruptManager) CleanupIfNoRunning(sessionID string, hasRunning bool) {
	if hasRunning {
		return
	}
	m.mu.Lock()
	delete(m.interruptCtx, sessionID)
	delete(m.interruptFn, sessionID)
	delete(m.interruptClaimed, sessionID)
	m.mu.Unlock()
}

// ExtractUserMessage extracts user message from interrupt error.
func ExtractUserMessage(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "user_message:") {
		return strings.TrimPrefix(msg, "user_message:")
	}
	return msg
}
