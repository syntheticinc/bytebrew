package config_repo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMLLMProviderRepository implements LLM provider CRUD using GORM.
type GORMLLMProviderRepository struct {
	db *gorm.DB
}

// NewGORMLLMProviderRepository creates a new GORMLLMProviderRepository.
func NewGORMLLMProviderRepository(db *gorm.DB) *GORMLLMProviderRepository {
	return &GORMLLMProviderRepository{db: db}
}

// List returns all LLM provider models.
func (r *GORMLLMProviderRepository) List(ctx context.Context) ([]models.LLMProviderModel, error) {
	var providers []models.LLMProviderModel
	if err := r.db.WithContext(ctx).Order("name").Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("list llm providers: %w", err)
	}
	return providers, nil
}

// Create inserts a new LLM provider model.
func (r *GORMLLMProviderRepository) Create(ctx context.Context, model *models.LLMProviderModel) error {
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create llm provider: %w", err)
	}
	return nil
}

// Update updates an LLM provider model by ID.
func (r *GORMLLMProviderRepository) Update(ctx context.Context, id uint, model *models.LLMProviderModel) error {
	result := r.db.WithContext(ctx).Model(&models.LLMProviderModel{}).Where("id = ?", id).Updates(model)
	if result.Error != nil {
		return fmt.Errorf("update llm provider: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("llm provider not found: %d", id)
	}
	return nil
}

// Delete removes an LLM provider model by ID.
func (r *GORMLLMProviderRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.LLMProviderModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete llm provider: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("llm provider not found: %d", id)
	}
	return nil
}
