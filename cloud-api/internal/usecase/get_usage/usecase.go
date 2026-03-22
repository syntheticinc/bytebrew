package get_usage

import (
	"context"
	"time"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// SubscriptionReader looks up a user's subscription.
type SubscriptionReader interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
}

// Input is the get_usage request.
type Input struct {
	UserID string
}

// Output is the get_usage response.
type Output struct {
	Tier                string
	ProxyStepsUsed      int
	ProxyStepsLimit     int
	ProxyStepsRemaining int
	BYOKEnabled         bool
	CurrentPeriodEnd    *time.Time
}

// Usecase returns proxy usage data for a user's subscription.
type Usecase struct {
	subReader SubscriptionReader
}

// New creates a new get_usage Usecase.
func New(subReader SubscriptionReader) *Usecase {
	return &Usecase{subReader: subReader}
}

// Execute returns usage data for the given user.
func (u *Usecase) Execute(ctx context.Context, in Input) (*Output, error) {
	if in.UserID == "" {
		return nil, errors.InvalidInput("user_id is required")
	}

	sub, err := u.subReader.GetByUserID(ctx, in.UserID)
	if err != nil {
		return nil, errors.Internal("get subscription", err)
	}

	if sub == nil {
		return nil, errors.Forbidden("no active subscription")
	}

	remaining := sub.ProxyStepsLimit - sub.ProxyStepsUsed
	if remaining < 0 {
		remaining = 0
	}

	return &Output{
		Tier:                string(sub.Tier),
		ProxyStepsUsed:      sub.ProxyStepsUsed,
		ProxyStepsLimit:     sub.ProxyStepsLimit,
		ProxyStepsRemaining: remaining,
		BYOKEnabled:         sub.BYOKEnabled,
		CurrentPeriodEnd:    sub.CurrentPeriodEnd,
	}, nil
}
