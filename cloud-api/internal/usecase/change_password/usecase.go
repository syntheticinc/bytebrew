package change_password

import (
	"context"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// Consumer-side interfaces

// UserReader provides user lookup needed by password change.
type UserReader interface {
	GetByID(ctx context.Context, id string) (*domain.User, error)
}

// PasswordUpdater persists a new password hash.
type PasswordUpdater interface {
	UpdatePassword(ctx context.Context, userID, newHash string) error
}

// PasswordHasher hashes and verifies passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

// Input is the change password request.
type Input struct {
	UserID          string
	CurrentPassword string
	NewPassword     string
}

// Usecase handles password change for authenticated users.
type Usecase struct {
	userReader  UserReader
	passUpdater PasswordUpdater
	hasher      PasswordHasher
}

// New creates a new ChangePassword usecase.
func New(userReader UserReader, passUpdater PasswordUpdater, hasher PasswordHasher) *Usecase {
	return &Usecase{
		userReader:  userReader,
		passUpdater: passUpdater,
		hasher:      hasher,
	}
}

// Execute changes the user's password after verifying the current one.
func (u *Usecase) Execute(ctx context.Context, input Input) error {
	if len(input.NewPassword) < 8 {
		return errors.InvalidInput("password must be at least 8 characters")
	}

	user, err := u.userReader.GetByID(ctx, input.UserID)
	if err != nil {
		return errors.Internal("get user", err)
	}
	if user == nil {
		return errors.NotFound("user not found")
	}

	if err := u.hasher.Compare(user.PasswordHash, input.CurrentPassword); err != nil {
		return errors.Unauthorized("current password is incorrect")
	}

	newHash, err := u.hasher.Hash(input.NewPassword)
	if err != nil {
		return errors.Internal("hash new password", err)
	}

	if err := u.passUpdater.UpdatePassword(ctx, input.UserID, newHash); err != nil {
		return errors.Internal("update password", err)
	}

	return nil
}
