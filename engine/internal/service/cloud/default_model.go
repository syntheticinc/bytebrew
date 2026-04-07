package cloud

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// ModelProxy proxies requests to the actual LLM API.
type ModelProxy interface {
	Chat(ctx context.Context, messages []map[string]string) (string, error)
}

// DefaultModelConfig holds configuration for the default model service.
type DefaultModelConfig struct {
	ModelName       string // "GLM-4.7"
	MaxReqsPerMonth int    // 100 per tenant (AC-PRICE-06)
}

// DefaultDefaultModelConfig returns the default configuration.
func DefaultDefaultModelConfig() DefaultModelConfig {
	return DefaultModelConfig{
		ModelName:       "GLM-4.7",
		MaxReqsPerMonth: 100,
	}
}

// tenantCounter tracks per-tenant monthly request counts.
type tenantCounter struct {
	count     int
	resetAt   time.Time // start of current month
}

// DefaultModelService provides the default model (GLM 4.7) with rate limiting (AC-PRICE-06).
type DefaultModelService struct {
	mu       sync.Mutex
	config   DefaultModelConfig
	proxy    ModelProxy
	counters map[string]*tenantCounter // tenant_id → counter
}

// NewDefaultModelService creates a new default model service.
func NewDefaultModelService(config DefaultModelConfig, proxy ModelProxy) *DefaultModelService {
	return &DefaultModelService{
		config:   config,
		proxy:    proxy,
		counters: make(map[string]*tenantCounter),
	}
}

// Chat sends a chat request using the default model.
// Returns an error if the tenant has exceeded the monthly limit.
func (s *DefaultModelService) Chat(ctx context.Context, tenantID string, messages []map[string]string) (string, error) {
	if err := s.checkLimit(tenantID); err != nil {
		return "", err
	}

	result, err := s.proxy.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("default model chat: %w", err)
	}

	s.incrementCounter(tenantID)
	slog.DebugContext(ctx, "[DefaultModel] request completed",
		"tenant", tenantID, "model", s.config.ModelName)

	return result, nil
}

// RemainingRequests returns the number of remaining default model requests for a tenant.
func (s *DefaultModelService) RemainingRequests(tenantID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	counter := s.getOrCreateCounter(tenantID)
	remaining := s.config.MaxReqsPerMonth - counter.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// ModelName returns the default model name.
func (s *DefaultModelService) ModelName() string {
	return s.config.ModelName
}

func (s *DefaultModelService) checkLimit(tenantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	counter := s.getOrCreateCounter(tenantID)
	if counter.count >= s.config.MaxReqsPerMonth {
		return fmt.Errorf("default model limit reached (%d/%d requests this month). Add your own API key to continue",
			counter.count, s.config.MaxReqsPerMonth)
	}
	return nil
}

func (s *DefaultModelService) incrementCounter(tenantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	counter := s.getOrCreateCounter(tenantID)
	counter.count++
}

func (s *DefaultModelService) getOrCreateCounter(tenantID string) *tenantCounter {
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	counter, ok := s.counters[tenantID]
	if !ok || counter.resetAt.Before(monthStart) {
		counter = &tenantCounter{count: 0, resetAt: monthStart}
		s.counters[tenantID] = counter
	}
	return counter
}
