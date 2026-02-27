package crypto

import (
	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/infrastructure/cache"
)

// JWTHasher produces cache keys from JWTs using SHA-256.
type JWTHasher struct{}

// NewJWTHasher creates a new JWTHasher.
func NewJWTHasher() *JWTHasher {
	return &JWTHasher{}
}

// Hash returns a short hash of the JWT for use as cache key.
func (h *JWTHasher) Hash(jwt string) string {
	return cache.HashJWT(jwt)
}
