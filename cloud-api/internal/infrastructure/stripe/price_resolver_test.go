package stripe

import (
	"testing"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTierForPriceID(t *testing.T) {
	r := NewPriceResolver(config.StripePricesConfig{
		PersonalMonthly: "price_personal_m",
		PersonalAnnual:  "price_personal_a",
		TeamsMonthly:    "price_teams_m",
		TeamsAnnual:     "price_teams_a",
		EngineEEMonthly: "price_engine_ee_m",
		EngineEEAnnual:  "price_engine_ee_a",
	})

	tests := []struct {
		priceID string
		want    domain.LicenseTier
		ok      bool
	}{
		{"price_personal_m", domain.TierPersonal, true},
		{"price_personal_a", domain.TierPersonal, true},
		{"price_teams_m", domain.TierTeams, true},
		{"price_teams_a", domain.TierTeams, true},
		{"price_engine_ee_m", domain.TierEngineEE, true},
		{"price_engine_ee_a", domain.TierEngineEE, true},
		{"price_unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.priceID, func(t *testing.T) {
			tier, ok := r.TierForPriceID(tt.priceID)
			assert.Equal(t, tt.ok, ok)
			if ok {
				assert.Equal(t, tt.want, tier)
			}
		})
	}
}

func TestPriceIDForPlan(t *testing.T) {
	r := NewPriceResolver(config.StripePricesConfig{
		PersonalMonthly: "price_personal_m",
		PersonalAnnual:  "price_personal_a",
		TeamsMonthly:    "price_teams_m",
		TeamsAnnual:     "price_teams_a",
		EngineEEMonthly: "price_engine_ee_m",
		EngineEEAnnual:  "price_engine_ee_a",
	})

	tests := []struct {
		plan    string
		period  string
		want    string
		wantErr bool
	}{
		{"personal", "monthly", "price_personal_m", false},
		{"personal", "annual", "price_personal_a", false},
		{"teams", "monthly", "price_teams_m", false},
		{"teams", "annual", "price_teams_a", false},
		{"engine_ee", "monthly", "price_engine_ee_m", false},
		{"engine_ee", "annual", "price_engine_ee_a", false},
		{"enterprise", "monthly", "", true},
		{"personal", "weekly", "", true},
		{"starter", "monthly", "", true},
		{"pro", "annual", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.plan+"_"+tt.period, func(t *testing.T) {
			got, err := r.PriceIDForPlan(tt.plan, tt.period)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPriceResolver_SkipsEmptyPriceIDs(t *testing.T) {
	r := NewPriceResolver(config.StripePricesConfig{
		PersonalMonthly: "price_personal",
		// all others empty
	})

	t.Run("empty string not mapped", func(t *testing.T) {
		_, ok := r.TierForPriceID("")
		assert.False(t, ok)
	})

	t.Run("configured price works", func(t *testing.T) {
		tier, ok := r.TierForPriceID("price_personal")
		assert.True(t, ok)
		assert.Equal(t, domain.TierPersonal, tier)
	})

	t.Run("unconfigured plan returns error", func(t *testing.T) {
		_, err := r.PriceIDForPlan("teams", "monthly")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no price configured")
	})

	t.Run("configured plan works", func(t *testing.T) {
		priceID, err := r.PriceIDForPlan("personal", "monthly")
		require.NoError(t, err)
		assert.Equal(t, "price_personal", priceID)
	})
}
