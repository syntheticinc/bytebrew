package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/configrepo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools"
	"github.com/syntheticinc/bytebrew/engine/pkg/config"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
)

// mcpServiceHTTPAdapter bridges GORMMCPServerRepository to the http.MCPService interface.
type mcpServiceHTTPAdapter struct {
	repo *configrepo.GORMMCPServerRepository
}

func (a *mcpServiceHTTPAdapter) ListMCPServers(ctx context.Context) ([]deliveryhttp.MCPServerResponse, error) {
	servers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Batch-load agent names for all servers (single query, no N+1)
	serverIDs := make([]string, 0, len(servers))
	for _, s := range servers {
		serverIDs = append(serverIDs, s.ID)
	}
	agentsByServer, err := a.repo.GetAgentNamesByServerIDs(ctx, serverIDs)
	if err != nil {
		return nil, fmt.Errorf("load agents for mcp servers: %w", err)
	}

	result := make([]deliveryhttp.MCPServerResponse, 0, len(servers))
	for _, s := range servers {
		agents := agentsByServer[s.ID]
		if agents == nil {
			agents = []string{}
		}
		resp := deliveryhttp.MCPServerResponse{
			ID:           s.ID,
			Name:         s.Name,
			Type:         s.Type,
			Command:      s.Command,
			URL:          s.URL,
			AuthType:     s.AuthType,
			AuthKeyEnv:   s.AuthKeyEnv,
			AuthTokenEnv: s.AuthTokenEnv,
			AuthClientID: s.AuthClientID,
			Agents:       agents,
		}
		if s.Args != nil && *s.Args != "" {
			_ = json.Unmarshal([]byte(*s.Args), &resp.Args)
		}
		if s.EnvVars != nil && *s.EnvVars != "" {
			_ = json.Unmarshal([]byte(*s.EnvVars), &resp.EnvVars)
		}
		if s.ForwardHeaders != nil && *s.ForwardHeaders != "" {
			_ = json.Unmarshal([]byte(*s.ForwardHeaders), &resp.ForwardHeaders)
		}
		// V2 Commit Group C (§5.6): connection status is no longer persisted
		// — callers query the live MCP client registry separately.
		result = append(result, resp)
	}
	return result, nil
}

