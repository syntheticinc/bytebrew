package billing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
)

// StripeClient handles Stripe API interactions (consumer-side interface).
type StripeClient interface {
	CreateCheckoutSession(ctx context.Context, tenantID string, plan domain.CloudPlan) (sessionURL string, err error)
	CreatePortalSession(ctx context.Context, tenantID string) (portalURL string, err error)
}

// PlanUpdater updates a tenant's plan in the database.
type PlanUpdater interface {
	UpdatePlan(ctx context.Context, tenantID string, plan domain.CloudPlan) error
	ActivateTrial(ctx context.Context, tenantID string, trial domain.TrialInfo) error
	GetTrialInfo(ctx context.Context, tenantID string) (*domain.TrialInfo, error)
}

// StripeService manages Stripe-related billing operations.
type StripeService struct {
	stripe  StripeClient
	updater PlanUpdater
}

// NewStripeService creates a new Stripe billing service.
func NewStripeService(stripe StripeClient, updater PlanUpdater) *StripeService {
	return &StripeService{
		stripe:  stripe,
		updater: updater,
	}
}

// CreateCheckout creates a Stripe Checkout session for plan upgrade (AC-PRICE-03).
func (s *StripeService) CreateCheckout(ctx context.Context, tenantID string, plan domain.CloudPlan) (string, error) {
	if !plan.IsValid() {
		return "", fmt.Errorf("invalid plan: %s", plan)
	}
	if plan == domain.PlanFree {
		return "", fmt.Errorf("cannot checkout for free plan")
	}

	url, err := s.stripe.CreateCheckoutSession(ctx, tenantID, plan)
	if err != nil {
		return "", fmt.Errorf("create checkout: %w", err)
	}

	slog.InfoContext(ctx, "[Stripe] checkout created", "tenant", tenantID, "plan", plan)
	return url, nil
}

// HandlePaymentSuccess handles a successful payment webhook (AC-PRICE-11).
// Updates the tenant's plan immediately.
func (s *StripeService) HandlePaymentSuccess(ctx context.Context, tenantID string, plan domain.CloudPlan) error {
	if err := s.updater.UpdatePlan(ctx, tenantID, plan); err != nil {
		return fmt.Errorf("update plan after payment: %w", err)
	}

	slog.InfoContext(ctx, "[Stripe] plan upgraded", "tenant", tenantID, "plan", plan)
	return nil
}

// ActivateProTrial activates a 14-day Pro trial (AC-PRICE-04).
func (s *StripeService) ActivateProTrial(ctx context.Context, tenantID string) error {
	// Check if trial already used
	existing, err := s.updater.GetTrialInfo(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("check trial: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("trial already used for tenant %s", tenantID)
	}

	trial := domain.NewTrial()
	if err := s.updater.ActivateTrial(ctx, tenantID, *trial); err != nil {
		return fmt.Errorf("activate trial: %w", err)
	}

	// Set plan to Pro during trial
	if err := s.updater.UpdatePlan(ctx, tenantID, domain.PlanPro); err != nil {
		return fmt.Errorf("set trial plan: %w", err)
	}

	slog.InfoContext(ctx, "[Stripe] Pro trial activated",
		"tenant", tenantID, "ends", trial.EndDate.Format(time.RFC3339))
	return nil
}

// CheckTrialExpiry checks if a trial has expired and reverts to Free (AC-PRICE-05).
func (s *StripeService) CheckTrialExpiry(ctx context.Context, tenantID string) error {
	trial, err := s.updater.GetTrialInfo(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("get trial: %w", err)
	}
	if trial == nil || !trial.Active {
		return nil
	}

	if trial.IsExpired() {
		// Revert to Free (AC-PRICE-05: not blocked, just downgraded)
		if err := s.updater.UpdatePlan(ctx, tenantID, domain.PlanFree); err != nil {
			return fmt.Errorf("revert trial plan: %w", err)
		}
		slog.InfoContext(ctx, "[Stripe] trial expired, reverted to Free", "tenant", tenantID)
	}
	return nil
}

// CreatePortal creates a Stripe Customer Portal session for plan management.
func (s *StripeService) CreatePortal(ctx context.Context, tenantID string) (string, error) {
	url, err := s.stripe.CreatePortalSession(ctx, tenantID)
	if err != nil {
		return "", fmt.Errorf("create portal: %w", err)
	}
	return url, nil
}
