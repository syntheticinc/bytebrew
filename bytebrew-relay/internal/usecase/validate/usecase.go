package validate

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/domain"
)

// CloudAPIClient validates licenses against the Cloud API.
type CloudAPIClient interface {
	ValidateLicense(ctx context.Context, licenseJWT string) (*CloudAPIResult, error)
}

// CloudAPIResult holds the result from Cloud API validation.
type CloudAPIResult struct {
	Tier         string
	SeatsAllowed int
	ExpiresAt    time.Time
}

// Cache provides cached license storage.
type Cache interface {
	Get(jwtHash string) (*domain.CachedLicense, bool)
	Set(jwtHash string, license *domain.CachedLicense)
	IsFresh(license *domain.CachedLicense) bool
	IsWithinGrace() bool
	UpdateSyncTime()
}

// JWTHasher produces cache keys from JWTs.
type JWTHasher interface {
	Hash(jwt string) string
}

// Result represents the outcome of a license validation.
type Result struct {
	Valid        bool
	Tier         string
	SeatsAllowed int
	Message      string
	FromCache    bool
}

// Usecase handles license validation through Cloud API with caching.
type Usecase struct {
	client    CloudAPIClient
	cache     Cache
	jwtHasher JWTHasher
	nowFunc   func() time.Time
}

// New creates a new validate usecase.
func New(client CloudAPIClient, cache Cache, jwtHasher JWTHasher) *Usecase {
	return &Usecase{
		client:    client,
		cache:     cache,
		jwtHasher: jwtHasher,
		nowFunc:   time.Now,
	}
}

// Execute validates a license JWT.
//
// Flow:
//  1. Check cache - if fresh, return cached result
//  2. Call Cloud API to validate
//  3. On success: cache result, return valid
//  4. On API failure + cache within grace: return cached + warning
//  5. On API failure + cache expired grace: return blocked
func (u *Usecase) Execute(ctx context.Context, licenseJWT string) (*Result, error) {
	if licenseJWT == "" {
		return &Result{Valid: false, Message: "license JWT is required"}, nil
	}

	hash := u.jwtHasher.Hash(licenseJWT)

	// Step 1: check cache
	if cached, ok := u.cache.Get(hash); ok && u.cache.IsFresh(cached) {
		slog.InfoContext(ctx, "license validated from cache", "tier", cached.Tier)
		return &Result{
			Valid:        true,
			Tier:         cached.Tier,
			SeatsAllowed: cached.SeatsAllowed,
			FromCache:    true,
		}, nil
	}

	// Step 2: call Cloud API
	apiResult, err := u.client.ValidateLicense(ctx, licenseJWT)
	if err != nil {
		slog.WarnContext(ctx, "cloud api validation failed", "error", err)
		return u.handleAPIFailure(ctx, hash, err)
	}

	// Step 3: success - cache and return
	cached := &domain.CachedLicense{
		JWT:          licenseJWT,
		Tier:         apiResult.Tier,
		SeatsAllowed: apiResult.SeatsAllowed,
		ValidatedAt:  u.nowFunc(),
		ExpiresAt:    apiResult.ExpiresAt,
	}
	u.cache.Set(hash, cached)
	u.cache.UpdateSyncTime()

	slog.InfoContext(ctx, "license validated via cloud api", "tier", apiResult.Tier)
	return &Result{
		Valid:        true,
		Tier:         apiResult.Tier,
		SeatsAllowed: apiResult.SeatsAllowed,
	}, nil
}

// handleAPIFailure returns cached result during grace period, or blocked.
func (u *Usecase) handleAPIFailure(ctx context.Context, hash string, apiErr error) (*Result, error) {
	// Step 4: check grace period
	cached, hasCached := u.cache.Get(hash)
	if hasCached && u.cache.IsWithinGrace() {
		slog.WarnContext(ctx, "using cached license during grace period",
			"tier", cached.Tier,
			"validated_at", cached.ValidatedAt,
		)
		return &Result{
			Valid:        true,
			Tier:         cached.Tier,
			SeatsAllowed: cached.SeatsAllowed,
			Message:      fmt.Sprintf("cloud api unavailable, using cached license (grace period): %v", apiErr),
			FromCache:    true,
		}, nil
	}

	// Step 5: no cache or grace expired
	slog.ErrorContext(ctx, "license validation blocked: no cache or grace expired", "error", apiErr)
	return &Result{
		Valid:   false,
		Message: fmt.Sprintf("license validation failed: %v", apiErr),
	}, nil
}
