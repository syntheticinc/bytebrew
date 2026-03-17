package refresh_license

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

type mockLicenseVerifier struct {
	claims *LicenseClaims
	err    error
}

func (m *mockLicenseVerifier) VerifyLicense(_ string) (*LicenseClaims, error) {
	return m.claims, m.err
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
	otherPeriodEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	personalClaims := &LicenseClaims{
		Subject:   "u-1",
		Email:     "user@test.com",
		Tier:      string(domain.TierPersonal),
		ExpiresAt: &periodEnd,
	}

	trialClaims := &LicenseClaims{
		Subject:   "u-1",
		Email:     "user@test.com",
		Tier:      string(domain.TierTrial),
		ExpiresAt: &periodEnd,
	}

	otherUserClaims := &LicenseClaims{
		Subject:   "u-other",
		Email:     "other@test.com",
		Tier:      string(domain.TierTrial),
		ExpiresAt: &periodEnd,
	}

	tests := []struct {
		name        string
		input       Input
		verifier    *mockLicenseVerifier
		subReader   *mockSubscriptionReader
		signerJWT   string
		signerErr   error
		wantJWT     string
		wantSameJWT bool // true if we expect the current license returned unchanged
		wantErr     string
		wantCode    string
	}{
		{
			name:     "empty user ID",
			input:    Input{UserID: "", CurrentLicense: "some-license"},
			wantErr:  "user ID is required",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "empty current license",
			input:    Input{UserID: "u-1", CurrentLicense: ""},
			wantErr:  "current license is required",
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "invalid license token",
			input:    Input{UserID: "u-1", CurrentLicense: "invalid.jwt.token"},
			verifier: &mockLicenseVerifier{err: fmt.Errorf("invalid token")},
			wantErr:  "invalid license token",
			wantCode: errors.CodeUnauthorized,
		},
		{
			name:     "license belongs to different user",
			input:    Input{UserID: "u-1", CurrentLicense: "other-user-license"},
			verifier: &mockLicenseVerifier{claims: otherUserClaims},
			wantErr:  "license does not belong to user",
			wantCode: errors.CodeForbidden,
		},
		{
			name:  "no change returns current license",
			input: Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "current-license"},
			verifier: &mockLicenseVerifier{claims: personalClaims},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &periodEnd,
			}},
			wantSameJWT: true,
		},
		{
			name:  "tier changed issues new license",
			input: Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "trial-license"},
			verifier: &mockLicenseVerifier{claims: trialClaims},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &periodEnd,
			}},
			signerJWT: "new-personal-jwt",
			wantJWT:   "new-personal-jwt",
		},
		{
			name:  "period end changed issues new license",
			input: Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "personal-license"},
			verifier: &mockLicenseVerifier{claims: personalClaims},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &otherPeriodEnd,
			}},
			signerJWT: "new-period-jwt",
			wantJWT:   "new-period-jwt",
		},
		{
			name:  "subscription canceled returns error",
			input: Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "personal-license"},
			verifier: &mockLicenseVerifier{claims: personalClaims},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID: "u-1",
				Tier:   domain.TierPersonal,
				Status: domain.StatusCanceled,
			}},
			wantErr:  "no active subscription",
			wantCode: errors.CodeForbidden,
		},
		{
			name:      "no subscription returns error",
			input:     Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "personal-license"},
			verifier:  &mockLicenseVerifier{claims: personalClaims},
			subReader: &mockSubscriptionReader{sub: nil},
			wantErr:   "no active subscription",
			wantCode:  errors.CodeForbidden,
		},
		{
			name:      "subscription reader error",
			input:     Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "trial-license"},
			verifier:  &mockLicenseVerifier{claims: trialClaims},
			subReader: &mockSubscriptionReader{err: fmt.Errorf("db down")},
			wantErr:   "get subscription",
			wantCode:  errors.CodeInternal,
		},
		{
			name:  "signer error",
			input: Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "trial-license"},
			verifier: &mockLicenseVerifier{claims: trialClaims},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &periodEnd,
			}},
			signerErr: fmt.Errorf("sign failed"),
			wantErr:   "sign license",
			wantCode:  errors.CodeInternal,
		},
		{
			name:  "past_due preserves paid tier",
			input: Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "trial-license"},
			verifier: &mockLicenseVerifier{claims: trialClaims},
			subReader: &mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusPastDue,
				CurrentPeriodEnd: &periodEnd,
			}},
			signerJWT: "personal-jwt",
			wantJWT:   "personal-jwt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subReader := tt.subReader
			if subReader == nil {
				subReader = &mockSubscriptionReader{}
			}
			verifier := tt.verifier
			if verifier == nil {
				verifier = &mockLicenseVerifier{claims: trialClaims}
			}
			signer := &mockSigner{jwt: tt.signerJWT, err: tt.signerErr}

			uc := New(subReader, signer, verifier, nil)
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
			if tt.wantSameJWT {
				assert.Equal(t, tt.input.CurrentLicense, got.LicenseJWT, "expected current license returned unchanged")
			} else {
				assert.Equal(t, tt.wantJWT, got.LicenseJWT)
			}
		})
	}
}

