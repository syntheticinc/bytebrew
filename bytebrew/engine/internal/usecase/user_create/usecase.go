package user_create

import (
	"context"
	"log/slog"

	"github.com/syntheticinc/bytebrew/bytebrew-srv/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-srv/pkg/errors"
)

// UserRepository defines interface for user persistence operations
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByUsername(ctx context.Context, username string) (*domain.User, error)
}

// Input represents input for user creation
type Input struct {
	Username     string
	Email        string
	PasswordHash string
}

// Output represents output from user creation
type Output struct {
	UserID string
}

// Usecase handles user creation
type Usecase struct {
	userRepo UserRepository
}

// New creates a new User Create use case
func New(userRepo UserRepository) (*Usecase, error) {
	if userRepo == nil {
		return nil, errors.New(errors.CodeInvalidInput, "user repository is required")
	}

	return &Usecase{
		userRepo: userRepo,
	}, nil
}

// Execute creates a new user
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	slog.InfoContext(ctx, "creating user", "username", input.Username, "email", input.Email)

	if input.Username == "" {
		return nil, errors.New(errors.CodeInvalidInput, "username is required")
	}
	if input.Email == "" {
		return nil, errors.New(errors.CodeInvalidInput, "email is required")
	}
	if input.PasswordHash == "" {
		return nil, errors.New(errors.CodeInvalidInput, "password hash is required")
	}

	// Check if user already exists
	existingUser, err := u.userRepo.GetByEmail(ctx, input.Email)
	if err != nil && !errors.Is(err, errors.CodeNotFound) {
		slog.ErrorContext(ctx, "failed to check existing user by email", "error", err)
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to check existing user")
	}
	if existingUser != nil {
		return nil, errors.New(errors.CodeAlreadyExists, "user with this email already exists")
	}

	existingUser, err = u.userRepo.GetByUsername(ctx, input.Username)
	if err != nil && !errors.Is(err, errors.CodeNotFound) {
		slog.ErrorContext(ctx, "failed to check existing user by username", "error", err)
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to check existing user")
	}
	if existingUser != nil {
		return nil, errors.New(errors.CodeAlreadyExists, "user with this username already exists")
	}

	// Create domain entity
	user, err := domain.NewUser(input.Username, input.Email, input.PasswordHash)
	if err != nil {
		slog.ErrorContext(ctx, "failed to create user entity", "error", err)
		return nil, errors.Wrap(err, errors.CodeInvalidInput, "invalid user data")
	}

	// Save user
	if err := u.userRepo.Create(ctx, user); err != nil {
		slog.ErrorContext(ctx, "failed to save user", "error", err)
		return nil, errors.Wrap(err, errors.CodeInternal, "failed to save user")
	}

	slog.InfoContext(ctx, "user created successfully", "user_id", user.ID)

	return &Output{
		UserID: user.ID,
	}, nil
}
