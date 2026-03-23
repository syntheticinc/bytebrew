package register

import (
	"context"
	"fmt"
	"testing"
	"time"

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

func (m *mockUserRepo) SetVerificationToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	return nil
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

type mockTokenGenerator struct{}

func (m *mockTokenGenerator) Generate() (string, error) {
	return "test-verification-token", nil
}

type mockEmailSender struct {
	sent bool
}

func (m *mockEmailSender) SendEmailVerification(ctx context.Context, to, verificationURL string) error {
	m.sent = true
	return nil
}

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name      string
		input     Input
		setupRepo func(*mockUserRepo)
		hashErr   error
		wantCode  string
		wantOut   *Output
	}{
		{
			name:  "happy path",
			input: Input{Email: "user@example.com", Password: "securepassword"},
			wantOut: &Output{
				UserID:  "generated-uuid",
				Message: "registration successful, please check your email to verify your account",
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

			emailSender := &mockEmailSender{}
			uc := New(
				repo,
				&mockPasswordHasher{hashErr: tt.hashErr},
				&mockTokenGenerator{},
				emailSender,
				"https://bytebrew.ai",
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
			if got.UserID != tt.wantOut.UserID {
				t.Errorf("UserID = %q, want %q", got.UserID, tt.wantOut.UserID)
			}
			if got.Message != tt.wantOut.Message {
				t.Errorf("Message = %q, want %q", got.Message, tt.wantOut.Message)
			}
			if tt.wantCode == "" && !emailSender.sent {
				t.Error("expected verification email to be sent")
			}
		})
	}
}
