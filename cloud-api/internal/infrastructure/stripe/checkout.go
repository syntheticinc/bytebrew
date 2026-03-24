package stripe

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/customer"
)

// CheckoutClient interacts with Stripe Checkout and Customer APIs.
type CheckoutClient struct {
	secretKey string
}

// NewCheckoutClient creates a new CheckoutClient.
// It sets the global stripe.Key once during initialization to avoid race conditions.
func NewCheckoutClient(secretKey string) *CheckoutClient {
	stripe.Key = secretKey
	return &CheckoutClient{secretKey: secretKey}
}

// CreateCustomer creates a Stripe Customer and returns the customer ID.
func (c *CheckoutClient) CreateCustomer(ctx context.Context, email string, metadata map[string]string) (string, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
	}
	for k, v := range metadata {
		params.AddMetadata(k, v)
	}

	cust, err := customer.New(params)
	if err != nil {
		return "", fmt.Errorf("create stripe customer: %w", err)
	}
	return cust.ID, nil
}

// CheckoutParams holds parameters for creating a checkout session.
type CheckoutParams struct {
	CustomerID string
	PriceID    string
	Plan       string // "personal", "teams", "engine_ee"
	TrialDays  int64
	SuccessURL string
	CancelURL  string
	Metadata   map[string]string
}

// CreateCheckoutSession creates a Stripe Checkout Session and returns the session URL.
func (c *CheckoutClient) CreateCheckoutSession(ctx context.Context, params CheckoutParams) (string, error) {
	lineItem := &stripe.CheckoutSessionLineItemParams{
		Price:    stripe.String(params.PriceID),
		Quantity: stripe.Int64(1),
	}

	sessionParams := &stripe.CheckoutSessionParams{
		Customer:   stripe.String(params.CustomerID),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems:  []*stripe.CheckoutSessionLineItemParams{lineItem},
		SuccessURL: stripe.String(params.SuccessURL),
		CancelURL:  stripe.String(params.CancelURL),
	}

	if params.TrialDays > 0 && params.Plan != "teams" && params.Plan != "engine_ee" {
		sessionParams.SubscriptionData = &stripe.CheckoutSessionSubscriptionDataParams{
			TrialPeriodDays: stripe.Int64(params.TrialDays),
		}
		sessionParams.PaymentMethodCollection = stripe.String("always")
	}

	for k, v := range params.Metadata {
		sessionParams.AddMetadata(k, v)
	}

	sess, err := session.New(sessionParams)
	if err != nil {
		return "", fmt.Errorf("create checkout session: %w", err)
	}
	return sess.URL, nil
}
