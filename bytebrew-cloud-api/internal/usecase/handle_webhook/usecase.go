package handle_webhook

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
)

// UserIDResolver resolves Stripe Customer ID to internal user ID via stripe_customers table.
type UserIDResolver interface {
	GetUserIDByCustomerID(ctx context.Context, customerID string) (string, error)
}

// SubscriptionReader reads subscription data by user ID.
type SubscriptionReader interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
}

// SubscriptionUpdater updates subscription data.
type SubscriptionUpdater interface {
	UpdateFull(ctx context.Context, userID string, tier domain.LicenseTier, status domain.SubscriptionStatus, periodStart, periodEnd *time.Time, stripeSubID string, proxyStepsLimit int) error
	UpdateStatus(ctx context.Context, userID string, status domain.SubscriptionStatus) error
}

// ProxyStepsResetter resets proxy steps counter for a user.
type ProxyStepsResetter interface {
	ResetProxySteps(ctx context.Context, userID string) error
}

// SubscriptionCreator creates subscriptions for users who don't have one yet.
type SubscriptionCreator interface {
	Create(ctx context.Context, sub *domain.Subscription) (*domain.Subscription, error)
}

// EventStore manages processed event idempotency.
type EventStore interface {
	IsProcessed(ctx context.Context, eventID string) (bool, error)
	MarkProcessed(ctx context.Context, eventID, eventType string) error
}

// TierResolver resolves Stripe Price ID to domain tier.
type TierResolver interface {
	TierForPriceID(priceID string) (domain.LicenseTier, bool)
}

// TeamSeatsUpdater finds and updates team seats.
type TeamSeatsUpdater interface {
	GetTeamByOwnerID(ctx context.Context, ownerID string) (*domain.Team, error)
	UpdateTeamMaxSeats(ctx context.Context, teamID string, maxSeats int) error
}

// Event represents a Stripe webhook event.
type Event struct {
	ID   string
	Type string
	Data EventData
}

// EventData holds the parsed event payload.
type EventData struct {
	// Subscription fields (for customer.subscription.* events)
	CustomerID         string
	SubscriptionID     string
	Status             string
	PriceID            string
	Quantity           int64 // seat count from subscription item
	CurrentPeriodStart *time.Time
	CurrentPeriodEnd   *time.Time

	// Invoice fields (for invoice.* events)
	InvoiceCustomerID string
}

// Usecase handles Stripe webhook events.
type Usecase struct {
	userIDResolver UserIDResolver
	subReader      SubscriptionReader
	subUpdater     SubscriptionUpdater
	subCreator     SubscriptionCreator
	eventStore     EventStore
	tierResolver   TierResolver
	proxyResetter  ProxyStepsResetter
	teamSeats      TeamSeatsUpdater
}

// New creates a new handle_webhook Usecase.
func New(
	userIDResolver UserIDResolver,
	subReader SubscriptionReader,
	subUpdater SubscriptionUpdater,
	subCreator SubscriptionCreator,
	eventStore EventStore,
	tierResolver TierResolver,
	proxyResetter ProxyStepsResetter,
	teamSeats TeamSeatsUpdater,
) *Usecase {
	return &Usecase{
		userIDResolver: userIDResolver,
		subReader:      subReader,
		subUpdater:     subUpdater,
		subCreator:     subCreator,
		eventStore:     eventStore,
		tierResolver:   tierResolver,
		proxyResetter:  proxyResetter,
		teamSeats:      teamSeats,
	}
}

// Execute processes a Stripe webhook event.
func (u *Usecase) Execute(ctx context.Context, event Event) error {
	processed, err := u.eventStore.IsProcessed(ctx, event.ID)
	if err != nil {
		return errors.Internal("check event idempotency", err)
	}
	if processed {
		slog.InfoContext(ctx, "skipping already processed event", "event_id", event.ID)
		return nil
	}

	if err := u.dispatch(ctx, event); err != nil {
		return err
	}

	if err := u.eventStore.MarkProcessed(ctx, event.ID, event.Type); err != nil {
		return errors.Internal("mark event as processed", err)
	}

	return nil
}

func (u *Usecase) dispatch(ctx context.Context, event Event) error {
	switch event.Type {
	case "customer.subscription.created", "customer.subscription.updated":
		return u.handleSubscriptionChange(ctx, event.Data)
	case "customer.subscription.deleted":
		return u.handleSubscriptionDeleted(ctx, event.Data)
	case "invoice.payment_failed":
		return u.handlePaymentFailed(ctx, event.Data)
	case "invoice.payment_succeeded":
		return u.handlePaymentSucceeded(ctx, event.Data)
	case "customer.subscription.trial_will_end":
		return u.handleTrialWillEnd(ctx, event.Data)
	default:
		slog.InfoContext(ctx, "ignoring unhandled event type", "type", event.Type)
		return nil
	}
}

