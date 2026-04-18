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

// GetByID looks up a user by primary key. Returns nil if not found.
func (r *GORMUserRepository) GetByID(ctx context.Context, id string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.WithContext(ctx).
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

// Update saves changes to an existing user.
func (r *GORMUserRepository) Update(ctx context.Context, user *models.UserModel) error {
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		return fmt.Errorf("update user: %w", result.Error)
	}
	return nil
}
