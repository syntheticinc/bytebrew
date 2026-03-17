package shell

import (
	"sync"
)

const PoolSize = 3

// SessionManager maintains pools of persistent shell sessions keyed by
// projectRoot and agentID. Each pool holds up to PoolSize sessions.
type SessionManager struct {
	pools     map[string][]*ShellSession
	bgManager *BackgroundProcessManager
	mu        sync.Mutex
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		pools:     make(map[string][]*ShellSession),
		bgManager: NewBackgroundProcessManager(),
	}
}

// GetAvailableSession returns a non-busy session from the pool, creating one
// if the pool is not full. Returns nil if all PoolSize sessions are busy.
func (m *SessionManager) GetAvailableSession(projectRoot, agentID string) *ShellSession {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := poolKey(projectRoot, agentID)
	pool := m.pools[key]

	// Find first non-executing session
	for _, s := range pool {
		if !s.IsExecuting() {
			return s
		}
	}

	// Pool not full: create new session
	if len(pool) < PoolSize {
		session := NewShellSession(projectRoot, DefaultMaxSize)
		m.pools[key] = append(pool, session)
		return session
	}

	// All busy
	return nil
}

// BackgroundManager returns the background process manager.
func (m *SessionManager) BackgroundManager() *BackgroundProcessManager {
	return m.bgManager
}

// DisposeAll destroys all sessions and background processes.
func (m *SessionManager) DisposeAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, pool := range m.pools {
		for _, s := range pool {
			s.Destroy()
		}
		delete(m.pools, key)
	}

	m.bgManager.DisposeAll()
}

// poolKey constructs the map key from projectRoot and agentID.
func poolKey(projectRoot, agentID string) string {
	if agentID == "" {
		return projectRoot
	}
	return projectRoot + "::" + agentID
}
