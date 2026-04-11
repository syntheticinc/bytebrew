package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	admintools "github.com/syntheticinc/bytebrew/engine/internal/infrastructure/tools/admin"
	"gorm.io/gorm"
)

// --- Agent adapter ---

type adminAgentRepoAdapter struct {
	repo *config_repo.GORMAgentRepository
}

func newAdminAgentRepoAdapter(repo *config_repo.GORMAgentRepository) *adminAgentRepoAdapter {
	return &adminAgentRepoAdapter{repo: repo}
}

func (a *adminAgentRepoAdapter) List(ctx context.Context) ([]admintools.AgentRecord, error) {
	records, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]admintools.AgentRecord, 0, len(records))
	for _, r := range records {
		out = append(out, toAdminAgentRecord(r))
	}
	return out, nil
}

func (a *adminAgentRepoAdapter) GetByName(ctx context.Context, name string) (*admintools.AgentRecord, error) {
	rec, err := a.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	out := toAdminAgentRecord(*rec)
	return &out, nil
}

func (a *adminAgentRepoAdapter) Create(ctx context.Context, record *admintools.AgentRecord) error {
	cr := fromAdminAgentRecord(record)
	return a.repo.Create(ctx, &cr)
}

func (a *adminAgentRepoAdapter) Update(ctx context.Context, name string, record *admintools.AgentRecord) error {
	cr := fromAdminAgentRecord(record)
	return a.repo.Update(ctx, name, &cr)
}

func (a *adminAgentRepoAdapter) Delete(ctx context.Context, name string) error {
	return a.repo.Delete(ctx, name)
}

func toAdminAgentRecord(r config_repo.AgentRecord) admintools.AgentRecord {
	return admintools.AgentRecord{
		Name:          r.Name,
		SystemPrompt:  r.SystemPrompt,
		ModelName:     r.ModelName,
		Lifecycle:     r.Lifecycle,
		ToolExecution: r.ToolExecution,
		MaxSteps:      r.MaxSteps,
		BuiltinTools:  r.BuiltinTools,
		MCPServers:    r.MCPServers,
		CanSpawn:      r.CanSpawn,
		IsSystem:      r.IsSystem,
	}
}

func fromAdminAgentRecord(r *admintools.AgentRecord) config_repo.AgentRecord {
	return config_repo.AgentRecord{
		Name:          r.Name,
		SystemPrompt:  r.SystemPrompt,
		ModelName:     r.ModelName,
		Lifecycle:     r.Lifecycle,
		ToolExecution: r.ToolExecution,
		MaxSteps:      r.MaxSteps,
		BuiltinTools:  r.BuiltinTools,
		MCPServers:    r.MCPServers,
		CanSpawn:      r.CanSpawn,
		IsSystem:      r.IsSystem,
	}
}

// --- Schema adapter ---

type adminSchemaRepoAdapter struct {
	repo *config_repo.GORMSchemaRepository
}

func newAdminSchemaRepoAdapter(repo *config_repo.GORMSchemaRepository) *adminSchemaRepoAdapter {
	return &adminSchemaRepoAdapter{repo: repo}
}

func (a *adminSchemaRepoAdapter) List(ctx context.Context) ([]admintools.SchemaRecord, error) {
	records, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]admintools.SchemaRecord, 0, len(records))
	for _, r := range records {
		out = append(out, admintools.SchemaRecord{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			AgentNames:  r.AgentNames,
		})
	}
	return out, nil
}

func (a *adminSchemaRepoAdapter) GetByID(ctx context.Context, id string) (*admintools.SchemaRecord, error) {
	rec, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &admintools.SchemaRecord{
		ID:          rec.ID,
		Name:        rec.Name,
		Description: rec.Description,
		AgentNames:  rec.AgentNames,
	}, nil
}

func (a *adminSchemaRepoAdapter) Create(ctx context.Context, record *admintools.SchemaRecord) error {
	cr := &config_repo.SchemaRecord{
		Name:        record.Name,
		Description: record.Description,
	}
	if err := a.repo.Create(ctx, cr); err != nil {
		return err
	}
	record.ID = cr.ID
	return nil
}

func (a *adminSchemaRepoAdapter) Update(ctx context.Context, id string, record *admintools.SchemaRecord) error {
	cr := &config_repo.SchemaRecord{
		Name:        record.Name,
		Description: record.Description,
	}
	return a.repo.Update(ctx, id, cr)
}

