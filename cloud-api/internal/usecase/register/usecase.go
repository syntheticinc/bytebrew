package register

import (
	"context"
	"time"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// Consumer-side interfaces

// UserRepository provides user persistence operations needed by registration.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	SetVerificationToken(ctx context.Context, userID, token string, expiresAt time.Time) error
}

// PasswordHasher hashes passwords.
type PasswordHasher interface {
	Hash(password string) (string, error)
}

// TokenGenerator generates cryptographically secure tokens.
type TokenGenerator interface {
	Generate() (string, error)
}

// EmailVerificationSender sends email verification emails.
type EmailVerificationSender interface {
	SendEmailVerification(ctx context.Context, to, verificationURL string) error
}

// Input is the register request.
type Input struct {
	Email    string
	Password string
}

// Output is the register response.
type Output struct {
	UserID  string
	Message string
}

// Usecase handles user registration.
type Usecase struct {
	userRepo       UserRepository
	passwordHasher PasswordHasher
	tokenGen       TokenGenerator
	emailSender    EmailVerificationSender
	frontendURL    string
}

// New creates a new Register usecase.
func New(
	userRepo UserRepository,
	passwordHasher PasswordHasher,
	tokenGen TokenGenerator,
	emailSender EmailVerificationSender,
	frontendURL string,
) *Usecase {
	return &Usecase{
		userRepo:       userRepo,
		passwordHasher: passwordHasher,
		tokenGen:       tokenGen,
		emailSender:    emailSender,
		frontendURL:    frontendURL,
	}
}

// Execute registers a new user and sends a verification email.
// No tokens are returned until the email is verified.
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

	token, err := u.tokenGen.Generate()
	if err != nil {
		return nil, errors.Internal("generate verification token", err)
	}

	expiresAt := time.Now().Add(24 * time.Hour)
	if err := u.userRepo.SetVerificationToken(ctx, created.ID, token, expiresAt); err != nil {
		return nil, errors.Internal("save verification token", err)
	}

	verificationURL := u.frontendURL + "/verify-email?token=" + token
	if err := u.emailSender.SendEmailVerification(ctx, created.Email, verificationURL); err != nil {
		return nil, errors.Internal("send verification email", err)
	}

	return &Output{
		UserID:  created.ID,
		Message: "registration successful, please check your email to verify your account",
	}, nil
}
