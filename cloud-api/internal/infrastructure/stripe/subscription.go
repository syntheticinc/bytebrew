package stripe

import (
	"context"
	"fmt"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/subscription"
	"github.com/stripe/stripe-go/v82/subscriptionitem"
)

// CancelSubscription cancels a Stripe subscription immediately.
func (c *CheckoutClient) CancelSubscription(ctx context.Context, stripeSubscriptionID string) error {
	params := &stripe.SubscriptionCancelParams{}
	params.Context = ctx
	_, err := subscription.Cancel(stripeSubscriptionID, params)
	if err != nil {
		return fmt.Errorf("cancel subscription: %w", err)
	}
	return nil
}

// UpdateSubscriptionQuantity updates the seat count on a Stripe subscription.
// Uses proration_behavior = "create_prorations" (Stripe default).
func (c *CheckoutClient) UpdateSubscriptionQuantity(ctx context.Context, subscriptionID string, quantity int64) error {
	getParams := &stripe.SubscriptionParams{}
	getParams.Context = ctx
	sub, err := subscription.Get(subscriptionID, getParams)
	if err != nil {
		return fmt.Errorf("get subscription: %w", err)
	}
	if sub.Items == nil || len(sub.Items.Data) == 0 {
		return fmt.Errorf("subscription has no items")
	}

	itemID := sub.Items.Data[0].ID
	itemParams := &stripe.SubscriptionItemParams{
		Quantity:          stripe.Int64(quantity),
		ProrationBehavior: stripe.String("create_prorations"),
	}
	itemParams.Context = ctx

	_, err = subscriptionitem.Update(itemID, itemParams)
	if err != nil {
		return fmt.Errorf("update subscription quantity: %w", err)
	}
	return nil
}
