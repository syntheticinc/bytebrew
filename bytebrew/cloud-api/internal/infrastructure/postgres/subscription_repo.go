package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/postgres/sqlcgen"
)

// SubscriptionRepository implements subscription persistence with PostgreSQL.
type SubscriptionRepository struct {
	queries *sqlcgen.Queries
}

// NewSubscriptionRepository creates a new SubscriptionRepository.
func NewSubscriptionRepository(db sqlcgen.DBTX) *SubscriptionRepository {
	return &SubscriptionRepository{
		queries: sqlcgen.New(db),
	}
}

// Create inserts a new subscription and returns the created subscription.
func (r *SubscriptionRepository) Create(ctx context.Context, sub *domain.Subscription) (*domain.Subscription, error) {
	userID, err := parseUUID(sub.UserID)
	if err != nil {
		return nil, fmt.Errorf("parse user ID: %w", err)
	}
	row, err := r.queries.CreateSubscription(ctx, sqlcgen.CreateSubscriptionParams{
		UserID:          userID,
		Tier:            string(sub.Tier),
		Status:          string(sub.Status),
		ProxyStepsLimit: int32(sub.ProxyStepsLimit),
		ByokEnabled:     sub.BYOKEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("create subscription: %w", err)
	}
	return mapCreateSubscriptionRow(row), nil
}

// GetByUserID returns subscription for a user, or nil if not found.
func (r *SubscriptionRepository) GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user ID: %w", err)
	}
	row, err := r.queries.GetSubscriptionByUserID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get subscription by user id: %w", err)
	}
	return mapGetSubscriptionRow(row), nil
}

// UpdateTier updates subscription tier and period.
func (r *SubscriptionRepository) UpdateTier(
	ctx context.Context,
	userID string,
	tier domain.LicenseTier,
	status domain.SubscriptionStatus,
	periodStart, periodEnd *time.Time,
) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	err = r.queries.UpdateSubscriptionTier(ctx, sqlcgen.UpdateSubscriptionTierParams{
		Tier:               string(tier),
		Status:             string(status),
		CurrentPeriodStart: timeToTimestamptz(periodStart),
		CurrentPeriodEnd:   timeToTimestamptz(periodEnd),
		UserID:             uid,
	})
	if err != nil {
		return fmt.Errorf("update subscription tier: %w", err)
	}
	return nil
}

// UpdateStripeSubscriptionID sets the Stripe Subscription ID on a subscription.
func (r *SubscriptionRepository) UpdateStripeSubscriptionID(ctx context.Context, userID, subID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	return r.queries.UpdateStripeSubscriptionID(ctx, sqlcgen.UpdateStripeSubscriptionIDParams{
		StripeSubscriptionID: pgtype.Text{String: subID, Valid: true},
		UserID:               uid,
	})
}

// UpdateStatus updates only the subscription status.
func (r *SubscriptionRepository) UpdateStatus(ctx context.Context, userID string, status domain.SubscriptionStatus) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	return r.queries.UpdateSubscriptionStatus(ctx, sqlcgen.UpdateSubscriptionStatusParams{
		Status: string(status),
		UserID: uid,
	})
}

// UpdateFull updates tier, status, period, Stripe subscription ID, and proxy steps limit.
func (r *SubscriptionRepository) UpdateFull(
	ctx context.Context,
	userID string,
	tier domain.LicenseTier,
	status domain.SubscriptionStatus,
	periodStart, periodEnd *time.Time,
	stripeSubID string,
	proxyStepsLimit int,
) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	return r.queries.UpdateSubscriptionFull(ctx, sqlcgen.UpdateSubscriptionFullParams{
		Tier:                 string(tier),
		Status:               string(status),
		CurrentPeriodStart:   timeToTimestamptz(periodStart),
		CurrentPeriodEnd:     timeToTimestamptz(periodEnd),
		StripeSubscriptionID: pgtype.Text{String: stripeSubID, Valid: stripeSubID != ""},
		ProxyStepsLimit:      int32(proxyStepsLimit),
		UserID:               uid,
	})
}

// IncrementProxySteps atomically increments the proxy steps counter for a user's subscription.
func (r *SubscriptionRepository) IncrementProxySteps(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	return r.queries.IncrementProxySteps(ctx, uid)
}

// ResetProxySteps resets the proxy steps counter for a user's subscription.
func (r *SubscriptionRepository) ResetProxySteps(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	return r.queries.ResetProxyStepsUsed(ctx, uid)
}

func mapCreateSubscriptionRow(row sqlcgen.CreateSubscriptionRow) *domain.Subscription {
	return &domain.Subscription{
		ID:                   uuidToString(row.ID),
		UserID:               uuidToString(row.UserID),
		Tier:                 domain.LicenseTier(row.Tier),
		Status:               domain.SubscriptionStatus(row.Status),
		CurrentPeriodStart:   timestamptzToTime(row.CurrentPeriodStart),
		CurrentPeriodEnd:     timestamptzToTime(row.CurrentPeriodEnd),
		StripeSubscriptionID: textToStringPtr(row.StripeSubscriptionID),
		ProxyStepsUsed:       int(row.ProxyStepsUsed),
		ProxyStepsLimit:      int(row.ProxyStepsLimit),
		BYOKEnabled:          row.ByokEnabled,
		CreatedAt:            timestamptzToTimeValue(row.CreatedAt),
		UpdatedAt:            timestamptzToTimeValue(row.UpdatedAt),
	}
}

func mapGetSubscriptionRow(row sqlcgen.GetSubscriptionByUserIDRow) *domain.Subscription {
	return &domain.Subscription{
		ID:                   uuidToString(row.ID),
		UserID:               uuidToString(row.UserID),
		Tier:                 domain.LicenseTier(row.Tier),
		Status:               domain.SubscriptionStatus(row.Status),
		CurrentPeriodStart:   timestamptzToTime(row.CurrentPeriodStart),
		CurrentPeriodEnd:     timestamptzToTime(row.CurrentPeriodEnd),
		StripeSubscriptionID: textToStringPtr(row.StripeSubscriptionID),
		ProxyStepsUsed:       int(row.ProxyStepsUsed),
		ProxyStepsLimit:      int(row.ProxyStepsLimit),
		BYOKEnabled:          row.ByokEnabled,
		CreatedAt:            timestamptzToTimeValue(row.CreatedAt),
		UpdatedAt:            timestamptzToTimeValue(row.UpdatedAt),
	}
}
