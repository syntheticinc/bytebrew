package domain

import (
	"strings"
	"testing"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name         string
		email        string
		passwordHash string
		wantErr      bool
		errContains  string
	}{
		{
			name:         "valid user",
			email:        "valid@email.com",
			passwordHash: "hash123",
			wantErr:      false,
		},
		{
			name:         "empty email",
			email:        "",
			passwordHash: "hash123",
			wantErr:      true,
			errContains:  "email is required",
		},
		{
			name:         "invalid email format",
			email:        "invalid",
			passwordHash: "hash123",
			wantErr:      true,
			errContains:  "invalid email format",
		},
		{
			name:         "empty password hash",
			email:        "a@b.com",
			passwordHash: "",
			wantErr:      true,
			errContains:  "password hash is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewUser(tt.email, tt.passwordHash)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Email != tt.email {
				t.Errorf("email = %q, want %q", got.Email, tt.email)
			}
			if got.PasswordHash != tt.passwordHash {
				t.Errorf("passwordHash = %q, want %q", got.PasswordHash, tt.passwordHash)
			}
			if got.CreatedAt.IsZero() {
				t.Error("createdAt should not be zero")
			}
		})
	}
}