func (a *adminSchemaRepoAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

func (a *adminSchemaRepoAdapter) AddAgent(ctx context.Context, schemaID string, agentName string) error {
	return a.repo.AddAgent(ctx, schemaID, agentName)
}

func (a *adminSchemaRepoAdapter) RemoveAgent(ctx context.Context, schemaID string, agentName string) error {
	return a.repo.RemoveAgent(ctx, schemaID, agentName)
}

// --- Trigger adapter ---

type adminTriggerRepoAdapter struct {
	repo *config_repo.GORMTriggerRepository
	db   *gorm.DB
}

func newAdminTriggerRepoAdapter(repo *config_repo.GORMTriggerRepository, db *gorm.DB) *adminTriggerRepoAdapter {
	return &adminTriggerRepoAdapter{repo: repo, db: db}
}

func (a *adminTriggerRepoAdapter) List(ctx context.Context) ([]admintools.TriggerRecord, error) {
	triggers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]admintools.TriggerRecord, 0, len(triggers))
	for _, t := range triggers {
		out = append(out, toAdminTriggerRecord(t))
	}
	return out, nil
}

func (a *adminTriggerRepoAdapter) GetByID(ctx context.Context, id string) (*admintools.TriggerRecord, error) {
	t, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rec := toAdminTriggerRecord(*t)
	return &rec, nil
}

func (a *adminTriggerRepoAdapter) Create(ctx context.Context, record *admintools.TriggerRecord) error {
	agentID, err := resolveAgentID(ctx, a.db, record.AgentName)
	if err != nil {
		return fmt.Errorf("resolve agent %q: %w", record.AgentName, err)
	}
	m := &models.TriggerModel{
		Type:        record.Type,
		Title:       record.Title,
		AgentID:     ptrString(agentID),
		Schedule:    record.Schedule,
		WebhookPath: record.WebhookPath,
		Description: record.Description,
		Enabled:     record.Enabled,
	}
	if err := a.repo.Create(ctx, m); err != nil {
		return err
	}
	record.ID = m.ID
	return nil
}

func (a *adminTriggerRepoAdapter) Update(ctx context.Context, id string, record *admintools.TriggerRecord) error {
	m := &models.TriggerModel{
		Type:        record.Type,
		Title:       record.Title,
		AgentID:     ptrString(record.AgentID),
		Schedule:    record.Schedule,
		WebhookPath: record.WebhookPath,
		Description: record.Description,
		Enabled:     record.Enabled,
	}
	if record.AgentName != "" {
		agentID, err := resolveAgentID(ctx, a.db, record.AgentName)
		if err != nil {
			return fmt.Errorf("resolve agent for trigger update: %w", err)
		}
		m.AgentID = ptrString(agentID)
	}
	return a.repo.Update(ctx, id, m)
}

func (a *adminTriggerRepoAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

func toAdminTriggerRecord(t models.TriggerModel) admintools.TriggerRecord {
	agentName := ""
	if t.Agent.Name != "" {
		agentName = t.Agent.Name
	}
	return admintools.TriggerRecord{
		ID:          t.ID,
		Type:        t.Type,
		Title:       t.Title,
		AgentName:   agentName,
		AgentID:     derefString(t.AgentID),
		SchemaID:    t.SchemaID,
		Schedule:    t.Schedule,
		WebhookPath: t.WebhookPath,
		Description: t.Description,
		Enabled:     t.Enabled,
	}
}

// --- MCP Server adapter ---

type adminMCPServerRepoAdapter struct {
	repo *config_repo.GORMMCPServerRepository
}

func newAdminMCPServerRepoAdapter(repo *config_repo.GORMMCPServerRepository) *adminMCPServerRepoAdapter {
	return &adminMCPServerRepoAdapter{repo: repo}
}

func (a *adminMCPServerRepoAdapter) List(ctx context.Context) ([]admintools.MCPServerRecord, error) {
	servers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]admintools.MCPServerRecord, 0, len(servers))
	for _, s := range servers {
		out = append(out, toAdminMCPServerRecord(s))
	}
	return out, nil
}

func (a *adminMCPServerRepoAdapter) GetByID(ctx context.Context, id string) (*admintools.MCPServerRecord, error) {
	s, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rec := toAdminMCPServerRecord(*s)
	return &rec, nil
}

func (a *adminMCPServerRepoAdapter) Create(ctx context.Context, record *admintools.MCPServerRecord) error {
	argsJSON, _ := json.Marshal(record.Args)
	envJSON, _ := json.Marshal(record.EnvVars)
	m := &models.MCPServerModel{
		Name:    record.Name,
		Type:    record.Type,
		Command: record.Command,
		URL:     record.URL,
		Args:    string(argsJSON),
		EnvVars: string(envJSON),
	}
	if err := a.repo.Create(ctx, m); err != nil {
		return err
	}
	record.ID = m.ID
	return nil
}

