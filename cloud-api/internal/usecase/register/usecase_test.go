package register

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/pkg/errors"
)

// --- Mocks ---

type mockUserRepo struct {
	users    map[string]*domain.User
	createFn func(ctx context.Context, user *domain.User) (*domain.User, error)
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, user)
	}
	created := &domain.User{
		ID:           "generated-uuid",
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
	}
	m.users[user.Email] = created
	return created, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return m.users[email], nil
}

type mockTokenSigner struct {
	accessErr  error
	refreshErr error
}

func (m *mockTokenSigner) SignAccessToken(userID, email string) (string, error) {
	if m.accessErr != nil {
		return "", m.accessErr
	}
	return "access-" + userID, nil
}

func (m *mockTokenSigner) SignRefreshToken(userID string) (string, error) {
	if m.refreshErr != nil {
		return "", m.refreshErr
	}
	return "refresh-" + userID, nil
}

type mockPasswordHasher struct {
	hashResult string
	hashErr    error
}

func (m *mockPasswordHasher) Hash(password string) (string, error) {
	if m.hashErr != nil {
		return "", m.hashErr
	}
	if m.hashResult != "" {
		return m.hashResult, nil
	}
	return "hashed-" + password, nil
}

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name       string
		input      Input
		setupRepo  func(*mockUserRepo)
		accessErr  error
		refreshErr error
		hashErr    error
		wantCode   string
		wantOut    *Output
	}{
		{
			name:  "happy path",
			input: Input{Email: "user@example.com", Password: "securepassword"},
			wantOut: &Output{
				AccessToken:  "access-generated-uuid",
				RefreshToken: "refresh-generated-uuid",
				UserID:       "generated-uuid",
			},
		},
		{
			name:     "empty email",
			input:    Input{Email: "", Password: "securepassword"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "short password",
			input:    Input{Email: "user@example.com", Password: "short"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:  "duplicate email",
			input: Input{Email: "existing@example.com", Password: "securepassword"},
			setupRepo: func(repo *mockUserRepo) {
				repo.users["existing@example.com"] = &domain.User{
					ID:    "existing-id",
					Email: "existing@example.com",
				}
			},
			wantCode: errors.CodeAlreadyExists,
		},
		{
			name:      "access token signing fails",
			input:     Input{Email: "user@example.com", Password: "securepassword"},
			accessErr: fmt.Errorf("signing error"),
			wantCode:  errors.CodeInternal,
		},
		{
			name:       "refresh token signing fails",
			input:      Input{Email: "user@example.com", Password: "securepassword"},
			refreshErr: fmt.Errorf("signing error"),
			wantCode:   errors.CodeInternal,
		},
		{
			name:     "password hashing fails",
			input:    Input{Email: "user@example.com", Password: "securepassword"},
			hashErr:  fmt.Errorf("hash error"),
			wantCode: errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockUserRepo()
			if tt.setupRepo != nil {
				tt.setupRepo(repo)
			}

			uc := New(
				repo,
				&mockTokenSigner{accessErr: tt.accessErr, refreshErr: tt.refreshErr},
				&mockPasswordHasher{hashErr: tt.hashErr},
			)

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
