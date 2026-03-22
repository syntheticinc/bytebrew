package refresh_auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// --- Mocks ---

type mockTokenVerifier struct {
	claims *RefreshClaims
	err    error
}

func (m *mockTokenVerifier) VerifyRefreshToken(tokenString string) (*RefreshClaims, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.claims, nil
}

type mockTokenSigner struct {
	err error
}

func (m *mockTokenSigner) SignAccessToken(userID, email string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return "access-" + userID, nil
}

type mockUserReader struct {
	user *domain.User
	err  error
}

func (m *mockUserReader) GetByID(ctx context.Context, id string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name     string
		input    Input
		verifier *mockTokenVerifier
		signer   *mockTokenSigner
		reader   *mockUserReader
		wantCode string
		wantOut  *Output
	}{
		{
			name:  "valid refresh token returns new access token",
			input: Input{RefreshToken: "valid-refresh-token"},
			verifier: &mockTokenVerifier{
				claims: &RefreshClaims{UserID: "user-1"},
			},
			signer: &mockTokenSigner{},
			reader: &mockUserReader{
				user: &domain.User{ID: "user-1", Email: "user@example.com"},
			},
			wantOut: &Output{AccessToken: "access-user-1"},
		},
		{
			name:     "empty refresh token",
			input:    Input{RefreshToken: ""},
			verifier: &mockTokenVerifier{},
			signer:   &mockTokenSigner{},
			reader:   &mockUserReader{},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:  "invalid or expired refresh token",
			input: Input{RefreshToken: "bad-token"},
			verifier: &mockTokenVerifier{
				err: fmt.Errorf("token expired"),
			},
			signer:   &mockTokenSigner{},
			reader:   &mockUserReader{},
			wantCode: errors.CodeUnauthorized,
		},
		{
			name:  "user not found",
			input: Input{RefreshToken: "valid-token"},
			verifier: &mockTokenVerifier{
				claims: &RefreshClaims{UserID: "deleted-user"},
			},
			signer:   &mockTokenSigner{},
			reader:   &mockUserReader{user: nil},
			wantCode: errors.CodeNotFound,
		},
		{
			name:  "user reader returns error",
			input: Input{RefreshToken: "valid-token"},
			verifier: &mockTokenVerifier{
				claims: &RefreshClaims{UserID: "user-1"},
			},
			signer:   &mockTokenSigner{},
			reader:   &mockUserReader{err: fmt.Errorf("db connection error")},
			wantCode: errors.CodeInternal,
		},
		{
			name:  "token sign failure",
			input: Input{RefreshToken: "valid-token"},
			verifier: &mockTokenVerifier{
				claims: &RefreshClaims{UserID: "user-1"},
			},
			signer: &mockTokenSigner{err: fmt.Errorf("signing error")},
			reader: &mockUserReader{
				user: &domain.User{ID: "user-1", Email: "user@example.com"},
			},
			wantCode: errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(tt.verifier, tt.signer, tt.reader)

			got, err := uc.Execute(context.Background(), tt.input)

			if tt.wantCode != "" {
				if err == nil {
					t.Fatalf("expected error with code %s, got nil", tt.wantCode)
				}
				if !errors.Is(err, tt.wantCode) {
					t.Fatalf("expected error code %s, got: %v", tt.wantCode, err)
				}
				if got != nil {
					t.Fatalf("expected nil output on error, got: %+v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("expected non-nil output")
			}
			if got.AccessToken != tt.wantOut.AccessToken {
				t.Errorf("AccessToken = %q, want %q", got.AccessToken, tt.wantOut.AccessToken)
			}
		})
	}
}
