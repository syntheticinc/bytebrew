package login

import (
	"context"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// Consumer-side interfaces

// UserReader provides user lookup needed by login.
type UserReader interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// AuthTokenSigner signs authentication tokens for authenticated users.
type AuthTokenSigner interface {
	SignAccessToken(userID, email string) (string, error)
	SignRefreshToken(userID string) (string, error)
}

// PasswordHasher verifies passwords against hashes.
type PasswordHasher interface {
	Compare(hash, password string) error
}

// Input is the login request.
type Input struct {
	Email    string
	Password string
}

// Output is the login response.
type Output struct {
	AccessToken  string
	RefreshToken string
	UserID       string
}

// Usecase handles user login.
type Usecase struct {
	userReader     UserReader
	tokenSigner    AuthTokenSigner
	passwordHasher PasswordHasher
}

// New creates a new Login usecase.
func New(userReader UserReader, tokenSigner AuthTokenSigner, passwordHasher PasswordHasher) *Usecase {
	return &Usecase{
		userReader:     userReader,
		tokenSigner:    tokenSigner,
		passwordHasher: passwordHasher,
	}
}

// Execute authenticates a user and returns tokens.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.Email == "" {
		return nil, errors.InvalidInput("email is required")
	}
	if input.Password == "" {
		return nil, errors.InvalidInput("password is required")
	}

	user, err := u.userReader.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, errors.Internal("get user", err)
	}
	if user == nil {
		return nil, errors.Unauthorized("invalid credentials")
	}

	if user.IsGoogleOnly() {
		return nil, errors.Unauthorized("this account uses Google sign-in, please use Google login")
	}

	if err := u.passwordHasher.Compare(user.PasswordHash, input.Password); err != nil {
		return nil, errors.Unauthorized("invalid credentials")
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