func TestExecute_FallbackEmail(t *testing.T) {
	ctx := context.Background()
	periodEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	newPeriodEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	claims := &LicenseClaims{
		Subject:   "u-1",
		Email:     "original@test.com",
		Tier:      string(domain.TierTrial),
		ExpiresAt: &periodEnd,
	}

	var captured domain.LicenseInfo
	signer := &capturingSigner{jwt: "new-jwt", captured: &captured}

	uc := New(
		&mockSubscriptionReader{sub: &domain.Subscription{
			UserID:           "u-1",
			Tier:             domain.TierPersonal,
			Status:           domain.StatusActive,
			CurrentPeriodEnd: &newPeriodEnd,
		}},
		signer,
		&mockLicenseVerifier{claims: claims},
		nil,
	)

	// Empty Email in input should fallback to email from current claims.
	_, err := uc.Execute(ctx, Input{UserID: "u-1", Email: "", CurrentLicense: "some-license"})
	require.NoError(t, err)
	assert.Equal(t, "original@test.com", captured.Email)
}

func TestExecute_TeamsMaxSeats(t *testing.T) {
	ctx := context.Background()
	periodEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	newPeriodEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	t.Run("teams tier sets MaxSeats from team", func(t *testing.T) {
		claims := &LicenseClaims{
			Subject:   "u-1",
			Email:     "user@test.com",
			Tier:      string(domain.TierPersonal),
			ExpiresAt: &periodEnd,
		}

		var captured domain.LicenseInfo
		signer := &capturingSigner{jwt: "teams-jwt", captured: &captured}

		uc := New(
			&mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierTeams,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &newPeriodEnd,
				ProxyStepsLimit:  300,
			}},
			signer,
			&mockLicenseVerifier{claims: claims},
			&mockTeamByUserReader{team: &domain.Team{MaxSeats: 15}},
		)

		_, err := uc.Execute(ctx, Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "old-license"})
		require.NoError(t, err)

		assert.Equal(t, domain.TierTeams, captured.Tier)
		assert.Equal(t, 15, captured.MaxSeats)
	})

	t.Run("personal tier has MaxSeats 1 even with team reader", func(t *testing.T) {
		claims := &LicenseClaims{
			Subject:   "u-1",
			Email:     "user@test.com",
			Tier:      string(domain.TierTrial),
			ExpiresAt: &periodEnd,
		}

		var captured domain.LicenseInfo
		signer := &capturingSigner{jwt: "personal-jwt", captured: &captured}

		uc := New(
			&mockSubscriptionReader{sub: &domain.Subscription{
				UserID:           "u-1",
				Tier:             domain.TierPersonal,
				Status:           domain.StatusActive,
				CurrentPeriodEnd: &newPeriodEnd,
			}},
			signer,
			&mockLicenseVerifier{claims: claims},
			&mockTeamByUserReader{team: &domain.Team{MaxSeats: 15}},
		)

		_, err := uc.Execute(ctx, Input{UserID: "u-1", Email: "user@test.com", CurrentLicense: "old-license"})
		require.NoError(t, err)

		assert.Equal(t, 1, captured.MaxSeats)
	})
}

// --- mock signer ---

type mockSigner struct {
	jwt string
	err error
}

func (m *mockSigner) SignLicense(_ domain.LicenseInfo) (string, error) {
	return m.jwt, m.err
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
