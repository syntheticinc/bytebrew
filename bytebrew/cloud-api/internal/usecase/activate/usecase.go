package activate

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
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

// Input is the activate request (userID and email come from auth context).
type Input struct {
	UserID string
	Email  string
}

// Output is the activate response.
type Output struct {
	LicenseJWT string
}

// Usecase handles license activation.
type Usecase struct {
	subReader     SubscriptionReader
	licenseSigner LicenseSigner
	teamReader    TeamByUserReader
}

// New creates a new Activate usecase.
func New(subReader SubscriptionReader, licenseSigner LicenseSigner, teamReader TeamByUserReader) *Usecase {
	return &Usecase{
		subReader:     subReader,
		licenseSigner: licenseSigner,
		teamReader:    teamReader,
	}
}

// Execute activates a license for the user.
// Returns an error if the user has no active subscription.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.UserID == "" {
		return nil, errors.InvalidInput("user ID is required")
	}
	if input.Email == "" {
		return nil, errors.InvalidInput("email is required")
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

	info := domain.LicenseInfo{
		UserID:              input.UserID,
		Email:               input.Email,
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
