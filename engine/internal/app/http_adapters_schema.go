package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"gorm.io/gorm"
)

// schemaServiceHTTPAdapter bridges GORMSchemaRepository to the http.SchemaService interface.
type schemaServiceHTTPAdapter struct {
	repo *config_repo.GORMSchemaRepository
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
		})
	}
	return result, nil
}

func (a *schemaServiceHTTPAdapter) GetSchema(ctx context.Context, id uint) (*deliveryhttp.SchemaInfo, error) {
	record, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound(fmt.Sprintf("schema not found: %d", id))
		}
		return nil, fmt.Errorf("get schema: %w", err)
	}

	return &deliveryhttp.SchemaInfo{
		ID:          record.ID,
		Name:        record.Name,
		Description: record.Description,
		Agents:      record.AgentNames,
	}, nil
}

func (a *schemaServiceHTTPAdapter) CreateSchema(ctx context.Context, req deliveryhttp.CreateSchemaRequest) (*deliveryhttp.SchemaInfo, error) {
	record := &config_repo.SchemaRecord{
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
	}, nil
}

func (a *schemaServiceHTTPAdapter) UpdateSchema(ctx context.Context, id uint, req deliveryhttp.UpdateSchemaRequest) error {
	record := &config_repo.SchemaRecord{
		Name:        req.Name,
		Description: req.Description,
	}
	if err := a.repo.Update(ctx, id, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("schema not found: %d", id))
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return pkgerrors.AlreadyExists(fmt.Sprintf("schema with name %q already exists", req.Name))
		}
		return fmt.Errorf("update schema: %w", err)
	}
	return nil
}

func (a *schemaServiceHTTPAdapter) DeleteSchema(ctx context.Context, id uint) error {
	if err := a.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("schema not found: %d", id))
		}
		return fmt.Errorf("delete schema: %w", err)
	}
	return nil
}

func (a *schemaServiceHTTPAdapter) AddSchemaAgent(ctx context.Context, schemaID uint, agentName string) error {
	if err := a.repo.AddAgent(ctx, schemaID, agentName); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("agent not found: %s", agentName))
		}
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return pkgerrors.AlreadyExists(fmt.Sprintf("agent %q already in schema", agentName))
		}
		return fmt.Errorf("add agent to schema: %w", err)
	}
	return nil
}

func (a *schemaServiceHTTPAdapter) RemoveSchemaAgent(ctx context.Context, schemaID uint, agentName string) error {
	if err := a.repo.RemoveAgent(ctx, schemaID, agentName); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("agent %q not in schema %d", agentName, schemaID))
		}
		return fmt.Errorf("remove agent from schema: %w", err)
	}
	return nil
}

func (a *schemaServiceHTTPAdapter) ListSchemaAgents(ctx context.Context, schemaID uint) ([]string, error) {
	names, err := a.repo.ListAgents(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("list schema agents: %w", err)
	}
	return names, nil
}

// gateServiceHTTPAdapter bridges GORMGateRepository to the http.GateService interface.
type gateServiceHTTPAdapter struct {
	repo *config_repo.GORMGateRepository
}

func (a *gateServiceHTTPAdapter) ListGates(ctx context.Context, schemaID uint) ([]deliveryhttp.GateInfo, error) {
	records, err := a.repo.List(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("list gates: %w", err)
	}

	result := make([]deliveryhttp.GateInfo, 0, len(records))
	for _, r := range records {
		result = append(result, deliveryhttp.GateInfo{
			ID:            r.ID,
			SchemaID:      r.SchemaID,
			Name:          r.Name,
			ConditionType: r.ConditionType,
			Config:        r.Config,
			MaxIterations: r.MaxIterations,
			Timeout:       r.Timeout,
		})
	}
	return result, nil
}

func (a *gateServiceHTTPAdapter) GetGate(ctx context.Context, id uint) (*deliveryhttp.GateInfo, error) {
	record, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound(fmt.Sprintf("gate not found: %d", id))
		}
		return nil, fmt.Errorf("get gate: %w", err)
	}

	return &deliveryhttp.GateInfo{
		ID:            record.ID,
		SchemaID:      record.SchemaID,
		Name:          record.Name,
		ConditionType: record.ConditionType,
		Config:        record.Config,
		MaxIterations: record.MaxIterations,
		Timeout:       record.Timeout,
	}, nil
}

func (a *gateServiceHTTPAdapter) CreateGate(ctx context.Context, schemaID uint, req deliveryhttp.CreateGateRequest) (*deliveryhttp.GateInfo, error) {
	condType := req.ConditionType
	if condType == "" {
		condType = "all"
	}

	record := &config_repo.GateRecord{
		SchemaID:      schemaID,
		Name:          req.Name,
		ConditionType: condType,
		Config:        req.Config,
		MaxIterations: req.MaxIterations,
		Timeout:       req.Timeout,
	}
	if err := a.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create gate: %w", err)
	}

	return &deliveryhttp.GateInfo{
		ID:            record.ID,
		SchemaID:      record.SchemaID,
		Name:          record.Name,
		ConditionType: record.ConditionType,
		Config:        record.Config,
		MaxIterations: record.MaxIterations,
		Timeout:       record.Timeout,
	}, nil
}

