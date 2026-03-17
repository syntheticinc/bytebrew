package get_usage

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockSubscriptionReader struct {
	sub *domain.Subscription
	err error
}

func (m *mockSubscriptionReader) GetByUserID(_ context.Context, _ string) (*domain.Subscription, error) {
	return m.sub, m.err
}

// --- tests ---

func TestExecute(t *testing.T) {
	ctx := context.Background()
	periodEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		input     Input
		subReader *mockSubscriptionReader
		want      *Output
		wantErr   string
		wantCode  string
	}{
		{
			name:     "empty user ID",
			input:    Input{UserID: ""},
			wantErr:  "user_id is required",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:      "no subscription returns error",
			input:     Input{UserID: "u-1"},
			subReader: &mockSubscriptionReader{sub: nil},
			wantErr:   "no active subscription",
			wantCode:  errors.CodeForbidden,
		},
		{
			name:      "subscription reader error",
			input:     Input{UserID: "u-1"},
			subReader: &mockSubscriptionReader{err: fmt.Errorf("db down")},
			wantErr:   "get subscription",
			wantCode:  errors.CodeInternal,
		},
		{
			name:  "personal subscription with usage",
			input: Input{UserID: "u-1"},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				ProxyStepsUsed:   47,
				ProxyStepsLimit:  300,
				BYOKEnabled:      true,
				CurrentPeriodEnd: &periodEnd,
			}},
			want: &Output{
				Tier:                "personal",
				ProxyStepsUsed:      47,
				ProxyStepsLimit:     300,
				ProxyStepsRemaining: 253,
				BYOKEnabled:         true,
				CurrentPeriodEnd:    &periodEnd,
			},
		},
		{
			name:  "trial subscription with zero limit",
			input: Input{UserID: "u-1"},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:          "u-1",
				Tier:            domain.TierTrial,
				Status:          domain.StatusTrialing,
				ProxyStepsUsed:  5,
				ProxyStepsLimit: 0,
				BYOKEnabled:     true,
			}},
			want: &Output{
				Tier:                "trial",
				ProxyStepsUsed:      5,
				ProxyStepsLimit:     0,
				ProxyStepsRemaining: 0, // 0 - 5 = -5, clamped to 0
				BYOKEnabled:         true,
				CurrentPeriodEnd:    nil,
			},
		},
		{
			name:  "over limit clamps remaining to zero",
			input: Input{UserID: "u-1"},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				ProxyStepsUsed:   350,
				ProxyStepsLimit:  300,
				BYOKEnabled:      true,
				CurrentPeriodEnd: &periodEnd,
			}},
			want: &Output{
				Tier:                "personal",
				ProxyStepsUsed:      350,
				ProxyStepsLimit:     300,
				ProxyStepsRemaining: 0,
				BYOKEnabled:         true,
				CurrentPeriodEnd:    &periodEnd,
			},
		},
		{
			name:  "teams subscription without period end",
			input: Input{UserID: "u-1"},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:          "u-1",
				Tier:            domain.TierTeams,
				Status:          domain.StatusActive,
				ProxyStepsUsed:  0,
				ProxyStepsLimit: 300,
				BYOKEnabled:     true,
			}},
			want: &Output{
				Tier:                "teams",
				ProxyStepsUsed:      0,
				ProxyStepsLimit:     300,
				ProxyStepsRemaining: 300,
				BYOKEnabled:         true,
				CurrentPeriodEnd:    nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subReader := tt.subReader
			if subReader == nil {
				subReader = &mockSubscriptionReader{}
			}

			uc := New(subReader)
			got, err := uc.Execute(ctx, tt.input)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				if tt.wantCode != "" {
					assert.True(t, errors.Is(err, tt.wantCode), "expected code %s, got %s", tt.wantCode, errors.GetCode(err))
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want.Tier, got.Tier)
			assert.Equal(t, tt.want.ProxyStepsUsed, got.ProxyStepsUsed)
			assert.Equal(t, tt.want.ProxyStepsLimit, got.ProxyStepsLimit)
			assert.Equal(t, tt.want.ProxyStepsRemaining, got.ProxyStepsRemaining)
			assert.Equal(t, tt.want.BYOKEnabled, got.BYOKEnabled)
			assert.Equal(t, tt.want.CurrentPeriodEnd, got.CurrentPeriodEnd)
		})
	}
}
