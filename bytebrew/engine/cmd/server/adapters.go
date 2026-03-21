package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	deliveryhttp "github.com/syntheticinc/bytebrew/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/delivery/ws"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/domain"
	pb "github.com/syntheticinc/bytebrew/bytebrew/engine/api/proto/gen"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/audit"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/session_processor"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/bytebrew/engine/internal/service/task"
	"gorm.io/gorm"
)

// dbPingerAdapter bridges *sql.DB to the http.DBPinger interface.
type dbPingerAdapter struct {
	db interface{ Ping() error }
}

func (a *dbPingerAdapter) Ping() error { return a.db.Ping() }

// agentCounterAdapter bridges AgentRegistry to the http.AgentCounter interface.
type agentCounterAdapter struct {
	registry *agent_registry.AgentRegistry
}

func (a *agentCounterAdapter) Count() int {
	return a.registry.Count()
}

// wsAgentListerAdapter bridges AgentRegistry to the ws.AgentLister interface.
type wsAgentListerAdapter struct {
	registry *agent_registry.AgentRegistry
}

func (a *wsAgentListerAdapter) ListAgentInfos() []ws.AgentInfo {
	agents := a.registry.GetAll()
	result := make([]ws.AgentInfo, 0, len(agents))
	for _, agent := range agents {
		result = append(result, ws.AgentInfo{
			Name:         agent.Record.Name,
			ToolsCount:   len(agent.Record.BuiltinTools) + len(agent.Record.CustomTools),
			Kit:          agent.Record.Kit,
			HasKnowledge: agent.Record.KnowledgePath != "",
		})
	}
	return result
}

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
		SystemPrompt:   agent.Record.SystemPrompt,
		KnowledgePath:  agent.Record.KnowledgePath,
		Tools:          tools,
		CanSpawn:       agent.Record.CanSpawn,
		Lifecycle:      agent.Record.Lifecycle,
		ToolExecution:  agent.Record.ToolExecution,
		MaxSteps:       agent.Record.MaxSteps,
		MaxContextSize: agent.Record.MaxContextSize,
		ConfirmBefore:  agent.Record.ConfirmBefore,
		MCPServers:     agent.Record.MCPServers,
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

func (a *tokenRepoAdapter) VerifyToken(ctx context.Context, tokenHash string) (string, int, error) {
	return a.repo.VerifyToken(ctx, tokenHash)
}

// agentManagerAdapter bridges AgentRepository + AgentRegistry to the http.AgentManager interface.
type agentManagerAdapter struct {
	agentListerAdapter
	repo     *config_repo.GORMAgentRepository
	registry *agent_registry.AgentRegistry
}

func (a *agentManagerAdapter) CreateAgent(ctx context.Context, req deliveryhttp.CreateAgentRequest) (*deliveryhttp.AgentDetail, error) {
	rec := agentRequestToRecord(req)
	if err := a.repo.Create(ctx, &rec); err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}
	if err := a.registry.Reload(ctx); err != nil {
		return nil, fmt.Errorf("reload registry: %w", err)
	}
	return a.GetAgent(ctx, req.Name)
}

func (a *agentManagerAdapter) UpdateAgent(ctx context.Context, name string, req deliveryhttp.CreateAgentRequest) (*deliveryhttp.AgentDetail, error) {
	rec := agentRequestToRecord(req)
	if rec.Name == "" {
		rec.Name = name // preserve original name if not provided in body
	}
	if err := a.repo.Update(ctx, name, &rec); err != nil {
		return nil, fmt.Errorf("update agent: %w", err)
	}
	if err := a.registry.Reload(ctx); err != nil {
		return nil, fmt.Errorf("reload registry: %w", err)
	}
	returnName := name
	if req.Name != "" {
		returnName = req.Name
	}
	return a.GetAgent(ctx, returnName)
}

func (a *agentManagerAdapter) DeleteAgent(ctx context.Context, name string) error {
	if err := a.repo.Delete(ctx, name); err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	if err := a.registry.Reload(ctx); err != nil {
		return fmt.Errorf("reload registry: %w", err)
	}
	return nil
}

