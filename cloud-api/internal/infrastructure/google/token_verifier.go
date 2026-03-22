package google

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/usecase/google_login"
)

// TokenVerifier verifies Google ID tokens using Google's public keys.
type TokenVerifier struct {
	clientID string
}

// NewTokenVerifier creates a new Google token verifier.
func NewTokenVerifier(clientID string) *TokenVerifier {
	return &TokenVerifier{clientID: clientID}
}

// Verify validates a Google ID token and extracts claims.
func (v *TokenVerifier) Verify(ctx context.Context, rawToken string) (*google_login.GoogleClaims, error) {
	payload, err := idtoken.Validate(ctx, rawToken, v.clientID)
	if err != nil {
		return nil, fmt.Errorf("validate google id token: %w", err)
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || email == "" {
		return nil, fmt.Errorf("google token missing email claim")
	}

	return &google_login.GoogleClaims{
		Sub:   payload.Subject,
		Email: email,
	}, nil
}
