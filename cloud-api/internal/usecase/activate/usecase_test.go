package activate

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"

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

type mockLicenseSigner struct {
	jwt string
	err error
}

func (m *mockLicenseSigner) SignLicense(_ domain.LicenseInfo) (string, error) {
	return m.jwt, m.err
}

type mockTeamByUserReader struct {
	team *domain.Team
	err  error
}

func (m *mockTeamByUserReader) GetTeamByUserID(_ context.Context, _ string) (*domain.Team, error) {
	return m.team, m.err
}

// --- tests ---

func TestExecute(t *testing.T) {
	ctx := context.Background()

	periodEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		input     Input
		subReader *mockSubscriptionReader
		signer    *mockLicenseSigner
		wantJWT   string
		wantErr   string
		wantCode  string
	}{
		{
			name:     "empty user ID",
			input:    Input{UserID: "", Email: "user@test.com"},
			wantErr:  "user ID is required",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "empty email",
			input:    Input{UserID: "u-1", Email: ""},
			wantErr:  "email is required",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:      "no subscription returns error",
			input:     Input{UserID: "u-1", Email: "user@test.com"},
			subReader: &mockSubscriptionReader{sub: nil},
			wantErr:   "no active subscription",
			wantCode:  errors.CodeForbidden,
		},
		{
			name:  "active personal subscription with period end",
			input: Input{UserID: "u-1", Email: "user@test.com"},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &periodEnd,
			}},
			signer:  &mockLicenseSigner{jwt: "personal-jwt"},
			wantJWT: "personal-jwt",
		},
		{
			name:  "active subscription without period end uses default expiry",
			input: Input{UserID: "u-1", Email: "user@test.com"},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             TierTeams,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: nil,
			}},
			signer:  &mockLicenseSigner{jwt: "teams-jwt"},
			wantJWT: "teams-jwt",
		},
		{
			name:  "canceled subscription returns error",
			input: Input{UserID: "u-1", Email: "user@test.com"},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID: "u-1",
				Tier:   domain.TierPersonal,
				Status: domain.StatusCanceled,
			}},
			wantErr:  "no active subscription",
			wantCode: errors.CodeForbidden,
		},
		{
			name:      "subscription reader error",
			input:     Input{UserID: "u-1", Email: "user@test.com"},
			subReader: &mockSubscriptionReader{err: fmt.Errorf("db down")},
			wantErr:   "get subscription",
			wantCode:  errors.CodeInternal,
		},
		{
			name:      "signer error",
			input:     Input{UserID: "u-1", Email: "user@test.com"},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &periodEnd,
			}},
			signer:   &mockLicenseSigner{err: fmt.Errorf("sign failed")},
			wantErr:  "sign license",
			wantCode: errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subReader := tt.subReader
			if subReader == nil {
				subReader = &mockSubscriptionReader{}
			}
			signer := tt.signer
			if signer == nil {
				signer = &mockLicenseSigner{}
			}

			uc := New(subReader, signer, nil)
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
			assert.Equal(t, tt.wantJWT, got.LicenseJWT)
		})
	}
}

// TierTeams alias for use in tests within this package.
const TierTeams = domain.TierTeams

func TestExecute_SignedLicenseInfo(t *testing.T) {
	ctx := context.Background()

	periodEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	t.Run("personal tier with period end has correct info", func(t *testing.T) {
		var captured domain.LicenseInfo
		signer := &capturingSigner{jwt: "test-jwt", captured: &captured}

		uc := New(
			&mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &periodEnd,
				ProxyStepsUsed:   47,
				ProxyStepsLimit:  300,
				BYOKEnabled:      true,
			}},
			signer,
			nil,
		)

		_, err := uc.Execute(ctx, Input{UserID: "u-1", Email: "user@test.com"})
		require.NoError(t, err)

		assert.Equal(t, domain.TierPersonal, captured.Tier)
		assert.Equal(t, periodEnd, captured.ExpiresAt)
		assert.Equal(t, domain.GraceFromExpiry(periodEnd), captured.GraceUntil)
		assert.True(t, captured.Features.FullAutonomy)
		assert.Equal(t, -1, captured.Features.ParallelAgents)
		assert.True(t, captured.Features.ExploreCodebase)
		assert.Equal(t, 253, captured.ProxyStepsRemaining)
		assert.Equal(t, 300, captured.ProxyStepsLimit)
		assert.True(t, captured.BYOKEnabled)
	})

	t.Run("trial tier has full features", func(t *testing.T) {
		var captured domain.LicenseInfo
		signer := &capturingSigner{jwt: "test-jwt", captured: &captured}

		uc := New(
			&mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierTrial,
				Status:           domain.StatusTrialing,
				CurrentPeriodEnd: &periodEnd,
			}},
			signer,
			nil,
		)

		_, err := uc.Execute(ctx, Input{UserID: "u-1", Email: "user@test.com"})
		require.NoError(t, err)

		assert.Equal(t, domain.TierTrial, captured.Tier)
		assert.True(t, captured.Features.FullAutonomy)
		assert.Equal(t, -1, captured.Features.ParallelAgents)
		assert.True(t, captured.Features.ExploreCodebase)
		assert.Equal(t, 1, captured.MaxSeats)
	})

	t.Run("teams tier sets MaxSeats from team", func(t *testing.T) {
		var captured domain.LicenseInfo
		signer := &capturingSigner{jwt: "test-jwt", captured: &captured}

		uc := New(
			&mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierTeams,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &periodEnd,
				ProxyStepsLimit:  300,
			}},
			signer,
			&mockTeamByUserReader{team: &domain.Team{MaxSeats: 10}},
		)

		_, err := uc.Execute(ctx, Input{UserID: "u-1", Email: "user@test.com"})
		require.NoError(t, err)

		assert.Equal(t, domain.TierTeams, captured.Tier)
		assert.Equal(t, 10, captured.MaxSeats)
	})

	t.Run("personal tier has MaxSeats 1", func(t *testing.T) {
		var captured domain.LicenseInfo
		signer := &capturingSigner{jwt: "test-jwt", captured: &captured}

		uc := New(
			&mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &periodEnd,
			}},
			signer,
			&mockTeamByUserReader{team: &domain.Team{MaxSeats: 10}},
		)

		_, err := uc.Execute(ctx, Input{UserID: "u-1", Email: "user@test.com"})
		require.NoError(t, err)

		assert.Equal(t, 1, captured.MaxSeats)
	})
}

// capturingSigner captures the LicenseInfo passed to SignLicense.
type capturingSigner struct {
	jwt      string
	captured *domain.LicenseInfo
}

func (s *capturingSigner) SignLicense(info domain.LicenseInfo) (string, error) {
	*s.captured = info
	return s.jwt, nil
}
