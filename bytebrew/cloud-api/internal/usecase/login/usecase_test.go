package login

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

// --- Mocks ---

type mockUserReader struct {
	users map[string]*domain.User
}

func newMockUserReader() *mockUserReader {
	return &mockUserReader{users: make(map[string]*domain.User)}
}

func (m *mockUserReader) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.users[email], nil
}

func (m *mockUserReader) addUser(email, password string) {
	m.users[email] = &domain.User{
		ID:           "user-123",
		Email:        email,
		PasswordHash: "hashed-" + password,
	}
}

type mockTokenSigner struct{}

func (m *mockTokenSigner) SignAccessToken(userID, email string) (string, error) {
	return "access-" + userID, nil
}

func (m *mockTokenSigner) SignRefreshToken(userID string) (string, error) {
	return "refresh-" + userID, nil
}

type mockPasswordHasher struct {
	compareErr error
}

func (m *mockPasswordHasher) Compare(hash, password string) error {
	if m.compareErr != nil {
		return m.compareErr
	}
	// Simple mock: hash should be "hashed-" + password
	if hash == "hashed-"+password {
		return nil
	}
	return fmt.Errorf("password mismatch")
}

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name      string
		input     Input
		setupRepo func(*mockUserReader)
		wantCode  string
		wantOut   *Output
	}{
		{
			name:  "happy path",
			input: Input{Email: "user@example.com", Password: "securepassword"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user@example.com", "securepassword")
			},
			wantOut: &Output{
				AccessToken:  "access-user-123",
				RefreshToken: "refresh-user-123",
				UserID:       "user-123",
			},
		},
		{
			name:     "empty email",
			input:    Input{Email: "", Password: "securepassword"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "empty password",
			input:    Input{Email: "user@example.com", Password: ""},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "user not found",
			input:    Input{Email: "nobody@example.com", Password: "securepassword"},
			wantCode: errors.CodeUnauthorized,
		},
		{
			name:  "wrong password",
			input: Input{Email: "user@example.com", Password: "wrongpassword"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user@example.com", "securepassword")
			},
			wantCode: errors.CodeUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := newMockUserReader()
			if tt.setupRepo != nil {
				tt.setupRepo(reader)
			}

			uc := New(reader, &mockTokenSigner{}, &mockPasswordHasher{})

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
			if got.RefreshToken != tt.wantOut.RefreshToken {
				t.Errorf("RefreshToken = %q, want %q", got.RefreshToken, tt.wantOut.RefreshToken)
			}
			if got.UserID != tt.wantOut.UserID {
				t.Errorf("UserID = %q, want %q", got.UserID, tt.wantOut.UserID)
			}
		})
	}
}
