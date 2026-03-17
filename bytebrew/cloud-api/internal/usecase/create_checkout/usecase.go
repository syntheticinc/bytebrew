package create_checkout

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
)

// StripeCustomerReader reads stripe customer data.
type StripeCustomerReader interface {
	GetByUserID(ctx context.Context, userID string) (*domain.StripeCustomer, error)
}

// StripeCustomerSaver persists stripe customer mappings.
type StripeCustomerSaver interface {
	Upsert(ctx context.Context, userID, customerID string) error
}

// CustomerCreator creates Stripe customers.
type CustomerCreator interface {
	CreateCustomer(ctx context.Context, email string, metadata map[string]string) (string, error)
}

// CheckoutSessionCreator creates Stripe checkout sessions.
type CheckoutSessionCreator interface {
	CreateCheckoutSession(ctx context.Context, params CheckoutParams) (string, error)
}

// CheckoutParams holds parameters for creating a checkout session.
type CheckoutParams struct {
	CustomerID string
	PriceID    string
	Plan       string // "personal", "teams" — used by infrastructure to decide trial eligibility
	TrialDays  int64
	SuccessURL string
	CancelURL  string
	Metadata   map[string]string
}

// PriceResolver resolves plan+period to Stripe Price ID.
type PriceResolver interface {
	PriceIDForPlan(plan, period string) (string, error)
}

// Input for creating a checkout session.
type Input struct {
	UserID string
	Email  string
	Plan   string // "personal", "teams"
	Period string // "monthly", "annual"
}

// Output contains the Stripe Checkout URL.
type Output struct {
	CheckoutURL string
}

// Usecase orchestrates Stripe Checkout Session creation.
type Usecase struct {
	customerReader  StripeCustomerReader
	customerSaver   StripeCustomerSaver
	customerCreator CustomerCreator
	sessionCreator  CheckoutSessionCreator
	priceResolver   PriceResolver
	successURL      string
	cancelURL       string
	trialDays       int64
}

// New creates a new create_checkout Usecase.
func New(
	customerReader StripeCustomerReader,
	customerSaver StripeCustomerSaver,
	customerCreator CustomerCreator,
	sessionCreator CheckoutSessionCreator,
	priceResolver PriceResolver,
	successURL, cancelURL string,
	trialDays int64,
) *Usecase {
	return &Usecase{
		customerReader:  customerReader,
		customerSaver:   customerSaver,
		customerCreator: customerCreator,
		sessionCreator:  sessionCreator,
		priceResolver:   priceResolver,
		successURL:      successURL,
		cancelURL:       cancelURL,
		trialDays:       trialDays,
	}
}

// Execute creates a Stripe Checkout Session.
func (u *Usecase) Execute(ctx context.Context, in Input) (*Output, error) {
	if err := validateInput(in); err != nil {
		return nil, err
	}

	customerID, err := u.ensureStripeCustomer(ctx, in)
	if err != nil {
		return nil, err
	}

	priceID, err := u.priceResolver.PriceIDForPlan(in.Plan, in.Period)
	if err != nil {
		return nil, errors.InvalidInput(fmt.Sprintf("invalid plan: %s", err))
	}

	checkoutURL, err := u.sessionCreator.CreateCheckoutSession(ctx, CheckoutParams{
		CustomerID: customerID,
		PriceID:    priceID,
		Plan:       in.Plan,
		TrialDays:  u.trialDays,
		SuccessURL: u.successURL,
		CancelURL:  u.cancelURL,
		Metadata: map[string]string{
			"user_id": in.UserID,
		},
	})
	if err != nil {
		return nil, errors.Internal("create checkout session", err)
	}

	return &Output{CheckoutURL: checkoutURL}, nil
}

func (u *Usecase) ensureStripeCustomer(ctx context.Context, in Input) (string, error) {
	existing, err := u.customerReader.GetByUserID(ctx, in.UserID)
	if err != nil {
		return "", errors.Internal("get stripe customer", err)
	}
	if existing != nil {
		return existing.CustomerID, nil
	}

	customerID, err := u.customerCreator.CreateCustomer(ctx, in.Email, map[string]string{
		"user_id": in.UserID,
	})
	if err != nil {
		return "", errors.Internal("create stripe customer", err)
	}

	if err := u.customerSaver.Upsert(ctx, in.UserID, customerID); err != nil {
		return "", errors.Internal("save stripe customer", err)
	}

	return customerID, nil
}

func validateInput(in Input) error {
	if in.UserID == "" {
		return errors.InvalidInput("user_id is required")
	}
	if in.Email == "" {
		return errors.InvalidInput("email is required")
	}
	switch in.Plan {
	case "personal", "teams":
		// valid
	default:
		return errors.InvalidInput(fmt.Sprintf("invalid plan: %q, must be personal or teams", in.Plan))
	}
	switch in.Period {
	case "monthly", "annual":
		// valid
	default:
		return errors.InvalidInput(fmt.Sprintf("invalid period: %q, must be monthly or annual", in.Period))
	}
	return nil
}