func (a *mcpServiceHTTPAdapter) CreateMCPServer(ctx context.Context, req deliveryhttp.CreateMCPServerRequest) (*deliveryhttp.MCPServerResponse, error) {
	model := &models.MCPServerModel{
		Name:         req.Name,
		Type:         req.Type,
		Command:      req.Command,
		URL:          req.URL,
		AuthType:     req.AuthType,
		AuthKeyEnv:   req.AuthKeyEnv,
		AuthTokenEnv: req.AuthTokenEnv,
		AuthClientID: req.AuthClientID,
	}
	if len(req.Args) > 0 {
		data, _ := json.Marshal(req.Args)
		s := string(data)
		model.Args = &s
	}
	if len(req.EnvVars) > 0 {
		data, _ := json.Marshal(req.EnvVars)
		s := string(data)
		model.EnvVars = &s
	}
	if len(req.ForwardHeaders) > 0 {
		data, _ := json.Marshal(req.ForwardHeaders)
		s := string(data)
		model.ForwardHeaders = &s
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
		AuthType:       model.AuthType,
		AuthKeyEnv:     model.AuthKeyEnv,
		AuthTokenEnv:   model.AuthTokenEnv,
		AuthClientID:   model.AuthClientID,
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
	var targetID string
	for _, s := range servers {
		if s.Name == name {
			targetID = s.ID
			break
		}
	}
	if targetID == "" {
		return nil, fmt.Errorf("mcp server not found: %s", name)
	}

	model := &models.MCPServerModel{
		Name:         req.Name,
		Type:         req.Type,
		Command:      req.Command,
		URL:          req.URL,
		AuthType:     req.AuthType,
		AuthKeyEnv:   req.AuthKeyEnv,
		AuthTokenEnv: req.AuthTokenEnv,
		AuthClientID: req.AuthClientID,
	}
	if len(req.Args) > 0 {
		data, _ := json.Marshal(req.Args)
		s := string(data)
		model.Args = &s
	}
	if len(req.EnvVars) > 0 {
		data, _ := json.Marshal(req.EnvVars)
		s := string(data)
		model.EnvVars = &s
	}
	if len(req.ForwardHeaders) > 0 {
		data, _ := json.Marshal(req.ForwardHeaders)
		s := string(data)
		model.ForwardHeaders = &s
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
			agents, err := a.repo.GetAgentNamesForServer(ctx, targetID)
			if err != nil {
				return nil, fmt.Errorf("load agents for mcp server: %w", err)
			}
			if agents == nil {
				agents = []string{}
			}
			resp := &deliveryhttp.MCPServerResponse{
				ID:           s.ID,
				Name:         s.Name,
				Type:         s.Type,
				Command:      s.Command,
				URL:          s.URL,
				AuthType:     s.AuthType,
				AuthKeyEnv:   s.AuthKeyEnv,
				AuthTokenEnv: s.AuthTokenEnv,
				AuthClientID: s.AuthClientID,
				Agents:       agents,
			}
			if s.Args != nil && *s.Args != "" {
				_ = json.Unmarshal([]byte(*s.Args), &resp.Args)
			}
			if s.EnvVars != nil && *s.EnvVars != "" {
				_ = json.Unmarshal([]byte(*s.EnvVars), &resp.EnvVars)
			}
			if s.ForwardHeaders != nil && *s.ForwardHeaders != "" {
				_ = json.Unmarshal([]byte(*s.ForwardHeaders), &resp.ForwardHeaders)
			}
			// V2 Commit Group C (§5.6): live status no longer persisted.
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

// ptrString converts a string to *string; returns nil when v == "" (no reference).
func ptrString(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// derefString dereferences a *string; returns "" when p is nil.
func derefString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// triggerServiceHTTPAdapter bridges GORMTriggerRepository to the http.TriggerService interface.
type triggerServiceHTTPAdapter struct {
	repo *configrepo.GORMTriggerRepository
	db   *gorm.DB
}

func triggerModelToResponse(t models.TriggerModel) deliveryhttp.TriggerResponse {
	var schemaIDPtr *string
	if t.SchemaID != "" {
		s := t.SchemaID
		schemaIDPtr = &s
	}
	resp := deliveryhttp.TriggerResponse{
		ID:          t.ID,
		Type:        t.Type,
		Title:       t.Title,
		SchemaID:    schemaIDPtr,
		Description: t.Description,
		Enabled:     t.Enabled,
		Config:      triggerConfigToMap(t.Config),
		CreatedAt:   t.CreatedAt.Format(time.RFC3339),
	}
	if t.LastFiredAt != nil {
		resp.LastFiredAt = t.LastFiredAt.Format(time.RFC3339)
	}
	return resp
}

// triggerConfigToMap flattens a typed TriggerConfig into the wire-format map
// served by the HTTP API. Empty fields are elided so `config` stays compact
// for chat-type triggers that carry no config.
func triggerConfigToMap(c models.TriggerConfig) map[string]interface{} {
	out := map[string]interface{}{}
	if c.Schedule != "" {
		out["schedule"] = c.Schedule
	}
	if c.WebhookPath != "" {
		out["webhook_path"] = c.WebhookPath
	}
	return out
}

// triggerConfigFromMap materialises a typed TriggerConfig from the wire map.
// Unknown keys are dropped silently — the API surface is deliberately narrow.
func triggerConfigFromMap(m map[string]interface{}) models.TriggerConfig {
	var c models.TriggerConfig
	if v, ok := m["schedule"].(string); ok {
		c.Schedule = v
	}
	if v, ok := m["webhook_path"].(string); ok {
		c.WebhookPath = v
	}
	return c
}

func (a *triggerServiceHTTPAdapter) ListTriggers(ctx context.Context) ([]deliveryhttp.TriggerResponse, error) {
	triggers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.TriggerResponse, 0, len(triggers))
	for _, t := range triggers {
		result = append(result, triggerModelToResponse(t))
	}
	return result, nil
}

func (a *triggerServiceHTTPAdapter) ListTriggersBySchema(ctx context.Context, schemaID string) ([]deliveryhttp.TriggerResponse, error) {
	triggers, err := a.repo.ListBySchemaID(ctx, schemaID)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.TriggerResponse, 0, len(triggers))
	for _, t := range triggers {
		result = append(result, triggerModelToResponse(t))
	}
	return result, nil
}

// Q.5: triggers no longer have agent_id — they target schemas only.

func (a *triggerServiceHTTPAdapter) CreateTrigger(ctx context.Context, req deliveryhttp.CreateTriggerRequest) (*deliveryhttp.TriggerResponse, error) {
	schemaID := ""
	if req.SchemaID != nil {
		schemaID = *req.SchemaID
	}
	model := &models.TriggerModel{
		Type:        req.Type,
		Title:       req.Title,
		SchemaID:    schemaID,
		Description: req.Description,
		Enabled:     true,
		Config:      triggerConfigFromMap(req.Config),
	}
	if req.Enabled != nil {
		model.Enabled = *req.Enabled
	}
	if err := a.repo.Create(ctx, model); err != nil {
		return nil, err
	}
	resp := triggerModelToResponse(*model)
	return &resp, nil
}

func (a *triggerServiceHTTPAdapter) UpdateTrigger(ctx context.Context, id string, req deliveryhttp.CreateTriggerRequest) (*deliveryhttp.TriggerResponse, error) {
	model := &models.TriggerModel{
		Type:        req.Type,
		Title:       req.Title,
		Description: req.Description,
		Config:      triggerConfigFromMap(req.Config),
	}
	if req.Enabled != nil {
		model.Enabled = *req.Enabled
	}
	if err := a.repo.Update(ctx, id, model); err != nil {
		return nil, err
	}

	t, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := triggerModelToResponse(*t)
	return &resp, nil
}

// SetTriggerTarget resolves agent → schema and sets the trigger's schema_id.
// Q.5: triggers no longer have agent_id; they target schemas only.
func (a *triggerServiceHTTPAdapter) SetTriggerTarget(ctx context.Context, id string, agentName string) (*deliveryhttp.TriggerResponse, error) {
	// Resolve agent name → find schema where this agent is the entry agent
	var schemaID string
	if err := a.db.WithContext(ctx).
		Raw("SELECT id FROM schemas WHERE entry_agent_id = (SELECT id FROM agents WHERE name = ? LIMIT 1) LIMIT 1", agentName).
		Scan(&schemaID).Error; err != nil || schemaID == "" {
		return nil, pkgerrors.NotFound(fmt.Sprintf("no schema with entry agent %q", agentName))
	}
	if err := a.repo.SetSchemaID(ctx, id, schemaID); err != nil {
		return nil, err
	}
	t, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := triggerModelToResponse(*t)
	return &resp, nil
}

// ClearTriggerTarget is no longer supported — triggers.schema_id is NOT NULL.
// Returns an error directing callers to delete and recreate the trigger instead.
func (a *triggerServiceHTTPAdapter) ClearTriggerTarget(ctx context.Context, id string) error {
	return fmt.Errorf("clearing trigger schema is not supported (schema_id is NOT NULL); delete and recreate the trigger instead")
}

func (a *triggerServiceHTTPAdapter) DeleteTrigger(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

// settingServiceHTTPAdapter bridges GORMSettingRepository to the http.SettingService interface.
type settingServiceHTTPAdapter struct {
	repo *configrepo.GORMSettingRepository
}

// settingValueAsString renders a jsonb value for the HTTP layer:
//   - jsonb string ("foo") → unwrapped Go string foo
//   - any other jsonb (number, bool, array, object) → raw JSON text
//
// This keeps the wire shape stable for the existing
// SettingResponse.Value:string contract while allowing structured values
// (byok.allowed_providers as a real array) to round-trip as JSON text.
func settingValueAsString(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
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
			Value:     settingValueAsString(s.Value),
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
		Value:     settingValueAsString(setting.Value),
		UpdatedAt: setting.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// sessionServiceHTTPAdapter bridges GORMSessionRepository to the http.SessionService interface.
type sessionServiceHTTPAdapter struct {
	repo        *configrepo.GORMSessionRepository
	messageRepo *configrepo.GORMMessageRepository
}

func (a *sessionServiceHTTPAdapter) ListSessions(ctx context.Context, agentName, userID, status, from, to string, page, perPage int) ([]deliveryhttp.SessionResponse, int64, error) {
	sessions, total, err := a.repo.List(ctx, agentName, userID, status, from, to, page, perPage)
	if err != nil {
		return nil, 0, err
	}
	result := make([]deliveryhttp.SessionResponse, 0, len(sessions))
	for _, s := range sessions {
		userID := ""
		if s.UserID != nil {
			userID = *s.UserID
		}
		result = append(result, deliveryhttp.SessionResponse{
			ID:        s.ID,
			Title:     s.Title,
			AgentName: "", // Q.5: agent_name dropped from sessions
			UserID:    userID,
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
	userID := ""
	if s.UserID != nil {
		userID = *s.UserID
	}
	return &deliveryhttp.SessionResponse{
		ID:        s.ID,
		Title:     s.Title,
		AgentName: "", // Q.5: agent_name dropped from sessions
		UserID:    userID,
		Status:    s.Status,
		CreatedAt: s.CreatedAt.Format(time.RFC3339),
		UpdatedAt: s.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (a *sessionServiceHTTPAdapter) CreateSession(ctx context.Context, req deliveryhttp.CreateSessionRequest) (*deliveryhttp.SessionResponse, error) {
	id := req.ID
	if id == "" {
		id = uuid.New().String()
	}
	var userID *string
	if req.UserID != "" {
		userID = &req.UserID
	}
	session := &models.SessionModel{
		ID:     id,
		Title:  req.Title,
		UserID: userID,
		Status: "active",
	}
	if err := a.repo.Create(ctx, session); err != nil {
		return nil, err
	}
	respUserID := ""
	if session.UserID != nil {
		respUserID = *session.UserID
	}
	return &deliveryhttp.SessionResponse{
		ID:        session.ID,
		Title:     session.Title,
		AgentName: "", // Q.5: agent_name dropped from sessions
		UserID:    respUserID,
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

// eventServiceHTTPAdapter bridges GORMMessageRepository to the http.EventService interface.
type eventServiceHTTPAdapter struct {
	repo *configrepo.GORMMessageRepository
}

func (a *eventServiceHTTPAdapter) ListEvents(ctx context.Context, sessionID string) ([]deliveryhttp.EventResponse, error) {
	events, err := a.repo.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	result := make([]deliveryhttp.EventResponse, 0, len(events))
	for _, ev := range events {
		agentID := ""
		if ev.AgentID != nil {
			agentID = *ev.AgentID
		}
		result = append(result, deliveryhttp.EventResponse{
			ID:        ev.ID,
			EventType: ev.EventType,
			AgentID:   agentID,
			CallID:    ev.CallID,
			Payload:   ev.Payload,
			CreatedAt: ev.CreatedAt.Format(time.RFC3339),
		})
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
