package forgot_password

import (
	"context"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
)

// Consumer-side interfaces

// UserReader provides user lookup by email needed by password reset initiation.
type UserReader interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// ResetTokenSaver persists a password reset token with expiration.
type ResetTokenSaver interface {
	SetResetToken(ctx context.Context, userID, token string, expiresAt time.Time) error
}

// EmailSender sends password reset emails.
type EmailSender interface {
	SendPasswordReset(ctx context.Context, email, token string) error
}

// TokenGenerator generates cryptographically secure tokens.
type TokenGenerator interface {
	Generate() (string, error)
}

// Input is the forgot password request.
type Input struct {
	Email string
}

// Usecase handles password reset initiation by generating a token and sending an email.
type Usecase struct {
	userReader UserReader
	tokenSaver ResetTokenSaver
	email      EmailSender
	tokenGen   TokenGenerator
	tokenTTL   time.Duration
}

// New creates a new ForgotPassword usecase.
func New(userReader UserReader, tokenSaver ResetTokenSaver, email EmailSender, tokenGen TokenGenerator, tokenTTL time.Duration) *Usecase {
	return &Usecase{
		userReader: userReader,
		tokenSaver: tokenSaver,
		email:      email,
		tokenGen:   tokenGen,
		tokenTTL:   tokenTTL,
	}
}

// Execute initiates a password reset flow.
// Returns nil even if the email is not registered to avoid leaking account existence.
func (u *Usecase) Execute(ctx context.Context, input Input) error {
	if input.Email == "" {
		return errors.InvalidInput("email is required")
	}

	user, err := u.userReader.GetByEmail(ctx, input.Email)
	if err != nil {
		return errors.Internal("get user by email", err)
	}
	if user == nil {
		return nil // Do not reveal whether the account exists
	}

	token, err := u.tokenGen.Generate()
	if err != nil {
		return errors.Internal("generate reset token", err)
	}

	expiresAt := time.Now().Add(u.tokenTTL)
	if err := u.tokenSaver.SetResetToken(ctx, user.ID, token, expiresAt); err != nil {
		return errors.Internal("save reset token", err)
	}

	if err := u.email.SendPasswordReset(ctx, input.Email, token); err != nil {
		return errors.Internal("send reset email", err)
	}

	return nil
}
