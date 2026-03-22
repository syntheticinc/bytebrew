package domain

import "time"

// LicenseFeatures describes the capabilities for a given tier.
// All tiers have full features; differentiation is via proxy steps and rate limiting.
type LicenseFeatures struct {
	FullAutonomy     bool
	ParallelAgents   int
	ExploreCodebase  bool
	TraceSymbol      bool
	CodebaseIndexing bool
}

// LicenseInfo contains all data needed to sign a license JWT.
type LicenseInfo struct {
	UserID              string
	Email               string
	Tier                LicenseTier
	ExpiresAt           time.Time
	GraceUntil          time.Time
	Features            LicenseFeatures
	ProxyStepsRemaining int
	ProxyStepsLimit     int
	BYOKEnabled         bool
	MaxSeats            int
}

// FeaturesForTier returns the feature set for a given license tier.
// All tiers have full features; differentiation is via proxy steps and rate limiting.
func FeaturesForTier(_ LicenseTier) LicenseFeatures {
	return LicenseFeatures{
		FullAutonomy:     true,
		ParallelAgents:   -1,
		ExploreCodebase:  true,
		TraceSymbol:      true,
		CodebaseIndexing: true,
	}
}

// DefaultLicenseExpiry returns the default expiry: 30 days from now.
func DefaultLicenseExpiry() time.Time {
	return time.Now().Add(30 * 24 * time.Hour)
}

// GraceFromExpiry returns expiry + 3 days grace period.
func GraceFromExpiry(expiry time.Time) time.Time {
	return expiry.Add(3 * 24 * time.Hour)
}

// ResolveTierAndExpiry determines the license tier and expiry from a subscription.
// Returns ErrNoActiveSubscription when subscription is nil or status is canceled/expired.
// For past_due, preserves the paid tier (grace period).
func ResolveTierAndExpiry(sub *Subscription) (LicenseTier, time.Time, error) {
	if sub == nil {
		return "", time.Time{}, ErrNoActiveSubscription
	}

	switch sub.Status {
	case StatusActive, StatusTrialing, StatusPastDue:
		if sub.CurrentPeriodEnd != nil {
			return sub.Tier, *sub.CurrentPeriodEnd, nil
		}
		return sub.Tier, DefaultLicenseExpiry(), nil
	default:
		return "", time.Time{}, ErrNoActiveSubscription
	}
}
