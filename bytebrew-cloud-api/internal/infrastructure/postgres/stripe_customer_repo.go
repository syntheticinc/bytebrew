package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/infrastructure/postgres/sqlcgen"
)

// StripeCustomerRepository implements stripe customer persistence with PostgreSQL.
type StripeCustomerRepository struct {
	queries *sqlcgen.Queries
}

// NewStripeCustomerRepository creates a new StripeCustomerRepository.
func NewStripeCustomerRepository(db sqlcgen.DBTX) *StripeCustomerRepository {
	return &StripeCustomerRepository{
		queries: sqlcgen.New(db),
	}
}

// Upsert creates or updates a stripe customer mapping.
func (r *StripeCustomerRepository) Upsert(ctx context.Context, userID, customerID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	return r.queries.UpsertStripeCustomer(ctx, sqlcgen.UpsertStripeCustomerParams{
		UserID:     uid,
		CustomerID: customerID,
	})
}

// GetByUserID returns the stripe customer for a user, or nil if not found.
func (r *StripeCustomerRepository) GetByUserID(ctx context.Context, userID string) (*domain.StripeCustomer, error) {
	uid, err := parseUUID(userID)
	if err != nil {
		return nil, fmt.Errorf("parse user ID: %w", err)
	}
	row, err := r.queries.GetStripeCustomerByUserID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get stripe customer by user id: %w", err)
	}
	return &domain.StripeCustomer{
		UserID:     uuidToString(row.UserID),
		CustomerID: row.CustomerID,
		CreatedAt:  timestamptzToTimeValue(row.CreatedAt),
	}, nil
}

// GetUserIDByCustomerID returns the user ID for a Stripe customer ID, or empty string if not found.
func (r *StripeCustomerRepository) GetUserIDByCustomerID(ctx context.Context, customerID string) (string, error) {
	uid, err := r.queries.GetUserIDByStripeCustomerID(ctx, customerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("get user id by stripe customer id: %w", err)
	}
	return uuidToString(uid), nil
}
