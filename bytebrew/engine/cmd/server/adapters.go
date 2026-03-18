package main

import (
	"context"
	"time"

	deliveryhttp "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/audit"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/config_repo"
)

// auditLoggerAdapter bridges audit.Logger to the http.AuditLogger interface.
type auditLoggerAdapter struct {
	logger *audit.Logger
}

func (a *auditLoggerAdapter) Log(ctx context.Context, entry deliveryhttp.AuditEntry) error {
	return a.logger.Log(ctx, audit.Entry{
		Timestamp: entry.Timestamp,
		ActorType: entry.ActorType,
		ActorID:   entry.ActorID,
		Action:    entry.Action,
		Resource:  entry.Resource,
		Details:   entry.Details,
		SessionID: entry.SessionID,
	})
}

// agentListerAdapter bridges AgentRegistry to the http.AgentLister interface.
type agentListerAdapter struct {
	registry *agent_registry.AgentRegistry
}

func (a *agentListerAdapter) ListAgents(_ context.Context) ([]deliveryhttp.AgentInfo, error) {
	agents := a.registry.GetAll()
	result := make([]deliveryhttp.AgentInfo, 0, len(agents))
	for _, agent := range agents {
		result = append(result, deliveryhttp.AgentInfo{
			Name:         agent.Record.Name,
			ToolsCount:   len(agent.Record.BuiltinTools) + len(agent.Record.CustomTools),
			Kit:          agent.Record.Kit,
			HasKnowledge: agent.Record.KnowledgePath != "",
		})
	}
	return result, nil
}

func (a *agentListerAdapter) GetAgent(_ context.Context, name string) (*deliveryhttp.AgentDetail, error) {
	agent, err := a.registry.Get(name)
	if err != nil {
		return nil, nil // not found
	}

	tools := make([]string, 0, len(agent.Record.BuiltinTools)+len(agent.Record.CustomTools))
	tools = append(tools, agent.Record.BuiltinTools...)
	for _, ct := range agent.Record.CustomTools {
		tools = append(tools, ct.Name)
	}

	return &deliveryhttp.AgentDetail{
		AgentInfo: deliveryhttp.AgentInfo{
			Name:         agent.Record.Name,
			ToolsCount:   len(tools),
			Kit:          agent.Record.Kit,
			HasKnowledge: agent.Record.KnowledgePath != "",
		},
		Tools:    tools,
		CanSpawn: agent.Record.CanSpawn,
	}, nil
}

// tokenRepoAdapter bridges GORMAPITokenRepository to the http.TokenRepository interface.
type tokenRepoAdapter struct {
	repo *config_repo.GORMAPITokenRepository
}

func (a *tokenRepoAdapter) Create(ctx context.Context, name, tokenHash string, scopesMask int) (uint, error) {
	return a.repo.Create(ctx, name, tokenHash, scopesMask)
}

func (a *tokenRepoAdapter) List(ctx context.Context) ([]deliveryhttp.TokenInfo, error) {
	tokens, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.TokenInfo, 0, len(tokens))
	for _, t := range tokens {
		result = append(result, deliveryhttp.TokenInfo{
			ID:         t.ID,
			Name:       t.Name,
			ScopesMask: t.ScopesMask,
			CreatedAt:  t.CreatedAt,
			LastUsedAt: t.LastUsedAt,
		})
	}
	return result, nil
}

func (a *tokenRepoAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

// configReloaderAdapter bridges AgentRegistry to the http.ConfigReloader interface.
type configReloaderAdapter struct {
	registry *agent_registry.AgentRegistry
}

func (a *configReloaderAdapter) Reload(ctx context.Context) error {
	return a.registry.Reload(ctx)
}

func (a *configReloaderAdapter) AgentsCount() int {
	return a.registry.Count()
}

// configImportExportAdapter bridges AgentRepository to the http.ConfigImportExporter interface.
// Phase 5 skeleton — full YAML import/export will be implemented when config_repo gains the methods.
type configImportExportAdapter struct {
	repo *config_repo.GORMAgentRepository
}

func (a *configImportExportAdapter) ImportYAML(_ context.Context, _ []byte) error {
	return nil // TODO: implement YAML import
}

func (a *configImportExportAdapter) ExportYAML(_ context.Context) ([]byte, error) {
	return []byte("# ByteBrew config export (not yet implemented)\n"), nil
}

// taskServiceAdapter bridges task infrastructure to the http.TaskService interface.
// Phase 5 skeleton — full task CRUD will be wired when TaskExecutor is available.
type taskServiceAdapter struct {
	repo *config_repo.GORMTaskRepository
}

func (a *taskServiceAdapter) CreateTask(_ context.Context, _ deliveryhttp.CreateTaskRequest) (uint, error) {
	return 0, nil // TODO: wire task creation through TaskWorker
}

func (a *taskServiceAdapter) ListTasks(ctx context.Context, filter deliveryhttp.TaskListFilter) ([]deliveryhttp.TaskResponse, error) {
	repoFilter := config_repo.TaskFilter{}
	if filter.AgentName != "" {
		repoFilter.AgentName = &filter.AgentName
	}

	tasks, err := a.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, err
	}

	result := make([]deliveryhttp.TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, deliveryhttp.TaskResponse{
			ID:        t.ID,
			Title:     t.Title,
			AgentName: t.AgentName,
			Status:    string(t.Status),
			Source:    string(t.Source),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (a *taskServiceAdapter) GetTask(ctx context.Context, id uint) (*deliveryhttp.TaskDetailResponse, error) {
	t, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &deliveryhttp.TaskDetailResponse{
		TaskResponse: deliveryhttp.TaskResponse{
			ID:        t.ID,
			Title:     t.Title,
			AgentName: t.AgentName,
			Status:    string(t.Status),
			Source:    string(t.Source),
			CreatedAt: t.CreatedAt.Format(time.RFC3339),
		},
		Description: t.Description,
		Mode:        string(t.Mode),
		Result:      t.Result,
		Error:       t.Error,
	}
	if t.StartedAt != nil {
		resp.StartedAt = t.StartedAt.Format(time.RFC3339)
	}
	if t.CompletedAt != nil {
		resp.CompletedAt = t.CompletedAt.Format(time.RFC3339)
	}
	return resp, nil
}

func (a *taskServiceAdapter) CancelTask(ctx context.Context, id uint) error {
	return a.repo.Cancel(ctx, id)
}

func (a *taskServiceAdapter) ProvideInput(_ context.Context, _ uint, _ string) error {
	return nil // TODO: wire input channel to task session
}
