package configrepo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMUserRepository implements user persistence using GORM.
// Users are admin/system records created explicitly via the `ce admin` CLI
// subcommand or equivalent — no lazy creation (password_hash is NOT NULL).
type GORMUserRepository struct {
	db *gorm.DB
}

// NewGORMUserRepository creates a new GORMUserRepository.
func NewGORMUserRepository(db *gorm.DB) *GORMUserRepository {
	return &GORMUserRepository{db: db}
}

// GetByID looks up a user by primary key, scoped to the current tenant.
// Returns nil if the user does not exist or belongs to a different tenant.
func (r *GORMUserRepository) GetByID(ctx context.Context, id string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Where("id = ?", id).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &user, nil
}

// GetByUsername looks up a user by tenant + username. Returns nil if not found.
func (r *GORMUserRepository) GetByUsername(ctx context.Context, tenantID, username string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND username = ?", tenantID, username).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &user, nil
}

// Update saves changes to an existing user, enforcing tenant ownership.
//
// Security: `Save` would happily rewrite any row that matches the PK passed in
// the argument, which is a cross-tenant admin-hijack vector if an attacker can
// supply another tenant's user ID. We therefore:
//
//  1. Verify the target row exists AND belongs to the current tenant.
//  2. Apply updates with an explicit `tenant_id` predicate so the UPDATE
//     is a no-op on mismatched rows even under race conditions.
//  3. Omit immutable/identity columns (`id`, `tenant_id`, `created_at`) so an
//     attacker cannot smuggle a tenant move through the struct.
func (r *GORMUserRepository) Update(ctx context.Context, user *models.UserModel) error {
	if user == nil || user.ID == "" {
		return fmt.Errorf("update user: id required")
	}
	tenantID := tenantIDFromCtx(ctx)

	var existing models.UserModel
	if err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", user.ID, tenantID).
		First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("update user: %w", gorm.ErrRecordNotFound)
		}
		return fmt.Errorf("update user: load existing: %w", err)
	}

	result := r.db.WithContext(ctx).
		Model(&models.UserModel{}).
		Where("id = ? AND tenant_id = ?", user.ID, tenantID).
		Omit("id", "tenant_id", "created_at").
		Updates(user)
	if result.Error != nil {
		return fmt.Errorf("update user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("update user: %w", gorm.ErrRecordNotFound)
	}
	return nil
}
