package config_repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GateRecord is an intermediate struct for DB <-> domain mapping.
type GateRecord struct {
	ID            uint
	SchemaID      uint
	Name          string
	ConditionType string
	Config        map[string]interface{}
	MaxIterations int
	Timeout       int
}

// GORMGateRepository implements gate CRUD using GORM.
type GORMGateRepository struct {
	db *gorm.DB
}

// NewGORMGateRepository creates a new GORMGateRepository.
func NewGORMGateRepository(db *gorm.DB) *GORMGateRepository {
	return &GORMGateRepository{db: db}
}

// List returns all gates for a schema.
func (r *GORMGateRepository) List(ctx context.Context, schemaID uint) ([]GateRecord, error) {
	var gates []models.GateModel
	if err := r.db.WithContext(ctx).Where("schema_id = ?", schemaID).Find(&gates).Error; err != nil {
		return nil, fmt.Errorf("list gates: %w", err)
	}

	records := make([]GateRecord, 0, len(gates))
	for _, g := range gates {
		rec, err := toGateRecord(g)
		if err != nil {
			return nil, fmt.Errorf("convert gate %d: %w", g.ID, err)
		}
		records = append(records, rec)
	}
	return records, nil
}

// GetByID returns a single gate by ID.
func (r *GORMGateRepository) GetByID(ctx context.Context, id uint) (*GateRecord, error) {
	var gate models.GateModel
	if err := r.db.WithContext(ctx).First(&gate, id).Error; err != nil {
		return nil, fmt.Errorf("get gate %d: %w", id, err)
	}
	rec, err := toGateRecord(gate)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Create inserts a new gate.
func (r *GORMGateRepository) Create(ctx context.Context, record *GateRecord) error {
	configJSON, err := json.Marshal(record.Config)
	if err != nil {
		return fmt.Errorf("marshal gate config: %w", err)
	}

	model := models.GateModel{
		SchemaID:      record.SchemaID,
		Name:          record.Name,
		ConditionType: record.ConditionType,
		Config:        string(configJSON),
		MaxIterations: record.MaxIterations,
		Timeout:       record.Timeout,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create gate: %w", err)
	}
	record.ID = model.ID
	return nil
}

// Update updates an existing gate by ID.
func (r *GORMGateRepository) Update(ctx context.Context, id uint, record *GateRecord) error {
	configJSON, err := json.Marshal(record.Config)
	if err != nil {
		return fmt.Errorf("marshal gate config: %w", err)
	}

	result := r.db.WithContext(ctx).Model(&models.GateModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":           record.Name,
		"condition_type": record.ConditionType,
		"config":         string(configJSON),
		"max_iterations": record.MaxIterations,
		"timeout":        record.Timeout,
	})
	if result.Error != nil {
		return fmt.Errorf("update gate %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes a gate by ID.
func (r *GORMGateRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.GateModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete gate %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func toGateRecord(g models.GateModel) (GateRecord, error) {
	var config map[string]interface{}
	if g.Config != "" {
		if err := json.Unmarshal([]byte(g.Config), &config); err != nil {
			return GateRecord{}, fmt.Errorf("unmarshal gate config: %w", err)
		}
	}
	return GateRecord{
		ID:            g.ID,
		SchemaID:      g.SchemaID,
		Name:          g.Name,
		ConditionType: g.ConditionType,
		Config:        config,
		MaxIterations: g.MaxIterations,
		Timeout:       g.Timeout,
	}, nil
}