func (a *adminMCPServerRepoAdapter) Update(ctx context.Context, id string, record *admintools.MCPServerRecord) error {
	argsJSON, _ := json.Marshal(record.Args)
	envJSON, _ := json.Marshal(record.EnvVars)
	m := &models.MCPServerModel{
		Name:    record.Name,
		Type:    record.Type,
		Command: record.Command,
		URL:     record.URL,
		Args:    string(argsJSON),
		EnvVars: string(envJSON),
	}
	return a.repo.Update(ctx, id, m)
}

func (a *adminMCPServerRepoAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

func toAdminMCPServerRecord(s models.MCPServerModel) admintools.MCPServerRecord {
	var args []string
	if s.Args != "" {
		_ = json.Unmarshal([]byte(s.Args), &args)
	}
	var envVars map[string]string
	if s.EnvVars != "" {
		_ = json.Unmarshal([]byte(s.EnvVars), &envVars)
	}
	return admintools.MCPServerRecord{
		ID:      s.ID,
		Name:    s.Name,
		Type:    s.Type,
		Command: s.Command,
		URL:     s.URL,
		Args:    args,
		EnvVars: envVars,
	}
}

// --- Model (LLM Provider) adapter ---

type adminModelRepoAdapter struct {
	repo *config_repo.GORMLLMProviderRepository
}

func newAdminModelRepoAdapter(repo *config_repo.GORMLLMProviderRepository) *adminModelRepoAdapter {
	return &adminModelRepoAdapter{repo: repo}
}

func (a *adminModelRepoAdapter) List(ctx context.Context) ([]admintools.ModelRecord, error) {
	providers, err := a.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]admintools.ModelRecord, 0, len(providers))
	for _, p := range providers {
		out = append(out, toAdminModelRecord(p))
	}
	return out, nil
}

func (a *adminModelRepoAdapter) GetByID(ctx context.Context, id string) (*admintools.ModelRecord, error) {
	p, err := a.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	rec := toAdminModelRecord(*p)
	return &rec, nil
}

func (a *adminModelRepoAdapter) Create(ctx context.Context, record *admintools.ModelRecord) error {
	m := &models.LLMProviderModel{
		Name:            record.Name,
		Type:            record.Type,
		BaseURL:         record.BaseURL,
		ModelName:       record.ModelName,
		APIKeyEncrypted: record.APIKey,
	}
	if err := a.repo.Create(ctx, m); err != nil {
		return err
	}
	record.ID = m.ID
	return nil
}

func (a *adminModelRepoAdapter) Update(ctx context.Context, id string, record *admintools.ModelRecord) error {
	m := &models.LLMProviderModel{
		Name:      record.Name,
		Type:      record.Type,
		BaseURL:   record.BaseURL,
		ModelName: record.ModelName,
	}
	if record.APIKey != "" {
		m.APIKeyEncrypted = record.APIKey
	}
	return a.repo.Update(ctx, id, m)
}

