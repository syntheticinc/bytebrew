package delete_account

import (
	"context"
	"log/slog"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// Consumer-side interfaces

// UserReader provides user lookup needed by account deletion.
type UserReader interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// UserDeleter removes a user from persistence.
type UserDeleter interface {
	Delete(ctx context.Context, userID string) error
}

// PasswordHasher verifies passwords against hashes.
type PasswordHasher interface {
	Compare(hash, password string) error
}

// SubscriptionReader looks up subscriptions by user ID.
type SubscriptionReader interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
}

// SubscriptionCanceller cancels a Stripe subscription.
type SubscriptionCanceller interface {
	CancelSubscription(ctx context.Context, stripeSubscriptionID string) error
}

// Input is the delete account request.
type Input struct {
	UserID   string
	Password string
}

// Usecase handles account deletion after password confirmation.
type Usecase struct {
	userReader   UserReader
	deleter      UserDeleter
	hasher       PasswordHasher
	subReader    SubscriptionReader
	subCanceller SubscriptionCanceller
}

// New creates a new DeleteAccount usecase.
func New(
	userReader UserReader,
	deleter UserDeleter,
	hasher PasswordHasher,
	subReader SubscriptionReader,
	subCanceller SubscriptionCanceller,
) *Usecase {
	return &Usecase{
		userReader:   userReader,
		deleter:      deleter,
		hasher:       hasher,
		subReader:    subReader,
		subCanceller: subCanceller,
	}
}

// Execute deletes the user's account after verifying the password.
func (u *Usecase) Execute(ctx context.Context, input Input) error {
	user, err := u.userReader.GetByID(ctx, input.UserID)
	if err != nil {
		return errors.Internal("get user", err)
	}
	if user == nil {
		return errors.NotFound("user not found")
	}

	if err := u.hasher.Compare(user.PasswordHash, input.Password); err != nil {
		return errors.Unauthorized("incorrect password")
	}

	u.cancelStripeSubscription(ctx, input.UserID)

	if err := u.deleter.Delete(ctx, input.UserID); err != nil {
		return errors.Internal("delete account", err)
	}

	return nil
}

// cancelStripeSubscription attempts to cancel the user's Stripe subscription.
// Errors are logged but do not block account deletion.
func (u *Usecase) cancelStripeSubscription(ctx context.Context, userID string) {
	sub, err := u.subReader.GetByUserID(ctx, userID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get subscription for cancellation", "user_id", userID, "error", err)
		return
	}
	if sub == nil || sub.StripeSubscriptionID == nil || *sub.StripeSubscriptionID == "" {
		return
	}

	if err := u.subCanceller.CancelSubscription(ctx, *sub.StripeSubscriptionID); err != nil {
		slog.ErrorContext(ctx, "failed to cancel stripe subscription", "user_id", userID, "stripe_subscription_id", *sub.StripeSubscriptionID, "error", err)
	}
}
