package callbacks

import (
	"sync"
)

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

// IncrementStep increments the step counter (thread-safe).
func (c *StepCounter) IncrementStep() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.step++
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
