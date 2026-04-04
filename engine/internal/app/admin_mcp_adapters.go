package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/admin_mcp"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/mcp"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
)

// ---------------------------------------------------------------------------
// Agent adapter (admin_mcp.AgentManager)
// ---------------------------------------------------------------------------

type adminMCPAgentAdapter struct {
	repo     *config_repo.GORMAgentRepository
	registry *agent_registry.AgentRegistry
	db       *gorm.DB
}

func (a *adminMCPAgentAdapter) ListAgents(ctx context.Context) ([]admin_mcp.AgentInfo, error) {
	records, err := a.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	result := make([]admin_mcp.AgentInfo, 0, len(records))
	for _, rec := range records {
		result = append(result, admin_mcp.AgentInfo{
			Name:         rec.Name,
			ToolsCount:   len(rec.BuiltinTools) + len(rec.CustomTools),
			Kit:          rec.Kit,
			HasKnowledge: rec.KnowledgePath != "",
		})
	}
	return result, nil
}

func (a *adminMCPAgentAdapter) GetAgent(ctx context.Context, name string) (*admin_mcp.AgentDetail, error) {
	rec, err := a.repo.GetByName(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get agent: %w", err)
	}

	agentTools := make([]string, 0, len(rec.BuiltinTools)+len(rec.CustomTools))
	agentTools = append(agentTools, rec.BuiltinTools...)
	for _, ct := range rec.CustomTools {
		agentTools = append(agentTools, ct.Name)
	}

	detail := &admin_mcp.AgentDetail{
		AgentInfo: admin_mcp.AgentInfo{
			Name:         rec.Name,
			ToolsCount:   len(agentTools),
			Kit:          rec.Kit,
			HasKnowledge: rec.KnowledgePath != "",
		},
		SystemPrompt:   rec.SystemPrompt,
		KnowledgePath:  rec.KnowledgePath,
		Tools:          agentTools,
		CanSpawn:       rec.CanSpawn,
		Lifecycle:      rec.Lifecycle,
		ToolExecution:  rec.ToolExecution,
		MaxSteps:       rec.MaxSteps,
		MaxContextSize: rec.MaxContextSize,
		ConfirmBefore:  rec.ConfirmBefore,
		MCPServers:     rec.MCPServers,
	}

	// Load model ID.
	if rec.ModelName != "" {
		var llmModel models.LLMProviderModel
		if err := a.db.Where("name = ?", rec.ModelName).First(&llmModel).Error; err == nil {
			id := llmModel.ID
			detail.ModelID = &id
		}
	}

	// Load MCP servers.
	var agentMCPs []models.AgentMCPServer
	if err := a.db.Preload("MCPServer").Where("agent_id = (SELECT id FROM agents WHERE name = ?)", name).Find(&agentMCPs).Error; err == nil {
		mcpNames := make([]string, 0, len(agentMCPs))
		for _, am := range agentMCPs {
			mcpNames = append(mcpNames, am.MCPServer.Name)
		}
		detail.MCPServers = mcpNames
	}

	if rec.Escalation != nil {
		detail.Escalation = &admin_mcp.AgentEscalation{
			Action:     rec.Escalation.Action,
			WebhookURL: rec.Escalation.WebhookURL,
			Triggers:   rec.Escalation.Triggers,
		}
	}

	return detail, nil
}

func (a *adminMCPAgentAdapter) CreateAgent(ctx context.Context, req admin_mcp.CreateAgentRequest) (*admin_mcp.AgentDetail, error) {
	record := a.toAgentRecord(req)
	if err := a.repo.Create(ctx, record); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, pkgerrors.AlreadyExists(fmt.Sprintf("agent with name %q already exists", req.Name))
		}
		return nil, fmt.Errorf("create agent: %w", err)
	}

	return a.GetAgent(ctx, req.Name)
}

func (a *adminMCPAgentAdapter) UpdateAgent(ctx context.Context, name string, req admin_mcp.CreateAgentRequest) (*admin_mcp.AgentDetail, error) {
	record := a.toAgentRecord(req)
	if err := a.repo.Update(ctx, name, record); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, pkgerrors.NotFound(fmt.Sprintf("agent not found: %s", name))
		}
		return nil, fmt.Errorf("update agent: %w", err)
	}

	return a.GetAgent(ctx, name)
}

