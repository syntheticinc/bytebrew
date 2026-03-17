//go:build integration

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/domain"
	"github.com/syntheticinc/bytebrew/bytebrew/cloud-api/internal/infrastructure/postgres"
)

func TestSubscriptionRepository_Integration(t *testing.T) {
	db := setupTestDB(t)
	userRepo := postgres.NewUserRepository(db.Pool)
	subRepo := postgres.NewSubscriptionRepository(db.Pool)
	ctx := context.Background()

	// createTestUser is a helper that creates a user and returns it.
	createTestUser := func(t *testing.T, email string) *domain.User {
		t.Helper()
		user, err := userRepo.Create(ctx, &domain.User{
			Email:        email,
			PasswordHash: "$2a$10$testhashfortesting00uAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		})
		require.NoError(t, err)
		return user
	}

	t.Run("Create and GetByUserID", func(t *testing.T) {
		truncateTables(t, db.Pool)

		user := createTestUser(t, "sub-user@example.com")

		sub := &domain.Subscription{
			UserID: user.ID,
			Tier:   domain.TierTrial,
			Status: domain.StatusActive,
		}

		created, err := subRepo.Create(ctx, sub)
		require.NoError(t, err)
		require.NotNil(t, created)
		assert.NotEmpty(t, created.ID)
		assert.Equal(t, user.ID, created.UserID)
		assert.Equal(t, domain.TierTrial, created.Tier)
		assert.Equal(t, domain.StatusActive, created.Status)
		assert.False(t, created.CreatedAt.IsZero())
		assert.False(t, created.UpdatedAt.IsZero())

		found, err := subRepo.GetByUserID(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, user.ID, found.UserID)
		assert.Equal(t, domain.TierTrial, found.Tier)
		assert.Equal(t, domain.StatusActive, found.Status)
	})

	t.Run("GetByUserID returns nil for nonexistent", func(t *testing.T) {
		truncateTables(t, db.Pool)

		found, err := subRepo.GetByUserID(ctx, "00000000-0000-0000-0000-000000000000")
		require.NoError(t, err)
		assert.Nil(t, found)
	})

	t.Run("UpdateTier", func(t *testing.T) {
		truncateTables(t, db.Pool)

		user := createTestUser(t, "upgrade-user@example.com")

		sub := &domain.Subscription{
			UserID: user.ID,
			Tier:   domain.TierTrial,
			Status: domain.StatusActive,
		}
		_, err := subRepo.Create(ctx, sub)
		require.NoError(t, err)

		periodStart := time.Now().Truncate(time.Microsecond)
		periodEnd := periodStart.Add(30 * 24 * time.Hour).Truncate(time.Microsecond)

		err = subRepo.UpdateTier(ctx, user.ID, domain.TierPersonal, domain.StatusActive, &periodStart, &periodEnd)
		require.NoError(t, err)

		found, err := subRepo.GetByUserID(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		assert.Equal(t, domain.TierPersonal, found.Tier)
		assert.Equal(t, domain.StatusActive, found.Status)
		require.NotNil(t, found.CurrentPeriodStart)
		require.NotNil(t, found.CurrentPeriodEnd)
		assert.WithinDuration(t, periodStart, *found.CurrentPeriodStart, time.Second)
		assert.WithinDuration(t, periodEnd, *found.CurrentPeriodEnd, time.Second)
	})

	t.Run("Create subscription with FK to user", func(t *testing.T) {
		truncateTables(t, db.Pool)

		// Attempt to create subscription for nonexistent user
		sub := &domain.Subscription{
			UserID: "00000000-0000-0000-0000-000000000000",
			Tier:   domain.TierTrial,
			Status: domain.StatusActive,
		}
		_, err := subRepo.Create(ctx, sub)
		require.Error(t, err, "should fail due to FK constraint")
	})
}
