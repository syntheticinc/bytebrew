package http

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/usecase/handle_webhook"
)

type webhookUsecase interface {
	Execute(ctx context.Context, event handle_webhook.Event) error
}

// WebhookHandler handles Stripe webhook events.
type WebhookHandler struct {
	webhookUC     webhookUsecase
	webhookSecret string
}

// NewWebhookHandler creates a new WebhookHandler.
func NewWebhookHandler(webhookUC webhookUsecase, webhookSecret string) *WebhookHandler {
	return &WebhookHandler{
		webhookUC:     webhookUC,
		webhookSecret: webhookSecret,
	}
}

// HandleStripe handles POST /api/v1/webhooks/stripe.
// No auth middleware -- verified via Stripe-Signature header.
func (h *WebhookHandler) HandleStripe(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to read webhook body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	event, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), h.webhookSecret)
	if err != nil {
		slog.WarnContext(r.Context(), "invalid stripe signature", "error", err)
		http.Error(w, "invalid signature", http.StatusBadRequest)
		return
	}

	domainEvent, err := mapStripeEvent(event)
	if err != nil {
		slog.ErrorContext(r.Context(), "failed to parse stripe event", "type", string(event.Type), "error", err)
		// Return 200 so Stripe does not retry events we cannot parse.
		w.WriteHeader(http.StatusOK)
		return
	}

	if err := h.webhookUC.Execute(r.Context(), domainEvent); err != nil {
		slog.ErrorContext(r.Context(), "failed to process webhook", "type", string(event.Type), "error", err)
		http.Error(w, "processing failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func mapStripeEvent(event stripe.Event) (handle_webhook.Event, error) {
	result := handle_webhook.Event{
		ID:   event.ID,
		Type: string(event.Type),
	}

	switch event.Type {
	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted",
		"customer.subscription.trial_will_end":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return result, err
		}
		result.Data = mapSubscriptionEventData(sub)

	case "invoice.payment_failed",
		"invoice.payment_succeeded":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return result, err
		}
		result.Data = mapInvoiceEventData(inv)
	}

	return result, nil
}

func mapSubscriptionEventData(sub stripe.Subscription) handle_webhook.EventData {
	data := handle_webhook.EventData{
		SubscriptionID: sub.ID,
		Status:         string(sub.Status),
	}

	if sub.Customer != nil {
		data.CustomerID = sub.Customer.ID
	}

	// In stripe-go v82, CurrentPeriodStart/End are on SubscriptionItem, not Subscription.
	if sub.Items != nil && len(sub.Items.Data) > 0 {
		item := sub.Items.Data[0]

		if item.Price != nil {
			data.PriceID = item.Price.ID
		}

		data.Quantity = item.Quantity

		if item.CurrentPeriodStart > 0 {
			t := time.Unix(item.CurrentPeriodStart, 0)
			data.CurrentPeriodStart = &t
		}
		if item.CurrentPeriodEnd > 0 {
			t := time.Unix(item.CurrentPeriodEnd, 0)
			data.CurrentPeriodEnd = &t
		}
	}

	return data
}

func mapInvoiceEventData(inv stripe.Invoice) handle_webhook.EventData {
	data := handle_webhook.EventData{}
	if inv.Customer != nil {
		data.InvoiceCustomerID = inv.Customer.ID
	}
	return data
}
