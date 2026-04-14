package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"gorm.io/gorm"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agentregistry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
)

// agentManagerHTTPAdapter bridges GORMAgentRepository + AgentRegistry to the http.AgentManager interface.
type agentManagerHTTPAdapter struct {
	repo       *configrepo.GORMAgentRepository
	registry   *agentregistry.AgentRegistry
	db         *gorm.DB
	schemaRepo *configrepo.GORMSchemaRepository
}

func (a *agentManagerHTTPAdapter) ListAgents(ctx context.Context) ([]deliveryhttp.AgentInfo, error) {
	records, err := a.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	result := make([]deliveryhttp.AgentInfo, 0, len(records))
	for _, rec := range records {
		info := deliveryhttp.AgentInfo{
			Name:         rec.Name,
			ToolsCount:   len(rec.BuiltinTools) + len(rec.CustomTools),
			Kit:          rec.Kit,
			HasKnowledge: rec.KnowledgePath != "",
			IsSystem:     rec.IsSystem,
		}
		if a.schemaRepo != nil {
			schemaNames, _ := a.schemaRepo.ListSchemasForAgent(ctx, rec.Name)
			info.UsedInSchemas = schemaNames
		}
		result = append(result, info)
	}
	return result, nil
}

func (a *agentManagerHTTPAdapter) GetAgent(ctx context.Context, name string) (*deliveryhttp.AgentDetail, error) {
	rec, err := a.repo.GetByName(ctx, name)
	if err != nil {
		return nil, nil
	}

	tools := make([]string, 0, len(rec.BuiltinTools)+len(rec.CustomTools))
	tools = append(tools, rec.BuiltinTools...)
	for _, ct := range rec.CustomTools {
		tools = append(tools, ct.Name)
	}

	detail := &deliveryhttp.AgentDetail{
		AgentInfo: deliveryhttp.AgentInfo{
			Name:         rec.Name,
			ToolsCount:   len(tools),
			Kit:          rec.Kit,
			HasKnowledge: rec.KnowledgePath != "",
			IsSystem:     rec.IsSystem,
		},
		SystemPrompt:    rec.SystemPrompt,
		KnowledgePath:   rec.KnowledgePath,
		Tools:           tools,
		CanSpawn:        rec.CanSpawn,
		Lifecycle:       rec.Lifecycle,
		ToolExecution:   rec.ToolExecution,
		MaxSteps:        rec.MaxSteps,
		MaxContextSize:  rec.MaxContextSize,
		MaxTurnDuration: rec.MaxTurnDuration,
		Temperature:     rec.Temperature,
		TopP:            rec.TopP,
		MaxTokens:       rec.MaxTokens,
		StopSequences:   rec.StopSequences,
		ConfirmBefore:   rec.ConfirmBefore,
		MCPServers:      rec.MCPServers,
	}

	// Load MCP servers separately (GORM many2many has naming issues).
	mcpNames, err := a.loadMCPServersForAgent(ctx, name)
	if err == nil {
		detail.MCPServers = mcpNames
	}

	// Resolve model ID for the response.
	detail.ModelID = a.resolveModelID(ctx, rec.ModelName)

	if rec.Escalation != nil {
		detail.Escalation = &deliveryhttp.AgentEscalation{
			Action:     rec.Escalation.Action,
			WebhookURL: rec.Escalation.WebhookURL,
			Triggers:   rec.Escalation.Triggers,
		}
	}

	// Populate used_in_schemas (AC-ENT-03)
	if a.schemaRepo != nil {
		schemaNames, _ := a.schemaRepo.ListSchemasForAgent(ctx, name)
		detail.UsedInSchemas = schemaNames
	}

	return detail, nil
}

func (a *agentManagerHTTPAdapter) CreateAgent(ctx context.Context, req deliveryhttp.CreateAgentRequest) (*deliveryhttp.AgentDetail, error) {
	// WP-4: Prevent using embedding models as agent model.
	if req.ModelID != nil {
		var llm models.LLMProviderModel
		if err := a.db.Where("id = ?", *req.ModelID).First(&llm).Error; err == nil && llm.Type == "embedding" {
			return nil, pkgerrors.InvalidInput("embedding models cannot be used as agent model, use a chat model instead")
		}
	}

	record := a.toAgentRecord(req)
	if err := a.repo.Create(ctx, record); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, pkgerrors.AlreadyExists(fmt.Sprintf("agent with name %q already exists", req.Name))
		}
		return nil, fmt.Errorf("create agent: %w", err)
	}

	if err := a.registry.Reload(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to reload agent registry after create", "error", err)
	}

	return a.GetAgent(ctx, req.Name)
}

