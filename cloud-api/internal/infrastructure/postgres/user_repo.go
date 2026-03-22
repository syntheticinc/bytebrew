package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/syntheticinc/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/cloud-api/internal/infrastructure/postgres/sqlcgen"
)

// UserRepository implements user persistence with PostgreSQL.
type UserRepository struct {
	queries *sqlcgen.Queries
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db sqlcgen.DBTX) *UserRepository {
	return &UserRepository{
		queries: sqlcgen.New(db),
	}
}

// Create inserts a new user and returns the created user.
func (r *UserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	row, err := r.queries.CreateUser(ctx, sqlcgen.CreateUserParams{
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return mapCreateUserRow(row), nil
}

// GetByEmail returns a user by email, or nil if not found.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return mapGetUserByEmailRow(row), nil
}

// GetByID returns a user by ID, or nil if not found.
func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return nil, fmt.Errorf("parse user ID: %w", err)
	}
	row, err := r.queries.GetUserByID(ctx, uid)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return mapGetUserByIDRow(row), nil
}

// UpdatePassword updates the password hash for a user.
func (r *UserRepository) UpdatePassword(ctx context.Context, userID, newHash string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	if err := r.queries.UpdateUserPassword(ctx, sqlcgen.UpdateUserPasswordParams{
		ID:           uid,
		PasswordHash: newHash,
	}); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return nil
}

// Delete removes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	if err := r.queries.DeleteUserByID(ctx, uid); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

// SetResetToken stores a password reset token with an expiration time.
func (r *UserRepository) SetResetToken(ctx context.Context, userID, token string, expiresAt time.Time) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	if err := r.queries.SetPasswordResetToken(ctx, sqlcgen.SetPasswordResetTokenParams{
		ID:                     uid,
		PasswordResetToken:     pgtype.Text{String: token, Valid: true},
		PasswordResetExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	}); err != nil {
		return fmt.Errorf("set reset token: %w", err)
	}
	return nil
}

// GetByResetToken returns a user by a valid (non-expired) reset token, or nil if not found.
func (r *UserRepository) GetByResetToken(ctx context.Context, token string) (*domain.User, error) {
	row, err := r.queries.GetUserByResetToken(ctx, pgtype.Text{String: token, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by reset token: %w", err)
	}
	return mapUser(row), nil
}

// ClearResetToken removes the password reset token for a user.
func (r *UserRepository) ClearResetToken(ctx context.Context, userID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	if err := r.queries.ClearPasswordResetToken(ctx, uid); err != nil {
		return fmt.Errorf("clear reset token: %w", err)
	}
	return nil
}

// UpdatePasswordAndClearResetToken atomically updates the password and clears the reset token.
func (r *UserRepository) UpdatePasswordAndClearResetToken(ctx context.Context, userID, newHash string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	if err := r.queries.UpdatePasswordAndClearResetToken(ctx, sqlcgen.UpdatePasswordAndClearResetTokenParams{
		ID:           uid,
		PasswordHash: newHash,
	}); err != nil {
		return fmt.Errorf("update password and clear token: %w", err)
	}
	return nil
}

// GetByGoogleID returns a user by Google ID, or nil if not found.
func (r *UserRepository) GetByGoogleID(ctx context.Context, googleID string) (*domain.User, error) {
	row, err := r.queries.GetUserByGoogleID(ctx, pgtype.Text{String: googleID, Valid: true})
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by google id: %w", err)
	}
	return mapGetUserByGoogleIDRow(row), nil
}

// CreateGoogleUser inserts a new user authenticated via Google.
func (r *UserRepository) CreateGoogleUser(ctx context.Context, email, googleID string) (*domain.User, error) {
	row, err := r.queries.CreateGoogleUser(ctx, sqlcgen.CreateGoogleUserParams{
		Email:    email,
		GoogleID: pgtype.Text{String: googleID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("create google user: %w", err)
	}
	return mapCreateGoogleUserRow(row), nil
}

// LinkGoogleID links a Google ID to an existing user.
func (r *UserRepository) LinkGoogleID(ctx context.Context, userID, googleID string) error {
	uid, err := parseUUID(userID)
	if err != nil {
		return fmt.Errorf("parse user ID: %w", err)
	}
	if err := r.queries.LinkGoogleID(ctx, sqlcgen.LinkGoogleIDParams{
		ID:       uid,
		GoogleID: pgtype.Text{String: googleID, Valid: true},
	}); err != nil {
		return fmt.Errorf("link google id: %w", err)
	}
	return nil
}

// mapGetUserByGoogleIDRow maps GetUserByGoogleIDRow to domain.User.
func mapGetUserByGoogleIDRow(row sqlcgen.GetUserByGoogleIDRow) *domain.User {
	return &domain.User{
		ID:           uuidToString(row.ID),
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		GoogleID:     textToStringPtr(row.GoogleID),
		CreatedAt:    timestamptzToTimeValue(row.CreatedAt),
	}
}

// mapCreateGoogleUserRow maps CreateGoogleUserRow to domain.User.
func mapCreateGoogleUserRow(row sqlcgen.CreateGoogleUserRow) *domain.User {
	return &domain.User{
		ID:           uuidToString(row.ID),
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		GoogleID:     textToStringPtr(row.GoogleID),
		CreatedAt:    timestamptzToTimeValue(row.CreatedAt),
	}
}

// mapUser maps a full sqlcgen.User (with reset fields) to domain.User.
func mapUser(row sqlcgen.User) *domain.User {
	return &domain.User{
		ID:                   uuidToString(row.ID),
		Email:                row.Email,
		PasswordHash:         row.PasswordHash,
		CreatedAt:            timestamptzToTimeValue(row.CreatedAt),
		PasswordResetToken:   textToStringPtr(row.PasswordResetToken),
		PasswordResetExpires: timestamptzToTime(row.PasswordResetExpiresAt),
	}
}

// mapCreateUserRow maps CreateUserRow to domain.User.
func mapCreateUserRow(row sqlcgen.CreateUserRow) *domain.User {
	return &domain.User{
		ID:           uuidToString(row.ID),
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    timestamptzToTimeValue(row.CreatedAt),
	}
}

// mapGetUserByEmailRow maps GetUserByEmailRow to domain.User.
func mapGetUserByEmailRow(row sqlcgen.GetUserByEmailRow) *domain.User {
	return &domain.User{
		ID:           uuidToString(row.ID),
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    timestamptzToTimeValue(row.CreatedAt),
	}
}

// mapGetUserByIDRow maps GetUserByIDRow to domain.User.
func mapGetUserByIDRow(row sqlcgen.GetUserByIDRow) *domain.User {
	return &domain.User{
		ID:           uuidToString(row.ID),
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		CreatedAt:    timestamptzToTimeValue(row.CreatedAt),
	}
}
