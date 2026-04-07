package app

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/service/assistant"
	"github.com/syntheticinc/bytebrew/engine/internal/service/capability"
	"gorm.io/gorm"
)

// assistantServiceHTTPAdapter bridges assistant.Builder to http.AssistantService.
type assistantServiceHTTPAdapter struct {
	builder *assistant.Builder
}

func (a *assistantServiceHTTPAdapter) HandleMessage(ctx context.Context, sessionID, message string,
	hasSchemas bool, eventStream domain.AgentEventStream) (string, error) {
	return a.builder.HandleMessage(ctx, sessionID, message, hasSchemas, eventStream)
}

// schemaCounterHTTPAdapter bridges GORMSchemaRepository to http.SchemaCounter.
type schemaCounterHTTPAdapter struct {
	repo *config_repo.GORMSchemaRepository
}

func (a *schemaCounterHTTPAdapter) HasSchemas(ctx context.Context) (bool, error) {
	schemas, err := a.repo.List(ctx)
	if err != nil {
		return false, err
	}
	return len(schemas) > 0, nil
}

// assistantAdminOpsAdapter bridges DB operations to assistant.AdminOperations.
type assistantAdminOpsAdapter struct {
	db       *gorm.DB
	registry *agent_registry.AgentRegistry
}

func (a *assistantAdminOpsAdapter) CreateSchema(ctx context.Context, name, description string) (uint, error) {
	repo := config_repo.NewGORMSchemaRepository(a.db)
	record := config_repo.SchemaRecord{Name: name, Description: description}
	if err := repo.Create(ctx, &record); err != nil {
		return 0, fmt.Errorf("create schema: %w", err)
	}
	return record.ID, nil
}

func (a *assistantAdminOpsAdapter) CreateAgent(ctx context.Context, name, systemPrompt, model string) error {
	agentRepo := config_repo.NewGORMAgentRepository(a.db)
	record := &config_repo.AgentRecord{
		Name:         name,
		SystemPrompt: systemPrompt,
		ModelName:    model,
	}
	if err := agentRepo.Create(ctx, record); err != nil {
		return fmt.Errorf("create agent: %w", err)
	}
	if a.registry != nil {
		_ = a.registry.Load(ctx)
	}
	return nil
}

func (a *assistantAdminOpsAdapter) AddAgentToSchema(ctx context.Context, schemaID uint, agentName string) error {
	repo := config_repo.NewGORMSchemaRepository(a.db)
	if err := repo.AddAgent(ctx, schemaID, agentName); err != nil {
		return fmt.Errorf("add agent to schema: %w", err)
	}
	return nil
}

func (a *assistantAdminOpsAdapter) CreateEdge(ctx context.Context, schemaID uint, source, target, edgeType string) error {
	repo := config_repo.NewGORMEdgeRepository(a.db)
	record := config_repo.EdgeRecord{
		SchemaID:        schemaID,
		SourceAgentName: source,
		TargetAgentName: target,
		Type:            edgeType,
	}
	if err := repo.Create(ctx, &record); err != nil {
		return fmt.Errorf("create edge: %w", err)
	}
	return nil
}

func (a *assistantAdminOpsAdapter) CreateTrigger(ctx context.Context, agentName, triggerType string) error {
	// Resolve agent ID
	var agent models.AgentModel
	if err := a.db.WithContext(ctx).Where("name = ?", agentName).First(&agent).Error; err != nil {
		return fmt.Errorf("resolve agent %q: %w", agentName, err)
	}
	triggerRepo := config_repo.NewGORMTriggerRepository(a.db)
	trigger := &models.TriggerModel{
		Type:    triggerType,
		Title:   agentName + " trigger",
		AgentID: agent.ID,
		Enabled: true,
	}
	if err := triggerRepo.Create(ctx, trigger); err != nil {
		return fmt.Errorf("create trigger: %w", err)
	}
	return nil
}

// capabilityInjectorAdapter bridges GORMCapabilityRepository to capability.CapabilityReader.
type capabilityInjectorAdapter struct {
	repo *config_repo.GORMCapabilityRepository
}

func (a *capabilityInjectorAdapter) ListEnabledByAgent(ctx context.Context, agentName string) ([]capability.CapabilityRecord, error) {
	records, err := a.repo.ListEnabledByAgent(ctx, agentName)
	if err != nil {
		return nil, err
	}
	result := make([]capability.CapabilityRecord, 0, len(records))
	for _, r := range records {
		result = append(result, capability.CapabilityRecord{
			ID:        r.ID,
			AgentName: r.AgentName,
			Type:      r.Type,
			Config:    r.Config,
			Enabled:   r.Enabled,
		})
	}
	return result, nil
}
