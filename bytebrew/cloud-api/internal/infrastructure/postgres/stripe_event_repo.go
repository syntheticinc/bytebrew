package postgres

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/postgres/sqlcgen"
)

// StripeEventRepository handles idempotency for Stripe webhook events.
type StripeEventRepository struct {
	queries *sqlcgen.Queries
}

// NewStripeEventRepository creates a new StripeEventRepository.
func NewStripeEventRepository(db sqlcgen.DBTX) *StripeEventRepository {
	return &StripeEventRepository{
		queries: sqlcgen.New(db),
	}
}

// IsProcessed checks if a Stripe event has already been processed.
func (r *StripeEventRepository) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	processed, err := r.queries.IsEventProcessed(ctx, eventID)
	if err != nil {
		return false, fmt.Errorf("check event processed: %w", err)
	}
	return processed, nil
}

// MarkProcessed records that a Stripe event has been processed.
func (r *StripeEventRepository) MarkProcessed(ctx context.Context, eventID, eventType string) error {
	err := r.queries.InsertProcessedEvent(ctx, sqlcgen.InsertProcessedEventParams{
		EventID:   eventID,
		EventType: eventType,
	})
	if err != nil {
		return fmt.Errorf("mark event processed: %w", err)
	}
	return nil
}
