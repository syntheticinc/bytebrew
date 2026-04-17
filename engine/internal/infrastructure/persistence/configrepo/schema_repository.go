package configrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// SchemaRecord is an intermediate struct for DB <-> domain mapping.
//
// V2: AgentNames is derived at read time from `agent_relations` (union of
// source and target agent names for the schema). There is no
// `schema_agents` join table — see docs/architecture/agent-first-runtime.md
// §2.1.
type SchemaRecord struct {
	ID           string
	Name         string
	Description  string
	IsSystem     bool
	AgentNames   []string // derived: distinct agents referenced by agent_relations of this schema
	EntryAgentID *string  // FK to agents.id; may be nil
	CreatedAt    time.Time
}

// GORMSchemaRepository implements schema CRUD using GORM.
type GORMSchemaRepository struct {
	db *gorm.DB
}

// NewGORMSchemaRepository creates a new GORMSchemaRepository.
func NewGORMSchemaRepository(db *gorm.DB) *GORMSchemaRepository {
	return &GORMSchemaRepository{db: db}
}

// List returns all schemas with their derived agent membership.
func (r *GORMSchemaRepository) List(ctx context.Context) ([]SchemaRecord, error) {
	var schemas []models.SchemaModel
	if err := r.db.WithContext(ctx).Order("created_at ASC").Find(&schemas).Error; err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}

	records := make([]SchemaRecord, 0, len(schemas))
	for _, s := range schemas {
		agentNames, err := r.deriveAgentNames(ctx, s.ID)
		if err != nil {
			return nil, fmt.Errorf("derive agents for schema %q: %w", s.Name, err)
		}
		records = append(records, SchemaRecord{
			ID:           s.ID,
			Name:         s.Name,
			Description:  s.Description,
			IsSystem:     s.IsSystem,
			AgentNames:   agentNames,
			EntryAgentID: s.EntryAgentID,
			CreatedAt:    s.CreatedAt,
		})
	}
	return records, nil
}

// GetByID returns a single schema by ID with derived agent membership.
func (r *GORMSchemaRepository) GetByID(ctx context.Context, id string) (*SchemaRecord, error) {
	var schema models.SchemaModel
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&schema).Error; err != nil {
		return nil, fmt.Errorf("get schema %s: %w", id, err)
	}

	agentNames, err := r.deriveAgentNames(ctx, schema.ID)
	if err != nil {
		return nil, fmt.Errorf("derive agents for schema %s: %w", id, err)
	}

	return &SchemaRecord{
		ID:           schema.ID,
		Name:         schema.Name,
		Description:  schema.Description,
		IsSystem:     schema.IsSystem,
		AgentNames:   agentNames,
		EntryAgentID: schema.EntryAgentID,
		CreatedAt:    schema.CreatedAt,
	}, nil
}

// Create inserts a new schema.
func (r *GORMSchemaRepository) Create(ctx context.Context, record *SchemaRecord) error {
	model := models.SchemaModel{
		Name:         record.Name,
		Description:  record.Description,
		IsSystem:     record.IsSystem,
		EntryAgentID: record.EntryAgentID,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create schema %q: %w", record.Name, err)
	}
	record.ID = model.ID
	return nil
}

// Update updates an existing schema by ID.
// Includes entry_agent_id so admins can re-point a schema's entry without a
// delete+recreate cycle. Nil EntryAgentID clears the column.
func (r *GORMSchemaRepository) Update(ctx context.Context, id string, record *SchemaRecord) error {
	updates := map[string]interface{}{
		"name":            record.Name,
		"description":     record.Description,
		"entry_agent_id":  record.EntryAgentID,
	}
	result := r.db.WithContext(ctx).Model(&models.SchemaModel{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("update schema %s: %w", id, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// Delete removes a schema and all its agent_relations + triggers by ID.
// V2: triggers are schema-scoped, so deleting the schema must also remove
// them; otherwise the trigger FK blocks the schema delete with a 500.
func (r *GORMSchemaRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete triggers bound to this schema (trigger.schema_id FK).
		if err := tx.Where("schema_id = ?", id).Delete(&models.TriggerModel{}).Error; err != nil {
			return fmt.Errorf("delete schema triggers: %w", err)
		}
		// Delete agent relations bound to this schema (membership cascade).
		if err := tx.Where("schema_id = ?", id).Delete(&models.AgentRelationModel{}).Error; err != nil {
			return fmt.Errorf("delete schema agent relations: %w", err)
		}
		// Delete schema itself.
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

// ListAgents returns the derived list of agent names that participate in the
// given schema (V2: union of source/target agents in agent_relations).
func (r *GORMSchemaRepository) ListAgents(ctx context.Context, schemaID string) ([]string, error) {
	return r.deriveAgentNames(ctx, schemaID)
}

// ListSchemasForAgent returns schema names that reference a given agent.
//
// V2 derivation: schemas where the agent appears as source or target of any
// agent_relation. See docs/architecture/agent-first-runtime.md §2.1.
//
// Q.5: agent_relations uses source_agent_id/target_agent_id UUIDs. We first
// resolve agentName → agent.id, then query agent_relations by UUID.
func (r *GORMSchemaRepository) ListSchemasForAgent(ctx context.Context, agentName string) ([]string, error) {
	var agentID string
	if err := r.db.WithContext(ctx).Raw("SELECT id FROM agents WHERE name = ?", agentName).Scan(&agentID).Error; err != nil || agentID == "" {
		return nil, nil
	}

	var schemaIDs []string
	if err := r.db.WithContext(ctx).
		Raw(`SELECT DISTINCT schema_id FROM agent_relations
			WHERE source_agent_id = ? OR target_agent_id = ?`, agentID, agentID).
		Scan(&schemaIDs).Error; err != nil {
		return nil, fmt.Errorf("list schema ids for agent %q: %w", agentName, err)
	}

	if len(schemaIDs) == 0 {
		return nil, nil
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

// deriveAgentNames returns the distinct agent names participating in a schema
// via agent_relations (union of source_agent_id and target_agent_id, joined to
// agents for the name).
//
// Per docs/architecture/agent-first-runtime.md §2.1, an isolated agent in a
// schema with no relations is not a supported state — schema membership is
// expressed through delegation relations.
//
// Q.5: queries by agent UUID, joins agents table to resolve names.
func (r *GORMSchemaRepository) deriveAgentNames(ctx context.Context, schemaID string) ([]string, error) {
	var names []string
	if err := r.db.WithContext(ctx).
		Raw(`SELECT DISTINCT a.name FROM (
				SELECT source_agent_id AS agent_id FROM agent_relations WHERE schema_id = ?
				UNION
				SELECT target_agent_id AS agent_id FROM agent_relations WHERE schema_id = ?
			) members JOIN agents a ON a.id = members.agent_id ORDER BY a.name`, schemaID, schemaID).
		Scan(&names).Error; err != nil {
		return nil, fmt.Errorf("derive agent names: %w", err)
	}
	if len(names) == 0 {
		return nil, nil
	}
	return names, nil
}
