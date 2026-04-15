// Package schematemplate implements the "Use template" fork operation for
// the V2 schema template catalog (§2.2). Given a curated template from the
// `schema_templates` table, ForkService clones its Definition into
// tenant-owned rows in schemas + agents + agent_relations + triggers +
// capabilities in a single transaction. Forked rows have no FK back — the
// copy is independent of the catalog (catalog updates never touch existing
// forks).
//
// See docs/architecture/agent-first-runtime.md §2.2 and
// docs/plan/v2-cleanup-checklist.md "Commit Group L".
package schematemplate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
)

// ErrTemplateNotFound is returned by Fork when the named template is not
// present in the catalog. Callers typically map this to HTTP 404.
var ErrTemplateNotFound = errors.New("schema template not found")

// ErrSchemaNameTaken is returned by Fork when the requested new schema name
// already exists. Schema names are globally unique in V2
// (`idx_schemas_tenant_name`), so duplicate names are rejected up front.
// Callers typically map this to HTTP 409.
var ErrSchemaNameTaken = errors.New("schema name already taken")

// ErrInvalidTemplate is returned when the loaded template definition is
// self-inconsistent (missing entry agent, dangling relation endpoint,
// empty required field). The catalog seeder should catch these at boot,
// but we double-check at fork time so a corrupt row never writes partial
// data.
var ErrInvalidTemplate = errors.New("invalid schema template")

// ForkedSchema is the lightweight result of a successful fork, returned to
// the caller so it can navigate to the new schema detail page.
type ForkedSchema struct {
	SchemaID   string
	SchemaName string
	AgentIDs   map[string]string // logical name → newly minted uuid
}

// TemplateReader is the consumer-side interface ForkService needs to load a
// catalog template by name. Implemented by
// configrepo.GORMSchemaTemplateRepository.
type TemplateReader interface {
	GetByName(ctx context.Context, name string) (*domain.SchemaTemplate, error)
}

// ForkService clones a catalog template into tenant-owned runtime rows.
// One instance is safe to reuse — state-free apart from the injected DB
// handle.
type ForkService struct {
	db    *gorm.DB
	repo  TemplateReader
}

// NewForkService constructs a ForkService backed by the given DB handle and
// template reader.
func NewForkService(db *gorm.DB, repo TemplateReader) *ForkService {
	return &ForkService{db: db, repo: repo}
}

