package config_repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// EdgeRecord is an intermediate struct for DB <-> domain mapping.
type EdgeRecord struct {
	ID              uint
	SchemaID        uint
	SourceAgentName string
	TargetAgentName string
	Type            string
	Config          map[string]interface{}
}

// GORMEdgeRepository implements edge CRUD using GORM.
type GORMEdgeRepository struct {
	db *gorm.DB
}

// NewGORMEdgeRepository creates a new GORMEdgeRepository.
func NewGORMEdgeRepository(db *gorm.DB) *GORMEdgeRepository {
	return &GORMEdgeRepository{db: db}
}

// List returns all edges for a schema.
func (r *GORMEdgeRepository) List(ctx context.Context, schemaID uint) ([]EdgeRecord, error) {
	var edges []models.EdgeModel
	if err := r.db.WithContext(ctx).Where("schema_id = ?", schemaID).Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("list edges: %w", err)
	}

	records := make([]EdgeRecord, 0, len(edges))
	for _, e := range edges {
		rec, err := toEdgeRecord(e)
		if err != nil {
			return nil, fmt.Errorf("convert edge %d: %w", e.ID, err)
		}
		records = append(records, rec)
	}
	return records, nil
}

// GetByID returns a single edge by ID.
func (r *GORMEdgeRepository) GetByID(ctx context.Context, id uint) (*EdgeRecord, error) {
	var edge models.EdgeModel
	if err := r.db.WithContext(ctx).First(&edge, id).Error; err != nil {
		return nil, fmt.Errorf("get edge %d: %w", id, err)
	}
	rec, err := toEdgeRecord(edge)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// Create inserts a new edge.
func (r *GORMEdgeRepository) Create(ctx context.Context, record *EdgeRecord) error {
	configJSON, err := json.Marshal(record.Config)
	if err != nil {
		return fmt.Errorf("marshal edge config: %w", err)
	}

	model := models.EdgeModel{
		SchemaID:        record.SchemaID,
		SourceAgentName: record.SourceAgentName,
		TargetAgentName: record.TargetAgentName,
		Type:            record.Type,
		Config:          string(configJSON),
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create edge: %w", err)
	}
	record.ID = model.ID
	return nil
}

// Update updates an existing edge by ID.
func (r *GORMEdgeRepository) Update(ctx context.Context, id uint, record *EdgeRecord) error {
	configJSON, err := json.Marshal(record.Config)
	if err != nil {
		return fmt.Errorf("marshal edge config: %w", err)
	}

	result := r.db.WithContext(ctx).Model(&models.EdgeModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"source_agent_name": record.SourceAgentName,
		"target_agent_name": record.TargetAgentName,
		"type":              record.Type,
		"config":            string(configJSON),
	})
	if result.Error != nil {
		return fmt.Errorf("update edge %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes an edge by ID.
func (r *GORMEdgeRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.EdgeModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete edge %d: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func toEdgeRecord(e models.EdgeModel) (EdgeRecord, error) {
	var config map[string]interface{}
	if e.Config != "" {
		if err := json.Unmarshal([]byte(e.Config), &config); err != nil {
			return EdgeRecord{}, fmt.Errorf("unmarshal edge config: %w", err)
		}
	}
	return EdgeRecord{
		ID:              e.ID,
		SchemaID:        e.SchemaID,
		SourceAgentName: e.SourceAgentName,
		TargetAgentName: e.TargetAgentName,
		Type:            e.Type,
		Config:          config,
	}, nil
}
