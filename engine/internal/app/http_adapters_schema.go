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

// schemaServiceHTTPAdapter bridges GORMSchemaRepository to the http.SchemaService interface.
type schemaServiceHTTPAdapter struct {
	repo *configrepo.GORMSchemaRepository
}

func (a *schemaServiceHTTPAdapter) ListSchemas(ctx context.Context) ([]deliveryhttp.SchemaInfo, error) {
	records, err := a.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list schemas: %w", err)
	}

	result := make([]deliveryhttp.SchemaInfo, 0, len(records))
	for _, r := range records {
		result = append(result, deliveryhttp.SchemaInfo{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Agents:      r.AgentNames,
			IsSystem:    r.IsSystem,
			CreatedAt:   r.CreatedAt,
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
		ID:          record.ID,
		Name:        record.Name,
		Description: record.Description,
		Agents:      record.AgentNames,
		IsSystem:    record.IsSystem,
		CreatedAt:   record.CreatedAt,
	}, nil
}

func (a *schemaServiceHTTPAdapter) CreateSchema(ctx context.Context, req deliveryhttp.CreateSchemaRequest) (*deliveryhttp.SchemaInfo, error) {
	record := &configrepo.SchemaRecord{
		Name:        req.Name,
		Description: req.Description,
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
	record := &configrepo.SchemaRecord{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := a.repo.Update(ctx, id, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("schema not found: %s", id))
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return pkgerrors.AlreadyExists(fmt.Sprintf("schema with name %q already exists", req.Name))
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
type agentRelationServiceHTTPAdapter struct {
	repo *configrepo.GORMAgentRelationRepository
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
	record := &configrepo.AgentRelationRecord{
		SchemaID:        schemaID,
		SourceAgentID: req.Source,
		TargetAgentID: req.Target,
		Config:          req.Config,
	}
	if err := a.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create agent relation: %w", err)
	}

	return &deliveryhttp.AgentRelationInfo{
		ID:              record.ID,
		SchemaID:        record.SchemaID,
		SourceAgentID: record.SourceAgentID,
		TargetAgentID: record.TargetAgentID,
		Config:          record.Config,
	}, nil
}

func (a *agentRelationServiceHTTPAdapter) UpdateAgentRelation(ctx context.Context, id string, req deliveryhttp.CreateAgentRelationRequest) error {
	record := &configrepo.AgentRelationRecord{
		SourceAgentID: req.Source,
		TargetAgentID: req.Target,
		Config:          req.Config,
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
