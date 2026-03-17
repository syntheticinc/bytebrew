package reset_password

import (
	"context"
	"fmt"
	"testing"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/pkg/errors"
)

// --- Mocks ---

type mockUserByTokenReader struct {
	users map[string]*domain.User
	err   error
}

func newMockUserByTokenReader() *mockUserByTokenReader {
	return &mockUserByTokenReader{users: make(map[string]*domain.User)}
}

func (m *mockUserByTokenReader) GetByResetToken(_ context.Context, token string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users[token], nil
}

func (m *mockUserByTokenReader) addUser(token, userID string) {
	m.users[token] = &domain.User{
		ID:           userID,
		Email:        userID + "@example.com",
		PasswordHash: "old-hash",
	}
}

type mockPasswordResetUpdater struct {
	updated []string
	err     error
}

func (m *mockPasswordResetUpdater) UpdatePasswordAndClearResetToken(_ context.Context, userID, _ string) error {
	if m.err != nil {
		return m.err
	}
	m.updated = append(m.updated, userID)
	return nil
}

type mockPasswordHasher struct {
	hashResult string
	hashErr    error
}

func (m *mockPasswordHasher) Hash(_ string) (string, error) {
	if m.hashErr != nil {
		return "", m.hashErr
	}
	return m.hashResult, nil
}

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name        string
		input       Input
		setupRepo   func(*mockUserByTokenReader)
		readerErr   error
		hashResult  string
		hashErr     error
		updaterErr  error
		wantCode    string
		wantUpdated bool
	}{
		{
			name:  "success",
			input: Input{Token: "valid-token", NewPassword: "newpass12"},
			setupRepo: func(r *mockUserByTokenReader) {
				r.addUser("valid-token", "user-1")
			},
			hashResult:  "hashed-newpass12",
			wantUpdated: true,
		},
		{
			name:     "empty token",
			input:    Input{Token: "", NewPassword: "newpass12"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "password too short",
			input:    Input{Token: "valid-token", NewPassword: "short"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "invalid or expired token",
			input:    Input{Token: "bad-token", NewPassword: "newpass12"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:      "token reader error",
			input:     Input{Token: "valid-token", NewPassword: "newpass12"},
			readerErr: fmt.Errorf("db error"),
			wantCode:  errors.CodeInternal,
		},
		{
			name:  "hash error",
			input: Input{Token: "valid-token", NewPassword: "newpass12"},
			setupRepo: func(r *mockUserByTokenReader) {
				r.addUser("valid-token", "user-1")
			},
			hashErr:  fmt.Errorf("bcrypt failure"),
			wantCode: errors.CodeInternal,
		},
		{
			name:  "update and clear error",
			input: Input{Token: "valid-token", NewPassword: "newpass12"},
			setupRepo: func(r *mockUserByTokenReader) {
				r.addUser("valid-token", "user-1")
			},
			hashResult: "hashed-newpass12",
			updaterErr: fmt.Errorf("db error"),
			wantCode:   errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := newMockUserByTokenReader()
			reader.err = tt.readerErr
			if tt.setupRepo != nil {
				tt.setupRepo(reader)
			}

			updater := &mockPasswordResetUpdater{err: tt.updaterErr}
			hasher := &mockPasswordHasher{
				hashResult: tt.hashResult,
				hashErr:    tt.hashErr,
			}

			uc := New(reader, updater, hasher)
			err := uc.Execute(context.Background(), tt.input)

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

			if tt.wantUpdated && len(updater.updated) == 0 {
				t.Fatal("expected password to be updated and token cleared")
			}
		})
	}
}
