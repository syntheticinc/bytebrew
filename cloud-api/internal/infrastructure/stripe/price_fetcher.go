package stripe

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/price"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/get_pricing"
)

// PriceFetcher fetches price details from Stripe with in-memory caching.
type PriceFetcher struct {
	mu    sync.RWMutex
	cache map[string]cachedPrice
	ttl   time.Duration
}

type cachedPrice struct {
	detail    *get_pricing.PriceDetail
	fetchedAt time.Time
}

// NewPriceFetcher creates a PriceFetcher with the given cache TTL.
func NewPriceFetcher(cacheTTL time.Duration) *PriceFetcher {
	return &PriceFetcher{
		cache: make(map[string]cachedPrice),
		ttl:   cacheTTL,
	}
}

// FetchPrice returns price details for a Stripe Price ID.
// Results are cached for the configured TTL.
func (f *PriceFetcher) FetchPrice(ctx context.Context, priceID string) (*get_pricing.PriceDetail, error) {
	if detail := f.fromCache(priceID); detail != nil {
		return detail, nil
	}

	params := &stripe.PriceParams{}
	params.Context = ctx

	p, err := price.Get(priceID, params)
	if err != nil {
		return nil, fmt.Errorf("stripe price.Get(%s): %w", priceID, err)
	}

	interval := ""
	if p.Recurring != nil {
		interval = string(p.Recurring.Interval)
	}

	detail := &get_pricing.PriceDetail{
		PriceID:  p.ID,
		Amount:   p.UnitAmount,
		Currency: string(p.Currency),
		Interval: interval,
	}

	f.toCache(priceID, detail)
	return detail, nil
}

func (f *PriceFetcher) fromCache(priceID string) *get_pricing.PriceDetail {
	f.mu.RLock()
	defer f.mu.RUnlock()

	entry, ok := f.cache[priceID]
	if !ok {
		return nil
	}
	if time.Since(entry.fetchedAt) > f.ttl {
		return nil
	}
	return entry.detail
}

func (f *PriceFetcher) toCache(priceID string, detail *get_pricing.PriceDetail) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.cache[priceID] = cachedPrice{
		detail:    detail,
		fetchedAt: time.Now(),
	}
}
