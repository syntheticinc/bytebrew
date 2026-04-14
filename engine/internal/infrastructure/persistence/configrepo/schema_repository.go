package configrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// SchemaRecord is an intermediate struct for DB <-> domain mapping.
type SchemaRecord struct {
	ID          string
	Name        string
	Description string
	IsSystem    bool
	AgentNames  []string // names of agents referenced by this schema
	CreatedAt   time.Time
}

// GORMSchemaRepository implements schema CRUD using GORM.
type GORMSchemaRepository struct {
	db *gorm.DB
}

// NewGORMSchemaRepository creates a new GORMSchemaRepository.
func NewGORMSchemaRepository(db *gorm.DB) *GORMSchemaRepository {
	return &GORMSchemaRepository{db: db}
}

// List returns all schemas with their agent references.
func (r *GORMSchemaRepository) List(ctx context.Context) ([]SchemaRecord, error) {
	var schemas []models.SchemaModel
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&schemas).Error; err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}

	records := make([]SchemaRecord, 0, len(schemas))
	for _, s := range schemas {
		agentNames, err := r.loadAgentNames(ctx, s.ID)
		if err != nil {
			return nil, fmt.Errorf("load agents for schema %q: %w", s.Name, err)
		}
		records = append(records, SchemaRecord{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			IsSystem:    s.IsSystem,
			AgentNames:  agentNames,
			CreatedAt:   s.CreatedAt,
		})
	}
	return records, nil
}

// GetByID returns a single schema by ID.
func (r *GORMSchemaRepository) GetByID(ctx context.Context, id string) (*SchemaRecord, error) {
	var schema models.SchemaModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		return nil, fmt.Errorf("get schema %s: %w", id, err)
	}

	agentNames, err := r.loadAgentNames(ctx, schema.ID)
	if err != nil {
		return nil, fmt.Errorf("load agents for schema %s: %w", id, err)
	}

	return &SchemaRecord{
		ID:          schema.ID,
		Name:        schema.Name,
		Description: schema.Description,
		IsSystem:    schema.IsSystem,
		AgentNames:  agentNames,
		CreatedAt:   schema.CreatedAt,
	}, nil
}

// Create inserts a new schema.
func (r *GORMSchemaRepository) Create(ctx context.Context, record *SchemaRecord) error {
	model := models.SchemaModel{
		Name:        record.Name,
		Description: record.Description,
		IsSystem:    record.IsSystem,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create schema %q: %w", record.Name, err)
	}
	record.ID = model.ID
	return nil
}

