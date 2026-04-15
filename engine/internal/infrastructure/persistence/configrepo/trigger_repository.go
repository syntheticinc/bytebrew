package configrepo

import (
	"context"
	"fmt"
	"time"

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

// ListBySchemaID returns triggers scoped to a specific schema.
func (r *GORMTriggerRepository) ListBySchemaID(ctx context.Context, schemaID string) ([]models.TriggerModel, error) {
	var triggers []models.TriggerModel
	if err := r.db.WithContext(ctx).Preload("Agent").Where("schema_id = ?", schemaID).Order("created_at DESC").Find(&triggers).Error; err != nil {
		return nil, fmt.Errorf("list triggers by schema %s: %w", schemaID, err)
	}
	return triggers, nil
}

// GetByID returns a single trigger model by ID with agent preloaded.
func (r *GORMTriggerRepository) GetByID(ctx context.Context, id string) (*models.TriggerModel, error) {
	var trigger models.TriggerModel
	if err := r.db.WithContext(ctx).Preload("Agent").Where("id = ?", id).First(&trigger).Error; err != nil {
		return nil, fmt.Errorf("get trigger %s: %w", id, err)
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
// SchemaID and IsSystem are omitted — they must not change via normal update.
func (r *GORMTriggerRepository) Update(ctx context.Context, id string, model *models.TriggerModel) error {
	// Select("*") ensures zero-value fields (e.g. Enabled=false) are persisted.
	// Omit schema_id so normal updates never overwrite schema scoping.
	result := r.db.WithContext(ctx).Model(&models.TriggerModel{}).Where("id = ?", id).Select("*").Omit("id", "created_at", "schema_id").Updates(model)
	if result.Error != nil {
		return fmt.Errorf("update trigger: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %s", id)
	}
	return nil
}

// SetSchemaID assigns a trigger to a specific schema. Used for explicit reassignment.
func (r *GORMTriggerRepository) SetSchemaID(ctx context.Context, triggerID string, schemaID *string) error {
	result := r.db.WithContext(ctx).Model(&models.TriggerModel{}).Where("id = ?", triggerID).Update("schema_id", schemaID)
	if result.Error != nil {
		return fmt.Errorf("set trigger schema: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %s", triggerID)
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

// FindEnabledChatTrigger returns the first enabled chat trigger for the given
// agent, or nil when there is none. Used by the chat dispatcher to stamp
// last_fired_at on the originating channel when a new session opens (§4.1).
func (r *GORMTriggerRepository) FindEnabledChatTrigger(ctx context.Context, agentName string) (*models.TriggerModel, error) {
	var trigger models.TriggerModel
	err := r.db.WithContext(ctx).
		Table("triggers").
		Joins("JOIN agents ON agents.id = triggers.agent_id").
		Where("agents.name = ? AND triggers.type = ? AND triggers.enabled = ?", agentName, models.TriggerTypeChat, true).
		Select("triggers.*").
		First(&trigger).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("find enabled chat trigger for %q: %w", agentName, err)
	}
	return &trigger, nil
}

// SetAgentID sets the target agent for a trigger (canvas edge → routing enabled).
func (r *GORMTriggerRepository) SetAgentID(ctx context.Context, triggerID string, agentID string) error {
	result := r.db.WithContext(ctx).Model(&models.TriggerModel{}).Where("id = ?", triggerID).Update("agent_id", agentID)
	if result.Error != nil {
		return fmt.Errorf("set trigger agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %s", triggerID)
	}
	return nil
}

// ClearAgentID removes the target agent from a trigger (canvas edge deleted → routing disabled).
func (r *GORMTriggerRepository) ClearAgentID(ctx context.Context, triggerID string) error {
	result := r.db.WithContext(ctx).Model(&models.TriggerModel{}).Where("id = ?", triggerID).Update("agent_id", nil)
	if result.Error != nil {
		return fmt.Errorf("clear trigger agent: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %s", triggerID)
	}
	return nil
}

// Delete removes a trigger model by ID.
func (r *GORMTriggerRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.TriggerModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete trigger: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %s", id)
	}
	return nil
}

// FindByWebhookPath resolves an enabled webhook-type trigger by its
// `config->>'webhook_path'` value. Returns nil (no error) when no trigger
// matches — callers decide whether that is a 404.
//
// V2 (§4.1): webhook_path lives inside `config` jsonb; the legacy flat
// column is gone.
func (r *GORMTriggerRepository) FindByWebhookPath(ctx context.Context, path string) (*models.TriggerModel, error) {
	var trigger models.TriggerModel
	err := r.db.WithContext(ctx).
		Preload("Agent").
		Where("type = ? AND enabled = ? AND config->>'webhook_path' = ?", models.TriggerTypeWebhook, true, path).
		First(&trigger).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("find webhook trigger %q: %w", path, err)
	}
	return &trigger, nil
}

// MarkFired stamps the trigger's last_fired_at to now(). Called from
// CronScheduler (every tick), the webhook handler (after validation), and the
// chat dispatcher (first message of a new session) — see
// docs/architecture/agent-first-runtime.md §4.1.
//
// Idempotent by design: every call just overwrites the timestamp; a missing
// trigger row is reported as an error so callers can distinguish the "stale
// trigger id" case (e.g. trigger deleted mid-flight) from a successful stamp.
func (r *GORMTriggerRepository) MarkFired(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Model(&models.TriggerModel{}).
		Where("id = ?", id).
		Update("last_fired_at", time.Now())
	if result.Error != nil {
		return fmt.Errorf("mark trigger fired: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("trigger not found: %s", id)
	}
	return nil
}
