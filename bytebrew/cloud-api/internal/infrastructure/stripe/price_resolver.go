package stripe

import (
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/config"
)

// PriceResolver maps Stripe Price IDs to domain tiers and vice versa.
type PriceResolver struct {
	priceToTier map[string]domain.LicenseTier
	planPrices  map[string]string // "personal_monthly" -> "price_xxx"
}

// NewPriceResolver creates a PriceResolver from config.
func NewPriceResolver(cfg config.StripePricesConfig) *PriceResolver {
	priceToTier := make(map[string]domain.LicenseTier)
	planPrices := make(map[string]string)

	prices := map[string]struct {
		priceID string
		tier    domain.LicenseTier
	}{
		"personal_monthly": {cfg.PersonalMonthly, domain.TierPersonal},
		"personal_annual":  {cfg.PersonalAnnual, domain.TierPersonal},
		"teams_monthly":    {cfg.TeamsMonthly, domain.TierTeams},
		"teams_annual":     {cfg.TeamsAnnual, domain.TierTeams},
	}

	for key, p := range prices {
		if p.priceID == "" {
			continue
		}
		priceToTier[p.priceID] = p.tier
		planPrices[key] = p.priceID
	}

	return &PriceResolver{
		priceToTier: priceToTier,
		planPrices:  planPrices,
	}
}

// TierForPriceID returns the domain tier for a Stripe Price ID.
func (r *PriceResolver) TierForPriceID(priceID string) (domain.LicenseTier, bool) {
	tier, ok := r.priceToTier[priceID]
	return tier, ok
}

// PriceIDForPlan returns the Stripe Price ID for a plan+period combination.
// plan: "personal", "teams"; period: "monthly", "annual".
func (r *PriceResolver) PriceIDForPlan(plan, period string) (string, error) {
	key := plan + "_" + period
	priceID, ok := r.planPrices[key]
	if !ok {
		return "", fmt.Errorf("no price configured for %s", key)
	}
	return priceID, nil
}
