package domain

import (
	"fmt"
	"net/mail"
	"time"
)

// User represents a registered user.
type User struct {
	ID                   string
	Email                string
	PasswordHash         string
	GoogleID             *string
	CreatedAt            time.Time
	PasswordResetToken   *string
	PasswordResetExpires *time.Time
}

// NewUser creates a new User with password-based authentication.
func NewUser(email, passwordHash string) (*User, error) {
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if passwordHash == "" {
		return nil, fmt.Errorf("password hash is required")
	}
	return &User{
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}, nil
}

// NewGoogleUser creates a new User authenticated via Google OAuth.
// No password is required for Google-only users.
func NewGoogleUser(email, googleID string) (*User, error) {
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if googleID == "" {
		return nil, fmt.Errorf("google ID is required")
	}
	return &User{
		Email:    email,
		GoogleID: &googleID,
		CreatedAt: time.Now(),
	}, nil
}

// IsGoogleOnly returns true if the user has no password set (Google-only account).
func (u *User) IsGoogleOnly() bool {
	return u.PasswordHash == "" && u.GoogleID != nil
}

func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}
	return nil
}
