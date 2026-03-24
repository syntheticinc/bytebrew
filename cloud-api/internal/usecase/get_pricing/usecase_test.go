package get_pricing

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockPriceFetcher struct {
	prices map[string]*PriceDetail
}

func (m *mockPriceFetcher) FetchPrice(_ context.Context, priceID string) (*PriceDetail, error) {
	p, ok := m.prices[priceID]
	if !ok {
		return nil, fmt.Errorf("price not found: %s", priceID)
	}
	return p, nil
}

type mockPriceIDResolver struct {
	mapping map[string]string // "plan_period" -> priceID
}

func (m *mockPriceIDResolver) PriceIDForPlan(plan, period string) (string, error) {
	key := plan + "_" + period
	id, ok := m.mapping[key]
	if !ok {
		return "", fmt.Errorf("no price configured for %s", key)
	}
	return id, nil
}

// --- tests ---

func TestExecute_AllPlansConfigured(t *testing.T) {
	fetcher := &mockPriceFetcher{
		prices: map[string]*PriceDetail{
			"price_personal_m": {PriceID: "price_personal_m", Amount: 2000, Currency: "usd", Interval: "month"},
			"price_personal_a": {PriceID: "price_personal_a", Amount: 20000, Currency: "usd", Interval: "year"},
			"price_teams_m":    {PriceID: "price_teams_m", Amount: 3000, Currency: "usd", Interval: "month"},
			"price_teams_a":    {PriceID: "price_teams_a", Amount: 30000, Currency: "usd", Interval: "year"},
			"price_ee_m":       {PriceID: "price_ee_m", Amount: 49900, Currency: "usd", Interval: "month"},
			"price_ee_a":       {PriceID: "price_ee_a", Amount: 499000, Currency: "usd", Interval: "year"},
		},
	}

	resolver := &mockPriceIDResolver{
		mapping: map[string]string{
			"personal_monthly":  "price_personal_m",
			"personal_annual":   "price_personal_a",
			"teams_monthly":     "price_teams_m",
			"teams_annual":      "price_teams_a",
			"engine_ee_monthly": "price_ee_m",
			"engine_ee_annual":  "price_ee_a",
		},
	}

	uc := New(fetcher, resolver)
	out, err := uc.Execute(context.Background())
	require.NoError(t, err)

	assert.Len(t, out.Plans, 3)

	// Personal
	assert.NotNil(t, out.Plans["personal"])
	assert.Equal(t, int64(2000), out.Plans["personal"].Monthly.Amount)
	assert.Equal(t, "month", out.Plans["personal"].Monthly.Interval)
	assert.Equal(t, int64(20000), out.Plans["personal"].Annual.Amount)
	assert.Equal(t, "year", out.Plans["personal"].Annual.Interval)

	// Teams
	assert.NotNil(t, out.Plans["teams"])
	assert.Equal(t, int64(3000), out.Plans["teams"].Monthly.Amount)
	assert.Equal(t, int64(30000), out.Plans["teams"].Annual.Amount)

	// Engine EE
	assert.NotNil(t, out.Plans["engine_ee"])
	assert.Equal(t, int64(49900), out.Plans["engine_ee"].Monthly.Amount)
	assert.Equal(t, int64(499000), out.Plans["engine_ee"].Annual.Amount)
}

func TestExecute_PartialConfig(t *testing.T) {
	fetcher := &mockPriceFetcher{
		prices: map[string]*PriceDetail{
			"price_ee_m": {PriceID: "price_ee_m", Amount: 49900, Currency: "usd", Interval: "month"},
		},
	}

	resolver := &mockPriceIDResolver{
		mapping: map[string]string{
			"engine_ee_monthly": "price_ee_m",
			// only engine_ee monthly configured
		},
	}

	uc := New(fetcher, resolver)
	out, err := uc.Execute(context.Background())
	require.NoError(t, err)

	// Only engine_ee should be present
	assert.Len(t, out.Plans, 1)
	assert.NotNil(t, out.Plans["engine_ee"])
	assert.NotNil(t, out.Plans["engine_ee"].Monthly)
	assert.Nil(t, out.Plans["engine_ee"].Annual)
}

func TestExecute_NoPricesConfigured(t *testing.T) {
	fetcher := &mockPriceFetcher{prices: map[string]*PriceDetail{}}
	resolver := &mockPriceIDResolver{mapping: map[string]string{}}

	uc := New(fetcher, resolver)
	out, err := uc.Execute(context.Background())
	require.NoError(t, err)
	assert.Empty(t, out.Plans)
}

func TestExecute_FetchError(t *testing.T) {
	fetcher := &mockPriceFetcher{
		prices: map[string]*PriceDetail{}, // empty — will return error
	}

	resolver := &mockPriceIDResolver{
		mapping: map[string]string{
			"personal_monthly": "price_missing",
		},
	}

	uc := New(fetcher, resolver)
	_, err := uc.Execute(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fetch price personal_monthly")
}

func TestExecute_CurrencyPreserved(t *testing.T) {
	fetcher := &mockPriceFetcher{
		prices: map[string]*PriceDetail{
			"price_eur": {PriceID: "price_eur", Amount: 1990, Currency: "eur", Interval: "month"},
		},
	}

	resolver := &mockPriceIDResolver{
		mapping: map[string]string{
			"personal_monthly": "price_eur",
		},
	}

	uc := New(fetcher, resolver)
	out, err := uc.Execute(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "eur", out.Plans["personal"].Monthly.Currency)
}
