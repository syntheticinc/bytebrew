package config_repo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// GORMMCPServerRepository implements MCP server CRUD using GORM.
type GORMMCPServerRepository struct {
	db *gorm.DB
}

// NewGORMMCPServerRepository creates a new GORMMCPServerRepository.
func NewGORMMCPServerRepository(db *gorm.DB) *GORMMCPServerRepository {
	return &GORMMCPServerRepository{db: db}
}

// List returns all MCP server models with runtime status.
func (r *GORMMCPServerRepository) List(ctx context.Context) ([]models.MCPServerModel, error) {
	var servers []models.MCPServerModel
	if err := r.db.WithContext(ctx).Preload("Runtime").Order("name").Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("list mcp servers: %w", err)
	}
	return servers, nil
}

// Create inserts a new MCP server model.
func (r *GORMMCPServerRepository) Create(ctx context.Context, model *models.MCPServerModel) error {
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create mcp server: %w", err)
	}
	return nil
}

// Update updates an MCP server model by ID.
func (r *GORMMCPServerRepository) Update(ctx context.Context, id uint, model *models.MCPServerModel) error {
	result := r.db.WithContext(ctx).Model(&models.MCPServerModel{}).Where("id = ?", id).Updates(model)
	if result.Error != nil {
		return fmt.Errorf("update mcp server: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("mcp server not found: %d", id)
	}
	return nil
}

// Delete removes an MCP server model by ID.
func (r *GORMMCPServerRepository) Delete(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.MCPServerModel{}, id)
	if result.Error != nil {
		return fmt.Errorf("delete mcp server: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("mcp server not found: %d", id)
	}
	return nil
}