func (a *adminMCPAgentAdapter) DeleteAgent(ctx context.Context, name string) error {
	if err := a.repo.Delete(ctx, name); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return pkgerrors.NotFound(fmt.Sprintf("agent not found: %s", name))
		}
		return fmt.Errorf("delete agent: %w", err)
	}

	return nil
}

func (a *adminMCPAgentAdapter) toAgentRecord(req admin_mcp.CreateAgentRequest) *config_repo.AgentRecord {
	rec := &config_repo.AgentRecord{
		Name:           req.Name,
		SystemPrompt:   req.SystemPrompt,
		ModelName:      req.Model,
		Kit:            req.Kit,
		KnowledgePath:  req.KnowledgePath,
		Lifecycle:      req.Lifecycle,
		ToolExecution:  req.ToolExecution,
		MaxSteps:       req.MaxSteps,
		MaxContextSize: req.MaxContextSize,
		ConfirmBefore:  req.ConfirmBefore,
		BuiltinTools:   req.Tools,
		CanSpawn:       req.CanSpawn,
		MCPServers:     req.MCPServers,
	}
	if req.Escalation != nil {
		rec.Escalation = &config_repo.EscalationRecord{
			Action:     req.Escalation.Action,
			WebhookURL: req.Escalation.WebhookURL,
			Triggers:   req.Escalation.Triggers,
		}
	}
	return rec
}

// ---------------------------------------------------------------------------
// Model adapter (admin_mcp.ModelManager)
// ---------------------------------------------------------------------------

type adminMCPModelAdapter struct {
	repo       *config_repo.GORMLLMProviderRepository
	modelCache ModelCacheInvalidator
}

