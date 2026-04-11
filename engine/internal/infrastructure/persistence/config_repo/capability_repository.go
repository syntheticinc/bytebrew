package config_repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// CapabilityRecord is an intermediate struct for DB <-> domain mapping.
type CapabilityRecord struct {
	ID        string
	AgentName string
	Type      string
	Config    map[string]interface{}
	Enabled   bool
}

// GORMCapabilityRepository implements capability CRUD using GORM.
type GORMCapabilityRepository struct {
	db *gorm.DB
}

// NewGORMCapabilityRepository creates a new GORMCapabilityRepository.
func NewGORMCapabilityRepository(db *gorm.DB) *GORMCapabilityRepository {
	return &GORMCapabilityRepository{db: db}
}

// ListByAgent returns all capabilities for an agent (by name).
func (r *GORMCapabilityRepository) ListByAgent(ctx context.Context, agentName string) ([]CapabilityRecord, error) {
	agentID, err := r.resolveAgentID(ctx, agentName)
	if err != nil {
		return nil, err
	}

	var caps []models.CapabilityModel
	if err := r.db.WithContext(ctx).Where("agent_id = ?", agentID).Find(&caps).Error; err != nil {
		return nil, fmt.Errorf("list capabilities for agent %q: %w", agentName, err)
	}

	records := make([]CapabilityRecord, 0, len(caps))
	for _, c := range caps {
		rec, err := toCapabilityRecord(c, agentName)
		if err != nil {
			return nil, fmt.Errorf("convert capability %s: %w", c.ID, err)
		}
		records = append(records, rec)
	}
	return records, nil
}

// ListEnabledByAgent returns only enabled capabilities for an agent (used at runtime).
func (r *GORMCapabilityRepository) ListEnabledByAgent(ctx context.Context, agentName string) ([]CapabilityRecord, error) {
	agentID, err := r.resolveAgentID(ctx, agentName)
	if err != nil {
		return nil, err
	}

	var caps []models.CapabilityModel
	if err := r.db.WithContext(ctx).Where("agent_id = ? AND enabled = ?", agentID, true).Find(&caps).Error; err != nil {
		return nil, fmt.Errorf("list enabled capabilities for agent %q: %w", agentName, err)
	}

	records := make([]CapabilityRecord, 0, len(caps))
	for _, c := range caps {
		rec, err := toCapabilityRecord(c, agentName)
		if err != nil {
			return nil, fmt.Errorf("convert capability %s: %w", c.ID, err)
		}
		records = append(records, rec)
	}
	return records, nil
}

// GetByID returns a single capability by ID.
func (r *GORMCapabilityRepository) GetByID(ctx context.Context, id string) (*CapabilityRecord, error) {
	var cap models.CapabilityModel
	if err := r.db.WithContext(ctx).Preload("Agent").Where("id = ?", id).First(&cap).Error; err != nil {
		return nil, fmt.Errorf("get capability %s: %w", id, err)
	}
	rec, err := toCapabilityRecord(cap, cap.Agent.Name)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Create inserts a new capability.
func (r *GORMCapabilityRepository) Create(ctx context.Context, record *CapabilityRecord) error {
	agentID, err := r.resolveAgentID(ctx, record.AgentName)
	if err != nil {
		return err
	}

	configJSON, err := json.Marshal(record.Config)
	if err != nil {
		return fmt.Errorf("marshal capability config: %w", err)
	}

	model := models.CapabilityModel{
		AgentID: agentID,
		Type:    record.Type,
		Config:  string(configJSON),
		Enabled: record.Enabled,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create capability: %w", err)
	}
	record.ID = model.ID
	return nil
}

// Update updates an existing capability by ID.
func (r *GORMCapabilityRepository) Update(ctx context.Context, id string, record *CapabilityRecord) error {
	configJSON, err := json.Marshal(record.Config)
	if err != nil {
		return fmt.Errorf("marshal capability config: %w", err)
	}

	result := r.db.WithContext(ctx).Model(&models.CapabilityModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"type":    record.Type,
		"config":  string(configJSON),
		"enabled": record.Enabled,
	})
	if result.Error != nil {
		return fmt.Errorf("update capability %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes a capability by ID.
func (r *GORMCapabilityRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.CapabilityModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete capability %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *GORMCapabilityRepository) resolveAgentID(ctx context.Context, agentName string) (string, error) {
	var agent models.AgentModel
	if err := r.db.WithContext(ctx).Where("name = ?", agentName).First(&agent).Error; err != nil {
		return "", fmt.Errorf("find agent %q: %w", agentName, err)
	}
	return agent.ID, nil
}

func toCapabilityRecord(c models.CapabilityModel, agentName string) (CapabilityRecord, error) {
	var config map[string]interface{}
	if c.Config != "" {
		if err := json.Unmarshal([]byte(c.Config), &config); err != nil {
			return CapabilityRecord{}, fmt.Errorf("unmarshal capability config: %w", err)
		}
	}
	return CapabilityRecord{
		ID:        c.ID,
		AgentName: agentName,
		Type:      c.Type,
		Config:    config,
		Enabled:   c.Enabled,
	}, nil
}
