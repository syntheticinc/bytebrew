package domain

import (
	"errors"
	"testing"
	"time"
)

func TestFeaturesForTier(t *testing.T) {
	tests := []struct {
		name string
		tier LicenseTier
	}{
		{"trial tier has full features", TierTrial},
		{"personal tier has full features", TierPersonal},
		{"teams tier has full features", TierTeams},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FeaturesForTier(tt.tier)

			if !got.FullAutonomy {
				t.Error("FullAutonomy should be true")
			}
			if got.ParallelAgents != -1 {
				t.Errorf("ParallelAgents = %d, want -1", got.ParallelAgents)
			}
			if !got.ExploreCodebase {
				t.Error("ExploreCodebase should be true")
			}
			if !got.TraceSymbol {
				t.Error("TraceSymbol should be true")
			}
			if !got.CodebaseIndexing {
				t.Error("CodebaseIndexing should be true")
			}
		})
	}
}

func TestGraceFromExpiry(t *testing.T) {
	expiry := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	got := GraceFromExpiry(expiry)
	want := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC) // 3 days, not 30

	if !got.Equal(want) {
		t.Errorf("GraceFromExpiry(%v) = %v, want %v", expiry, got, want)
	}
}

func TestDefaultLicenseExpiry(t *testing.T) {
	before := time.Now().Add(30 * 24 * time.Hour)
	got := DefaultLicenseExpiry()
	after := time.Now().Add(30 * 24 * time.Hour)

	if got.Before(before.Add(-time.Second)) || got.After(after.Add(time.Second)) {
		t.Errorf("DefaultLicenseExpiry() = %v, expected ~30 days from now", got)
	}
}

func TestResolveTierAndExpiry(t *testing.T) {
	periodEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		sub        *Subscription
		wantTier   LicenseTier
		wantExpiry *time.Time // nil means check default (30 days from now)
		wantErr    error
	}{
		{
			name:    "nil subscription returns error",
			sub:     nil,
			wantErr: ErrNoActiveSubscription,
		},
		{
			name: "active subscription with period end",
			sub: &Subscription{
				Tier:             TierPersonal,
				Status:           StatusActive,
				CurrentPeriodEnd: &periodEnd,
			},
			wantTier:   TierPersonal,
			wantExpiry: &periodEnd,
		},
		{
			name: "active subscription without period end uses default",
			sub: &Subscription{
				Tier:   TierPersonal,
				Status: StatusActive,
			},
			wantTier: TierPersonal,
		},
		{
			name: "trialing subscription preserves tier",
			sub: &Subscription{
				Tier:             TierPersonal,
				Status:           StatusTrialing,
				CurrentPeriodEnd: &periodEnd,
			},
			wantTier:   TierPersonal,
			wantExpiry: &periodEnd,
		},
		{
			name: "past_due preserves paid tier (grace period)",
			sub: &Subscription{
				Tier:             TierPersonal,
				Status:           StatusPastDue,
				CurrentPeriodEnd: &periodEnd,
			},
			wantTier:   TierPersonal,
			wantExpiry: &periodEnd,
		},
		{
			name: "past_due without period end uses default expiry",
			sub: &Subscription{
				Tier:   TierTeams,
				Status: StatusPastDue,
			},
			wantTier: TierTeams,
		},
		{
			name: "canceled subscription returns error",
			sub: &Subscription{
				Tier:   TierPersonal,
				Status: StatusCanceled,
			},
			wantErr: ErrNoActiveSubscription,
		},
		{
			name: "expired subscription returns error",
			sub: &Subscription{
				Tier:   TierPersonal,
				Status: StatusExpired,
			},
			wantErr: ErrNoActiveSubscription,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTier, gotExpiry, err := ResolveTierAndExpiry(tt.sub)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if gotTier != tt.wantTier {
				t.Errorf("tier = %q, want %q", gotTier, tt.wantTier)
			}

			if tt.wantExpiry != nil {
				if !gotExpiry.Equal(*tt.wantExpiry) {
					t.Errorf("expiry = %v, want %v", gotExpiry, *tt.wantExpiry)
				}
			} else {
				// Should be approximately 30 days from now
				expected := time.Now().Add(30 * 24 * time.Hour)
				diff := gotExpiry.Sub(expected)
				if diff < -time.Second || diff > time.Second {
					t.Errorf("expiry = %v, want ~%v (default)", gotExpiry, expected)
				}
			}
		})
	}
}