func (m *adminMCPModelAdapter) ListModels(ctx context.Context) ([]admin_mcp.ModelResponse, error) {
	providers, err := m.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}

	result := make([]admin_mcp.ModelResponse, 0, len(providers))
	for _, p := range providers {
		result = append(result, admin_mcp.ModelResponse{
			ID:         p.ID,
			Name:       p.Name,
			Type:       p.Type,
			BaseURL:    p.BaseURL,
			ModelName:  p.ModelName,
			HasAPIKey:  p.APIKeyEncrypted != "",
			APIVersion: p.APIVersion,
			CreatedAt:  p.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (m *adminMCPModelAdapter) CreateModel(ctx context.Context, req admin_mcp.CreateModelRequest) (*admin_mcp.ModelResponse, error) {
	provider := &models.LLMProviderModel{
		Name:            req.Name,
		Type:            req.Type,
		BaseURL:         req.BaseURL,
		ModelName:       req.ModelName,
		APIKeyEncrypted: req.APIKey,
		APIVersion:      req.APIVersion,
	}

	if err := m.repo.Create(ctx, provider); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, pkgerrors.AlreadyExists(fmt.Sprintf("model with name %q already exists", req.Name))
		}
		return nil, fmt.Errorf("create model: %w", err)
	}

	return &admin_mcp.ModelResponse{
		ID:         provider.ID,
		Name:       provider.Name,
		Type:       provider.Type,
		BaseURL:    provider.BaseURL,
		ModelName:  provider.ModelName,
		HasAPIKey:  provider.APIKeyEncrypted != "",
		APIVersion: provider.APIVersion,
		CreatedAt:  provider.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (m *adminMCPModelAdapter) UpdateModel(ctx context.Context, name string, req admin_mcp.CreateModelRequest) (*admin_mcp.ModelResponse, error) {
	providers, err := m.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models for update: %w", err)
	}

	var existing *models.LLMProviderModel
	for i := range providers {
		if providers[i].Name == name {
			existing = &providers[i]
			break
		}
	}
	if existing == nil {
		return nil, pkgerrors.NotFound(fmt.Sprintf("model not found: %s", name))
	}

	update := &models.LLMProviderModel{
		Name:       req.Name,
		Type:       req.Type,
		BaseURL:    req.BaseURL,
		ModelName:  req.ModelName,
		APIVersion: req.APIVersion,
	}
	if req.APIKey != "" {
		update.APIKeyEncrypted = req.APIKey
	}

	if err := m.repo.Update(ctx, existing.ID, update); err != nil {
		return nil, fmt.Errorf("update model: %w", err)
	}

	if m.modelCache != nil {
		m.modelCache.Invalidate(existing.ID)
	}

	hasKey := existing.APIKeyEncrypted != ""
	if req.APIKey != "" {
		hasKey = true
	}

	return &admin_mcp.ModelResponse{
		ID:         existing.ID,
		Name:       coalesce(req.Name, existing.Name),
		Type:       coalesce(req.Type, existing.Type),
		BaseURL:    coalesce(req.BaseURL, existing.BaseURL),
		ModelName:  coalesce(req.ModelName, existing.ModelName),
		HasAPIKey:  hasKey,
		APIVersion: coalesce(req.APIVersion, existing.APIVersion),
		CreatedAt:  existing.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (m *adminMCPModelAdapter) DeleteModel(ctx context.Context, name string) error {
	providers, err := m.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("list models for delete: %w", err)
	}

	for _, p := range providers {
		if p.Name == name {
			return m.repo.Delete(ctx, p.ID)
		}
	}
	return pkgerrors.NotFound(fmt.Sprintf("model not found: %s", name))
}

// coalesce returns the first non-empty string.
func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Trigger adapter (admin_mcp.TriggerManager)
// ---------------------------------------------------------------------------

type adminMCPTriggerAdapter struct {
	repo *config_repo.GORMTriggerRepository
}

func (a *adminMCPTriggerAdapter) ListTriggers(ctx context.Context) ([]admin_mcp.TriggerResponse, error) {
	triggers, err := a.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list triggers: %w", err)
	}

	result := make([]admin_mcp.TriggerResponse, 0, len(triggers))
	for _, t := range triggers {
		result = append(result, admin_mcp.TriggerResponse{
			ID:          t.ID,
			Type:        t.Type,
			Title:       t.Title,
			AgentID:     t.AgentID,
			AgentName:   t.Agent.Name,
			Schedule:    t.Schedule,
			WebhookPath: t.WebhookPath,
			Description: t.Description,
			Enabled:     t.Enabled,
			LastFiredAt: formatTimePtr(t.LastFiredAt),
			CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (a *adminMCPTriggerAdapter) CreateTrigger(ctx context.Context, req admin_mcp.CreateTriggerRequest) (*admin_mcp.TriggerResponse, error) {
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	trigger := &models.TriggerModel{
		Type:        req.Type,
		Title:       req.Title,
		AgentID:     req.AgentID,
		Schedule:    req.Schedule,
		WebhookPath: req.WebhookPath,
		Description: req.Description,
		Enabled:     enabled,
	}

	if err := a.repo.Create(ctx, trigger); err != nil {
		return nil, fmt.Errorf("create trigger: %w", err)
	}

	return &admin_mcp.TriggerResponse{
		ID:          trigger.ID,
		Type:        trigger.Type,
		Title:       trigger.Title,
		AgentID:     trigger.AgentID,
		Schedule:    trigger.Schedule,
		WebhookPath: trigger.WebhookPath,
		Description: trigger.Description,
		Enabled:     trigger.Enabled,
		CreatedAt:   trigger.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (a *adminMCPTriggerAdapter) UpdateTrigger(ctx context.Context, id uint, req admin_mcp.CreateTriggerRequest) (*admin_mcp.TriggerResponse, error) {
	updateModel := &models.TriggerModel{
		Type:        req.Type,
		Title:       req.Title,
		AgentID:     req.AgentID,
		Schedule:    req.Schedule,
		WebhookPath: req.WebhookPath,
		Description: req.Description,
	}
	if req.Enabled != nil {
		updateModel.Enabled = *req.Enabled
	}

	if err := a.repo.Update(ctx, id, updateModel); err != nil {
		return nil, fmt.Errorf("update trigger: %w", err)
	}

	// Re-read from DB to get full record with agent preloaded.
	triggers, err := a.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list triggers after update: %w", err)
	}
	for _, t := range triggers {
		if t.ID == id {
			return &admin_mcp.TriggerResponse{
				ID:          t.ID,
				Type:        t.Type,
				Title:       t.Title,
				AgentID:     t.AgentID,
				AgentName:   t.Agent.Name,
				Schedule:    t.Schedule,
				WebhookPath: t.WebhookPath,
				Description: t.Description,
				Enabled:     t.Enabled,
				LastFiredAt: formatTimePtr(t.LastFiredAt),
				CreatedAt:   t.CreatedAt.Format(time.RFC3339),
			}, nil
		}
	}
	return nil, fmt.Errorf("trigger not found after update: %d", id)
}

func (a *adminMCPTriggerAdapter) DeleteTrigger(ctx context.Context, id uint) error {
	return a.repo.Delete(ctx, id)
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// ---------------------------------------------------------------------------
// MCP Server Lister adapter (admin_mcp.MCPServerLister)
// ---------------------------------------------------------------------------

type adminMCPServerListerAdapter struct {
	repo *config_repo.GORMMCPServerRepository
}

func (a *adminMCPServerListerAdapter) ListMCPServers(ctx context.Context) ([]admin_mcp.MCPServerResponse, error) {
	servers, err := a.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list mcp servers: %w", err)
	}

	result := make([]admin_mcp.MCPServerResponse, 0, len(servers))
	for _, s := range servers {
		resp := admin_mcp.MCPServerResponse{
			ID:   s.ID,
			Name: s.Name,
			Type: s.Type,
		}

		if s.Command != "" {
			resp.Command = s.Command
		}
		if s.URL != "" {
			resp.URL = s.URL
		}
		if s.Args != "" {
			var args []string
			if err := json.Unmarshal([]byte(s.Args), &args); err == nil {
				resp.Args = args
			}
		}
		if s.EnvVars != "" {
			var envVars map[string]string
			if err := json.Unmarshal([]byte(s.EnvVars), &envVars); err == nil {
				// Mask env var values to prevent leaking secrets (API keys, passwords).
				masked := make(map[string]string, len(envVars))
				for k := range envVars {
					masked[k] = "***"
				}
				resp.EnvVars = masked
			}
		}
		if s.ForwardHeaders != "" {
			var fh []string
			if err := json.Unmarshal([]byte(s.ForwardHeaders), &fh); err == nil {
				resp.ForwardHeaders = fh
			}
		}

		result = append(result, resp)
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Tool metadata adapter (admin_mcp.ToolMetadataProvider)
// ---------------------------------------------------------------------------

type adminMCPToolMetadataAdapter struct{}

func (a *adminMCPToolMetadataAdapter) GetAllToolMetadata() []admin_mcp.ToolMetadataResponse {
	all := tools.GetAllToolMetadata()
	result := make([]admin_mcp.ToolMetadataResponse, len(all))
	for i, m := range all {
		result[i] = admin_mcp.ToolMetadataResponse{
			Name:         m.Name,
			Description:  m.Description,
			SecurityZone: string(m.SecurityZone),
			RiskWarning:  m.RiskWarning,
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Config exporter adapter (admin_mcp.ConfigExporter)
// ---------------------------------------------------------------------------

type adminMCPConfigAdapter struct {
	db *gorm.DB
}

func (a *adminMCPConfigAdapter) ExportYAML(ctx context.Context) ([]byte, error) {
	inner := &configImportExportHTTPAdapter{db: a.db}
	return inner.ExportYAML(ctx)
}

func (a *adminMCPConfigAdapter) ImportYAML(ctx context.Context, yamlData []byte) error {
	inner := &configImportExportHTTPAdapter{db: a.db}
	return inner.ImportYAML(ctx, yamlData)
}

// ---------------------------------------------------------------------------
// Reloader adapter (admin_mcp.Reloader)
// ---------------------------------------------------------------------------

type adminMCPReloaderAdapter struct {
	registry            *agent_registry.AgentRegistry
	mcpRegistry         *mcp.ClientRegistry
	db                  *gorm.DB
	forwardHeadersStore *atomic.Value
}

func (a *adminMCPReloaderAdapter) Reload(ctx context.Context) error {
	inner := &configReloaderHTTPAdapter{
		registry:            a.registry,
		mcpRegistry:         a.mcpRegistry,
		db:                  a.db,
		forwardHeadersStore: a.forwardHeadersStore,
	}
	return inner.Reload(ctx)
}
