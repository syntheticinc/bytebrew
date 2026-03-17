package http

import (
	"context"
	"net/http"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/delivery/http/middleware"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/usecase/get_usage"
)

// usageUseCase defines the interface consumed by UsageHandler.
type usageUseCase interface {
	Execute(ctx context.Context, in get_usage.Input) (*get_usage.Output, error)
}

// UsageHandler handles subscription usage endpoints.
type UsageHandler struct {
	usecase usageUseCase
}

// NewUsageHandler creates a new UsageHandler.
func NewUsageHandler(uc usageUseCase) *UsageHandler {
	return &UsageHandler{usecase: uc}
}

type usageResponse struct {
	Tier                string  `json:"tier"`
	ProxyStepsUsed      int     `json:"proxy_steps_used"`
	ProxyStepsLimit     int     `json:"proxy_steps_limit"`
	ProxyStepsRemaining int     `json:"proxy_steps_remaining"`
	BYOKEnabled         bool    `json:"byok_enabled"`
	CurrentPeriodEnd    *string `json:"current_period_end,omitempty"`
}

// GetUsage handles GET /api/v1/subscription/usage.
func (h *UsageHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	out, err := h.usecase.Execute(r.Context(), get_usage.Input{
		UserID: userID,
	})
	if err != nil {
		Error(w, err)
		return
	}

	resp := usageResponse{
		Tier:                out.Tier,
		ProxyStepsUsed:      out.ProxyStepsUsed,
		ProxyStepsLimit:     out.ProxyStepsLimit,
		ProxyStepsRemaining: out.ProxyStepsRemaining,
		BYOKEnabled:         out.BYOKEnabled,
	}
	if out.CurrentPeriodEnd != nil {
		s := out.CurrentPeriodEnd.Format(time.RFC3339)
		resp.CurrentPeriodEnd = &s
	}

	JSON(w, http.StatusOK, resp)
}