func (a *adminModelRepoAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

func toAdminModelRecord(p models.LLMProviderModel) admintools.ModelRecord {
	apiKey := ""
	if p.APIKeyEncrypted != "" {
		apiKey = "***"
	}
	return admintools.ModelRecord{
		ID:        p.ID,
		Name:      p.Name,
		Type:      p.Type,
		BaseURL:   p.BaseURL,
		ModelName: p.ModelName,
		APIKey:    apiKey,
	}
}

// --- Edge adapter ---

type adminEdgeRepoAdapter struct {
	repo *config_repo.GORMEdgeRepository
}

func newAdminEdgeRepoAdapter(repo *config_repo.GORMEdgeRepository) *adminEdgeRepoAdapter {
	return &adminEdgeRepoAdapter{repo: repo}
}

func (a *adminEdgeRepoAdapter) List(ctx context.Context, schemaID string) ([]admintools.EdgeRecord, error) {
	records, err := a.repo.List(ctx, schemaID)
	if err != nil {
		return nil, err
	}
	out := make([]admintools.EdgeRecord, 0, len(records))
	for _, r := range records {
		label, _ := r.Config["label"].(string)
		out = append(out, admintools.EdgeRecord{
			ID:        r.ID,
			SchemaID:  r.SchemaID,
			FromAgent: r.SourceAgentName,
			ToAgent:   r.TargetAgentName,
			Type:      r.Type,
			Label:     label,
		})
	}
	return out, nil
}

func (a *adminEdgeRepoAdapter) Create(ctx context.Context, record *admintools.EdgeRecord) error {
	config := map[string]interface{}{}
	if record.Label != "" {
		config["label"] = record.Label
	}
	cr := &config_repo.EdgeRecord{
		SchemaID:        record.SchemaID,
		SourceAgentName: record.FromAgent,
		TargetAgentName: record.ToAgent,
		Type:            record.Type,
		Config:          config,
	}
	if err := a.repo.Create(ctx, cr); err != nil {
		return err
	}
	record.ID = cr.ID
	return nil
}

func (a *adminEdgeRepoAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

// --- Session adapter ---

type adminSessionRepoAdapter struct {
	repo *config_repo.GORMSessionRepository
}

func newAdminSessionRepoAdapter(repo *config_repo.GORMSessionRepository) *adminSessionRepoAdapter {
	return &adminSessionRepoAdapter{repo: repo}
}

func (a *adminSessionRepoAdapter) List(ctx context.Context) ([]admintools.SessionRecord, error) {
	sessions, _, err := a.repo.List(ctx, "", "", "", "", "", 1, 100)
	if err != nil {
		return nil, err
	}
	out := make([]admintools.SessionRecord, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, admintools.SessionRecord{
			ID:        s.ID,
			AgentName: s.AgentName,
			UserID:    s.UserID,
			StartedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Status:    s.Status,
		})
	}
	return out, nil
}

func (a *adminSessionRepoAdapter) GetByID(ctx context.Context, id string) (*admintools.SessionRecord, error) {
	s, err := a.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, fmt.Errorf("session %q not found", id)
	}
	return &admintools.SessionRecord{
		ID:        s.ID,
		AgentName: s.AgentName,
		UserID:    s.UserID,
		StartedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Status:    s.Status,
	}, nil
}

// --- Capability adapter ---

type adminCapabilityRepoAdapter struct {
	repo *config_repo.GORMCapabilityRepository
}

func newAdminCapabilityRepoAdapter(repo *config_repo.GORMCapabilityRepository) *adminCapabilityRepoAdapter {
	return &adminCapabilityRepoAdapter{repo: repo}
}

func (a *adminCapabilityRepoAdapter) ListByAgent(ctx context.Context, agentName string) ([]admintools.CapabilityRecord, error) {
	records, err := a.repo.ListByAgent(ctx, agentName)
	if err != nil {
		return nil, err
	}
	out := make([]admintools.CapabilityRecord, 0, len(records))
	for _, r := range records {
		out = append(out, admintools.CapabilityRecord{
			ID:        r.ID,
			AgentName: r.AgentName,
			Type:      r.Type,
			Config:    r.Config,
			Enabled:   r.Enabled,
		})
	}
	return out, nil
}

func (a *adminCapabilityRepoAdapter) Create(ctx context.Context, record *admintools.CapabilityRecord) error {
	cr := &config_repo.CapabilityRecord{
		AgentName: record.AgentName,
		Type:      record.Type,
		Config:    record.Config,
		Enabled:   record.Enabled,
	}
	if err := a.repo.Create(ctx, cr); err != nil {
		return err
	}
	record.ID = cr.ID
	return nil
}

func (a *adminCapabilityRepoAdapter) Update(ctx context.Context, id string, record *admintools.CapabilityRecord) error {
	cr := &config_repo.CapabilityRecord{
		AgentName: record.AgentName,
		Type:      record.Type,
		Config:    record.Config,
		Enabled:   record.Enabled,
	}
	return a.repo.Update(ctx, id, cr)
}

func (a *adminCapabilityRepoAdapter) Delete(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}

// --- Builder-assistant restorer adapter ---

type builderAssistantRestorerAdapter struct {
	db       *gorm.DB
	registry interface{ Reload(ctx context.Context) error }
}

func (a *builderAssistantRestorerAdapter) RestoreBuilderAssistant(ctx context.Context) error {
	if err := restoreBuilderSchema(ctx, a.db); err != nil {
		return err
	}
	// Reload in-memory agent registry so restored tools are available at runtime.
	if a.registry != nil {
		if err := a.registry.Reload(ctx); err != nil {
			slog.WarnContext(ctx, "failed to reload registry after restore", "error", err)
		}
	}
	return nil
}

// --- Helpers ---

func resolveAgentID(ctx context.Context, db *gorm.DB, agentName string) (string, error) {
	var agent models.AgentModel
	if err := db.WithContext(ctx).Where("name = ?", agentName).First(&agent).Error; err != nil {
		return "", fmt.Errorf("find agent %q: %w", agentName, err)
	}
	return agent.ID, nil
}
