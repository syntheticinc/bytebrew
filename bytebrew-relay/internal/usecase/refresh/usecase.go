package refresh

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-relay/internal/domain"
)

// LicenseCache provides access to cached licenses for background refresh.
type LicenseCache interface {
	Entries() map[string]*domain.CachedLicense
	Set(jwtHash string, license *domain.CachedLicense)
	Remove(jwtHash string)
	UpdateSyncTime()
}

// CloudAPIClient refreshes and validates licenses via Cloud API.
type CloudAPIClient interface {
	RefreshLicense(ctx context.Context, currentJWT string) (newJWT string, err error)
	ValidateLicense(ctx context.Context, licenseJWT string) (*ValidationResult, error)
}

// ValidationResult holds the result of Cloud API license validation.
type ValidationResult struct {
	Tier         string
	SeatsAllowed int
	ExpiresAt    time.Time
}

// JWTHasher produces cache keys from JWTs.
type JWTHasher interface {
	Hash(jwt string) string
}

// Usecase handles background refresh of all cached licenses via Cloud API.
type Usecase struct {
	cache     LicenseCache
	cloudAPI  CloudAPIClient
	jwtHasher JWTHasher
	nowFunc   func() time.Time
}

// New creates a new refresh usecase.
func New(cache LicenseCache, cloudAPI CloudAPIClient, jwtHasher JWTHasher) *Usecase {
	return &Usecase{
		cache:     cache,
		cloudAPI:  cloudAPI,
		jwtHasher: jwtHasher,
		nowFunc:   time.Now,
	}
}

// RefreshAll refreshes all licenses in the cache through Cloud API.
//
// For each cached license:
//  1. Call RefreshLicense to get a new JWT
//  2. Call ValidateLicense to validate the refreshed JWT
//  3. Store the new entry in cache (with new hash)
//  4. Remove old entry if hash changed
//
// Updates sync time if at least one license was refreshed successfully.
// Continues processing remaining licenses if one fails.
func (u *Usecase) RefreshAll(ctx context.Context) error {
	entries := u.cache.Entries()
	if len(entries) == 0 {
		return nil
	}

	slog.InfoContext(ctx, "background refresh starting", "licenses", len(entries))

	successCount := 0
	for hash, entry := range entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if err := u.refreshEntry(ctx, hash, entry); err != nil {
			slog.WarnContext(ctx, "failed to refresh license", "hash", hash, "error", err)
			continue
		}

		successCount++
	}

	if successCount > 0 {
		u.cache.UpdateSyncTime()
	}

	slog.InfoContext(ctx, "background refresh completed", "refreshed", successCount, "total", len(entries))
	return nil
}

// refreshEntry refreshes a single license entry: refresh JWT, validate, update cache.
func (u *Usecase) refreshEntry(ctx context.Context, oldHash string, entry *domain.CachedLicense) error {
	newJWT, err := u.cloudAPI.RefreshLicense(ctx, entry.JWT)
	if err != nil {
		return fmt.Errorf("refresh license: %w", err)
	}

	info, err := u.cloudAPI.ValidateLicense(ctx, newJWT)
	if err != nil {
		return fmt.Errorf("validate refreshed license: %w", err)
	}

	newHash := u.jwtHasher.Hash(newJWT)
	u.cache.Set(newHash, &domain.CachedLicense{
		JWT:          newJWT,
		Tier:         info.Tier,
		SeatsAllowed: info.SeatsAllowed,
		ValidatedAt:  u.nowFunc(),
		ExpiresAt:    info.ExpiresAt,
	})

	if newHash != oldHash {
		u.cache.Remove(oldHash)
	}

	return nil
}
