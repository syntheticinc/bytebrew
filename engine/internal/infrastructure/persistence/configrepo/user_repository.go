package configrepo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMUserRepository implements user persistence using GORM.
type GORMUserRepository struct {
	db *gorm.DB
}

// NewGORMUserRepository creates a new GORMUserRepository.
func NewGORMUserRepository(db *gorm.DB) *GORMUserRepository {
	return &GORMUserRepository{db: db}
}

// GetOrCreate ensures a user row exists for the given tenant + external ID.
// Uses GORM's FirstOrCreate which maps to a single SELECT + conditional INSERT,
// safe for the hot path (every authenticated request). The unique index on
// (tenant_id, external_id) guarantees idempotency under concurrent requests.
func (r *GORMUserRepository) GetOrCreate(ctx context.Context, tenantID, externalID string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.WithContext(ctx).
		Where(models.UserModel{TenantID: tenantID, ExternalID: externalID}).
		FirstOrCreate(&user).Error; err != nil {
		return nil, fmt.Errorf("get or create user: %w", err)
	}
	return &user, nil
}

// GetByExternalID looks up a user by tenant + external ID. Returns nil if not found.
func (r *GORMUserRepository) GetByExternalID(ctx context.Context, tenantID, externalID string) (*models.UserModel, error) {
	var user models.UserModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND external_id = ?", tenantID, externalID).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get user by external id: %w", err)
	}
	return &user, nil
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

// Update saves changes to an existing user.
func (r *GORMUserRepository) Update(ctx context.Context, user *models.UserModel) error {
	result := r.db.WithContext(ctx).Save(user)
	if result.Error != nil {
		return fmt.Errorf("update user: %w", result.Error)
	}
	return nil
}