// Update updates an existing schema by ID.
func (r *GORMSchemaRepository) Update(ctx context.Context, id string, record *SchemaRecord) error {
	result := r.db.WithContext(ctx).Model(&models.SchemaModel{}).Where("id = ?", id).Updates(map[string]interface{}{
		"name":        record.Name,
		"description": record.Description,
	})
	if result.Error != nil {
		return fmt.Errorf("update schema %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes a schema and all its associations by ID.
func (r *GORMSchemaRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete schema-agent refs
		if err := tx.Where("schema_id = ?", id).Delete(&models.SchemaAgentModel{}).Error; err != nil {
			return fmt.Errorf("delete schema agent refs: %w", err)
		}
		// Delete edges
		if err := tx.Where("schema_id = ?", id).Delete(&models.EdgeModel{}).Error; err != nil {
			return fmt.Errorf("delete schema edges: %w", err)
		}
		// Delete gates
		if err := tx.Where("schema_id = ?", id).Delete(&models.GateModel{}).Error; err != nil {
			return fmt.Errorf("delete schema gates: %w", err)
		}
		// Delete schema itself
		result := tx.Delete(&models.SchemaModel{}, "id = ?", id)
		if result.Error != nil {
			return fmt.Errorf("delete schema %s: %w", id, result.Error)
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// AddAgent adds an agent reference to a schema.
func (r *GORMSchemaRepository) AddAgent(ctx context.Context, schemaID string, agentName string) error {
	// Resolve agent ID by name
	var agent models.AgentModel
	if err := r.db.WithContext(ctx).Where("name = ?", agentName).First(&agent).Error; err != nil {
		return fmt.Errorf("find agent %q: %w", agentName, err)
	}

	ref := models.SchemaAgentModel{
		SchemaID: schemaID,
		AgentID:  agent.ID,
	}
	if err := r.db.WithContext(ctx).Create(&ref).Error; err != nil {
		return fmt.Errorf("add agent %q to schema %s: %w", agentName, schemaID, err)
	}
	return nil
}

// RemoveAgent removes an agent reference from a schema.
func (r *GORMSchemaRepository) RemoveAgent(ctx context.Context, schemaID string, agentName string) error {
	// Resolve agent ID by name
	var agent models.AgentModel
	if err := r.db.WithContext(ctx).Where("name = ?", agentName).First(&agent).Error; err != nil {
		return fmt.Errorf("find agent %q: %w", agentName, err)
	}

	result := r.db.WithContext(ctx).
		Where("schema_id = ? AND agent_id = ?", schemaID, agent.ID).
		Delete(&models.SchemaAgentModel{})
	if result.Error != nil {
		return fmt.Errorf("remove agent %q from schema %s: %w", agentName, schemaID, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// ListAgents returns agent names for a schema.
func (r *GORMSchemaRepository) ListAgents(ctx context.Context, schemaID string) ([]string, error) {
	return r.loadAgentNames(ctx, schemaID)
}

// ListSchemasForAgent returns schema names that reference a given agent.
func (r *GORMSchemaRepository) ListSchemasForAgent(ctx context.Context, agentName string) ([]string, error) {
	var agent models.AgentModel
	if err := r.db.WithContext(ctx).Where("name = ?", agentName).First(&agent).Error; err != nil {
		return nil, fmt.Errorf("find agent %q: %w", agentName, err)
	}

	var refs []models.SchemaAgentModel
	if err := r.db.WithContext(ctx).Where("agent_id = ?", agent.ID).Find(&refs).Error; err != nil {
		return nil, fmt.Errorf("list schema refs for agent %q: %w", agentName, err)
	}

	if len(refs) == 0 {
		return nil, nil
	}

	schemaIDs := make([]string, 0, len(refs))
	for _, ref := range refs {
		schemaIDs = append(schemaIDs, ref.SchemaID)
	}

	var schemas []models.SchemaModel
	if err := r.db.WithContext(ctx).Where("id IN ?", schemaIDs).Find(&schemas).Error; err != nil {
		return nil, fmt.Errorf("load schemas: %w", err)
	}

	names := make([]string, 0, len(schemas))
	for _, s := range schemas {
		names = append(names, s.Name)
	}
	return names, nil
}

func (r *GORMSchemaRepository) loadAgentNames(ctx context.Context, schemaID string) ([]string, error) {
	var refs []models.SchemaAgentModel
	if err := r.db.WithContext(ctx).Where("schema_id = ?", schemaID).Order("position ASC").Find(&refs).Error; err != nil {
		return nil, err
	}

	if len(refs) == 0 {
		return nil, nil
	}

	agentIDs := make([]string, 0, len(refs))
	for _, ref := range refs {
		agentIDs = append(agentIDs, ref.AgentID)
	}

	var agents []models.AgentModel
	if err := r.db.WithContext(ctx).Where("id IN ?", agentIDs).Find(&agents).Error; err != nil {
		return nil, err
	}

	// Build ID->name map and return in position order
	nameByID := make(map[string]string, len(agents))
	for _, a := range agents {
		nameByID[a.ID] = a.Name
	}

	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		if name, ok := nameByID[ref.AgentID]; ok {
			names = append(names, name)
		}
	}
	return names, nil
}
