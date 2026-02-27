package forgot_password

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew-cloud-api/pkg/errors"
)

// --- Mocks ---

type mockUserReader struct {
	users map[string]*domain.User
	err   error
}

func newMockUserReader() *mockUserReader {
	return &mockUserReader{users: make(map[string]*domain.User)}
}

func (m *mockUserReader) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users[email], nil
}

func (m *mockUserReader) addUser(email string) {
	m.users[email] = &domain.User{
		ID:    "user-" + email,
		Email: email,
	}
}

type mockResetTokenSaver struct {
	saved []string
	err   error
}

func (m *mockResetTokenSaver) SetResetToken(_ context.Context, userID, token string, _ time.Time) error {
	if m.err != nil {
		return m.err
	}
	m.saved = append(m.saved, userID+":"+token)
	return nil
}

type mockEmailSender struct {
	sent []string
	err  error
}

func (m *mockEmailSender) SendPasswordReset(_ context.Context, email, token string) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, email+":"+token)
	return nil
}

type mockTokenGenerator struct {
	token string
	err   error
}

func (m *mockTokenGenerator) Generate() (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.token, nil
}

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name       string
		input      Input
		setupRepo  func(*mockUserReader)
		readerErr  error
		token      string
		tokenErr   error
		saverErr   error
		senderErr  error
		wantCode   string
		wantNilErr bool // expect nil error even though no user exists (anti-enumeration)
		wantSent   bool
		wantSaved  bool
	}{
		{
			name:  "success",
			input: Input{Email: "alice@example.com"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("alice@example.com")
			},
			token:     "reset-token-abc",
			wantSent:  true,
			wantSaved: true,
		},
		{
			name:     "empty email",
			input:    Input{Email: ""},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:       "user not found returns nil",
			input:      Input{Email: "unknown@example.com"},
			wantNilErr: true,
		},
		{
			name:      "user reader error",
			input:     Input{Email: "alice@example.com"},
			readerErr: fmt.Errorf("db connection lost"),
			wantCode:  errors.CodeInternal,
		},
		{
			name:  "token generation error",
			input: Input{Email: "alice@example.com"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("alice@example.com")
			},
			tokenErr: fmt.Errorf("entropy exhausted"),
			wantCode: errors.CodeInternal,
		},
		{
			name:  "save token error",
			input: Input{Email: "alice@example.com"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("alice@example.com")
			},
			token:    "reset-token-abc",
			saverErr: fmt.Errorf("db error"),
			wantCode: errors.CodeInternal,
		},
		{
			name:  "send email error",
			input: Input{Email: "alice@example.com"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("alice@example.com")
			},
			token:     "reset-token-abc",
			senderErr: fmt.Errorf("smtp failure"),
			wantCode:  errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := newMockUserReader()
			reader.err = tt.readerErr
			if tt.setupRepo != nil {
				tt.setupRepo(reader)
			}

			saver := &mockResetTokenSaver{err: tt.saverErr}
			sender := &mockEmailSender{err: tt.senderErr}
			tokenGen := &mockTokenGenerator{token: tt.token, err: tt.tokenErr}

			uc := New(reader, saver, sender, tokenGen, 1*time.Hour)
			err := uc.Execute(context.Background(), tt.input)

			if tt.wantNilErr {
				if err != nil {
					t.Fatalf("expected nil error, got: %v", err)
				}
				if len(sender.sent) > 0 {
					t.Fatal("expected no email to be sent for unknown user")
				}
				return
			}

			if tt.wantCode != "" {
				if err == nil {
					t.Fatalf("expected error with code %s, got nil", tt.wantCode)
				}
				if !errors.Is(err, tt.wantCode) {
					t.Fatalf("expected error code %s, got: %v", tt.wantCode, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantSaved && len(saver.saved) == 0 {
				t.Fatal("expected token to be saved")
			}

			if tt.wantSent && len(sender.sent) == 0 {
				t.Fatal("expected email to be sent")
			}
		})
	}
}
