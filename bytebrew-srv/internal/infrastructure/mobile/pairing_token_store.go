package mobile

import (
	"context"
	"log/slog"
	"sync"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// InMemoryPairingTokenStore is a thread-safe in-memory implementation of pair_device.PairingTokenStore.
type InMemoryPairingTokenStore struct {
	mu          sync.RWMutex
	tokens      map[string]*domain.PairingToken // full token -> PairingToken
	byShortCode map[string]string               // short code -> full token (for lookup by code)
}

// NewInMemoryPairingTokenStore creates a new InMemoryPairingTokenStore.
func NewInMemoryPairingTokenStore() *InMemoryPairingTokenStore {
	return &InMemoryPairingTokenStore{
		tokens:      make(map[string]*domain.PairingToken),
		byShortCode: make(map[string]string),
	}
}

// SaveToken stores a pairing token indexed by both full token and short code.
func (s *InMemoryPairingTokenStore) SaveToken(_ context.Context, token *domain.PairingToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token.Token] = token

	if token.ShortCode != "" {
		s.byShortCode[token.ShortCode] = token.Token
	}

	slog.Debug("pairing token saved", "short_code", token.ShortCode)
	return nil
}

// GetToken looks up a pairing token by full token first, then by short code.
func (s *InMemoryPairingTokenStore) GetToken(_ context.Context, tokenOrCode string) (*domain.PairingToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.findTokenLocked(tokenOrCode), nil
}

// UseToken atomically finds, validates, and marks a pairing token as used.
// The entire operation runs under an exclusive lock to prevent race conditions.
func (s *InMemoryPairingTokenStore) UseToken(_ context.Context, tokenOrCode string) (*domain.PairingToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.findTokenLocked(tokenOrCode)
	if t == nil {
		return nil, errors.New(errors.CodeNotFound, "pairing token not found")
	}

	if !t.IsValid() {
		return nil, errors.New(errors.CodeInvalidInput, "pairing token is expired or already used")
	}

	t.MarkUsed()

	slog.Debug("pairing token used atomically", "short_code", t.ShortCode)
	return t, nil
}

// findTokenLocked looks up a token by full token or short code.
// Caller MUST hold the lock.
func (s *InMemoryPairingTokenStore) findTokenLocked(tokenOrCode string) *domain.PairingToken {
	if t, ok := s.tokens[tokenOrCode]; ok {
		return t
	}

	if fullToken, ok := s.byShortCode[tokenOrCode]; ok {
		if t, ok := s.tokens[fullToken]; ok {
			return t
		}
	}

	return nil
}

// DeleteToken deletes a pairing token by its full token value.
func (s *InMemoryPairingTokenStore) DeleteToken(_ context.Context, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.tokens[token]; ok {
		if t.ShortCode != "" {
			delete(s.byShortCode, t.ShortCode)
		}
		delete(s.tokens, token)
		prefix := token
		if len(prefix) > 8 {
			prefix = prefix[:8]
		}
		slog.Debug("pairing token deleted", "token_prefix", prefix)
	}

	return nil
}
