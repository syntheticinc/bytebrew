package config_repo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GORMSettingRepository implements settings key-value CRUD using GORM.
type GORMSettingRepository struct {
	db *gorm.DB
}

// NewGORMSettingRepository creates a new GORMSettingRepository.
func NewGORMSettingRepository(db *gorm.DB) *GORMSettingRepository {
	return &GORMSettingRepository{db: db}
}

// List returns all settings.
func (r *GORMSettingRepository) List(ctx context.Context) ([]models.SettingModel, error) {
	var settings []models.SettingModel
	if err := r.db.WithContext(ctx).Order(`"key"`).Find(&settings).Error; err != nil {
		return nil, fmt.Errorf("list settings: %w", err)
	}
	return settings, nil
}

// Get returns a single setting by key. Returns nil if not found.
func (r *GORMSettingRepository) Get(ctx context.Context, key string) (*models.SettingModel, error) {
	var setting models.SettingModel
	err := r.db.WithContext(ctx).Where(`"key" = ?`, key).First(&setting).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get setting %q: %w", key, err)
	}
	return &setting, nil
}

// Set upserts a setting value by key.
func (r *GORMSettingRepository) Set(ctx context.Context, key, value string) error {
	setting := models.SettingModel{Key: key, Value: value}
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&setting).Error
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}
