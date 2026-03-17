//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/postgres"
)

func TestUserRepository_Integration(t *testing.T) {
	db := setupTestDB(t)
	repo := postgres.NewUserRepository(db.Pool)
	ctx := context.Background()

	t.Run("Create and GetByEmail", func(t *testing.T) {
		truncateTables(t, db.Pool)

		user := &domain.User{
			Email:        "alice@example.com",
			PasswordHash: "$2a$10$abcdefghijklmnopqrstuuAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		}

		created, err := repo.Create(ctx, user)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, user.Email, created.Email)
		assert.Equal(t, user.PasswordHash, created.PasswordHash)
		assert.False(t, created.CreatedAt.IsZero())

		found, err := repo.GetByEmail(ctx, "alice@example.com")
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, created.Email, found.Email)
		assert.Equal(t, created.PasswordHash, found.PasswordHash)
		assert.Equal(t, created.CreatedAt.Unix(), found.CreatedAt.Unix())
	})

	t.Run("Create and GetByID", func(t *testing.T) {
		truncateTables(t, db.Pool)

		user := &domain.User{
			Email:        "bob@example.com",
			PasswordHash: "$2a$10$xyzxyzxyzxyzxyzxyzxyzuBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
		}

		created, err := repo.Create(ctx, user)
		require.NoError(t, err)
		require.NotNil(t, created)

		found, err := repo.GetByID(ctx, created.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, created.Email, found.Email)
		assert.Equal(t, created.PasswordHash, found.PasswordHash)
	})

	t.Run("GetByEmail returns nil for nonexistent", func(t *testing.T) {
		truncateTables(t, db.Pool)

		found, err := repo.GetByEmail(ctx, "nobody@example.com")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("GetByID returns nil for nonexistent", func(t *testing.T) {
		truncateTables(t, db.Pool)

		found, err := repo.GetByID(ctx, "00000000-0000-0000-0000-000000000000")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("Create duplicate email fails", func(t *testing.T) {
		truncateTables(t, db.Pool)

		user := &domain.User{
			Email:        "dup@example.com",
			PasswordHash: "$2a$10$hashhashhashhashhashhuCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
		}

		_, err := repo.Create(ctx, user)
		require.NoError(t, err)

		_, err = repo.Create(ctx, user)
		require.Error(t, err)
	})

	t.Run("Multiple users independent", func(t *testing.T) {
		truncateTables(t, db.Pool)

		user1 := &domain.User{
			Email:        "first@example.com",
			PasswordHash: "$2a$10$hash1hash1hash1hash1uDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD",
		}
		user2 := &domain.User{
			Email:        "second@example.com",
			PasswordHash: "$2a$10$hash2hash2hash2hash2uEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE",
		}

		created1, err := repo.Create(ctx, user1)
		require.NoError(t, err)

		created2, err := repo.Create(ctx, user2)
		require.NoError(t, err)

		assert.NotEqual(t, created1.ID, created2.ID)

		found1, err := repo.GetByEmail(ctx, "first@example.com")
		require.NoError(t, err)
		require.NotNil(t, found1)
		assert.Equal(t, created1.ID, found1.ID)

		found2, err := repo.GetByEmail(ctx, "second@example.com")
		require.NoError(t, err)
		require.NotNil(t, found2)
		assert.Equal(t, created2.ID, found2.ID)
	})
}
