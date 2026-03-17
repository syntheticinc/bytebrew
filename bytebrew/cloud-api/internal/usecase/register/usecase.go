package register

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

// Consumer-side interfaces

// UserRepository provides user persistence operations needed by registration.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// AuthTokenSigner signs authentication tokens for newly registered users.
type AuthTokenSigner interface {
	SignAccessToken(userID, email string) (string, error)
	SignRefreshToken(userID string) (string, error)
}

// PasswordHasher hashes passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// Input is the register request.
type Input struct {
	Email    string
	Password string
}

// Output is the register response.
type Output struct {
	AccessToken  string
	RefreshToken string
	UserID       string
}

// Usecase handles user registration.
type Usecase struct {
	userRepo       UserRepository
	tokenSigner    AuthTokenSigner
	passwordHasher PasswordHasher
}

// New creates a new Register usecase.
func New(userRepo UserRepository, tokenSigner AuthTokenSigner, passwordHasher PasswordHasher) *Usecase {
	return &Usecase{
		userRepo:       userRepo,
		tokenSigner:    tokenSigner,
		passwordHasher: passwordHasher,
	}
}

// Execute registers a new user and returns auth tokens.
// No subscription is created — user must start Trial via Stripe.
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.Email == "" {
		return nil, errors.InvalidInput("email is required")
	}
	if len(input.Password) < 8 {
		return nil, errors.InvalidInput("password must be at least 8 characters")
	}

	existing, err := u.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		return nil, errors.Internal("check email uniqueness", err)
	}
	if existing != nil {
		return nil, errors.AlreadyExists("email already registered")
	}

	hash, err := u.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, errors.Internal("hash password", err)
	}

	user, err := domain.NewUser(input.Email, hash)
	if err != nil {
		return nil, errors.InvalidInput(err.Error())
	}

	created, err := u.userRepo.Create(ctx, user)
	if err != nil {
		return nil, errors.Internal("create user", err)
	}

	accessToken, err := u.tokenSigner.SignAccessToken(created.ID, created.Email)
	if err != nil {
		return nil, errors.Internal("sign access token", err)
	}

	refreshToken, err := u.tokenSigner.SignRefreshToken(created.ID)
	if err != nil {
		return nil, errors.Internal("sign refresh token", err)
	}

	return &Output{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		UserID:       created.ID,
	}, nil
}
