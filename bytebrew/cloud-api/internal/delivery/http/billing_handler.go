package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/delivery/http/middleware"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/create_checkout"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/create_portal"
)

type checkoutUsecase interface {
	Execute(ctx context.Context, input create_checkout.Input) (*create_checkout.Output, error)
}

type portalUsecase interface {
	Execute(ctx context.Context, input create_portal.Input) (*create_portal.Output, error)
}

// BillingHandler handles billing endpoints.
type BillingHandler struct {
	checkoutUC checkoutUsecase
	portalUC   portalUsecase
}

// NewBillingHandler creates a new BillingHandler.
func NewBillingHandler(checkoutUC checkoutUsecase, portalUC portalUsecase) *BillingHandler {
	return &BillingHandler{
		checkoutUC: checkoutUC,
		portalUC:   portalUC,
	}
}

type checkoutRequest struct {
	Plan   string `json:"plan"`
	Period string `json:"period"`
}

type checkoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

type portalResponse struct {
	PortalURL string `json:"portal_url"`
}

// Checkout handles POST /api/v1/billing/checkout.
func (h *BillingHandler) Checkout(w http.ResponseWriter, r *http.Request) {
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Error(w, invalidBodyError)
		return
	}

	out, err := h.checkoutUC.Execute(r.Context(), create_checkout.Input{
		UserID: middleware.GetUserID(r.Context()),
		Email:  middleware.GetEmail(r.Context()),
		Plan:   req.Plan,
		Period: req.Period,
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, checkoutResponse{CheckoutURL: out.CheckoutURL})
}

// Portal handles POST /api/v1/billing/portal.
func (h *BillingHandler) Portal(w http.ResponseWriter, r *http.Request) {
	out, err := h.portalUC.Execute(r.Context(), create_portal.Input{
		UserID: middleware.GetUserID(r.Context()),
	})
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, portalResponse{PortalURL: out.PortalURL})
}
