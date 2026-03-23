package verify_email

import (
	"context"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// Consumer-side interfaces

// UserFinder looks up a user by a valid (non-expired) verification token.
type UserFinder interface {
	GetByVerificationToken(ctx context.Context, token string) (*domain.User, error)
}

// UserUpdater marks a user's email as verified.
type UserUpdater interface {
	SetEmailVerified(ctx context.Context, userID string) error
}

// TokenSigner signs authentication tokens for the verified user.
type TokenSigner interface {
	SignAccessToken(userID, email string) (string, error)
	SignRefreshToken(userID string) (string, error)
}

// Input is the verify email request.
type Input struct {
	Token string
}

// Output is the verify email response.
type Output struct {
	AccessToken  string
	RefreshToken string
	UserID       string
}

// Usecase handles email verification.
type Usecase struct {
	userFinder  UserFinder
	userUpdater UserUpdater
	tokenSigner TokenSigner
}

// New creates a new VerifyEmail usecase.
func New(userFinder UserFinder, userUpdater UserUpdater, tokenSigner TokenSigner) *Usecase {
	return &Usecase{
		userFinder:  userFinder,
		userUpdater: userUpdater,
		tokenSigner: tokenSigner,
	}
}

// Execute verifies the email using the token and returns auth tokens.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.Token == "" {
		return nil, errors.InvalidInput("verification token is required")
	}

	user, err := u.userFinder.GetByVerificationToken(ctx, input.Token)
	if err != nil {
		return nil, errors.Internal("get user by verification token", err)
	}
	if user == nil {
		return nil, errors.InvalidInput("invalid or expired verification token")
	}

	if err := u.userUpdater.SetEmailVerified(ctx, user.ID); err != nil {
		return nil, errors.Internal("set email verified", err)
	}

	accessToken, err := u.tokenSigner.SignAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, errors.Internal("sign access token", err)
	}

	refreshToken, err := u.tokenSigner.SignRefreshToken(user.ID)
	if err != nil {
		return nil, errors.Internal("sign refresh token", err)
	}

	return &Output{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserID:       user.ID,
	}, nil
}