func (u *Usecase) handleSubscriptionChange(ctx context.Context, data EventData) error {
	userID, err := u.resolveUserID(ctx, data.CustomerID)
	if err != nil {
		return err
	}
	if userID == "" {
		slog.WarnContext(ctx, "no user found for stripe customer", "customer_id", data.CustomerID)
		return nil
	}

	tier, ok := u.tierResolver.TierForPriceID(data.PriceID)
	if !ok {
		return errors.Internal("unknown price ID", fmt.Errorf("price_id: %s", data.PriceID))
	}

	status := mapStripeStatus(data.Status)

	sub, err := u.subReader.GetByUserID(ctx, userID)
	if err != nil {
		return errors.Internal("get subscription by user", err)
	}

	proxyStepsLimit := proxyStepsLimitForTier(tier)

	if sub == nil {
		newSub := &domain.Subscription{
			UserID:          userID,
			Tier:            tier,
			Status:          status,
			ProxyStepsLimit: proxyStepsLimit,
			BYOKEnabled:     true,
		}
		if _, err := u.subCreator.Create(ctx, newSub); err != nil {
			return errors.Internal("create subscription", err)
		}
	}

	if err := u.subUpdater.UpdateFull(ctx, userID, tier, status, data.CurrentPeriodStart, data.CurrentPeriodEnd, data.SubscriptionID, proxyStepsLimit); err != nil {
		return errors.Internal("update subscription", err)
	}

	// Sync team seats from Stripe quantity for Teams tier
	if tier == domain.TierTeams && data.Quantity > 0 {
		team, err := u.teamSeats.GetTeamByOwnerID(ctx, userID)
		if err != nil {
			slog.ErrorContext(ctx, "get team by owner for seat sync", "error", err, "user_id", userID)
			return nil // non-fatal: subscription updated, seat sync can retry
		}
		if team != nil {
			if err := u.teamSeats.UpdateTeamMaxSeats(ctx, team.ID, int(data.Quantity)); err != nil {
				slog.ErrorContext(ctx, "update team max seats from webhook", "error", err, "team_id", team.ID)
			}
		}
	}

	return nil
}

func (u *Usecase) handleSubscriptionDeleted(ctx context.Context, data EventData) error {
	userID, err := u.resolveUserID(ctx, data.CustomerID)
	if err != nil {
		return err
	}
	if userID == "" {
		return nil
	}

	sub, err := u.subReader.GetByUserID(ctx, userID)
	if err != nil {
		return errors.Internal("get subscription by user", err)
	}
	if sub == nil {
		return nil
	}

	return u.subUpdater.UpdateStatus(ctx, sub.UserID, domain.StatusCanceled)
}

func (u *Usecase) handlePaymentFailed(ctx context.Context, data EventData) error {
	sub, err := u.findSubscription(ctx, data)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
	}

	return u.subUpdater.UpdateStatus(ctx, sub.UserID, domain.StatusPastDue)
}

func (u *Usecase) handlePaymentSucceeded(ctx context.Context, data EventData) error {
	sub, err := u.findSubscription(ctx, data)
	if err != nil {
		return err
	}
	if sub == nil {
		return nil
	}

	// Reset proxy steps counter at the start of a new billing period.
	if err := u.proxyResetter.ResetProxySteps(ctx, sub.UserID); err != nil {
		return errors.Internal("reset proxy steps", err)
	}

	if sub.Status == domain.StatusPastDue {
		return u.subUpdater.UpdateStatus(ctx, sub.UserID, domain.StatusActive)
	}

	return nil
}

func (u *Usecase) handleTrialWillEnd(ctx context.Context, data EventData) error {
	slog.InfoContext(ctx, "trial ending soon",
		"customer_id", data.CustomerID,
		"subscription_id", data.SubscriptionID,
	)
	return nil
}

// resolveUserID resolves Stripe Customer ID to internal user ID.
func (u *Usecase) resolveUserID(ctx context.Context, customerID string) (string, error) {
	if customerID == "" {
		return "", nil
	}
	userID, err := u.userIDResolver.GetUserIDByCustomerID(ctx, customerID)
	if err != nil {
		return "", errors.Internal("resolve user by stripe customer", err)
	}
	return userID, nil
}

// findSubscription resolves the subscription from invoice or subscription customer ID.
func (u *Usecase) findSubscription(ctx context.Context, data EventData) (*domain.Subscription, error) {
	customerID := data.InvoiceCustomerID
	if customerID == "" {
		customerID = data.CustomerID
	}

	userID, err := u.resolveUserID(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if userID == "" {
		return nil, nil
	}

	sub, err := u.subReader.GetByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Internal("get subscription by user", err)
	}
	return sub, nil
}

// proxyStepsLimitForTier delegates to domain.ProxyStepsLimitForTier.
func proxyStepsLimitForTier(tier domain.LicenseTier) int {
	return domain.ProxyStepsLimitForTier(tier)
}

func mapStripeStatus(s string) domain.SubscriptionStatus {
	switch s {
	case "active":
		return domain.StatusActive
	case "trialing":
		return domain.StatusTrialing
	case "past_due":
		return domain.StatusPastDue
	case "canceled", "unpaid", "incomplete_expired":
		return domain.StatusCanceled
	default:
		return domain.StatusCanceled
	}
}