func (a *gateServiceHTTPAdapter) UpdateGate(ctx context.Context, id uint, req deliveryhttp.CreateGateRequest) error {
	record := &config_repo.GateRecord{
		Name:          req.Name,
		ConditionType: req.ConditionType,
		Config:        req.Config,
		MaxIterations: req.MaxIterations,
		Timeout:       req.Timeout,
	}
	if err := a.repo.Update(ctx, id, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("gate not found: %d", id))
		}
		return fmt.Errorf("update gate: %w", err)
	}
	return nil
}

func (a *gateServiceHTTPAdapter) DeleteGate(ctx context.Context, id uint) error {
	if err := a.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("gate not found: %d", id))
		}
		return fmt.Errorf("delete gate: %w", err)
	}
	return nil
}

// edgeServiceHTTPAdapter bridges GORMEdgeRepository to the http.EdgeService interface.
type edgeServiceHTTPAdapter struct {
	repo *config_repo.GORMEdgeRepository
}

func (a *edgeServiceHTTPAdapter) ListEdges(ctx context.Context, schemaID uint) ([]deliveryhttp.EdgeInfo, error) {
	records, err := a.repo.List(ctx, schemaID)
	if err != nil {
		return nil, fmt.Errorf("list edges: %w", err)
	}

	result := make([]deliveryhttp.EdgeInfo, 0, len(records))
	for _, r := range records {
		result = append(result, deliveryhttp.EdgeInfo{
			ID:              r.ID,
			SchemaID:        r.SchemaID,
			SourceAgentName: r.SourceAgentName,
			TargetAgentName: r.TargetAgentName,
			Type:            r.Type,
			Config:          r.Config,
		})
	}
	return result, nil
}

func (a *edgeServiceHTTPAdapter) GetEdge(ctx context.Context, id uint) (*deliveryhttp.EdgeInfo, error) {
	record, err := a.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound(fmt.Sprintf("edge not found: %d", id))
		}
		return nil, fmt.Errorf("get edge: %w", err)
	}

	return &deliveryhttp.EdgeInfo{
		ID:              record.ID,
		SchemaID:        record.SchemaID,
		SourceAgentName: record.SourceAgentName,
		TargetAgentName: record.TargetAgentName,
		Type:            record.Type,
		Config:          record.Config,
	}, nil
}

func (a *edgeServiceHTTPAdapter) CreateEdge(ctx context.Context, schemaID uint, req deliveryhttp.CreateEdgeRequest) (*deliveryhttp.EdgeInfo, error) {
	edgeType := req.Type
	if edgeType == "" {
		edgeType = "flow"
	}

	record := &config_repo.EdgeRecord{
		SchemaID:        schemaID,
		SourceAgentName: req.Source,
		TargetAgentName: req.Target,
		Type:            edgeType,
		Config:          req.Config,
	}
	if err := a.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("create edge: %w", err)
	}

	return &deliveryhttp.EdgeInfo{
		ID:              record.ID,
		SchemaID:        record.SchemaID,
		SourceAgentName: record.SourceAgentName,
		TargetAgentName: record.TargetAgentName,
		Type:            record.Type,
		Config:          record.Config,
	}, nil
}

func (a *edgeServiceHTTPAdapter) UpdateEdge(ctx context.Context, id uint, req deliveryhttp.CreateEdgeRequest) error {
	record := &config_repo.EdgeRecord{
		SourceAgentName: req.Source,
		TargetAgentName: req.Target,
		Type:            req.Type,
		Config:          req.Config,
	}
	if err := a.repo.Update(ctx, id, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("edge not found: %d", id))
		}
		return fmt.Errorf("update edge: %w", err)
	}
	return nil
}

func (a *edgeServiceHTTPAdapter) DeleteEdge(ctx context.Context, id uint) error {
	if err := a.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("edge not found: %d", id))
		}
		return fmt.Errorf("delete edge: %w", err)
	}
	return nil
}

// agentSchemaListerHTTPAdapter bridges GORMSchemaRepository to the http.AgentSchemaLister interface.
type agentSchemaListerHTTPAdapter struct {
	repo *config_repo.GORMSchemaRepository
}

func (a *agentSchemaListerHTTPAdapter) ListSchemasForAgent(ctx context.Context, agentName string) ([]string, error) {
	return a.repo.ListSchemasForAgent(ctx, agentName)
}
