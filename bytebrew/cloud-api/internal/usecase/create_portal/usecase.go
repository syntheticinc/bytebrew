package create_portal

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

// StripeCustomerReader reads stripe customer data.
type StripeCustomerReader interface {
	GetByUserID(ctx context.Context, userID string) (*domain.StripeCustomer, error)
}

// PortalCreator creates Stripe Customer Portal sessions.
type PortalCreator interface {
	CreatePortalSession(ctx context.Context, customerID, returnURL string) (string, error)
}

// Input for creating a portal session.
type Input struct {
	UserID string
}

// Output contains the Stripe Portal URL.
type Output struct {
	PortalURL string
}

// Usecase orchestrates Stripe Customer Portal Session creation.
type Usecase struct {
	customerReader StripeCustomerReader
	portalCreator  PortalCreator
	returnURL      string
}

// New creates a new create_portal Usecase.
func New(customerReader StripeCustomerReader, portalCreator PortalCreator, returnURL string) *Usecase {
	return &Usecase{
		customerReader: customerReader,
		portalCreator:  portalCreator,
		returnURL:      returnURL,
	}
}

// Execute creates a Stripe Customer Portal Session.
func (u *Usecase) Execute(ctx context.Context, in Input) (*Output, error) {
	if in.UserID == "" {
		return nil, errors.InvalidInput("user_id is required")
	}

	customer, err := u.customerReader.GetByUserID(ctx, in.UserID)
	if err != nil {
		return nil, errors.Internal("get stripe customer", err)
	}

	if customer == nil {
		return nil, errors.NotFound("no active Stripe subscription found")
	}

	portalURL, err := u.portalCreator.CreatePortalSession(ctx, customer.CustomerID, u.returnURL)
	if err != nil {
		return nil, errors.Internal("create portal session", err)
	}

	return &Output{PortalURL: portalURL}, nil
}
