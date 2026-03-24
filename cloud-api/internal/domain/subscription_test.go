package domain

import (
	"strings"
	"testing"
)

func TestNewSubscription(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		tier        LicenseTier
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid trial subscription",
			userID:  "user-id",
			tier:    TierTrial,
			wantErr: false,
		},
		{
			name:    "valid personal subscription",
			userID:  "user-id",
			tier:    TierPersonal,
			wantErr: false,
		},
		{
			name:    "valid teams subscription",
			userID:  "user-id",
			tier:    TierTeams,
			wantErr: false,
		},
		{
			name:    "valid engine_ee subscription",
			userID:  "user-id",
			tier:    TierEngineEE,
			wantErr: false,
		},
		{
			name:        "empty user ID",
			userID:      "",
			tier:        TierTrial,
			wantErr:     true,
			errContains: "user ID is required",
		},
		{
			name:        "invalid tier",
			userID:      "user-id",
			tier:        "invalid",
			wantErr:     true,
			errContains: "invalid tier",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSubscription(tt.userID, tt.tier)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.UserID != tt.userID {
				t.Errorf("userID = %q, want %q", got.UserID, tt.userID)
			}
			if got.Tier != tt.tier {
				t.Errorf("tier = %q, want %q", got.Tier, tt.tier)
			}
			if got.Status != StatusActive {
				t.Errorf("status = %q, want %q", got.Status, StatusActive)
			}
			if got.CreatedAt.IsZero() {
				t.Error("createdAt should not be zero")
			}
			if got.UpdatedAt.IsZero() {
				t.Error("updatedAt should not be zero")
			}
		})
	}
}

func TestLicenseTier_IsValid(t *testing.T) {
	tests := []struct {
		tier LicenseTier
		want bool
	}{
		{TierTrial, true},
		{TierPersonal, true},
		{TierTeams, true},
		{TierEngineEE, true},
		{"invalid", false},
		{"", false},
		{"free", false},
		{"starter", false},
		{"pro", false},
		{"max", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			if got := tt.tier.IsValid(); got != tt.want {
				t.Errorf("LicenseTier(%q).IsValid() = %v, want %v", tt.tier, got, tt.want)
			}
		})
	}
}

func TestLicenseTier_IsPaid(t *testing.T) {
	tests := []struct {
		tier LicenseTier
		want bool
	}{
		{TierTrial, false},
		{TierPersonal, true},
		{TierTeams, true},
		{TierEngineEE, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			if got := tt.tier.IsPaid(); got != tt.want {
				t.Errorf("LicenseTier(%q).IsPaid() = %v, want %v", tt.tier, got, tt.want)
			}
		})
	}
}

func TestProxyStepsLimitForTier(t *testing.T) {
	tests := []struct {
		tier LicenseTier
		want int
	}{
		{TierTrial, 0},
		{TierPersonal, 300},
		{TierTeams, 300},
		{TierEngineEE, 300},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier), func(t *testing.T) {
			if got := ProxyStepsLimitForTier(tt.tier); got != tt.want {
				t.Errorf("ProxyStepsLimitForTier(%q) = %d, want %d", tt.tier, got, tt.want)
			}
		})
	}
}

func TestSubscriptionStatus_IsActive(t *testing.T) {
	tests := []struct {
		status SubscriptionStatus
		want   bool
	}{
		{StatusActive, true},
		{StatusTrialing, true},
		{StatusPastDue, false},
		{StatusCanceled, false},
		{StatusExpired, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsActive(); got != tt.want {
				t.Errorf("SubscriptionStatus(%q).IsActive() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
