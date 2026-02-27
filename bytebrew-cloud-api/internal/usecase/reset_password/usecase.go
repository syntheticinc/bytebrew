package reset_password

import (
	"context"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
)

// Consumer-side interfaces

// UserByTokenReader looks up a user by a valid (non-expired) reset token.
type UserByTokenReader interface {
	GetByResetToken(ctx context.Context, token string) (*domain.User, error)
}

// PasswordResetUpdater atomically updates the password and clears the reset token.
type PasswordResetUpdater interface {
	UpdatePasswordAndClearResetToken(ctx context.Context, userID, newHash string) error
}

// PasswordHasher hashes passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// Input is the reset password request.
type Input struct {
	Token       string
	NewPassword string
}

// Usecase handles password reset using a valid token.
type Usecase struct {
	tokenReader UserByTokenReader
	updater     PasswordResetUpdater
	hasher      PasswordHasher
}

// New creates a new ResetPassword usecase.
func New(tokenReader UserByTokenReader, updater PasswordResetUpdater, hasher PasswordHasher) *Usecase {
	return &Usecase{
		tokenReader: tokenReader,
		updater:     updater,
		hasher:      hasher,
	}
}

// Execute resets the user's password using a one-time token.
func (u *Usecase) Execute(ctx context.Context, input Input) error {
	if input.Token == "" {
		return errors.InvalidInput("reset token is required")
	}
	if len(input.NewPassword) < 8 {
		return errors.InvalidInput("password must be at least 8 characters")
	}

	user, err := u.tokenReader.GetByResetToken(ctx, input.Token)
	if err != nil {
		return errors.Internal("get user by reset token", err)
	}
	if user == nil {
		return errors.InvalidInput("invalid or expired reset token")
	}

	newHash, err := u.hasher.Hash(input.NewPassword)
	if err != nil {
		return errors.Internal("hash new password", err)
	}

	if err := u.updater.UpdatePasswordAndClearResetToken(ctx, user.ID, newHash); err != nil {
		return errors.Internal("reset password", err)
	}

	return nil
}
