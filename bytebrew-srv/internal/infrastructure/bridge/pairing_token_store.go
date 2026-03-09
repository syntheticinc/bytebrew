package bridge

import (
	"sync"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
)

// PairingTokenStore holds active pairing tokens in memory.
// Tokens are ephemeral and do not survive server restarts.
type PairingTokenStore struct {
	tokens map[string]*domain.PairingToken // full token → PairingToken
	codes  map[string]string               // shortCode → full token
	mu     sync.Mutex
}

// NewPairingTokenStore creates an empty token store.
func NewPairingTokenStore() *PairingTokenStore {
	return &PairingTokenStore{
		tokens: make(map[string]*domain.PairingToken),
		codes:  make(map[string]string),
	}
}

// Add registers a new pairing token.
func (s *PairingTokenStore) Add(token *domain.PairingToken) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tokens[token.Token] = token
	if token.ShortCode != "" {
		s.codes[token.ShortCode] = token.Token
	}
}

// Get looks up a token by full token string or short code.
// Returns nil if not found.
func (s *PairingTokenStore) Get(tokenOrCode string) *domain.PairingToken {
	s.mu.Lock()
	defer s.mu.Unlock()

	if t, ok := s.tokens[tokenOrCode]; ok {
		return t
	}

	fullToken, ok := s.codes[tokenOrCode]
	if !ok {
		return nil
	}

	return s.tokens[fullToken]
}

// UseToken atomically finds, validates, and marks a token as used.
// Returns nil if the token is not found or not valid.
func (s *PairingTokenStore) UseToken(tokenOrCode string) *domain.PairingToken {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := s.resolve(tokenOrCode)
	if token == nil {
		return nil
	}
	if !token.IsValid() {
		return nil
	}

	token.MarkUsed()
	return token
}

// Remove deletes a token by its full token string.
func (s *PairingTokenStore) Remove(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tokens[token]
	if !ok {
		return
	}

	if t.ShortCode != "" {
		delete(s.codes, t.ShortCode)
	}
	delete(s.tokens, token)
}

// RemoveExpired removes all expired tokens and returns the count removed.
func (s *PairingTokenStore) RemoveExpired() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	var removed int
	for key, t := range s.tokens {
		if !t.IsExpired() {
			continue
		}
		if t.ShortCode != "" {
			delete(s.codes, t.ShortCode)
		}
		delete(s.tokens, key)
		removed++
	}

	return removed
}

// resolve finds a token by full token or short code (must be called under lock).
func (s *PairingTokenStore) resolve(tokenOrCode string) *domain.PairingToken {
	if t, ok := s.tokens[tokenOrCode]; ok {
		return t
	}

	fullToken, ok := s.codes[tokenOrCode]
	if !ok {
		return nil
	}

	return s.tokens[fullToken]
}
