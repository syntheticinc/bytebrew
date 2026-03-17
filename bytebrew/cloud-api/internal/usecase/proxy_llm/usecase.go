package proxy_llm

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

// SubscriptionReader reads subscription data.
type SubscriptionReader interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
}

// RateLimiter checks rate limits for trial users.
type RateLimiter interface {
	Check(userID string) error
}

// ModelRouter resolves the target LLM model for a given agent role.
type ModelRouter interface {
	RouteModel(role string) string
}

// Result holds the proxy authorization result.
type Result struct {
	TargetModel string
	Allowed     bool
}

// Usecase handles proxy LLM request authorization.
type Usecase struct {
	subReader   SubscriptionReader
	rateLimiter RateLimiter
	modelRouter ModelRouter
}

// New creates a new proxy_llm Usecase.
func New(subReader SubscriptionReader, rateLimiter RateLimiter, modelRouter ModelRouter) *Usecase {
	return &Usecase{
		subReader:   subReader,
		rateLimiter: rateLimiter,
		modelRouter: modelRouter,
	}
}

// Authorize checks if a user is authorized for proxy LLM and returns the target model.
func (u *Usecase) Authorize(ctx context.Context, userID, role, modelOverride string) (*Result, error) {
	if userID == "" {
		return nil, errors.InvalidInput("user_id is required")
	}

	sub, err := u.subReader.GetByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Internal("get subscription", err)
	}
	if sub == nil {
		return nil, errors.Forbidden("no active subscription")
	}
	if !sub.Status.IsActive() {
		return nil, errors.Forbidden(fmt.Sprintf("subscription not active: %s", sub.Status))
	}

	if sub.Tier == domain.TierTrial {
		if err := u.rateLimiter.Check(userID); err != nil {
			return nil, errors.New("RATE_LIMITED", err.Error())
		}
	}

	// Paid tiers: monthly quota check
	if sub.Tier != domain.TierTrial && sub.ProxyStepsLimit > 0 && sub.ProxyStepsUsed >= sub.ProxyStepsLimit {
		return nil, errors.New("QUOTA_EXHAUSTED",
			fmt.Sprintf("proxy steps exhausted: %d/%d", sub.ProxyStepsUsed, sub.ProxyStepsLimit))
	}

	targetModel := modelOverride
	if targetModel == "" {
		targetModel = u.modelRouter.RouteModel(role)
	}

	return &Result{TargetModel: targetModel, Allowed: true}, nil
}