func (a *agentManagerHTTPAdapter) UpdateAgent(ctx context.Context, name string, req deliveryhttp.CreateAgentRequest) (*deliveryhttp.AgentDetail, error) {
	// WP-4: Prevent using embedding models as agent model.
	if req.ModelID != nil {
		var llm models.LLMProviderModel
		if err := a.db.Where("id = ?", *req.ModelID).First(&llm).Error; err == nil && llm.Type == "embedding" {
			return nil, pkgerrors.InvalidInput("embedding models cannot be used as agent model, use a chat model instead")
		}
	}

	record := a.toAgentRecord(req)

	// Preserve is_system and builtin tools from the existing record.
	// is_system is not settable via HTTP.
	// For system agents: if the request doesn't specify tools, preserve existing builtin tools
	// to prevent accidental tool erasure during model/prompt updates.
	if existing, err := a.repo.GetByName(ctx, name); err == nil && existing != nil {
		record.IsSystem = existing.IsSystem
		if existing.IsSystem && len(record.BuiltinTools) == 0 && len(existing.BuiltinTools) > 0 {
			record.BuiltinTools = existing.BuiltinTools
		}
		if !existing.IsSystem {
			for _, toolName := range record.BuiltinTools {
				if strings.HasPrefix(toolName, "admin_") {
					return nil, pkgerrors.InvalidInput("admin tools are reserved for system agents")
				}
			}
		}
	}

	if err := a.repo.Update(ctx, name, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound(fmt.Sprintf("agent not found: %s", name))
		}
		return nil, fmt.Errorf("update agent: %w", err)
	}

	if err := a.registry.Reload(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to reload agent registry after update", "error", err)
	}

	// Use the updated name (could have been renamed).
	lookupName := req.Name
	if lookupName == "" {
		lookupName = name
	}
	return a.GetAgent(ctx, lookupName)
}

func (a *agentManagerHTTPAdapter) DeleteAgent(ctx context.Context, name string) error {
	// System agents cannot be deleted via API.
	existing, err := a.repo.GetByName(ctx, name)
	if err == nil && existing != nil && existing.IsSystem {
		return pkgerrors.Forbidden(fmt.Sprintf("system agent %q cannot be deleted", name))
	}

	// BUG-014: Delete triggers before agent to avoid FK constraint on fk_triggers_agent.
	if err := a.db.WithContext(ctx).
		Where("agent_id IN (SELECT id FROM agents WHERE name = ?)", name).
		Delete(&models.TriggerModel{}).Error; err != nil {
		slog.WarnContext(ctx, "failed to cascade-delete triggers", "agent", name, "error", err)
	}

	// BUG-004: Delete capabilities before agent to avoid FK constraint violation.
	if err := a.db.WithContext(ctx).
		Where("agent_id IN (SELECT id FROM agents WHERE name = ?)", name).
		Delete(&models.CapabilityModel{}).Error; err != nil {
		slog.WarnContext(ctx, "failed to cascade-delete capabilities", "agent", name, "error", err)
	}

	if err := a.repo.Delete(ctx, name); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("agent not found: %s", name))
		}
		return fmt.Errorf("delete agent: %w", err)
	}

	if err := a.registry.Reload(ctx); err != nil {
		slog.ErrorContext(ctx, "failed to reload agent registry after delete", "error", err)
	}

	return nil
}

func (a *agentManagerHTTPAdapter) toAgentRecord(req deliveryhttp.CreateAgentRequest) *configrepo.AgentRecord {
	rec := &configrepo.AgentRecord{
		Name:            req.Name,
		SystemPrompt:    req.SystemPrompt,
		Kit:             req.Kit,
		KnowledgePath:   req.KnowledgePath,
		Lifecycle:       req.Lifecycle,
		ToolExecution:   req.ToolExecution,
		MaxSteps:        req.MaxSteps,
		MaxContextSize:  req.MaxContextSize,
		MaxTurnDuration: req.MaxTurnDuration,
		Temperature:     req.Temperature,
		TopP:            req.TopP,
		MaxTokens:       req.MaxTokens,
		StopSequences:   req.StopSequences,
		ConfirmBefore:   req.ConfirmBefore,
		BuiltinTools:    req.Tools,
		CanSpawn:        req.CanSpawn,
		MCPServers:      req.MCPServers,
	}

	// Resolve model: by ID or by name.
	if req.ModelID != nil {
		rec.ModelID = req.ModelID
		var llm models.LLMProviderModel
		if err := a.db.Where("id = ?", *req.ModelID).First(&llm).Error; err == nil {
			rec.ModelName = llm.Name
		}
	} else if req.Model != "" {
		rec.ModelName = req.Model
	}

	if req.Escalation != nil {
		rec.Escalation = &configrepo.EscalationRecord{
			Action:     req.Escalation.Action,
			WebhookURL: req.Escalation.WebhookURL,
			Triggers:   req.Escalation.Triggers,
		}
	}

	// Apply defaults.
	if rec.Lifecycle == "" {
		rec.Lifecycle = "persistent"
	}
	if rec.ToolExecution == "" {
		rec.ToolExecution = "sequential"
	}
	if rec.MaxSteps == 0 {
		rec.MaxSteps = 50
	}
	if rec.MaxContextSize == 0 {
		rec.MaxContextSize = 16000
	}
	if rec.MaxTurnDuration == 0 {
		rec.MaxTurnDuration = 120
	}

	return rec
}

func (a *agentManagerHTTPAdapter) loadMCPServersForAgent(_ context.Context, name string) ([]string, error) {
	var agent models.AgentModel
	if err := a.db.Where("name = ?", name).First(&agent).Error; err != nil {
		return nil, err
	}

	var agentMCPs []models.AgentMCPServer
	if err := a.db.Preload("MCPServer").Where("agent_id = ?", agent.ID).Find(&agentMCPs).Error; err != nil {
		return nil, err
	}

	names := make([]string, 0, len(agentMCPs))
	for _, am := range agentMCPs {
		names = append(names, am.MCPServer.Name)
	}
	return names, nil
}

func (a *agentManagerHTTPAdapter) resolveModelID(_ context.Context, modelName string) *string {
	if modelName == "" {
		return nil
	}
	var llm models.LLMProviderModel
	if err := a.db.Where("name = ?", modelName).First(&llm).Error; err != nil {
		return nil
	}
	return &llm.ID
}
