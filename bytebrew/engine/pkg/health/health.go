package health

import (
	"context"
	"sync"
	"time"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// Check represents a health check result
type Check struct {
	Name      string                 `json:"name"`
	Status    Status                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Checker is an interface for health checks
type Checker interface {
	Check(ctx context.Context) Check
	Name() string
}

// Manager manages health checks
type Manager struct {
	mu       sync.RWMutex
	checkers map[string]Checker
}

// NewManager creates a new health check manager
func NewManager() *Manager {
	return &Manager{
		checkers: make(map[string]Checker),
	}
}

// Register registers a health checker
func (m *Manager) Register(checker Checker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.checkers[checker.Name()] = checker
}

// Unregister removes a health checker
func (m *Manager) Unregister(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.checkers, name)
}

// CheckAll runs all registered health checks
func (m *Manager) CheckAll(ctx context.Context) []Check {
	m.mu.RLock()
	defer m.mu.RUnlock()

	checks := make([]Check, 0, len(m.checkers))
	for _, checker := range m.checkers {
		checks = append(checks, checker.Check(ctx))
	}
	return checks
}

// IsHealthy returns true if all checks are healthy
func (m *Manager) IsHealthy(ctx context.Context) bool {
	checks := m.CheckAll(ctx)
	for _, check := range checks {
		if check.Status != StatusHealthy {
			return false
		}
	}
	return true
}

// DatabaseChecker checks database connectivity
type DatabaseChecker struct {
	name   string
	pingFn func(ctx context.Context) error
}

// NewDatabaseChecker creates a new database health checker
func NewDatabaseChecker(name string, pingFn func(ctx context.Context) error) *DatabaseChecker {
	return &DatabaseChecker{
		name:   name,
		pingFn: pingFn,
	}
}

// Name returns the checker name
func (c *DatabaseChecker) Name() string {
	return c.name
}

// Check performs the health check
func (c *DatabaseChecker) Check(ctx context.Context) Check {
	check := Check{
		Name:      c.name,
		Timestamp: time.Now(),
	}

	if err := c.pingFn(ctx); err != nil {
		check.Status = StatusUnhealthy
		check.Message = err.Error()
	} else {
		check.Status = StatusHealthy
		check.Message = "database connection is healthy"
	}

	return check
}

// LLMChecker checks LLM provider connectivity
type LLMChecker struct {
	name   string
	pingFn func(ctx context.Context) error
}

// NewLLMChecker creates a new LLM health checker
func NewLLMChecker(name string, pingFn func(ctx context.Context) error) *LLMChecker {
	return &LLMChecker{
		name:   name,
		pingFn: pingFn,
	}
}

// Name returns the checker name
func (c *LLMChecker) Name() string {
	return c.name
}

// Check performs the health check
func (c *LLMChecker) Check(ctx context.Context) Check {
	check := Check{
		Name:      c.name,
		Timestamp: time.Now(),
	}

	if err := c.pingFn(ctx); err != nil {
		check.Status = StatusDegraded
		check.Message = err.Error()
	} else {
		check.Status = StatusHealthy
		check.Message = "LLM provider is healthy"
	}

	return check
}
