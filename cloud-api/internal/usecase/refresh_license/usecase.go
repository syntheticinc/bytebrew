package refresh_license

import (
	"context"
	"time"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// Consumer-side interfaces

// SubscriptionReader looks up a user's subscription.
type SubscriptionReader interface {
	GetByUserID(ctx context.Context, userID string) (*domain.Subscription, error)
}

// LicenseSigner signs license JWTs.
type LicenseSigner interface {
	SignLicense(info domain.LicenseInfo) (string, error)
}

// TeamByUserReader reads the team for a user (used to resolve MaxSeats for Teams tier).
type TeamByUserReader interface {
	GetTeamByUserID(ctx context.Context, userID string) (*domain.Team, error)
}

// LicenseClaims holds the verified license data needed by the usecase.
type LicenseClaims struct {
	Subject             string
	Email               string
	Tier                string
	ExpiresAt           *time.Time
	ProxyStepsRemaining int
}

// LicenseVerifier verifies license JWT tokens.
type LicenseVerifier interface {
	VerifyLicense(tokenString string) (*LicenseClaims, error)
}

// Input is the refresh request.
type Input struct {
	UserID         string
	Email          string
	CurrentLicense string
}

// Output is the refresh response.
type Output struct {
	LicenseJWT string
}

// Usecase handles license refresh.
type Usecase struct {
	subReader       SubscriptionReader
	licenseSigner   LicenseSigner
	licenseVerifier LicenseVerifier
	teamReader      TeamByUserReader
}

// New creates a new RefreshLicense usecase.
func New(subReader SubscriptionReader, licenseSigner LicenseSigner, licenseVerifier LicenseVerifier, teamReader TeamByUserReader) *Usecase {
	return &Usecase{
		subReader:       subReader,
		licenseSigner:   licenseSigner,
		licenseVerifier: licenseVerifier,
		teamReader:      teamReader,
	}
}

// Execute refreshes a license if the subscription has changed since the current token was issued.
// Returns the current license unchanged when tier and expiry match.
// Returns an error if the user has no active subscription.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.UserID == "" {
		return nil, errors.InvalidInput("user ID is required")
	}
	if input.CurrentLicense == "" {
		return nil, errors.InvalidInput("current license is required")
	}

	currentClaims, err := u.licenseVerifier.VerifyLicense(input.CurrentLicense)
	if err != nil {
		return nil, errors.Unauthorized("invalid license token")
	}

	if currentClaims.Subject != input.UserID {
		return nil, errors.Forbidden("license does not belong to user")
	}

	sub, err := u.subReader.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, errors.Internal("get subscription", err)
	}

	tier, expiresAt, err := domain.ResolveTierAndExpiry(sub)
	if err != nil {
		return nil, errors.Forbidden("no active subscription")
	}

	remaining := sub.ProxyStepsLimit - sub.ProxyStepsUsed
	if remaining < 0 {
		remaining = 0
	}

	if !hasChanged(currentClaims, tier, expiresAt, remaining) {
		return &Output{LicenseJWT: input.CurrentLicense}, nil
	}

	email := input.Email
	if email == "" {
		email = currentClaims.Email
	}

	info := domain.LicenseInfo{
		UserID:              input.UserID,
		Email:               email,
		Tier:                tier,
		ExpiresAt:           expiresAt,
		GraceUntil:          domain.GraceFromExpiry(expiresAt),
		Features:            domain.FeaturesForTier(tier),
		ProxyStepsRemaining: remaining,
		ProxyStepsLimit:     sub.ProxyStepsLimit,
		BYOKEnabled:         sub.BYOKEnabled,
		MaxSeats:            1,
	}

	if tier == domain.TierTeams && u.teamReader != nil {
		team, teamErr := u.teamReader.GetTeamByUserID(ctx, input.UserID)
		if teamErr == nil && team != nil {
			info.MaxSeats = team.MaxSeats
		}
	}

	jwt, err := u.licenseSigner.SignLicense(info)
	if err != nil {
		return nil, errors.Internal("sign license", err)
	}

	return &Output{LicenseJWT: jwt}, nil
}

// hasChanged checks whether the subscription state differs from the current license claims.
func hasChanged(claims *LicenseClaims, tier domain.LicenseTier, expiresAt time.Time, proxyStepsRemaining int) bool {
	if string(tier) != claims.Tier {
		return true
	}

	if claims.ExpiresAt == nil {
		return true
	}

	diff := expiresAt.Sub(*claims.ExpiresAt)
	if diff < -time.Minute || diff > time.Minute {
		return true
	}

	if proxyStepsRemaining != claims.ProxyStepsRemaining {
		return true
	}

	return false
}
