package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"gorm.io/gorm"
)

// resolveAgentNameByID resolves an agent UUID to its name via a raw DB query.
// Returns an empty string when the ID is nil or the agent is not found.
func (a *schemaServiceHTTPAdapter) resolveAgentNameByID(ctx context.Context, agentID *string) string {
	if agentID == nil || *agentID == "" {
		return ""
	}
	var name string
	_ = a.db.WithContext(ctx).Raw("SELECT name FROM agents WHERE id = ? LIMIT 1", *agentID).Scan(&name).Error
	return name
}

// countAgentsInSchema returns the number of distinct agents linked to the schema
// through agent_relations (union of source and target).
func (a *schemaServiceHTTPAdapter) countAgentsInSchema(ctx context.Context, schemaID string) int {
	var count int64
	_ = a.db.WithContext(ctx).Raw(`
		SELECT COUNT(DISTINCT agent_id) FROM (
			SELECT source_agent_id AS agent_id FROM agent_relations WHERE schema_id = ?
			UNION
			SELECT target_agent_id AS agent_id FROM agent_relations WHERE schema_id = ?
		) members`, schemaID, schemaID).Scan(&count).Error
	return int(count)
}

// schemaServiceHTTPAdapter bridges GORMSchemaRepository to the http.SchemaService interface.
type schemaServiceHTTPAdapter struct {
	repo *configrepo.GORMSchemaRepository
	db   *gorm.DB
}

func (a *schemaServiceHTTPAdapter) ListSchemas(ctx context.Context) ([]deliveryhttp.SchemaInfo, error) {
	records, err := a.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}

	result := make([]deliveryhttp.SchemaInfo, 0, len(records))
	for _, r := range records {
		result = append(result, deliveryhttp.SchemaInfo{
			ID:             r.ID,
			Name:           r.Name,
			Description:    r.Description,
			Agents:         r.AgentNames,
			IsSystem:       r.IsSystem,
			EntryAgentName: a.resolveAgentNameByID(ctx, r.EntryAgentID),
			AgentsCount:    a.countAgentsInSchema(ctx, r.ID),
			CreatedAt:      r.CreatedAt,
		})
	}
	return result, nil
}

func (a *schemaServiceHTTPAdapter) GetSchema(ctx context.Context, id string) (*deliveryhttp.SchemaInfo, error) {
	record, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound(fmt.Sprintf("schema not found: %s", id))
		}
		return nil, fmt.Errorf("get schema: %w", err)
	}

	return &deliveryhttp.SchemaInfo{
		ID:             record.ID,
		Name:           record.Name,
		Description:    record.Description,
		Agents:         record.AgentNames,
		IsSystem:       record.IsSystem,
		EntryAgentName: a.resolveAgentNameByID(ctx, record.EntryAgentID),
		AgentsCount:    a.countAgentsInSchema(ctx, record.ID),
		CreatedAt:      record.CreatedAt,
	}, nil
}

func (a *schemaServiceHTTPAdapter) CreateSchema(ctx context.Context, req deliveryhttp.CreateSchemaRequest) (*deliveryhttp.SchemaInfo, error) {
	record := &configrepo.SchemaRecord{
		Name:         req.Name,
		Description:  req.Description,
		EntryAgentID: req.EntryAgentID,
	}
	if err := a.repo.Create(ctx, record); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, pkgerrors.AlreadyExists(fmt.Sprintf("schema with name %q already exists", req.Name))
		}
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &deliveryhttp.SchemaInfo{
		ID:          record.ID,
		Name:        record.Name,
		Description: record.Description,
		CreatedAt:   record.CreatedAt,
	}, nil
}

func (a *schemaServiceHTTPAdapter) UpdateSchema(ctx context.Context, id string, req deliveryhttp.UpdateSchemaRequest) error {
	existing, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("schema not found: %s", id))
		}
		return fmt.Errorf("load schema for update: %w", err)
	}

	record := &configrepo.SchemaRecord{
		Name:         existing.Name,
		Description:  existing.Description,
		EntryAgentID: existing.EntryAgentID,
	}
	if req.Name != nil {
		record.Name = *req.Name
	}
	if req.Description != nil {
		record.Description = *req.Description
	}
	if req.EntryAgentID != nil {
		if *req.EntryAgentID == "" {
			record.EntryAgentID = nil
		} else {
			v := *req.EntryAgentID
			record.EntryAgentID = &v
		}
	}

	if err := a.repo.Update(ctx, id, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("schema not found: %s", id))
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return pkgerrors.AlreadyExists(fmt.Sprintf("schema with name %q already exists", record.Name))
		}
		return fmt.Errorf("update schema: %w", err)
	}
	return nil
}

func (a *schemaServiceHTTPAdapter) DeleteSchema(ctx context.Context, id string) error {
	if err := a.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("schema not found: %s", id))
		}
		return fmt.Errorf("delete schema: %w", err)
	}
	return nil
}

// ListSchemaAgents returns the derived membership list for a schema (V2:
// union of source/target agents in agent_relations — see
// docs/architecture/agent-first-runtime.md §2.1).
func (a *schemaServiceHTTPAdapter) ListSchemaAgents(ctx context.Context, schemaID string) ([]string, error) {
	names, err := a.repo.ListAgents(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("list schema agents: %w", err)
	}
	if names == nil {
		return []string{}, nil
	}
	return names, nil
}

