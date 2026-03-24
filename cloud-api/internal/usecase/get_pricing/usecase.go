package get_pricing

import (
	"context"
	"fmt"
)

// PriceFetcher retrieves current price details from the payment provider.
type PriceFetcher interface {
	FetchPrice(ctx context.Context, priceID string) (*PriceDetail, error)
}

// PriceIDResolver maps plan+period combinations to price IDs.
type PriceIDResolver interface {
	PriceIDForPlan(plan, period string) (string, error)
}

// PriceDetail holds the details of a single price.
type PriceDetail struct {
	PriceID  string `json:"price_id"`
	Amount   int64  `json:"amount"`   // in smallest currency unit (cents)
	Currency string `json:"currency"` // e.g. "usd"
	Interval string `json:"interval"` // "month" or "year"
}

// PlanPricing holds monthly and annual pricing for a single plan.
type PlanPricing struct {
	Monthly *PriceDetail `json:"monthly,omitempty"`
	Annual  *PriceDetail `json:"annual,omitempty"`
}

// Output contains pricing for all configured plans.
type Output struct {
	Plans map[string]*PlanPricing `json:"plans"`
}

// Usecase fetches current pricing from the payment provider.
type Usecase struct {
	fetcher  PriceFetcher
	resolver PriceIDResolver
}

// New creates a new get_pricing Usecase.
func New(fetcher PriceFetcher, resolver PriceIDResolver) *Usecase {
	return &Usecase{
		fetcher:  fetcher,
		resolver: resolver,
	}
}

// planPeriod defines a plan + period combination to fetch.
type planPeriod struct {
	plan   string
	period string
}

// allPlanPeriods lists all plan+period combinations we want to expose pricing for.
var allPlanPeriods = []planPeriod{
	{"personal", "monthly"},
	{"personal", "annual"},
	{"teams", "monthly"},
	{"teams", "annual"},
	{"engine_ee", "monthly"},
	{"engine_ee", "annual"},
}

// Execute fetches pricing for all configured plans.
func (u *Usecase) Execute(ctx context.Context) (*Output, error) {
	plans := make(map[string]*PlanPricing)

	for _, pp := range allPlanPeriods {
		priceID, err := u.resolver.PriceIDForPlan(pp.plan, pp.period)
		if err != nil {
			// Plan not configured — skip
			continue
		}

		detail, err := u.fetcher.FetchPrice(ctx, priceID)
		if err != nil {
			return nil, fmt.Errorf("fetch price %s_%s: %w", pp.plan, pp.period, err)
		}

		if plans[pp.plan] == nil {
			plans[pp.plan] = &PlanPricing{}
		}

		switch pp.period {
		case "monthly":
			plans[pp.plan].Monthly = detail
		case "annual":
			plans[pp.plan].Annual = detail
		}
	}

	return &Output{Plans: plans}, nil
}
