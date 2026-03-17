package change_password

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
	err   error
}

func newMockUserReader() *mockUserReader {
	return &mockUserReader{users: make(map[string]*domain.User)}
}

func (m *mockUserReader) GetByID(_ context.Context, id string) (*domain.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users[id], nil
}

func (m *mockUserReader) addUser(id, password string) {
	m.users[id] = &domain.User{
		ID:           id,
		Email:        id + "@example.com",
		PasswordHash: "hashed-" + password,
	}
}

type mockPasswordUpdater struct {
	updated []string
	err     error
}

func (m *mockPasswordUpdater) UpdatePassword(_ context.Context, userID, _ string) error {
	if m.err != nil {
		return m.err
	}
	m.updated = append(m.updated, userID)
	return nil
}

type mockPasswordHasher struct {
	hashResult string
	hashErr    error
	compareErr error
}

func (m *mockPasswordHasher) Hash(_ string) (string, error) {
	if m.hashErr != nil {
		return "", m.hashErr
	}
	return m.hashResult, nil
}

func (m *mockPasswordHasher) Compare(hash, password string) error {
	if m.compareErr != nil {
		return m.compareErr
	}
	if hash == "hashed-"+password {
		return nil
	}
	return fmt.Errorf("password mismatch")
}

// --- Tests ---

func TestExecute(t *testing.T) {
	tests := []struct {
		name        string
		input       Input
		setupRepo   func(*mockUserReader)
		hashResult  string
		hashErr     error
		compareErr  error
		updaterErr  error
		wantCode    string
		wantUpdated bool
	}{
		{
			name:  "success",
			input: Input{UserID: "user-1", CurrentPassword: "oldpass1", NewPassword: "newpass12"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "oldpass1")
			},
			hashResult:  "hashed-newpass12",
			wantUpdated: true,
		},
		{
			name:     "password too short",
			input:    Input{UserID: "user-1", CurrentPassword: "oldpass1", NewPassword: "short"},
			wantCode: errors.CodeInvalidInput,
		},
		{
			name:     "user not found",
			input:    Input{UserID: "nobody", CurrentPassword: "oldpass1", NewPassword: "newpass12"},
			wantCode: errors.CodeNotFound,
		},
		{
			name:  "wrong current password",
			input: Input{UserID: "user-1", CurrentPassword: "wrong", NewPassword: "newpass12"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "oldpass1")
			},
			wantCode: errors.CodeUnauthorized,
		},
		{
			name:  "hash error",
			input: Input{UserID: "user-1", CurrentPassword: "oldpass1", NewPassword: "newpass12"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "oldpass1")
			},
			hashErr:  fmt.Errorf("bcrypt failure"),
			wantCode: errors.CodeInternal,
		},
		{
			name:  "update error",
			input: Input{UserID: "user-1", CurrentPassword: "oldpass1", NewPassword: "newpass12"},
			setupRepo: func(r *mockUserReader) {
				r.addUser("user-1", "oldpass1")
			},
			hashResult: "hashed-newpass12",
			updaterErr: fmt.Errorf("db error"),
			wantCode:   errors.CodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := newMockUserReader()
			if tt.setupRepo != nil {
				tt.setupRepo(reader)
			}

			updater := &mockPasswordUpdater{err: tt.updaterErr}
			hasher := &mockPasswordHasher{
				hashResult: tt.hashResult,
				hashErr:    tt.hashErr,
				compareErr: tt.compareErr,
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
				t.Fatal("expected password to be updated")
			}
		})
	}
}
