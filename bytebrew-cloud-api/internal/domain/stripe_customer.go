package domain

import "time"

// StripeCustomer maps a user to their Stripe customer ID.
type StripeCustomer struct {
	UserID     string
	CustomerID string
	CreatedAt  time.Time
}
