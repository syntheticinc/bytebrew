package domain

import (
	"errors"
	"fmt"
	"time"
)

// ErrNoActiveSubscription is returned when a user has no active subscription.
var ErrNoActiveSubscription = errors.New("no active subscription")

// LicenseTier represents the subscription tier.
type LicenseTier string

const (
	TierTrial    LicenseTier = "trial"
	TierPersonal LicenseTier = "personal"
	TierTeams    LicenseTier = "teams"
)

// ValidTiers returns all valid tiers.
func ValidTiers() []LicenseTier {
	return []LicenseTier{TierTrial, TierPersonal, TierTeams}
}

// IsValid checks if the tier is a known value.
func (t LicenseTier) IsValid() bool {
	switch t {
	case TierTrial, TierPersonal, TierTeams:
		return true
	}
	return false
}

// IsPaid returns true for paid tiers (Personal, Teams).
func (t LicenseTier) IsPaid() bool {
	return t == TierPersonal || t == TierTeams
}

// SubscriptionStatus represents the current state of a subscription.
type SubscriptionStatus string

const (
	StatusActive   SubscriptionStatus = "active"
	StatusTrialing SubscriptionStatus = "trialing"
	StatusPastDue  SubscriptionStatus = "past_due"
	StatusCanceled SubscriptionStatus = "canceled"
	StatusExpired  SubscriptionStatus = "expired"
)

// IsActive returns true if the subscription grants access.
func (s SubscriptionStatus) IsActive() bool {
	return s == StatusActive || s == StatusTrialing
}

// Subscription represents a user's subscription.
type Subscription struct {
	ID                   string
	UserID               string
	Tier                 LicenseTier
	Status               SubscriptionStatus
	CurrentPeriodStart   *time.Time
	CurrentPeriodEnd     *time.Time
	StripeSubscriptionID *string
	ProxyStepsUsed       int
	ProxyStepsLimit      int
	BYOKEnabled          bool
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ProxyStepsLimitForTier returns the monthly proxy steps limit for a tier.
// Trial: 0 (rate-limited by hour, not monthly cap).
// Personal/Teams: 300 steps per month.
func ProxyStepsLimitForTier(tier LicenseTier) int {
	switch tier {
	case TierTrial:
		return 0
	default:
		return 300
	}
}

// NewSubscription creates a new subscription with the given tier.
func NewSubscription(userID string, tier LicenseTier) (*Subscription, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if !tier.IsValid() {
		return nil, fmt.Errorf("invalid tier: %s", tier)
	}

	now := time.Now()
	return &Subscription{
		UserID:    userID,
		Tier:      tier,
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
