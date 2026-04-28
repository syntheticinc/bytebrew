package callbacks

import (
	"context"
	"sync"
)

// stepCallback is optionally set at process startup (server.go) and invoked
// after every step increment. Used by plugins for usage observation. Global
// because the callbacks package is deep in the agent infrastructure and
// plumbing a callback through 4 layers of constructors would be
// disproportionate for a single observer hook.
var (
	stepCallback   func(ctx context.Context) error
	stepCallbackMu sync.RWMutex
)

// SetStepCallback installs cb as the post-IncrementStep callback. Passing nil
// removes it. Safe to call at any time; idempotent.
func SetStepCallback(cb func(ctx context.Context) error) {
	stepCallbackMu.Lock()
	stepCallback = cb
	stepCallbackMu.Unlock()
}

// fireStepCallback invokes the global callback if set. Called from IncrementStep.
func fireStepCallback(ctx context.Context) error {
	stepCallbackMu.RLock()
	cb := stepCallback
	stepCallbackMu.RUnlock()
	if cb != nil {
		return cb(ctx)
	}
	return nil
}

// StepCounter tracks step number, model call count, and pending assistant content.
// All methods are thread-safe.
type StepCounter struct {
	step                    int
	modelCallCount          int
	pendingAssistantContent string
	mu                      sync.Mutex
}

// NewStepCounter creates a new StepCounter starting at step 0.
func NewStepCounter() *StepCounter {
	return &StepCounter{}
}

// GetStep returns the current step (thread-safe).
func (c *StepCounter) GetStep() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.step
}

// IncrementStep increments the step counter and fires the global step
// callback (if set) for observability/quota enforcement. Returns the callback
// error so callers can cancel the request context. Thread-safe.
func (c *StepCounter) IncrementStep(ctx context.Context) error {
	c.mu.Lock()
	c.step++
	c.mu.Unlock()
	return fireStepCallback(ctx)
}

// GetModelCallCount returns how many times the model has been called (thread-safe).
func (c *StepCounter) GetModelCallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.modelCallCount
}

// IncrementModelCallCount increments the model call counter (thread-safe).
func (c *StepCounter) IncrementModelCallCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.modelCallCount++
}

// SetPendingAssistantContent stores assistant content for the next onToolStart call (thread-safe).
func (c *StepCounter) SetPendingAssistantContent(content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pendingAssistantContent = content
}

// ConsumePendingAssistantContent returns and clears the pending assistant content (thread-safe).
func (c *StepCounter) ConsumePendingAssistantContent() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	content := c.pendingAssistantContent
	c.pendingAssistantContent = ""
	return content
}
