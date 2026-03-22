package google_login

import (
	"context"
	"log/slog"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// GoogleTokenVerifier verifies a Google ID token and extracts claims.
type GoogleTokenVerifier interface {
	Verify(ctx context.Context, idToken string) (*GoogleClaims, error)
}

// GoogleClaims holds the verified claims from a Google ID token.
type GoogleClaims struct {
	Sub   string // Google user ID
	Email string
}

// UserRepository provides user persistence operations needed by Google login.
type UserRepository interface {
	GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	CreateGoogleUser(ctx context.Context, email, googleID string) (*domain.User, error)
	LinkGoogleID(ctx context.Context, userID, googleID string) error
}

// AuthTokenSigner signs authentication tokens.
type AuthTokenSigner interface {
	SignAccessToken(userID, email string) (string, error)
	SignRefreshToken(userID string) (string, error)
}

// Input is the Google login request.
type Input struct {
	IDToken string
}

// Output is the Google login response.
type Output struct {
	AccessToken  string
	RefreshToken string
	UserID       string
}

// Usecase handles Google OAuth login.
type Usecase struct {
	tokenVerifier GoogleTokenVerifier
	userRepo      UserRepository
	tokenSigner   AuthTokenSigner
}

// New creates a new Google login usecase.
func New(tokenVerifier GoogleTokenVerifier, userRepo UserRepository, tokenSigner AuthTokenSigner) *Usecase {
	return &Usecase{
		tokenVerifier: tokenVerifier,
		userRepo:      userRepo,
		tokenSigner:   tokenSigner,
	}
}

// Execute verifies a Google ID token and returns auth tokens.
// Flow:
//  1. Verify ID token
//  2. Lookup user by Google ID -> login
//  3. Lookup user by email -> link Google ID + login
//  4. No user found -> create new Google user + login
func (u *Usecase) Execute(ctx context.Context, input Input) (*Output, error) {
	if input.IDToken == "" {
		return nil, errors.InvalidInput("id_token is required")
	}

	claims, err := u.tokenVerifier.Verify(ctx, input.IDToken)
	if err != nil {
		return nil, errors.Unauthorized("invalid google token")
	}

	// Try to find existing user by Google ID.
	user, err := u.userRepo.GetByGoogleID(ctx, claims.Sub)
	if err != nil {
		return nil, errors.Internal("get user by google id", err)
	}
	if user != nil {
		return u.signTokens(user)
	}

	// Try to find existing user by email (account linking).
	user, err = u.userRepo.GetByEmail(ctx, claims.Email)
	if err != nil {
		return nil, errors.Internal("get user by email", err)
	}
	if user != nil {
		if err := u.userRepo.LinkGoogleID(ctx, user.ID, claims.Sub); err != nil {
			return nil, errors.Internal("link google id", err)
		}
		slog.InfoContext(ctx, "linked google account to existing user", "user_id", user.ID, "email", claims.Email)
		return u.signTokens(user)
	}

	// Create a new Google-only user.
	user, err = u.userRepo.CreateGoogleUser(ctx, claims.Email, claims.Sub)
	if err != nil {
		return nil, errors.Internal("create google user", err)
	}
	slog.InfoContext(ctx, "created new google user", "user_id", user.ID, "email", claims.Email)

	return u.signTokens(user)
}

func (u *Usecase) signTokens(user *domain.User) (*Output, error) {
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
