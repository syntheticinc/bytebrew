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
	CreatedAt            time.Time
	PasswordResetToken   *string
	PasswordResetExpires *time.Time
}

// NewUser creates a new User with validation.
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

func validateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return fmt.Errorf("invalid email format: %w", err)
	}
	return nil
}
