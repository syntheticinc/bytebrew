package configrepo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMLLMProviderRepository implements LLM provider CRUD using GORM.
// Note: `models` (model configurations) ARE tenant-scoped — they carry
// per-tenant API keys, base URLs, etc. Global provider-kind enumerations are
// not persisted in this table.
type GORMLLMProviderRepository struct {
	db *gorm.DB
}

// NewGORMLLMProviderRepository creates a new GORMLLMProviderRepository.
func NewGORMLLMProviderRepository(db *gorm.DB) *GORMLLMProviderRepository {
	return &GORMLLMProviderRepository{db: db}
}

// List returns all LLM provider models for the current tenant.
func (r *GORMLLMProviderRepository) List(ctx context.Context) ([]models.LLMProviderModel, error) {
	var providers []models.LLMProviderModel
	if err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Order("name").
		Find(&providers).Error; err != nil {
		return nil, fmt.Errorf("list llm providers: %w", err)
	}
	return providers, nil
}

// GetByID returns a single LLM provider model by ID (tenant-scoped).
func (r *GORMLLMProviderRepository) GetByID(ctx context.Context, id string) (*models.LLMProviderModel, error) {
	var provider models.LLMProviderModel
	if err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Where("id = ?", id).
		First(&provider).Error; err != nil {
		return nil, fmt.Errorf("get llm provider %s: %w", id, err)
	}
	return &provider, nil
}

// Create inserts a new LLM provider model, stamping tenant from context.
func (r *GORMLLMProviderRepository) Create(ctx context.Context, model *models.LLMProviderModel) error {
	model.TenantID = tenantIDFromCtx(ctx)
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create llm provider: %w", err)
	}
	return nil
}

// Update updates an LLM provider model by ID (tenant-scoped).
func (r *GORMLLMProviderRepository) Update(ctx context.Context, id string, model *models.LLMProviderModel) error {
	result := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Model(&models.LLMProviderModel{}).
		Where("id = ?", id).
		Updates(model)
	if result.Error != nil {
		return fmt.Errorf("update llm provider: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("llm provider not found: %s", id)
	}
	return nil
}

// Delete removes an LLM provider model by ID (tenant-scoped).
func (r *GORMLLMProviderRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Delete(&models.LLMProviderModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete llm provider: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("llm provider not found: %s", id)
	}
	return nil
}

// GetModelKind returns the kind ('chat' or 'embedding') for the given model ID (tenant-scoped).
// Returns an empty string and an error when the model does not exist in the tenant.
func (r *GORMLLMProviderRepository) GetModelKind(ctx context.Context, id string) (string, error) {
	var kind string
	if err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Model(&models.LLMProviderModel{}).
		Where("id = ?", id).
		Pluck("kind", &kind).Error; err != nil {
		return "", fmt.Errorf("get model kind %s: %w", id, err)
	}
	if kind == "" {
		return "", fmt.Errorf("model not found: %s", id)
	}
	return kind, nil
}

// AgentsUsingModel returns the names of agents that reference the given model ID (tenant-scoped).
func (r *GORMLLMProviderRepository) AgentsUsingModel(ctx context.Context, modelID string) ([]string, error) {
	var names []string
	if err := r.db.WithContext(ctx).
		Scopes(tenantScope(ctx)).
		Model(&models.AgentModel{}).
		Where("model_id = ?", modelID).
		Order("name").
		Pluck("name", &names).Error; err != nil {
		return nil, fmt.Errorf("list agents using model %s: %w", modelID, err)
	}
	return names, nil
}
