package app

import (
	"context"
	"time"

	deliveryhttp "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/audit"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/config_repo"
)

// agentCounterHTTPAdapter bridges AgentRegistry to the http.AgentCounter interface.
type agentCounterHTTPAdapter struct {
	registry *agent_registry.AgentRegistry
}

func (a *agentCounterHTTPAdapter) Count() int {
	return a.registry.Count()
}

// auditHTTPAdapter bridges audit.Logger to the http.AuditLogger interface.
type auditHTTPAdapter struct {
	logger *audit.Logger
}

func (a *auditHTTPAdapter) Log(ctx context.Context, entry deliveryhttp.AuditEntry) error {
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

// agentListerHTTPAdapter bridges AgentRegistry to the http.AgentLister interface.
type agentListerHTTPAdapter struct {
	registry *agent_registry.AgentRegistry
}

func (a *agentListerHTTPAdapter) ListAgents(_ context.Context) ([]deliveryhttp.AgentInfo, error) {
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

func (a *agentListerHTTPAdapter) GetAgent(_ context.Context, name string) (*deliveryhttp.AgentDetail, error) {
	agent, err := a.registry.Get(name)
	if err != nil {
		return nil, nil
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

// tokenRepoHTTPAdapter bridges GORMAPITokenRepository to the http.TokenRepository interface.
type tokenRepoHTTPAdapter struct {
	repo *config_repo.GORMAPITokenRepository
}

func (a *tokenRepoHTTPAdapter) Create(ctx context.Context, name, tokenHash string, scopesMask int) (uint, error) {
	return a.repo.Create(ctx, name, tokenHash, scopesMask)
}

func (a *tokenRepoHTTPAdapter) List(ctx context.Context) ([]deliveryhttp.TokenInfo, error) {
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

func (a *tokenRepoHTTPAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

func (a *tokenRepoHTTPAdapter) VerifyToken(ctx context.Context, tokenHash string) (string, int, error) {
	return a.repo.VerifyToken(ctx, tokenHash)
}

// configReloaderHTTPAdapter bridges AgentRegistry to the http.ConfigReloader interface.
type configReloaderHTTPAdapter struct {
	registry *agent_registry.AgentRegistry
}

func (a *configReloaderHTTPAdapter) Reload(ctx context.Context) error {
	return a.registry.Reload(ctx)
}

func (a *configReloaderHTTPAdapter) AgentsCount() int {
	return a.registry.Count()
}

// configImportExportHTTPAdapter — skeleton for YAML import/export.
type configImportExportHTTPAdapter struct{}

func (a *configImportExportHTTPAdapter) ImportYAML(_ context.Context, _ []byte) error {
	return nil
}

func (a *configImportExportHTTPAdapter) ExportYAML(_ context.Context) ([]byte, error) {
	return []byte("# ByteBrew config export\n"), nil
}

// taskServiceHTTPAdapter bridges task infrastructure to the http.TaskService interface.
type taskServiceHTTPAdapter struct {
	repo *config_repo.GORMTaskRepository
}

func (a *taskServiceHTTPAdapter) CreateTask(_ context.Context, _ deliveryhttp.CreateTaskRequest) (uint, error) {
	return 0, nil
}

func (a *taskServiceHTTPAdapter) ListTasks(ctx context.Context, filter deliveryhttp.TaskListFilter) ([]deliveryhttp.TaskResponse, error) {
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

func (a *taskServiceHTTPAdapter) GetTask(ctx context.Context, id uint) (*deliveryhttp.TaskDetailResponse, error) {
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

func (a *taskServiceHTTPAdapter) CancelTask(ctx context.Context, id uint) error {
	return a.repo.Cancel(ctx, id)
}

func (a *taskServiceHTTPAdapter) ProvideInput(_ context.Context, _ uint, _ string) error {
	return nil
}
