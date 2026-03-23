package resend_verification

import (
	"context"
	"time"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// Consumer-side interfaces

// UserFinder looks up a user by email.
type UserFinder interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

// UserUpdater stores a new verification token.
type UserUpdater interface {
	SetVerificationToken(ctx context.Context, userID, token string, expiresAt time.Time) error
}

// TokenGenerator generates cryptographically secure tokens.
type TokenGenerator interface {
	Generate() (string, error)
}

// EmailSender sends email verification emails.
type EmailSender interface {
	SendEmailVerification(ctx context.Context, to, verificationURL string) error
}

// Input is the resend verification request.
type Input struct {
	Email string
}

// Usecase handles resending the email verification.
type Usecase struct {
	userFinder  UserFinder
	userUpdater UserUpdater
	tokenGen    TokenGenerator
	emailSender EmailSender
	frontendURL string
}

// New creates a new ResendVerification usecase.
func New(
	userFinder UserFinder,
	userUpdater UserUpdater,
	tokenGen TokenGenerator,
	emailSender EmailSender,
	frontendURL string,
) *Usecase {
	return &Usecase{
		userFinder:  userFinder,
		userUpdater: userUpdater,
		tokenGen:    tokenGen,
		emailSender: emailSender,
		frontendURL: frontendURL,
	}
}

// Execute resends the verification email.
// Returns nil even if the email is not registered to avoid leaking account existence.
func (u *Usecase) Execute(ctx context.Context, input Input) error {
	if input.Email == "" {
		return errors.InvalidInput("email is required")
	}

	user, err := u.userFinder.GetByEmail(ctx, input.Email)
	if err != nil {
		return errors.Internal("get user by email", err)
	}
	if user == nil {
		return nil // Do not reveal whether the account exists
	}

	if user.EmailVerified {
		return errors.InvalidInput("email is already verified")
	}

	token, err := u.tokenGen.Generate()
	if err != nil {
		return errors.Internal("generate verification token", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if err := u.userUpdater.SetVerificationToken(ctx, user.ID, token, expiresAt); err != nil {
		return errors.Internal("save verification token", err)
	}

	verificationURL := u.frontendURL + "/verify-email?token=" + token
	if err := u.emailSender.SendEmailVerification(ctx, input.Email, verificationURL); err != nil {
		return errors.Internal("send verification email", err)
	}

	return nil
}