// agentRelationServiceHTTPAdapter bridges GORMAgentRelationRepository to the
// http.AgentRelationService interface.
//
// agentRepo is used to resolve agent names to UUIDs — the API accepts either
// form in source/target fields so admin UI can work directly with agent names.
type agentRelationServiceHTTPAdapter struct {
	repo      *configrepo.GORMAgentRelationRepository
	agentRepo *configrepo.GORMAgentRepository
}

// resolveAgentRef returns the agent UUID for a name or UUID reference.
// UUIDs pass through verbatim. Names are looked up via agentRepo.
// Returns InvalidInput error for unknown names so the caller can surface 400.
func (a *agentRelationServiceHTTPAdapter) resolveAgentRef(ctx context.Context, ref string) (string, error) {
	if ref == "" {
		return "", pkgerrors.InvalidInput("agent reference is empty")
	}
	if isUUID(ref) {
		return ref, nil
	}
	rec, err := a.agentRepo.GetByName(ctx, ref)
	if err != nil || rec == nil {
		return "", pkgerrors.InvalidInput(fmt.Sprintf("agent not found: %s", ref))
	}
	return rec.ID, nil
}

// isUUID returns true for canonical 8-4-4-4-12 hex strings.
func isUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') && !(c >= 'A' && c <= 'F') {
				return false
			}
		}
	}
	return true
}

func (a *agentRelationServiceHTTPAdapter) ListAgentRelations(ctx context.Context, schemaID string) ([]deliveryhttp.AgentRelationInfo, error) {
	records, err := a.repo.List(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("list agent relations: %w", err)
	}

	result := make([]deliveryhttp.AgentRelationInfo, 0, len(records))
	for _, r := range records {
		result = append(result, deliveryhttp.AgentRelationInfo{
			ID:              r.ID,
			SchemaID:        r.SchemaID,
			SourceAgentID: r.SourceAgentID,
			TargetAgentID: r.TargetAgentID,
			Config:          r.Config,
		})
	}
	return result, nil
}

func (a *agentRelationServiceHTTPAdapter) GetAgentRelation(ctx context.Context, id string) (*deliveryhttp.AgentRelationInfo, error) {
	record, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound(fmt.Sprintf("agent relation not found: %s", id))
		}
		return nil, fmt.Errorf("get agent relation: %w", err)
	}

	return &deliveryhttp.AgentRelationInfo{
		ID:              record.ID,
		SchemaID:        record.SchemaID,
		SourceAgentID: record.SourceAgentID,
		TargetAgentID: record.TargetAgentID,
		Config:          record.Config,
	}, nil
}

func (a *agentRelationServiceHTTPAdapter) CreateAgentRelation(ctx context.Context, schemaID string, req deliveryhttp.CreateAgentRelationRequest) (*deliveryhttp.AgentRelationInfo, error) {
	sourceID, err := a.resolveAgentRef(ctx, req.Source)
	if err != nil {
		return nil, err
	}
	targetID, err := a.resolveAgentRef(ctx, req.Target)
	if err != nil {
		return nil, err
	}
	if sourceID == targetID {
		return nil, pkgerrors.InvalidInput("source and target must be different agents")
	}

	record := &configrepo.AgentRelationRecord{
		SchemaID:      schemaID,
		SourceAgentID: sourceID,
		TargetAgentID: targetID,
		Config:        req.Config,
	}
	if err := a.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create agent relation: %w", err)
	}

	return &deliveryhttp.AgentRelationInfo{
		ID:            record.ID,
		SchemaID:      record.SchemaID,
		SourceAgentID: record.SourceAgentID,
		TargetAgentID: record.TargetAgentID,
		Config:        record.Config,
	}, nil
}

func (a *agentRelationServiceHTTPAdapter) UpdateAgentRelation(ctx context.Context, id string, req deliveryhttp.CreateAgentRelationRequest) error {
	sourceID, err := a.resolveAgentRef(ctx, req.Source)
	if err != nil {
		return err
	}
	targetID, err := a.resolveAgentRef(ctx, req.Target)
	if err != nil {
		return err
	}
	if sourceID == targetID {
		return pkgerrors.InvalidInput("source and target must be different agents")
	}

	record := &configrepo.AgentRelationRecord{
		SourceAgentID: sourceID,
		TargetAgentID: targetID,
		Config:        req.Config,
	}
	if err := a.repo.Update(ctx, id, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("agent relation not found: %s", id))
		}
		return fmt.Errorf("update agent relation: %w", err)
	}
	return nil
}

func (a *agentRelationServiceHTTPAdapter) DeleteAgentRelation(ctx context.Context, id string) error {
	if err := a.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("agent relation not found: %s", id))
		}
		return fmt.Errorf("delete agent relation: %w", err)
	}
	return nil
}

// agentSchemaListerHTTPAdapter bridges GORMSchemaRepository to the http.AgentSchemaLister interface.
type agentSchemaListerHTTPAdapter struct {
	repo *configrepo.GORMSchemaRepository
}

func (a *agentSchemaListerHTTPAdapter) ListSchemasForAgent(ctx context.Context, agentName string) ([]string, error) {
	return a.repo.ListSchemasForAgent(ctx, agentName)
}