func agentRequestToRecord(req deliveryhttp.CreateAgentRequest) config_repo.AgentRecord {
	rec := config_repo.AgentRecord{
		Name:           req.Name,
		SystemPrompt:   req.SystemPrompt,
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

// modelServiceAdapter bridges GORMLLMProviderRepository to the http.ModelService interface.
type modelServiceAdapter struct {
	repo *config_repo.GORMLLMProviderRepository
}

func (a *modelServiceAdapter) ListModels(ctx context.Context) ([]deliveryhttp.ModelResponse, error) {
	providers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.ModelResponse, 0, len(providers))
	for _, p := range providers {
		result = append(result, deliveryhttp.ModelResponse{
			ID:        p.ID,
			Name:      p.Name,
			Type:      p.Type,
			BaseURL:   p.BaseURL,
			ModelName: p.ModelName,
			HasAPIKey: p.APIKeyEncrypted != "",
			CreatedAt: p.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (a *modelServiceAdapter) CreateModel(ctx context.Context, req deliveryhttp.CreateModelRequest) (*deliveryhttp.ModelResponse, error) {
	model := &models.LLMProviderModel{
		Name:            req.Name,
		Type:            req.Type,
		BaseURL:         req.BaseURL,
		ModelName:       req.ModelName,
		APIKeyEncrypted: req.APIKey, // TODO: encrypt API key
	}
	if err := a.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	return &deliveryhttp.ModelResponse{
		ID:        model.ID,
		Name:      model.Name,
		Type:      model.Type,
		BaseURL:   model.BaseURL,
		ModelName: model.ModelName,
		HasAPIKey: model.APIKeyEncrypted != "",
		CreatedAt: model.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (a *modelServiceAdapter) UpdateModel(ctx context.Context, name string, req deliveryhttp.CreateModelRequest) (*deliveryhttp.ModelResponse, error) {
	providers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	var targetID uint
	for _, p := range providers {
		if p.Name == name {
			targetID = p.ID
			break
		}
	}
	if targetID == 0 {
		return nil, fmt.Errorf("model not found: %s", name)
	}

	model := &models.LLMProviderModel{
		Name:            req.Name,
		Type:            req.Type,
		BaseURL:         req.BaseURL,
		ModelName:       req.ModelName,
		APIKeyEncrypted: req.APIKey,
	}
	if err := a.repo.Update(ctx, targetID, model); err != nil {
		return nil, err
	}

	// Re-read to get updated timestamps
	updated, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, p := range updated {
		if p.ID == targetID {
			return &deliveryhttp.ModelResponse{
				ID:        p.ID,
				Name:      p.Name,
				Type:      p.Type,
				BaseURL:   p.BaseURL,
				ModelName: p.ModelName,
				HasAPIKey: p.APIKeyEncrypted != "",
				CreatedAt: p.CreatedAt.Format(time.RFC3339),
			}, nil
		}
	}
	return nil, fmt.Errorf("model not found after update: %s", name)
}

func (a *modelServiceAdapter) DeleteModel(ctx context.Context, name string) error {
	// Find by name first, then delete by ID
	providers, err := a.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, p := range providers {
		if p.Name == name {
			return a.repo.Delete(ctx, p.ID)
		}
	}
	return fmt.Errorf("model not found: %s", name)
}

func (a *modelServiceAdapter) VerifyModel(ctx context.Context, name string) (*deliveryhttp.ModelVerifyResult, error) {
	providers, err := a.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models for verify: %w", err)
	}

	var dbModel *models.LLMProviderModel
	for i := range providers {
		if providers[i].Name == name {
			dbModel = &providers[i]
			break
		}
	}
	if dbModel == nil {
		return nil, fmt.Errorf("model not found: %s", name)
	}

	client, err := llm.CreateClientFromDBModel(*dbModel)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create client: %s", err.Error())
		return &deliveryhttp.ModelVerifyResult{
			Connectivity: "error",
			ToolCalling:  "skipped",
			ModelName:    dbModel.ModelName,
			Provider:     dbModel.Type,
			Error:        &errMsg,
		}, nil
	}

	verifyCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	vr := llm.VerifyModel(verifyCtx, client, dbModel.ModelName, dbModel.Type)
	return &deliveryhttp.ModelVerifyResult{
		Connectivity:   vr.Connectivity,
		ToolCalling:    vr.ToolCalling,
		ResponseTimeMs: vr.ResponseTimeMs,
		ModelName:      vr.ModelName,
		Provider:       vr.Provider,
		Error:          vr.Error,
	}, nil
}

// mcpServiceAdapter bridges GORMMCPServerRepository to the http.MCPService interface.
type mcpServiceAdapter struct {
	repo *config_repo.GORMMCPServerRepository
}

func (a *mcpServiceAdapter) ListMCPServers(ctx context.Context) ([]deliveryhttp.MCPServerResponse, error) {
	servers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.MCPServerResponse, 0, len(servers))
	for _, s := range servers {
		resp := deliveryhttp.MCPServerResponse{
			ID:          s.ID,
			Name:        s.Name,
			Type:        s.Type,
			Command:     s.Command,
			URL:         s.URL,
			IsWellKnown: s.IsWellKnown,
			Agents:      []string{}, // TODO: load agent associations
		}
		if s.Args != "" {
			_ = json.Unmarshal([]byte(s.Args), &resp.Args)
		}
		if s.EnvVars != "" {
			_ = json.Unmarshal([]byte(s.EnvVars), &resp.EnvVars)
		}
		if s.Runtime != nil {
			resp.Status = &deliveryhttp.MCPStatusInfo{
				Status:        s.Runtime.Status,
				StatusMessage: s.Runtime.StatusMessage,
				ToolsCount:    s.Runtime.ToolsCount,
			}
			if s.Runtime.ConnectedAt != nil {
				resp.Status.ConnectedAt = s.Runtime.ConnectedAt.Format(time.RFC3339)
			}
		}
		result = append(result, resp)
	}
	return result, nil
}

func (a *mcpServiceAdapter) CreateMCPServer(ctx context.Context, req deliveryhttp.CreateMCPServerRequest) (*deliveryhttp.MCPServerResponse, error) {
	model := &models.MCPServerModel{
		Name:    req.Name,
		Type:    req.Type,
		Command: req.Command,
		URL:     req.URL,
	}
	if len(req.Args) > 0 {
		data, _ := json.Marshal(req.Args)
		model.Args = string(data)
	}
	if len(req.EnvVars) > 0 {
		data, _ := json.Marshal(req.EnvVars)
		model.EnvVars = string(data)
	}
	if err := a.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	resp := &deliveryhttp.MCPServerResponse{
		ID:          model.ID,
		Name:        model.Name,
		Type:        model.Type,
		Command:     model.Command,
		URL:         model.URL,
		IsWellKnown: model.IsWellKnown,
		Args:        req.Args,
		EnvVars:     req.EnvVars,
		Agents:      []string{},
	}
	return resp, nil
}

func (a *mcpServiceAdapter) UpdateMCPServer(ctx context.Context, name string, req deliveryhttp.CreateMCPServerRequest) (*deliveryhttp.MCPServerResponse, error) {
	servers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	var targetID uint
	for _, s := range servers {
		if s.Name == name {
			targetID = s.ID
			break
		}
	}
	if targetID == 0 {
		return nil, fmt.Errorf("mcp server not found: %s", name)
	}

	model := &models.MCPServerModel{
		Name:    req.Name,
		Type:    req.Type,
		Command: req.Command,
		URL:     req.URL,
	}
	if len(req.Args) > 0 {
		data, _ := json.Marshal(req.Args)
		model.Args = string(data)
	}
	if len(req.EnvVars) > 0 {
		data, _ := json.Marshal(req.EnvVars)
		model.EnvVars = string(data)
	}
	if err := a.repo.Update(ctx, targetID, model); err != nil {
		return nil, err
	}

	// Re-read to get updated state
	updated, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range updated {
		if s.ID == targetID {
			resp := &deliveryhttp.MCPServerResponse{
				ID:          s.ID,
				Name:        s.Name,
				Type:        s.Type,
				Command:     s.Command,
				URL:         s.URL,
				IsWellKnown: s.IsWellKnown,
				Agents:      []string{},
			}
			if s.Args != "" {
				_ = json.Unmarshal([]byte(s.Args), &resp.Args)
			}
			if s.EnvVars != "" {
				_ = json.Unmarshal([]byte(s.EnvVars), &resp.EnvVars)
			}
			if s.Runtime != nil {
				resp.Status = &deliveryhttp.MCPStatusInfo{
					Status:        s.Runtime.Status,
					StatusMessage: s.Runtime.StatusMessage,
					ToolsCount:    s.Runtime.ToolsCount,
				}
				if s.Runtime.ConnectedAt != nil {
					resp.Status.ConnectedAt = s.Runtime.ConnectedAt.Format(time.RFC3339)
				}
			}
			return resp, nil
		}
	}
	return nil, fmt.Errorf("mcp server not found after update: %s", name)
}

func (a *mcpServiceAdapter) DeleteMCPServer(ctx context.Context, name string) error {
	servers, err := a.repo.List(ctx)
	if err != nil {
		return err
	}
	for _, s := range servers {
		if s.Name == name {
			return a.repo.Delete(ctx, s.ID)
		}
	}
	return fmt.Errorf("mcp server not found: %s", name)
}

// triggerServiceAdapter bridges GORMTriggerRepository to the http.TriggerService interface.
type triggerServiceAdapter struct {
	repo *config_repo.GORMTriggerRepository
}

func (a *triggerServiceAdapter) ListTriggers(ctx context.Context) ([]deliveryhttp.TriggerResponse, error) {
	triggers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.TriggerResponse, 0, len(triggers))
	for _, t := range triggers {
		resp := deliveryhttp.TriggerResponse{
			ID:          t.ID,
			Type:        t.Type,
			Title:       t.Title,
			AgentID:     t.AgentID,
			AgentName:   t.Agent.Name,
			Schedule:    t.Schedule,
			WebhookPath: t.WebhookPath,
			Description: t.Description,
			Enabled:     t.Enabled,
			CreatedAt:   t.CreatedAt.Format(time.RFC3339),
		}
		if t.LastFiredAt != nil {
			resp.LastFiredAt = t.LastFiredAt.Format(time.RFC3339)
		}
		result = append(result, resp)
	}
	return result, nil
}

func (a *triggerServiceAdapter) CreateTrigger(ctx context.Context, req deliveryhttp.CreateTriggerRequest) (*deliveryhttp.TriggerResponse, error) {
	model := &models.TriggerModel{
		Type:        req.Type,
		Title:       req.Title,
		AgentID:     req.AgentID,
		Schedule:    req.Schedule,
		WebhookPath: req.WebhookPath,
		Description: req.Description,
		Enabled:     true,
	}
	if req.Enabled != nil {
		model.Enabled = *req.Enabled
	}
	if err := a.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	return &deliveryhttp.TriggerResponse{
		ID:          model.ID,
		Type:        model.Type,
		Title:       model.Title,
		AgentID:     model.AgentID,
		Schedule:    model.Schedule,
		WebhookPath: model.WebhookPath,
		Description: model.Description,
		Enabled:     model.Enabled,
		CreatedAt:   model.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (a *triggerServiceAdapter) UpdateTrigger(ctx context.Context, id uint, req deliveryhttp.CreateTriggerRequest) (*deliveryhttp.TriggerResponse, error) {
	model := &models.TriggerModel{
		Type:        req.Type,
		Title:       req.Title,
		AgentID:     req.AgentID,
		Schedule:    req.Schedule,
		WebhookPath: req.WebhookPath,
		Description: req.Description,
	}
	if req.Enabled != nil {
		model.Enabled = *req.Enabled
	}
	if err := a.repo.Update(ctx, id, model); err != nil {
		return nil, err
	}

	// Re-read to get updated state with agent preloaded
	triggers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, t := range triggers {
		if t.ID == id {
			resp := &deliveryhttp.TriggerResponse{
				ID:          t.ID,
				Type:        t.Type,
				Title:       t.Title,
				AgentID:     t.AgentID,
				AgentName:   t.Agent.Name,
				Schedule:    t.Schedule,
				WebhookPath: t.WebhookPath,
				Description: t.Description,
				Enabled:     t.Enabled,
				CreatedAt:   t.CreatedAt.Format(time.RFC3339),
			}
			if t.LastFiredAt != nil {
				resp.LastFiredAt = t.LastFiredAt.Format(time.RFC3339)
			}
			return resp, nil
		}
	}
	return nil, fmt.Errorf("trigger not found after update: %d", id)
}

func (a *triggerServiceAdapter) DeleteTrigger(ctx context.Context, id uint) error {
	return a.repo.Delete(ctx, id)
}

// settingServiceAdapter bridges GORMSettingRepository to the http.SettingService interface.
type settingServiceAdapter struct {
	repo *config_repo.GORMSettingRepository
}

func (a *settingServiceAdapter) ListSettings(ctx context.Context) ([]deliveryhttp.SettingResponse, error) {
	settings, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.SettingResponse, 0, len(settings))
	for _, s := range settings {
		result = append(result, deliveryhttp.SettingResponse{
			Key:       s.Key,
			Value:     s.Value,
			UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (a *settingServiceAdapter) UpdateSetting(ctx context.Context, key, value string) (*deliveryhttp.SettingResponse, error) {
	if err := a.repo.Set(ctx, key, value); err != nil {
		return nil, err
	}
	setting, err := a.repo.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &deliveryhttp.SettingResponse{
		Key:       setting.Key,
		Value:     setting.Value,
		UpdatedAt: setting.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// stubJSONHandler returns a handler that writes a static JSON response.
func stubJSONHandler(jsonStr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(jsonStr))
	}
}

// chatServiceAdapter bridges agent execution to the http.ChatService interface.
// Uses SessionRegistry + SessionProcessor for real agent execution.
type chatServiceAdapter struct {
	sessionRegistry *flow_registry.SessionRegistry
	sessProcessor   *session_processor.Processor
	sessionRepo     *config_repo.GORMSessionRepository
	messageRepo     *config_repo.GORMMessageRepository
}

func (a *chatServiceAdapter) Chat(agentName, message, userID, sessionID string) (<-chan deliveryhttp.SSEEvent, error) {
	ch := make(chan deliveryhttp.SSEEvent, 10)

	if a.sessionRegistry == nil || a.sessProcessor == nil {
		go func() {
			defer close(ch)
			ch <- deliveryhttp.SSEEvent{Type: "thinking", Data: `{"content":"Processing..."}`}
			ch <- deliveryhttp.SSEEvent{Type: "message", Data: `{"content":"Chat REST API: SessionProcessor not wired. Use CLI."}`}
			ch <- deliveryhttp.SSEEvent{Type: "done", Data: `{"status":"completed"}`}
		}()
		return ch, nil
	}

	if sessionID == "" {
		sessionID = fmt.Sprintf("rest-%d", time.Now().UnixNano())
	}

	// Reuse existing session or create new one
	if !a.sessionRegistry.HasSession(sessionID) {
		a.sessionRegistry.CreateSession(sessionID, "", userID, "", "", agentName)

		// Persist session to DB for query by web-client
		if a.sessionRepo != nil {
			_ = a.sessionRepo.Create(context.Background(), &models.SessionModel{
				ID:        sessionID,
				AgentName: agentName,
				UserID:    userID,
				Status:    "active",
			})
		}
	} else if a.sessionRepo != nil {
		// Touch updated_at for existing session
		_ = a.sessionRepo.TouchUpdatedAt(context.Background(), sessionID)
	}

	// Save user message to DB (non-blocking)
	if a.messageRepo != nil {
		go func() {
			_ = a.messageRepo.SaveMessage(context.Background(), &models.RuntimeMessageModel{
				ID:          fmt.Sprintf("msg-%d", time.Now().UnixNano()),
				SessionID:   sessionID,
				MessageType: "user",
				Sender:      userID,
				Content:     message,
			})
		}()
	}

	// Subscribe to SSE events from session
	eventCh, cleanup := a.sessionRegistry.Subscribe(sessionID)

	go func() {
		defer close(ch)
		defer cleanup()

		// Start processing loop, wait for goroutine to be ready, then enqueue
		a.sessProcessor.StartProcessing(context.Background(), sessionID)
		time.Sleep(50 * time.Millisecond) // let processor goroutine start
		if err := a.sessionRegistry.EnqueueMessage(sessionID, message); err != nil {
			ch <- deliveryhttp.SSEEvent{Type: "error", Data: fmt.Sprintf(`{"message":%q}`, err.Error())}
			return
		}

		// Stream events until done (with timeout)
		timeout := time.After(2 * time.Minute)
		for {
			select {
			case evt, ok := <-eventCh:
				if !ok {
					ch <- deliveryhttp.SSEEvent{Type: "done", Data: `{"status":"completed"}`}
					return
				}
				sseEvt := convertSessionEventToSSE(evt)
				if sseEvt != nil {
					ch <- *sseEvt
					// Save persistent message types to DB (async, don't block stream)
					a.saveSSEEventMessage(sessionID, evt, sseEvt)
					if sseEvt.Type == "done" {
						return
					}
				}
			case <-timeout:
				ch <- deliveryhttp.SSEEvent{Type: "done", Data: `{"status":"timeout"}`}
				return
			}
		}
	}()

	return ch, nil
}

// saveSSEEventMessage persists relevant SSE events as messages in the database.
// Runs asynchronously to avoid blocking the SSE stream.
func (a *chatServiceAdapter) saveSSEEventMessage(sessionID string, evt *pb.SessionEvent, sseEvt *deliveryhttp.SSEEvent) {
	if a.messageRepo == nil || evt == nil || sseEvt == nil {
		return
	}

	var msg *models.RuntimeMessageModel

	switch sseEvt.Type {
	case "message":
		if evt.Content == "" {
			return // skip empty messages
		}
		msg = &models.RuntimeMessageModel{
			ID:          fmt.Sprintf("msg-%d", time.Now().UnixNano()),
			SessionID:   sessionID,
			MessageType: "assistant",
			Content:     evt.Content,
		}
	case "tool_call":
		metadata := fmt.Sprintf(`{"tool":"%s"}`, evt.ToolName)
		msg = &models.RuntimeMessageModel{
			ID:          fmt.Sprintf("msg-%d", time.Now().UnixNano()),
			SessionID:   sessionID,
			MessageType: "tool_call",
			Content:     evt.Content,
			Metadata:    metadata,
		}
	case "tool_result":
		content := evt.Content
		if content == "" {
			content = evt.ToolResultSummary
		}
		metadata := fmt.Sprintf(`{"tool":"%s"}`, evt.ToolName)
		msg = &models.RuntimeMessageModel{
			ID:          fmt.Sprintf("msg-%d", time.Now().UnixNano()),
			SessionID:   sessionID,
			MessageType: "tool_result",
			Content:     content,
			Metadata:    metadata,
		}
	case "error":
		msg = &models.RuntimeMessageModel{
			ID:          fmt.Sprintf("msg-%d", time.Now().UnixNano()),
			SessionID:   sessionID,
			MessageType: "error",
			Content:     evt.Content,
		}
	default:
		return
	}

	go func() {
		_ = a.messageRepo.SaveMessage(context.Background(), msg)
	}()
}

// convertSessionEventToSSE converts a gRPC SessionEvent to an SSE event.
func convertSessionEventToSSE(evt *pb.SessionEvent) *deliveryhttp.SSEEvent {
	if evt == nil {
		return nil
	}

	switch evt.Type {
	case pb.SessionEventType_SESSION_EVENT_REASONING:
		return &deliveryhttp.SSEEvent{Type: "thinking", Data: fmt.Sprintf(`{"content":%q}`, evt.Content)}
	case pb.SessionEventType_SESSION_EVENT_ANSWER:
		return &deliveryhttp.SSEEvent{Type: "message", Data: fmt.Sprintf(`{"content":%q}`, evt.Content)}
	case pb.SessionEventType_SESSION_EVENT_ANSWER_CHUNK:
		return &deliveryhttp.SSEEvent{Type: "message_delta", Data: fmt.Sprintf(`{"content":%q}`, evt.Content)}
	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START:
		args := evt.Content
		if args == "" && len(evt.ToolArguments) > 0 {
			argsJSON, _ := json.Marshal(evt.ToolArguments)
			args = string(argsJSON)
		}
		return &deliveryhttp.SSEEvent{Type: "tool_call", Data: fmt.Sprintf(`{"tool":%q,"content":%q}`, evt.ToolName, args)}
	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END:
		result := evt.Content
		if result == "" {
			result = evt.ToolResultSummary
		}
		return &deliveryhttp.SSEEvent{Type: "tool_result", Data: fmt.Sprintf(`{"tool":%q,"content":%q}`, evt.ToolName, result)}
	case pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED:
		return &deliveryhttp.SSEEvent{Type: "done", Data: `{"status":"completed"}`}
	default:
		return nil
	}
}

// mcpServerRow holds an MCP server config loaded from DB.
type mcpServerRow struct {
	Name    string
	Type    string
	Command string
	Args    string
	URL     string
}

func loadMCPServersFromDB(db *gorm.DB) ([]mcpServerRow, error) {
	if db == nil {
		return nil, nil
	}
	var rows []models.MCPServerModel
	if err := db.Find(&rows).Error; err != nil {
		return nil, err
	}
	servers := make([]mcpServerRow, 0, len(rows))
	for _, r := range rows {
		servers = append(servers, mcpServerRow{
			Name:    r.Name,
			Type:    r.Type,
			Command: r.Command,
			Args:    r.Args,
			URL:     r.URL,
		})
	}
	return servers, nil
}

// triggerRow holds a trigger record loaded from DB.
type triggerRow struct {
	ID          uint
	Type        string
	Schedule    string
	Title       string
	Description string
	AgentName   string
}

// loadTriggersFromDB loads trigger definitions from PostgreSQL.
func loadTriggersFromDB(db *gorm.DB) ([]triggerRow, error) {
	if db == nil {
		return nil, nil
	}
	var rows []models.TriggerModel
	if err := db.Preload("Agent").Find(&rows).Error; err != nil {
		return nil, err
	}
	triggers := make([]triggerRow, 0, len(rows))
	for _, r := range rows {
		triggers = append(triggers, triggerRow{
			ID:          r.ID,
			Type:        r.Type,
			Schedule:    r.Schedule,
			Title:       r.Title,
			Description: r.Description,
			AgentName:   r.Agent.Name,
		})
	}
	return triggers, nil
}

// cronTaskCreatorAdapter bridges GORMTaskRepository to task.TaskCreator for CronScheduler.
type cronTaskCreatorAdapter struct {
	repo *config_repo.GORMTaskRepository
}

func (a *cronTaskCreatorAdapter) CreateFromTrigger(ctx context.Context, params task.TriggerTaskParams) (uint, error) {
	t := &domain.EngineTask{
		Title:       params.Title,
		Description: params.Description,
		AgentName:   params.AgentName,
		Source:      domain.TaskSource(params.Source),
		SourceID:    params.SourceID,
		Status:      domain.EngineTaskStatusPending,
		Mode:        domain.TaskModeBackground,
	}
	if err := a.repo.Create(ctx, t); err != nil {
		return 0, err
	}
	return t.ID, nil
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

// configImportExportAdapter bridges repositories to the http.ConfigImportExporter interface.
type configImportExportAdapter struct {
	agentRepo *config_repo.GORMAgentRepository
	db        *gorm.DB
}

func (a *configImportExportAdapter) ImportYAML(ctx context.Context, data []byte) error {
	// Parse YAML into agent definitions
	var importData struct {
		Agents []struct {
			Name         string   `yaml:"name"`
			SystemPrompt string   `yaml:"system_prompt"`
			Kit          string   `yaml:"kit"`
			Lifecycle    string   `yaml:"lifecycle"`
			MaxSteps     int      `yaml:"max_steps"`
			Tools        []string `yaml:"tools"`
		} `yaml:"agents"`
	}
	if err := yaml.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("parse YAML: %w", err)
	}

	for _, agent := range importData.Agents {
		rec := config_repo.AgentRecord{
			Name:         agent.Name,
			SystemPrompt: agent.SystemPrompt,
			Kit:          agent.Kit,
			Lifecycle:    agent.Lifecycle,
			MaxSteps:     agent.MaxSteps,
			BuiltinTools: agent.Tools,
		}
		if rec.Lifecycle == "" {
			rec.Lifecycle = "persistent"
		}
		if rec.MaxSteps == 0 {
			rec.MaxSteps = 50
		}
		_ = a.agentRepo.Create(ctx, &rec) // ignore duplicate errors
	}
	return nil
}

func (a *configImportExportAdapter) ExportYAML(ctx context.Context) ([]byte, error) {
	agents, err := a.agentRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("# ByteBrew Engine Configuration Export\n")
	sb.WriteString(fmt.Sprintf("# Generated: %s\n\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("agents:\n")
	for _, a := range agents {
		sb.WriteString(fmt.Sprintf("  - name: %q\n", a.Name))
		sb.WriteString(fmt.Sprintf("    system_prompt: %q\n", a.SystemPrompt))
		if a.Kit != "" {
			sb.WriteString(fmt.Sprintf("    kit: %q\n", a.Kit))
		}
		sb.WriteString(fmt.Sprintf("    lifecycle: %q\n", a.Lifecycle))
		sb.WriteString(fmt.Sprintf("    max_steps: %d\n", a.MaxSteps))
		if len(a.BuiltinTools) > 0 {
			sb.WriteString("    tools:\n")
			for _, t := range a.BuiltinTools {
				sb.WriteString(fmt.Sprintf("      - %s\n", t))
			}
		}
	}
	return []byte(sb.String()), nil
}

// taskServiceAdapter bridges task infrastructure to the http.TaskService interface.
// Phase 5 skeleton — full task CRUD will be wired when TaskExecutor is available.
type taskServiceAdapter struct {
	repo *config_repo.GORMTaskRepository
}

func (a *taskServiceAdapter) CreateTask(_ context.Context, _ deliveryhttp.CreateTaskRequest) (uint, error) {
	return 0, nil // TODO: wire task creation through TaskWorker
}

func (a *taskServiceAdapter) buildRepoFilter(filter deliveryhttp.TaskListFilter) config_repo.TaskFilter {
	repoFilter := config_repo.TaskFilter{
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}
	if filter.AgentName != "" {
		repoFilter.AgentName = &filter.AgentName
	}
	if filter.Source != "" {
		src := domain.TaskSource(filter.Source)
		repoFilter.Source = &src
	}
	if filter.Status != "" {
		st := domain.EngineTaskStatus(filter.Status)
		repoFilter.Status = &st
	}
	return repoFilter
}

func (a *taskServiceAdapter) ListTasks(ctx context.Context, filter deliveryhttp.TaskListFilter) ([]deliveryhttp.TaskResponse, error) {
	tasks, err := a.repo.List(ctx, a.buildRepoFilter(filter))
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

func (a *taskServiceAdapter) CountTasks(ctx context.Context, filter deliveryhttp.TaskListFilter) (int64, error) {
	return a.repo.Count(ctx, a.buildRepoFilter(filter))
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

// sessionServiceAdapter bridges GORMSessionRepository to the http.SessionService interface.
type sessionServiceAdapter struct {
	repo        *config_repo.GORMSessionRepository
	messageRepo *config_repo.GORMMessageRepository
}

func (a *sessionServiceAdapter) ListSessions(ctx context.Context, agentName, userID, status string, page, perPage int) ([]deliveryhttp.SessionResponse, int64, error) {
	sessions, total, err := a.repo.List(ctx, agentName, userID, status, page, perPage)
	if err != nil {
		return nil, 0, err
	}
	result := make([]deliveryhttp.SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		result = append(result, deliveryhttp.SessionResponse{
			ID:        s.ID,
			Title:     s.Title,
			AgentName: s.AgentName,
			UserID:    s.UserID,
			Status:    s.Status,
			CreatedAt: s.CreatedAt.Format(time.RFC3339),
			UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result, total, nil
}

func (a *sessionServiceAdapter) GetSession(ctx context.Context, id string) (*deliveryhttp.SessionResponse, error) {
	s, err := a.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, nil
	}
	return &deliveryhttp.SessionResponse{
		ID:        s.ID,
		Title:     s.Title,
		AgentName: s.AgentName,
		UserID:    s.UserID,
		Status:    s.Status,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (a *sessionServiceAdapter) CreateSession(ctx context.Context, req deliveryhttp.CreateSessionRequest) (*deliveryhttp.SessionResponse, error) {
	id := req.ID
	if id == "" {
		id = fmt.Sprintf("web-%d", time.Now().UnixNano())
	}
	session := &models.SessionModel{
		ID:        id,
		Title:     req.Title,
		AgentName: req.AgentName,
		UserID:    req.UserID,
		Status:    "active",
	}
	if err := a.repo.Create(ctx, session); err != nil {
		return nil, err
	}
	return &deliveryhttp.SessionResponse{
		ID:        session.ID,
		Title:     session.Title,
		AgentName: session.AgentName,
		UserID:    session.UserID,
		Status:    session.Status,
		CreatedAt: session.CreatedAt.Format(time.RFC3339),
		UpdatedAt: session.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (a *sessionServiceAdapter) UpdateSession(ctx context.Context, id string, req deliveryhttp.UpdateSessionRequest) (*deliveryhttp.SessionResponse, error) {
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if len(updates) == 0 {
		return a.GetSession(ctx, id)
	}
	if err := a.repo.Update(ctx, id, updates); err != nil {
		return nil, err
	}
	return a.GetSession(ctx, id)
}

func (a *sessionServiceAdapter) DeleteSession(ctx context.Context, id string) error {
	if a.messageRepo != nil {
		_ = a.messageRepo.DeleteBySession(ctx, id)
	}
	return a.repo.Delete(ctx, id)
}

// messageServiceAdapter bridges GORMMessageRepository to the http.MessageService interface.
type messageServiceAdapter struct {
	repo *config_repo.GORMMessageRepository
}

func (a *messageServiceAdapter) ListMessages(ctx context.Context, sessionID string) ([]deliveryhttp.MessageResponse, error) {
	messages, err := a.repo.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.MessageResponse, 0, len(messages))
	seen := make(map[string]bool) // deduplicate by content+role
	for _, m := range messages {
		// Skip empty messages
		if m.Content == "" {
			continue
		}

		// Map role: "agent" → "assistant"
		role := m.MessageType
		if role == "agent" {
			role = "assistant"
		}

		// Deduplicate: same role + same content = skip
		dedup := role + ":" + m.Content
		if seen[dedup] {
			continue
		}
		seen[dedup] = true

		resp := deliveryhttp.MessageResponse{
			ID:        m.ID,
			Role:      role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt.Format(time.RFC3339),
		}
		// Extract tool_name from metadata JSON if present
		if m.Metadata != "" {
			var meta struct {
				Tool string `json:"tool"`
			}
			if json.Unmarshal([]byte(m.Metadata), &meta) == nil && meta.Tool != "" {
				resp.ToolName = meta.Tool
			}
		}
		result = append(result, resp)
	}
	return result, nil
}

// toolMetadataAdapter bridges tools.GetAllToolMetadata to the http.ToolMetadataProvider interface.
type toolMetadataAdapter struct{}

func (a *toolMetadataAdapter) GetAllToolMetadata() []deliveryhttp.ToolMetadataResponse {
	all := tools.GetAllToolMetadata()
	result := make([]deliveryhttp.ToolMetadataResponse, len(all))
	for i, m := range all {
		result[i] = deliveryhttp.ToolMetadataResponse{
			Name:         m.Name,
			Description:  m.Description,
			SecurityZone: string(m.SecurityZone),
			RiskWarning:  m.RiskWarning,
		}
	}
	return result
}

// knowledgeStatsAdapter bridges GORMKnowledgeRepository to http.KnowledgeStats.
type knowledgeStatsAdapter struct {
	repo *config_repo.GORMKnowledgeRepository
}

func (a *knowledgeStatsAdapter) GetStats(ctx context.Context, agentName string) (docCount int, chunkCount int, lastIndexed *time.Time, err error) {
	return a.repo.GetStats(ctx, agentName)
}

// knowledgeReindexerAdapter bridges Knowledge Indexer to http.KnowledgeReindexer.
type knowledgeReindexerAdapter struct {
	indexer  interface{ IndexFolder(ctx context.Context, agentName string, folderPath string) error }
	registry *agent_registry.AgentRegistry
}

func (a *knowledgeReindexerAdapter) Reindex(ctx context.Context, agentName string) error {
	agent, err := a.registry.Get(agentName)
	if err != nil {
		return fmt.Errorf("agent not found: %s", agentName)
	}
	if agent.Record.KnowledgePath == "" {
		return fmt.Errorf("agent %s has no knowledge_path configured", agentName)
	}
	return a.indexer.IndexFolder(ctx, agentName, agent.Record.KnowledgePath)
}

// auditServiceHTTPAdapter bridges GORMAuditRepository to http.AuditService.
type auditServiceHTTPAdapter struct {
	repo *config_repo.GORMAuditRepository
}

func (a *auditServiceHTTPAdapter) ListAuditLogs(ctx context.Context, actorType, action, resource string, from, to *time.Time, page, perPage int) ([]deliveryhttp.AuditResponse, int64, error) {
	filters := config_repo.AuditFilters{
		ActorType: actorType,
		Action:    action,
		Resource:  resource,
		From:      from,
		To:        to,
	}
	logs, total, err := a.repo.List(ctx, filters, page, perPage)
	if err != nil {
		return nil, 0, err
	}
	result := make([]deliveryhttp.AuditResponse, 0, len(logs))
	for _, l := range logs {
		result = append(result, deliveryhttp.AuditResponse{
			ID:        l.ID,
			Timestamp: l.Timestamp.Format(time.RFC3339),
			ActorType: l.ActorType,
			ActorID:   l.ActorID,
			Action:    l.Action,
			Resource:  l.Resource,
			Details:   l.Details,
		})
	}
	return result, total, nil
}
