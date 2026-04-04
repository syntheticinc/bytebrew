package config_repo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMTriggerRepository implements trigger CRUD using GORM.
type GORMTriggerRepository struct {
	db *gorm.DB
}

// NewGORMTriggerRepository creates a new GORMTriggerRepository.
func NewGORMTriggerRepository(db *gorm.DB) *GORMTriggerRepository {
	return &GORMTriggerRepository{db: db}
}

// List returns all trigger models with agent preloaded.
func (r *GORMTriggerRepository) List(ctx context.Context) ([]models.TriggerModel, error) {
	var triggers []models.TriggerModel
	if err := r.db.WithContext(ctx).Preload("Agent").Order("created_at DESC").Find(&triggers).Error; err != nil {
		return nil, fmt.Errorf("list triggers: %w", err)
	}
	return triggers, nil
}

// Create inserts a new trigger model.
func (r *GORMTriggerRepository) Create(ctx context.Context, model *models.TriggerModel) error {
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create trigger: %w", err)
	}
	return nil
}

// Update updates a trigger model by ID.
func (r *GORMTriggerRepository) Update(ctx context.Context, id uint, model *models.TriggerModel) error {
	// Select("*") ensures zero-value fields (e.g. Enabled=false) are persisted.
	result := r.db.WithContext(ctx).Model(&models.TriggerModel{}).Where("id = ?", id).Select("*").Omit("id", "created_at").Updates(model)
	if result.Error != nil {
		return fmt.Errorf("update trigger: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %d", id)
	}
	return nil
}

// Delete removes a trigger model by ID.
func (r *GORMTriggerRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.TriggerModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete trigger: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %d", id)
	}
	return nil
}
