package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/engine/internal/service/task"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
)

// mcpServiceHTTPAdapter bridges GORMMCPServerRepository to the http.MCPService interface.
type mcpServiceHTTPAdapter struct {
	repo *config_repo.GORMMCPServerRepository
}

func (a *mcpServiceHTTPAdapter) ListMCPServers(ctx context.Context) ([]deliveryhttp.MCPServerResponse, error) {
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
			Agents:      []string{},
		}
		if s.Args != "" {
			_ = json.Unmarshal([]byte(s.Args), &resp.Args)
		}
		if s.EnvVars != "" {
			_ = json.Unmarshal([]byte(s.EnvVars), &resp.EnvVars)
		}
		if s.ForwardHeaders != "" {
			_ = json.Unmarshal([]byte(s.ForwardHeaders), &resp.ForwardHeaders)
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

func (a *mcpServiceHTTPAdapter) CreateMCPServer(ctx context.Context, req deliveryhttp.CreateMCPServerRequest) (*deliveryhttp.MCPServerResponse, error) {
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
	if len(req.ForwardHeaders) > 0 {
		data, _ := json.Marshal(req.ForwardHeaders)
		model.ForwardHeaders = string(data)
	}
	if err := a.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	return &deliveryhttp.MCPServerResponse{
		ID:             model.ID,
		Name:           model.Name,
		Type:           model.Type,
		Command:        model.Command,
		URL:            model.URL,
		IsWellKnown:    model.IsWellKnown,
		Args:           req.Args,
		EnvVars:        req.EnvVars,
		ForwardHeaders: req.ForwardHeaders,
		Agents:         []string{},
	}, nil
}

func (a *mcpServiceHTTPAdapter) UpdateMCPServer(ctx context.Context, name string, req deliveryhttp.CreateMCPServerRequest) (*deliveryhttp.MCPServerResponse, error) {
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
	if len(req.ForwardHeaders) > 0 {
		data, _ := json.Marshal(req.ForwardHeaders)
		model.ForwardHeaders = string(data)
	}
	if err := a.repo.Update(ctx, targetID, model); err != nil {
		return nil, err
	}

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
			if s.ForwardHeaders != "" {
				_ = json.Unmarshal([]byte(s.ForwardHeaders), &resp.ForwardHeaders)
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

func (a *mcpServiceHTTPAdapter) DeleteMCPServer(ctx context.Context, name string) error {
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

// triggerServiceHTTPAdapter bridges GORMTriggerRepository to the http.TriggerService interface.
type triggerServiceHTTPAdapter struct {
	repo *config_repo.GORMTriggerRepository
}

func (a *triggerServiceHTTPAdapter) ListTriggers(ctx context.Context) ([]deliveryhttp.TriggerResponse, error) {
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

func (a *triggerServiceHTTPAdapter) CreateTrigger(ctx context.Context, req deliveryhttp.CreateTriggerRequest) (*deliveryhttp.TriggerResponse, error) {
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

func (a *triggerServiceHTTPAdapter) UpdateTrigger(ctx context.Context, id uint, req deliveryhttp.CreateTriggerRequest) (*deliveryhttp.TriggerResponse, error) {
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

func (a *triggerServiceHTTPAdapter) DeleteTrigger(ctx context.Context, id uint) error {
	return a.repo.Delete(ctx, id)
}

// settingServiceHTTPAdapter bridges GORMSettingRepository to the http.SettingService interface.
type settingServiceHTTPAdapter struct {
	repo *config_repo.GORMSettingRepository
}

func (a *settingServiceHTTPAdapter) ListSettings(ctx context.Context) ([]deliveryhttp.SettingResponse, error) {
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

func (a *settingServiceHTTPAdapter) UpdateSetting(ctx context.Context, key, value string) (*deliveryhttp.SettingResponse, error) {
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

// sessionServiceHTTPAdapter bridges GORMSessionRepository to the http.SessionService interface.
type sessionServiceHTTPAdapter struct {
	repo        *config_repo.GORMSessionRepository
	messageRepo *config_repo.GORMMessageRepository
}

func (a *sessionServiceHTTPAdapter) ListSessions(ctx context.Context, agentName, userID, status string, page, perPage int) ([]deliveryhttp.SessionResponse, int64, error) {
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

func (a *sessionServiceHTTPAdapter) GetSession(ctx context.Context, id string) (*deliveryhttp.SessionResponse, error) {
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

func (a *sessionServiceHTTPAdapter) CreateSession(ctx context.Context, req deliveryhttp.CreateSessionRequest) (*deliveryhttp.SessionResponse, error) {
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

func (a *sessionServiceHTTPAdapter) UpdateSession(ctx context.Context, id string, req deliveryhttp.UpdateSessionRequest) (*deliveryhttp.SessionResponse, error) {
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

func (a *sessionServiceHTTPAdapter) DeleteSession(ctx context.Context, id string) error {
	if a.messageRepo != nil {
		_ = a.messageRepo.DeleteBySession(ctx, id)
	}
	return a.repo.Delete(ctx, id)
}

// messageServiceHTTPAdapter bridges GORMMessageRepository to the http.MessageService interface.
type messageServiceHTTPAdapter struct {
	repo *config_repo.GORMMessageRepository
}

func (a *messageServiceHTTPAdapter) ListMessages(ctx context.Context, sessionID string) ([]deliveryhttp.MessageResponse, error) {
	messages, err := a.repo.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.MessageResponse, 0, len(messages))
	seen := make(map[string]bool)
	for _, m := range messages {
		if m.Content == "" {
			continue
		}

		role := m.MessageType
		if role == "agent" {
			role = "assistant"
		}

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

// toolMetadataHTTPAdapter bridges tools.GetAllToolMetadata to the http.ToolMetadataProvider interface.
type toolMetadataHTTPAdapter struct{}

func (a *toolMetadataHTTPAdapter) GetAllToolMetadata() []deliveryhttp.ToolMetadataResponse {
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

// cronTaskCreatorHTTPAdapter bridges GORMTaskRepository to task.TaskCreator for CronScheduler.
type cronTaskCreatorHTTPAdapter struct {
	repo *config_repo.GORMTaskRepository
}

func (a *cronTaskCreatorHTTPAdapter) CreateFromTrigger(ctx context.Context, params task.TriggerTaskParams) (uint, error) {
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

// convertRateLimitRules converts config rate limit rules to delivery HTTP types.
func convertRateLimitRules(cfgRules []config.RateLimitRule) []deliveryhttp.RateLimitRule {
	rules := make([]deliveryhttp.RateLimitRule, 0, len(cfgRules))
	for _, cr := range cfgRules {
		tiers := make(map[string]deliveryhttp.RateLimitTier, len(cr.Tiers))
		for name, ct := range cr.Tiers {
			tiers[name] = deliveryhttp.RateLimitTier{
				Requests:  ct.Requests,
				Window:    ct.Window,
				Unlimited: ct.Unlimited,
			}
		}
		rules = append(rules, deliveryhttp.RateLimitRule{
			Name:        cr.Name,
			KeyHeader:   cr.KeyHeader,
			TierHeader:  cr.TierHeader,
			Tiers:       tiers,
			DefaultTier: cr.DefaultTier,
		})
	}
	return rules
}
