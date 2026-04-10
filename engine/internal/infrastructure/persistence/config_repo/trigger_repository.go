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

// GetByID returns a single trigger model by ID with agent preloaded.
func (r *GORMTriggerRepository) GetByID(ctx context.Context, id uint) (*models.TriggerModel, error) {
	var trigger models.TriggerModel
	if err := r.db.WithContext(ctx).Preload("Agent").First(&trigger, id).Error; err != nil {
		return nil, fmt.Errorf("get trigger %d: %w", id, err)
	}
	return &trigger, nil
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

// HasEnabledChatTrigger returns true if the agent has at least one enabled chat trigger.
func (r *GORMTriggerRepository) HasEnabledChatTrigger(ctx context.Context, agentName string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("triggers").
		Joins("JOIN agents ON agents.id = triggers.agent_id").
		Where("agents.name = ? AND triggers.type = ? AND triggers.enabled = ?", agentName, models.TriggerTypeChat, true).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("check chat trigger for %q: %w", agentName, err)
	}
	return count > 0, nil
}

// SetAgentID sets the target agent for a trigger (canvas edge → routing enabled).
func (r *GORMTriggerRepository) SetAgentID(ctx context.Context, triggerID uint, agentID uint) error {
	result := r.db.WithContext(ctx).Model(&models.TriggerModel{}).Where("id = ?", triggerID).Update("agent_id", agentID)
	if result.Error != nil {
		return fmt.Errorf("set trigger agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %d", triggerID)
	}
	return nil
}

// ClearAgentID removes the target agent from a trigger (canvas edge deleted → routing disabled).
func (r *GORMTriggerRepository) ClearAgentID(ctx context.Context, triggerID uint) error {
	result := r.db.WithContext(ctx).Model(&models.TriggerModel{}).Where("id = ?", triggerID).Update("agent_id", nil)
	if result.Error != nil {
		return fmt.Errorf("clear trigger agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %d", triggerID)
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
