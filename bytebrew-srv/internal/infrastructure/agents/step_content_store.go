package agents

import (
	"sync"
)

// StepContentStore stores accumulated content per step
// This is shared between callback handler and message modifier
// to recover content that gets lost in eino's streaming mode
type StepContentStore struct {
	content map[int]string
	mu      sync.RWMutex
}

// NewStepContentStore creates a new step content store
func NewStepContentStore() *StepContentStore {
	return &StepContentStore{
		content: make(map[int]string),
	}
}

// Append adds content to a specific step
func (s *StepContentStore) Append(step int, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.content[step] += content
}

// Get returns content for a specific step
func (s *StepContentStore) Get(step int) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.content[step]
}

// GetAll returns a copy of all step content
func (s *StepContentStore) GetAll() map[int]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[int]string, len(s.content))
	for k, v := range s.content {
		result[k] = v
	}
	return result
}

// Clear removes all stored content
func (s *StepContentStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.content = make(map[int]string)
}

// Count returns the number of stored steps
func (s *StepContentStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.content)
}
