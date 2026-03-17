package refresh_auth

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

// Consumer-side interfaces

// RefreshClaims represents the claims extracted from a refresh token.
type RefreshClaims struct {
	UserID string
}

// TokenVerifier verifies refresh tokens.
type TokenVerifier interface {
	VerifyRefreshToken(tokenString string) (*RefreshClaims, error)
}

// TokenSigner signs new access tokens.
type TokenSigner interface {
	SignAccessToken(userID, email string) (string, error)
}

// UserReader looks up users by ID.
type UserReader interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// Input is the refresh auth request.
type Input struct {
	RefreshToken string
}

// Output is the refresh auth response.
type Output struct {
	AccessToken string
}

// Usecase handles access token refresh using a refresh token.
type Usecase struct {
	tokenVerifier TokenVerifier
	tokenSigner   TokenSigner
	userReader    UserReader
}

// New creates a new RefreshAuth usecase.
func New(tokenVerifier TokenVerifier, tokenSigner TokenSigner, userReader UserReader) *Usecase {
	return &Usecase{
		tokenVerifier: tokenVerifier,
		tokenSigner:   tokenSigner,
		userReader:    userReader,
	}
}

// Execute verifies the refresh token and returns a new access token.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.RefreshToken == "" {
		return nil, errors.InvalidInput("refresh_token is required")
	}

	claims, err := u.tokenVerifier.VerifyRefreshToken(input.RefreshToken)
	if err != nil {
		return nil, errors.Unauthorized("invalid or expired refresh token")
	}

	user, err := u.userReader.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, errors.Internal("get user", err)
	}
	if user == nil {
		return nil, errors.NotFound("user not found")
	}

	accessToken, err := u.tokenSigner.SignAccessToken(user.ID, user.Email)
	if err != nil {
		return nil, errors.Internal("sign access token", err)
	}

	return &Output{
		AccessToken: accessToken,
	}, nil
}
