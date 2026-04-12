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

// GetByID returns a single MCP server model by ID.
func (r *GORMMCPServerRepository) GetByID(ctx context.Context, id string) (*models.MCPServerModel, error) {
	var server models.MCPServerModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&server).Error; err != nil {
		return nil, fmt.Errorf("get mcp server %s: %w", id, err)
	}
	return &server, nil
}

// Create inserts a new MCP server model.
func (r *GORMMCPServerRepository) Create(ctx context.Context, model *models.MCPServerModel) error {
	if err := r.db.WithContext(ctx).Create(model).Error; err != nil {
		return fmt.Errorf("create mcp server: %w", err)
	}
	return nil
}

// Update updates an MCP server model by ID.
// Select("*") ensures zero-value fields (e.g. cleared ForwardHeaders) are written.
func (r *GORMMCPServerRepository) Update(ctx context.Context, id string, model *models.MCPServerModel) error {
	result := r.db.WithContext(ctx).Model(&models.MCPServerModel{}).Where("id = ?", id).
		Select("*").Omit("id", "created_at", "updated_at", "Runtime").Updates(model)
	if result.Error != nil {
		return fmt.Errorf("update mcp server: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("mcp server not found: %s", id)
	}
	return nil
}

// Delete removes an MCP server model by ID.
func (r *GORMMCPServerRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.MCPServerModel{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete mcp server: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("mcp server not found: %s", id)
	}
	return nil
}

// GetAgentNamesByServerIDs returns a map of MCP server ID -> []agent names
// by querying the agent_mcp_servers join table. Loads all servers in one query.
func (r *GORMMCPServerRepository) GetAgentNamesByServerIDs(ctx context.Context, serverIDs []string) (map[string][]string, error) {
	if len(serverIDs) == 0 {
		return make(map[string][]string), nil
	}

	var joins []models.AgentMCPServer
	if err := r.db.WithContext(ctx).
		Preload("Agent").
		Where("mcp_server_id IN ?", serverIDs).
		Find(&joins).Error; err != nil {
		return nil, fmt.Errorf("load agent names for mcp servers: %w", err)
	}

	result := make(map[string][]string, len(serverIDs))
	for _, j := range joins {
		result[j.MCPServerID] = append(result[j.MCPServerID], j.Agent.Name)
	}
	return result, nil
}

// GetAgentNamesForServer returns agent names assigned to a single MCP server.
func (r *GORMMCPServerRepository) GetAgentNamesForServer(ctx context.Context, serverID string) ([]string, error) {
	m, err := r.GetAgentNamesByServerIDs(ctx, []string{serverID})
	if err != nil {
		return nil, err
	}
	return m[serverID], nil
}