// Fork clones `templateName` into a new schema called `newSchemaName`. The
// optional `tenantID` is accepted for future per-tenant ownership (V2
// current schema keeps single-tenant columns — the field is threaded
// through for forward compatibility but not persisted today).
//
// All writes happen inside a single transaction; any error rolls back the
// entire fork so a failed attempt never leaves half-built rows.
func (s *ForkService) Fork(ctx context.Context, tenantID, templateName, newSchemaName string) (*ForkedSchema, error) {
	newSchemaName = strings.TrimSpace(newSchemaName)
	if newSchemaName == "" {
		return nil, fmt.Errorf("schema name is required")
	}
	templateName = strings.TrimSpace(templateName)
	if templateName == "" {
		return nil, fmt.Errorf("template name is required")
	}

	tmpl, err := s.repo.GetByName(ctx, templateName)
	if err != nil {
		return nil, fmt.Errorf("load template %q: %w", templateName, err)
	}
	if tmpl == nil {
		return nil, ErrTemplateNotFound
	}

	if err := validateDefinition(tmpl.Definition); err != nil {
		return nil, fmt.Errorf("template %q: %w", templateName, err)
	}

	var result ForkedSchema
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Guard against a duplicate schema name up front — the unique index
		// would catch it on insert, but the early check gives a clean
		// typed error for the HTTP 409 path.
		var existing int64
		if err := tx.Model(&models.SchemaModel{}).
			Where("name = ?", newSchemaName).
			Count(&existing).Error; err != nil {
			return fmt.Errorf("check schema name: %w", err)
		}
		if existing > 0 {
			return ErrSchemaNameTaken
		}

		forked, err := s.forkInTx(tx, tmpl, newSchemaName)
		if err != nil {
			return err
		}
		result = forked
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// forkInTx performs the actual row-creation inside an open transaction. All
// errors propagate to the outer Transaction closure → rollback.
func (s *ForkService) forkInTx(tx *gorm.DB, tmpl *domain.SchemaTemplate, newSchemaName string) (ForkedSchema, error) {
	def := tmpl.Definition

	// 1. Create the schema row.
	schema := models.SchemaModel{
		Name:        newSchemaName,
		Description: tmpl.Description,
	}
	if err := tx.Create(&schema).Error; err != nil {
		return ForkedSchema{}, fmt.Errorf("create schema: %w", err)
	}

	// 2. Create each agent with a freshly namespaced name. Agent names are
	//    globally unique (§5.1 — agents are a global library), so we
	//    prefix with the schema name to avoid collisions across forks of
	//    the same template.
	agentIDByLogical := make(map[string]string, len(def.Agents))
	agentNameByLogical := make(map[string]string, len(def.Agents))
	for _, a := range def.Agents {
		newAgentName := fmt.Sprintf("%s__%s", newSchemaName, a.Name)
		model := models.AgentModel{
			Name:         newAgentName,
			SystemPrompt: a.SystemPrompt,
			Lifecycle:    "persistent",
			ToolExecution: "sequential",
			MaxContextSize:  16000,
			MaxTurnDuration: 120,
		}
		if err := tx.Create(&model).Error; err != nil {
			return ForkedSchema{}, fmt.Errorf("create agent %q: %w", newAgentName, err)
		}
		agentIDByLogical[a.Name] = model.ID
		agentNameByLogical[a.Name] = newAgentName

		// 3. Attach capabilities to the newly created agent.
		for _, cap := range a.Capabilities {
			configJSON := ""
			if len(cap.Config) > 0 {
				raw, err := json.Marshal(cap.Config)
				if err != nil {
					return ForkedSchema{}, fmt.Errorf("marshal capability %q config: %w", cap.Type, err)
				}
				configJSON = string(raw)
			}
			capModel := models.CapabilityModel{
				AgentID: model.ID,
				Type:    cap.Type,
				Config:  configJSON,
				Enabled: true,
			}
			if err := tx.Create(&capModel).Error; err != nil {
				return ForkedSchema{}, fmt.Errorf("attach capability %q to agent %q: %w", cap.Type, newAgentName, err)
			}
		}
	}

	// 4. Resolve the entry agent. Validation has already asserted it
	//    exists in the agents list.
	entryAgentID := agentIDByLogical[def.EntryAgentName]

	// 5. Create delegation relations. Each relation stores the new
	//    namespaced agent names (AgentRelationModel is name-based, not
	//    id-based — see agent_relation.go).
	for _, rel := range def.Relations {
		sourceName, ok := agentNameByLogical[rel.Source]
		if !ok {
			return ForkedSchema{}, fmt.Errorf("relation source %q: %w", rel.Source, ErrInvalidTemplate)
		}
		targetName, ok := agentNameByLogical[rel.Target]
		if !ok {
			return ForkedSchema{}, fmt.Errorf("relation target %q: %w", rel.Target, ErrInvalidTemplate)
		}
		relModel := models.AgentRelationModel{
			SchemaID:        schema.ID,
			SourceAgentName: sourceName,
			TargetAgentName: targetName,
		}
		if err := tx.Create(&relModel).Error; err != nil {
			return ForkedSchema{}, fmt.Errorf("create relation %s → %s: %w", sourceName, targetName, err)
		}
	}

	// 6. Create triggers. All template triggers target the entry agent +
	//    the newly forked schema, so the delegation tree has exactly one
	//    implicit entry point (§4 — triggers can only target entry
	//    agents).
	for _, t := range def.Triggers {
		triggerType := strings.TrimSpace(t.Type)
		if triggerType == "" {
			return ForkedSchema{}, fmt.Errorf("trigger with no type: %w", ErrInvalidTemplate)
		}
		config, err := toTriggerConfig(t.Config)
		if err != nil {
			return ForkedSchema{}, fmt.Errorf("trigger %q: %w", t.Title, err)
		}
		agentID := entryAgentID
		schemaID := schema.ID
		trigger := models.TriggerModel{
			Type:     triggerType,
			Title:    t.Title,
			AgentID:  &agentID,
			SchemaID: &schemaID,
			Enabled:  t.Enabled,
			Config:   config,
		}
		if err := tx.Create(&trigger).Error; err != nil {
			return ForkedSchema{}, fmt.Errorf("create trigger %q: %w", t.Title, err)
		}
	}

	return ForkedSchema{
		SchemaID:   schema.ID,
		SchemaName: schema.Name,
		AgentIDs:   agentIDByLogical,
	}, nil
}

// toTriggerConfig projects the template's untyped config map onto the
// strongly-typed jsonb shape used by TriggerModel (schedule / webhook_path).
// Unknown keys are ignored — the trigger jsonb is the source of truth, not
// the template.
func toTriggerConfig(raw map[string]interface{}) (models.TriggerConfig, error) {
	if raw == nil {
		return models.TriggerConfig{}, nil
	}
	buf, err := json.Marshal(raw)
	if err != nil {
		return models.TriggerConfig{}, fmt.Errorf("marshal trigger config: %w", err)
	}
	var out models.TriggerConfig
	if err := json.Unmarshal(buf, &out); err != nil {
		return models.TriggerConfig{}, fmt.Errorf("unmarshal trigger config: %w", err)
	}
	return out, nil
}

// validateDefinition asserts self-consistency of the template before the
// fork transaction starts. Any failure short-circuits with
// ErrInvalidTemplate so the DB never sees partial writes.
func validateDefinition(def domain.SchemaTemplateDefinition) error {
	if len(def.Agents) == 0 {
		return fmt.Errorf("no agents defined: %w", ErrInvalidTemplate)
	}
	if strings.TrimSpace(def.EntryAgentName) == "" {
		return fmt.Errorf("entry_agent_name is empty: %w", ErrInvalidTemplate)
	}

	names := make(map[string]struct{}, len(def.Agents))
	for _, a := range def.Agents {
		if strings.TrimSpace(a.Name) == "" {
			return fmt.Errorf("agent with empty name: %w", ErrInvalidTemplate)
		}
		if _, dup := names[a.Name]; dup {
			return fmt.Errorf("duplicate agent name %q: %w", a.Name, ErrInvalidTemplate)
		}
		names[a.Name] = struct{}{}
	}
	if _, ok := names[def.EntryAgentName]; !ok {
		return fmt.Errorf("entry agent %q not in agents list: %w", def.EntryAgentName, ErrInvalidTemplate)
	}

	for _, rel := range def.Relations {
		if _, ok := names[rel.Source]; !ok {
			return fmt.Errorf("relation source %q not in agents list: %w", rel.Source, ErrInvalidTemplate)
		}
		if _, ok := names[rel.Target]; !ok {
			return fmt.Errorf("relation target %q not in agents list: %w", rel.Target, ErrInvalidTemplate)
		}
	}

	return nil
}
