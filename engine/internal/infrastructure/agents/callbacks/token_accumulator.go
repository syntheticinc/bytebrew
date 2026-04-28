package callbacks

import (
	"sync"

	"github.com/cloudwego/eino/components/model"
)

// TokenAccumulator accumulates token usage across multiple model calls within a turn.
// All methods are thread-safe.
type TokenAccumulator struct {
	promptTokens     int
	completionTokens int
	totalTokens      int
	mu               sync.Mutex
}

// NewTokenAccumulator creates a new TokenAccumulator.
func NewTokenAccumulator() *TokenAccumulator {
	return &TokenAccumulator{}
}

// Add adds token usage from a single model call.
func (a *TokenAccumulator) Add(usage *model.TokenUsage) {
	if usage == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.promptTokens += usage.PromptTokens
	a.completionTokens += usage.CompletionTokens
	a.totalTokens += usage.TotalTokens
}

// TotalTokens returns the accumulated total token count.
func (a *TokenAccumulator) TotalTokens() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.totalTokens
}

// PromptTokens returns the accumulated prompt token count.
func (a *TokenAccumulator) PromptTokens() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.promptTokens
}

// CompletionTokens returns the accumulated completion token count.
func (a *TokenAccumulator) CompletionTokens() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.completionTokens
}
