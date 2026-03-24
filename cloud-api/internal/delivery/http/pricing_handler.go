package http

import (
	"context"
	"net/http"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/get_pricing"
)

type pricingUsecase interface {
	Execute(ctx context.Context) (*get_pricing.Output, error)
}

// PricingHandler handles public pricing endpoints.
type PricingHandler struct {
	pricingUC pricingUsecase
}

// NewPricingHandler creates a new PricingHandler.
func NewPricingHandler(pricingUC pricingUsecase) *PricingHandler {
	return &PricingHandler{
		pricingUC: pricingUC,
	}
}

// GetPricing handles GET /api/v1/pricing.
func (h *PricingHandler) GetPricing(w http.ResponseWriter, r *http.Request) {
	out, err := h.pricingUC.Execute(r.Context())
	if err != nil {
		Error(w, err)
		return
	}

	JSON(w, http.StatusOK, out.Plans)
}
