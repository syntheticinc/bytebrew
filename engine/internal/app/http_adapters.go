package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"

	pb "github.com/syntheticinc/bytebrew/engine/api/proto/gen"
	deliveryhttp "github.com/syntheticinc/bytebrew/engine/internal/delivery/http"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/agent_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/audit"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/flow_registry"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/llm"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/mcp"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/config_repo"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	pkgerrors "github.com/syntheticinc/bytebrew/engine/pkg/errors"
	"github.com/syntheticinc/bytebrew/engine/internal/service/session_processor"
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
			Public:       agent.Record.Public,
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
			Public:       agent.Record.Public,
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

// configReloaderHTTPAdapter bridges AgentRegistry and MCP reconnection to the http.ConfigReloader interface.
type configReloaderHTTPAdapter struct {
	registry            *agent_registry.AgentRegistry
	mcpRegistry         *mcp.ClientRegistry
	db                  *gorm.DB
	forwardHeadersStore *atomic.Value // shared with ChatHandler for dynamic forward header updates
}

func (a *configReloaderHTTPAdapter) Reload(ctx context.Context) error {
	if err := a.registry.Reload(ctx); err != nil {
		return err
	}

	a.reconnectMCPServers(ctx)
	return nil
}

func (a *configReloaderHTTPAdapter) reconnectMCPServers(ctx context.Context) {
	if a.mcpRegistry == nil || a.db == nil {
		return
	}

	mcpServerRepo := config_repo.NewGORMMCPServerRepository(a.db)
	mcpServers, err := mcpServerRepo.List(ctx)
	if err != nil {
		slog.ErrorContext(ctx, "failed to load MCP servers for reload", "error", err)
		return
	}

	a.mcpRegistry.CloseAll()
	connectMCPServers(ctx, mcpServers, a.mcpRegistry)

	// Update forward headers so ChatHandler picks up changes immediately
	if a.forwardHeadersStore != nil {
		a.forwardHeadersStore.Store(collectForwardHeaders(mcpServers))
		slog.InfoContext(ctx, "forward headers updated after config reload")
	}

	slog.InfoContext(ctx, "MCP servers reconnected after config reload", "count", len(mcpServers))
}

func (a *configReloaderHTTPAdapter) AgentsCount() int {
	return a.registry.Count()
}

// collectForwardHeaders returns the deduplicated union of forward_headers
// configured across all MCP servers.
func collectForwardHeaders(mcpServers []models.MCPServerModel) []string {
	seen := make(map[string]bool)
	var headers []string
	for _, srv := range mcpServers {
		if srv.ForwardHeaders == "" {
			continue
		}
		var fh []string
		if err := json.Unmarshal([]byte(srv.ForwardHeaders), &fh); err != nil {
			continue
		}
		for _, h := range fh {
			if !seen[h] {
				seen[h] = true
				headers = append(headers, h)
			}
		}
	}
	return headers
}

// configImportExportHTTPAdapter bridges GORM DB to the http.ConfigImportExporter interface.
type configImportExportHTTPAdapter struct {
	db *gorm.DB
}

// --- YAML structs ---

type configYAML struct {
	Agents     flexList[agentYAML]     `yaml:"agents,omitempty"`
	Models     flexList[modelYAML]     `yaml:"models,omitempty"`
	MCPServers flexList[mcpServerYAML] `yaml:"mcp_servers,omitempty"`
	Triggers   flexList[triggerYAML]   `yaml:"triggers,omitempty"`
}

// namedItem is implemented by YAML structs that can be keyed by name in map format.
type namedItem interface {
	agentYAML | modelYAML | mcpServerYAML | triggerYAML
}

// flexList accepts both YAML array format and map format (where map keys become the Name/Title field).
// Map format example:
//
//	agents:
//	  my-agent:
//	    model_name: glm-5
//
// Array format example:
//
//	agents:
//	  - name: my-agent
//	    model_name: glm-5
type flexList[T namedItem] struct {
	Items []T
}

// MarshalYAML marshals as a plain array so export always uses the array format.
func (f flexList[T]) MarshalYAML() (interface{}, error) {
	return f.Items, nil
}

// UnmarshalYAML tries map format first (documented format), then falls back to array.
func (f *flexList[T]) UnmarshalYAML(node *yaml.Node) error {
	// Try array format first (sequence node)
	if node.Kind == yaml.SequenceNode {
		return node.Decode(&f.Items)
	}

	// Map format (mapping node): keys become the Name field
	if node.Kind == yaml.MappingNode {
		return f.decodeMap(node)
	}

	// Null or empty
	if node.Kind == yaml.ScalarNode && (node.Tag == "!!null" || node.Value == "") {
		return nil
	}

	return fmt.Errorf("expected sequence or mapping, got %v", node.Kind)
}

func (f *flexList[T]) decodeMap(node *yaml.Node) error {
	// Mapping nodes have key-value pairs: [key1, val1, key2, val2, ...]
	for i := 0; i+1 < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]

		var item T
		if err := valNode.Decode(&item); err != nil {
			return fmt.Errorf("decode item %q: %w", keyNode.Value, err)
		}

		// Set the name field from the map key
		setNameFromKey(&item, keyNode.Value)

		f.Items = append(f.Items, item)
	}
	return nil
}

// setNameFromKey injects the map key into the appropriate name field of the item.
func setNameFromKey(item interface{}, key string) {
	switch v := item.(type) {
	case *agentYAML:
		if v.Name == "" {
			v.Name = key
		}
	case *modelYAML:
		if v.Name == "" {
			v.Name = key
		}
	case *mcpServerYAML:
		if v.Name == "" {
			v.Name = key
		}
	case *triggerYAML:
		if v.Title == "" {
			v.Title = key
		}
	}
}

type agentYAML struct {
	Name           string          `yaml:"name"`
	SystemPrompt   string          `yaml:"system_prompt"`
	ModelName      string          `yaml:"model_name,omitempty"`
	Kit            string          `yaml:"kit,omitempty"`
	KnowledgePath  string          `yaml:"knowledge_path,omitempty"`
	Lifecycle      string          `yaml:"lifecycle"`
	ToolExecution  string          `yaml:"tool_execution"`
	MaxSteps       int             `yaml:"max_steps"`
	MaxContextSize int             `yaml:"max_context_size"`
	ConfirmBefore  []string        `yaml:"confirm_before,omitempty"`
	Tools          []string        `yaml:"tools,omitempty"`
	CanSpawn       []string        `yaml:"can_spawn,omitempty"`
	MCPServers     []string        `yaml:"mcp_servers,omitempty"`
	Escalation     *escalationYAML `yaml:"escalation,omitempty"`
}

// UnmarshalYAML supports field aliases used in documentation:
//   - "system" as alias for "system_prompt"
//   - "model" as alias for "model_name"
//   - "knowledge" as alias for "knowledge_path"
func (a *agentYAML) UnmarshalYAML(node *yaml.Node) error {
	// Use a shadow type to prevent infinite recursion.
	type agentYAMLAlias agentYAML
	var alias agentYAMLAlias
	if err := node.Decode(&alias); err != nil {
		return err
	}

	// Extract alias fields from the raw YAML node.
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i].Value
			val := node.Content[i+1]
			switch key {
			case "system":
				if alias.SystemPrompt == "" {
					alias.SystemPrompt = val.Value
				}
			case "model":
				if alias.ModelName == "" {
					alias.ModelName = val.Value
				}
			case "knowledge":
				if alias.KnowledgePath == "" {
					alias.KnowledgePath = val.Value
				}
			}
		}
	}

	*a = agentYAML(alias)
	return nil
}

type escalationYAML struct {
	Action     string   `yaml:"action"`
	WebhookURL string   `yaml:"webhook_url,omitempty"`
	Triggers   []string `yaml:"triggers,omitempty"`
}

type modelYAML struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	Provider  string `yaml:"provider,omitempty"` // alias for type (used in agents.yaml)
	BaseURL   string `yaml:"base_url,omitempty"`
	ModelName string `yaml:"model_name"`
	APIKey    string `yaml:"api_key,omitempty"`
}

// resolvedType returns Type, falling back to Provider for backwards compatibility.
func (m modelYAML) resolvedType() string {
	if m.Type != "" {
		return m.Type
	}
	return m.Provider
}

type mcpServerYAML struct {
	Name           string            `yaml:"name"`
	Type           string            `yaml:"type"`
	Command        string            `yaml:"command,omitempty"`
	Args           []string          `yaml:"args,omitempty"`
	URL            string            `yaml:"url,omitempty"`
	EnvVars        map[string]string `yaml:"env_vars,omitempty"`
	ForwardHeaders []string          `yaml:"forward_headers,omitempty"`
}

type triggerYAML struct {
	Title       string `yaml:"title"`
	Type        string `yaml:"type"`
	AgentName   string `yaml:"agent_name"`
	Schedule    string `yaml:"schedule,omitempty"`
	WebhookPath string `yaml:"webhook_path,omitempty"`
	Description string `yaml:"description,omitempty"`
	Enabled     bool   `yaml:"enabled"`
}

// ExportYAML reads all config from DB and marshals to YAML.
func (a *configImportExportHTTPAdapter) ExportYAML(ctx context.Context) ([]byte, error) {
	cfg, err := a.buildExportConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("build export config: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal yaml: %w", err)
	}

	header := fmt.Sprintf("# ByteBrew Engine Configuration\n# Exported: %s\n\n", time.Now().UTC().Format(time.RFC3339))
	return append([]byte(header), data...), nil
}

func (a *configImportExportHTTPAdapter) buildExportConfig(ctx context.Context) (*configYAML, error) {
	var cfg configYAML

	agentsYAML, err := a.exportAgents(ctx)
	if err != nil {
		return nil, fmt.Errorf("export agents: %w", err)
	}
	cfg.Agents.Items = agentsYAML

	modelsYAML, err := a.exportModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("export models: %w", err)
	}
	cfg.Models.Items = modelsYAML

	mcpYAML, err := a.exportMCPServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("export mcp servers: %w", err)
	}
	cfg.MCPServers.Items = mcpYAML

	triggersYAML, err := a.exportTriggers(ctx)
	if err != nil {
		return nil, fmt.Errorf("export triggers: %w", err)
	}
	cfg.Triggers.Items = triggersYAML

	return &cfg, nil
}

func (a *configImportExportHTTPAdapter) exportAgents(_ context.Context) ([]agentYAML, error) {
	var agents []models.AgentModel
	if err := a.db.Preload("Model").Preload("Tools", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort_order ASC")
	}).Preload("SpawnTargets.TargetAgent").Preload("Escalation.Triggers").Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}

	// Load MCP server associations separately (comment in model explains why).
	var agentMCPs []models.AgentMCPServer
	if err := a.db.Preload("MCPServer").Find(&agentMCPs).Error; err != nil {
		return nil, fmt.Errorf("query agent mcp servers: %w", err)
	}
	mcpByAgent := make(map[uint][]string)
	for _, am := range agentMCPs {
		mcpByAgent[am.AgentID] = append(mcpByAgent[am.AgentID], am.MCPServer.Name)
	}

	result := make([]agentYAML, 0, len(agents))
	for _, ag := range agents {
		ay := agentYAML{
			Name:           ag.Name,
			SystemPrompt:   ag.SystemPrompt,
			Kit:            ag.Kit,
			KnowledgePath:  ag.KnowledgePath,
			Lifecycle:      ag.Lifecycle,
			ToolExecution:  ag.ToolExecution,
			MaxSteps:       ag.MaxSteps,
			MaxContextSize: ag.MaxContextSize,
			MCPServers:     mcpByAgent[ag.ID],
		}

		if ag.Model != nil {
			ay.ModelName = ag.Model.Name
		}

		if ag.ConfirmBefore != "" {
			ay.ConfirmBefore = splitCSV(ag.ConfirmBefore)
		}

		for _, t := range ag.Tools {
			ay.Tools = append(ay.Tools, t.ToolName)
		}

		for _, st := range ag.SpawnTargets {
			ay.CanSpawn = append(ay.CanSpawn, st.TargetAgent.Name)
		}

		if ag.Escalation != nil {
			esc := &escalationYAML{
				Action:     ag.Escalation.Action,
				WebhookURL: ag.Escalation.WebhookURL,
			}
			for _, et := range ag.Escalation.Triggers {
				esc.Triggers = append(esc.Triggers, et.Keyword)
			}
			ay.Escalation = esc
		}

		result = append(result, ay)
	}
	return result, nil
}

func (a *configImportExportHTTPAdapter) exportModels(_ context.Context) ([]modelYAML, error) {
	var llms []models.LLMProviderModel
	if err := a.db.Find(&llms).Error; err != nil {
		return nil, fmt.Errorf("query models: %w", err)
	}

	result := make([]modelYAML, 0, len(llms))
	for _, m := range llms {
		result = append(result, modelYAML{
			Name:      m.Name,
			Type:      m.Type,
			BaseURL:   m.BaseURL,
			ModelName: m.ModelName,
			// API key intentionally not exported.
		})
	}
	return result, nil
}

func (a *configImportExportHTTPAdapter) exportMCPServers(_ context.Context) ([]mcpServerYAML, error) {
	var servers []models.MCPServerModel
	if err := a.db.Find(&servers).Error; err != nil {
		return nil, fmt.Errorf("query mcp servers: %w", err)
	}

	result := make([]mcpServerYAML, 0, len(servers))
	for _, s := range servers {
		my := mcpServerYAML{
			Name:    s.Name,
			Type:    s.Type,
			Command: s.Command,
			URL:     s.URL,
		}
		if s.Args != "" {
			var args []string
			if err := json.Unmarshal([]byte(s.Args), &args); err == nil {
				my.Args = args
			}
		}
		if s.EnvVars != "" {
			var envVars map[string]string
			if err := json.Unmarshal([]byte(s.EnvVars), &envVars); err == nil {
				// Mask env var values for security.
				masked := make(map[string]string, len(envVars))
				for k := range envVars {
					masked[k] = fmt.Sprintf("${%s}", k)
				}
				my.EnvVars = masked
			}
		}
		if s.ForwardHeaders != "" {
			var fh []string
			if err := json.Unmarshal([]byte(s.ForwardHeaders), &fh); err == nil {
				my.ForwardHeaders = fh
			}
		}
		result = append(result, my)
	}
	return result, nil
}

func (a *configImportExportHTTPAdapter) exportTriggers(_ context.Context) ([]triggerYAML, error) {
	var triggers []models.TriggerModel
	if err := a.db.Preload("Agent").Find(&triggers).Error; err != nil {
		return nil, fmt.Errorf("query triggers: %w", err)
	}

	result := make([]triggerYAML, 0, len(triggers))
	for _, t := range triggers {
		result = append(result, triggerYAML{
			Title:       t.Title,
			Type:        t.Type,
			AgentName:   t.Agent.Name,
			Schedule:    t.Schedule,
			WebhookPath: t.WebhookPath,
			Description: t.Description,
			Enabled:     t.Enabled,
		})
	}
	return result, nil
}

// ImportYAML parses YAML config and writes to DB in a transaction.
func (a *configImportExportHTTPAdapter) ImportYAML(ctx context.Context, yamlData []byte) error {
	var cfg configYAML
	if err := yaml.Unmarshal(yamlData, &cfg); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	return a.db.Transaction(func(tx *gorm.DB) error {
		if err := a.importModels(tx, cfg.Models.Items); err != nil {
			return fmt.Errorf("import models: %w", err)
		}

		if err := a.importMCPServers(tx, cfg.MCPServers.Items); err != nil {
			return fmt.Errorf("import mcp servers: %w", err)
		}

		if err := a.importAgents(tx, cfg.Agents.Items); err != nil {
			return fmt.Errorf("import agents: %w", err)
		}

		if err := a.importTriggers(tx, cfg.Triggers.Items); err != nil {
			return fmt.Errorf("import triggers: %w", err)
		}

		slog.InfoContext(ctx, "config imported",
			"agents", len(cfg.Agents.Items),
			"models", len(cfg.Models.Items),
			"mcp_servers", len(cfg.MCPServers.Items),
			"triggers", len(cfg.Triggers.Items),
		)
		return nil
	})
}

func (a *configImportExportHTTPAdapter) importModels(tx *gorm.DB, items []modelYAML) error {
	for _, m := range items {
		var existing models.LLMProviderModel
		err := tx.Where("name = ?", m.Name).First(&existing).Error
		if err == nil {
			// Update existing (preserve API key).
			existing.Type = m.resolvedType()
			existing.BaseURL = m.BaseURL
			existing.ModelName = m.ModelName
			if err := tx.Save(&existing).Error; err != nil {
				return fmt.Errorf("update model %q: %w", m.Name, err)
			}
			continue
		}

		newModel := models.LLMProviderModel{
			Name:      m.Name,
			Type:      m.resolvedType(),
			BaseURL:   m.BaseURL,
			ModelName: m.ModelName,
		}
		if err := tx.Create(&newModel).Error; err != nil {
			return fmt.Errorf("create model %q: %w", m.Name, err)
		}
	}
	return nil
}

func (a *configImportExportHTTPAdapter) importMCPServers(tx *gorm.DB, items []mcpServerYAML) error {
	for _, s := range items {
		var existing models.MCPServerModel
		err := tx.Where("name = ?", s.Name).First(&existing).Error

		argsJSON := ""
		if len(s.Args) > 0 {
			data, _ := json.Marshal(s.Args)
			argsJSON = string(data)
		}

		envJSON := ""
		if len(s.EnvVars) > 0 {
			// Filter out placeholder values like "${VAR_NAME}".
			clean := make(map[string]string)
			for k, v := range s.EnvVars {
				if !isEnvPlaceholder(v) {
					clean[k] = v
				}
			}
			if len(clean) > 0 {
				data, _ := json.Marshal(clean)
				envJSON = string(data)
			}
		}

		forwardHeadersJSON := ""
		if len(s.ForwardHeaders) > 0 {
			data, _ := json.Marshal(s.ForwardHeaders)
			forwardHeadersJSON = string(data)
		}

		if err == nil {
			existing.Type = s.Type
			existing.Command = s.Command
			existing.URL = s.URL
			if argsJSON != "" {
				existing.Args = argsJSON
			}
			if envJSON != "" {
				existing.EnvVars = envJSON
			}
			if forwardHeadersJSON != "" {
				existing.ForwardHeaders = forwardHeadersJSON
			}
			if err := tx.Save(&existing).Error; err != nil {
				return fmt.Errorf("update mcp server %q: %w", s.Name, err)
			}
			continue
		}

		newServer := models.MCPServerModel{
			Name:           s.Name,
			Type:           s.Type,
			Command:        s.Command,
			Args:           argsJSON,
			URL:            s.URL,
			EnvVars:        envJSON,
			ForwardHeaders: forwardHeadersJSON,
		}
		if err := tx.Create(&newServer).Error; err != nil {
			return fmt.Errorf("create mcp server %q: %w", s.Name, err)
		}
	}
	return nil
}

func applyAgentImportDefaults(ag *agentYAML) {
	if ag.Lifecycle == "" {
		ag.Lifecycle = "persistent"
	}
	if ag.ToolExecution == "" {
		ag.ToolExecution = "sequential"
	}
	if ag.MaxSteps == 0 {
		ag.MaxSteps = 50
	}
	if ag.MaxContextSize == 0 {
		ag.MaxContextSize = 16000
	}
}

func (a *configImportExportHTTPAdapter) importAgents(tx *gorm.DB, items []agentYAML) error {
	// Pass 1: create/update all agent records (without spawn targets that reference other agents).
	agentIDs := make(map[string]uint, len(items))
	for _, ag := range items {
		applyAgentImportDefaults(&ag)
		var modelID *uint
		if ag.ModelName != "" {
			var llm models.LLMProviderModel
			if err := tx.Where("name = ?", ag.ModelName).First(&llm).Error; err != nil {
				return fmt.Errorf("model %q referenced by agent %q not found: %w", ag.ModelName, ag.Name, err)
			}
			modelID = &llm.ID
		}

		var existing models.AgentModel
		err := tx.Where("name = ?", ag.Name).First(&existing).Error
		if err == nil {
			existing.SystemPrompt = ag.SystemPrompt
			existing.ModelID = modelID
			existing.Kit = ag.Kit
			existing.KnowledgePath = ag.KnowledgePath
			existing.Lifecycle = ag.Lifecycle
			existing.ToolExecution = ag.ToolExecution
			existing.MaxSteps = ag.MaxSteps
			existing.MaxContextSize = ag.MaxContextSize
			if cbJSON, err := json.Marshal(ag.ConfirmBefore); err == nil && len(ag.ConfirmBefore) > 0 {
				existing.ConfirmBefore = string(cbJSON)
			} else {
				existing.ConfirmBefore = ""
			}
			if err := tx.Save(&existing).Error; err != nil {
				return fmt.Errorf("update agent %q: %w", ag.Name, err)
			}
			agentIDs[ag.Name] = existing.ID
			continue
		}

		newAgent := models.AgentModel{
			Name:           ag.Name,
			SystemPrompt:   ag.SystemPrompt,
			ModelID:        modelID,
			Kit:            ag.Kit,
			KnowledgePath:  ag.KnowledgePath,
			Lifecycle:      ag.Lifecycle,
			ToolExecution:  ag.ToolExecution,
			MaxSteps:       ag.MaxSteps,
			MaxContextSize: ag.MaxContextSize,
			ConfirmBefore: func() string {
				if len(ag.ConfirmBefore) == 0 {
					return ""
				}
				d, _ := json.Marshal(ag.ConfirmBefore)
				return string(d)
			}(),
		}
		if err := tx.Create(&newAgent).Error; err != nil {
			return fmt.Errorf("create agent %q: %w", ag.Name, err)
		}
		agentIDs[ag.Name] = newAgent.ID
	}

	// Pass 2: sync relations (tools, spawn targets, MCP servers, escalation).
	for _, ag := range items {
		agentID := agentIDs[ag.Name]
		if err := a.syncAgentRelations(tx, agentID, ag); err != nil {
			return fmt.Errorf("sync relations for agent %q: %w", ag.Name, err)
		}
	}

	return nil
}

func (a *configImportExportHTTPAdapter) syncAgentRelations(tx *gorm.DB, agentID uint, ag agentYAML) error {
	// Tools: delete old, insert new.
	if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentToolModel{}).Error; err != nil {
		return fmt.Errorf("delete old tools: %w", err)
	}
	for i, toolName := range ag.Tools {
		tool := models.AgentToolModel{
			AgentID:   agentID,
			ToolType:  models.ToolTypeBuiltin,
			ToolName:  toolName,
			SortOrder: i,
		}
		if err := tx.Create(&tool).Error; err != nil {
			return fmt.Errorf("create tool %q: %w", toolName, err)
		}
	}

	// Spawn targets: delete old, insert new.
	if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentSpawnTarget{}).Error; err != nil {
		return fmt.Errorf("delete old spawn targets: %w", err)
	}
	for _, targetName := range ag.CanSpawn {
		var target models.AgentModel
		if err := tx.Where("name = ?", targetName).First(&target).Error; err != nil {
			return fmt.Errorf("spawn target %q not found: %w", targetName, err)
		}
		st := models.AgentSpawnTarget{
			AgentID:       agentID,
			TargetAgentID: target.ID,
		}
		if err := tx.Create(&st).Error; err != nil {
			return fmt.Errorf("create spawn target %q: %w", targetName, err)
		}
	}

	// MCP servers: delete old, insert new.
	if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentMCPServer{}).Error; err != nil {
		return fmt.Errorf("delete old mcp server links: %w", err)
	}
	for _, mcpName := range ag.MCPServers {
		var mcp models.MCPServerModel
		if err := tx.Where("name = ?", mcpName).First(&mcp).Error; err != nil {
			return fmt.Errorf("mcp server %q not found: %w", mcpName, err)
		}
		link := models.AgentMCPServer{
			AgentID:     agentID,
			MCPServerID: mcp.ID,
		}
		if err := tx.Create(&link).Error; err != nil {
			return fmt.Errorf("link mcp server %q: %w", mcpName, err)
		}
	}

	// Escalation: delete old, insert new.
	if err := tx.Where("agent_id = ?", agentID).Delete(&models.AgentEscalation{}).Error; err != nil {
		return fmt.Errorf("delete old escalation: %w", err)
	}
	if ag.Escalation != nil {
		esc := models.AgentEscalation{
			AgentID:    agentID,
			Action:     ag.Escalation.Action,
			WebhookURL: ag.Escalation.WebhookURL,
		}
		if err := tx.Create(&esc).Error; err != nil {
			return fmt.Errorf("create escalation: %w", err)
		}
		for _, keyword := range ag.Escalation.Triggers {
			trigger := models.AgentEscalationTrigger{
				EscalationID: esc.ID,
				Keyword:      keyword,
			}
			if err := tx.Create(&trigger).Error; err != nil {
				return fmt.Errorf("create escalation trigger %q: %w", keyword, err)
			}
		}
	}

	return nil
}

func (a *configImportExportHTTPAdapter) importTriggers(tx *gorm.DB, items []triggerYAML) error {
	for _, t := range items {
		var agent models.AgentModel
		if err := tx.Where("name = ?", t.AgentName).First(&agent).Error; err != nil {
			return fmt.Errorf("agent %q referenced by trigger %q not found: %w", t.AgentName, t.Title, err)
		}

		var existing models.TriggerModel
		err := tx.Where("title = ? AND agent_id = ?", t.Title, agent.ID).First(&existing).Error
		if err == nil {
			existing.Type = t.Type
			existing.Schedule = t.Schedule
			existing.WebhookPath = t.WebhookPath
			existing.Description = t.Description
			existing.Enabled = t.Enabled
			if err := tx.Save(&existing).Error; err != nil {
				return fmt.Errorf("update trigger %q: %w", t.Title, err)
			}
			continue
		}

		newTrigger := models.TriggerModel{
			Type:        t.Type,
			Title:       t.Title,
			AgentID:     agent.ID,
			Schedule:    t.Schedule,
			WebhookPath: t.WebhookPath,
			Description: t.Description,
			Enabled:     t.Enabled,
		}
		if err := tx.Create(&newTrigger).Error; err != nil {
			return fmt.Errorf("create trigger %q: %w", t.Title, err)
		}
	}
	return nil
}

// splitCSV splits a comma-separated string into a slice, trimming whitespace.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// auditServiceHTTPAdapter bridges GORMAuditRepository to the http.AuditService interface.
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

// toolCallLogHTTPAdapter bridges ToolCallEventRepository to the http.ToolCallEventQuerier interface.
type toolCallLogHTTPAdapter struct {
	repo *config_repo.ToolCallEventRepository
}

func (a *toolCallLogHTTPAdapter) QueryToolCalls(ctx context.Context, filters deliveryhttp.ToolCallFilters, page, perPage int) ([]deliveryhttp.ToolCallEntry, int64, error) {
	repoFilters := config_repo.ToolCallFilters{
		SessionID: filters.SessionID,
		AgentName: filters.AgentName,
		ToolName:  filters.ToolName,
		Status:    filters.Status,
		UserID:    filters.UserID,
		From:      filters.From,
		To:        filters.To,
	}

	entries, total, err := a.repo.QueryToolCalls(ctx, repoFilters, page, perPage)
	if err != nil {
		return nil, 0, err
	}

	result := make([]deliveryhttp.ToolCallEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, deliveryhttp.ToolCallEntry{
			ID:         e.ID,
			SessionID:  e.SessionID,
			AgentName:  e.AgentName,
			ToolName:   e.ToolName,
			Input:      e.Input,
			Output:     e.Output,
			Status:     e.Status,
			DurationMs: e.DurationMs,
			UserID:     e.UserID,
			CreatedAt:  e.CreatedAt,
		})
	}
	return result, total, nil
}

// isEnvPlaceholder checks if a value is an env var placeholder like "${VAR_NAME}".
func isEnvPlaceholder(v string) bool {
	return strings.HasPrefix(v, "${") && strings.HasSuffix(v, "}")
}

// agentManagerHTTPAdapter bridges GORMAgentRepository + AgentRegistry to the http.AgentManager interface.
type agentManagerHTTPAdapter struct {
	repo     *config_repo.GORMAgentRepository
	registry *agent_registry.AgentRegistry
	db       *gorm.DB
}

func (a *agentManagerHTTPAdapter) ListAgents(ctx context.Context) ([]deliveryhttp.AgentInfo, error) {
	records, err := a.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	result := make([]deliveryhttp.AgentInfo, 0, len(records))
	for _, rec := range records {
		result = append(result, deliveryhttp.AgentInfo{
			Name:         rec.Name,
			ToolsCount:   len(rec.BuiltinTools) + len(rec.CustomTools),
			Kit:          rec.Kit,
			HasKnowledge: rec.KnowledgePath != "",
			Public:       rec.Public,
		})
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
			Public:       rec.Public,
		},
		SystemPrompt:   rec.SystemPrompt,
		KnowledgePath:  rec.KnowledgePath,
		Tools:          tools,
		CanSpawn:       rec.CanSpawn,
		Lifecycle:      rec.Lifecycle,
		ToolExecution:  rec.ToolExecution,
		MaxSteps:       rec.MaxSteps,
		MaxContextSize: rec.MaxContextSize,
		ConfirmBefore:  rec.ConfirmBefore,
		MCPServers:     rec.MCPServers,
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

	return detail, nil
}

func (a *agentManagerHTTPAdapter) CreateAgent(ctx context.Context, req deliveryhttp.CreateAgentRequest) (*deliveryhttp.AgentDetail, error) {
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
	record := a.toAgentRecord(req)
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

func (a *agentManagerHTTPAdapter) toAgentRecord(req deliveryhttp.CreateAgentRequest) *config_repo.AgentRecord {
	rec := &config_repo.AgentRecord{
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
		Public:         req.Public,
	}

	// Resolve model: by ID or by name.
	if req.ModelID != nil {
		var llm models.LLMProviderModel
		if err := a.db.First(&llm, *req.ModelID).Error; err == nil {
			rec.ModelName = llm.Name
		}
	} else if req.Model != "" {
		rec.ModelName = req.Model
	}

	if req.Escalation != nil {
		rec.Escalation = &config_repo.EscalationRecord{
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

func (a *agentManagerHTTPAdapter) resolveModelID(_ context.Context, modelName string) *uint {
	if modelName == "" {
		return nil
	}
	var llm models.LLMProviderModel
	if err := a.db.Where("name = ?", modelName).First(&llm).Error; err != nil {
		return nil
	}
	return &llm.ID
}

// agentPublicCheckerDB implements deliveryhttp.AgentPublicChecker using the database.
type agentPublicCheckerDB struct {
	db *gorm.DB
}

func (c *agentPublicCheckerDB) IsAgentPublic(_ context.Context, name string) (bool, bool, error) {
	var agent models.AgentModel
	err := c.db.Select("id", "public").Where("name = ?", name).First(&agent).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, false, nil
		}
		return false, false, fmt.Errorf("check agent public: %w", err)
	}
	return true, agent.Public, nil
}

// agentPublicCheckerRegistry implements deliveryhttp.AgentPublicChecker using the in-memory agent registry.
type agentPublicCheckerRegistry struct {
	registry *agent_registry.AgentRegistry
}

func (c *agentPublicCheckerRegistry) IsAgentPublic(_ context.Context, name string) (bool, bool, error) {
	agent, err := c.registry.Get(name)
	if err != nil {
		return false, false, nil
	}
	return true, agent.Record.Public, nil
}

// ModelCacheInvalidator allows invalidating cached model clients when models are modified.
type ModelCacheInvalidator interface {
	Invalidate(modelID uint)
}

// modelServiceHTTPAdapter bridges GORMLLMProviderRepository to the http.ModelService interface.
type modelServiceHTTPAdapter struct {
	repo       *config_repo.GORMLLMProviderRepository
	modelCache ModelCacheInvalidator
}

func (m *modelServiceHTTPAdapter) ListModels(ctx context.Context) ([]deliveryhttp.ModelResponse, error) {
	providers, err := m.repo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}

	result := make([]deliveryhttp.ModelResponse, 0, len(providers))
	for _, p := range providers {
		result = append(result, deliveryhttp.ModelResponse{
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

func (m *modelServiceHTTPAdapter) CreateModel(ctx context.Context, req deliveryhttp.CreateModelRequest) (*deliveryhttp.ModelResponse, error) {
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

	return &deliveryhttp.ModelResponse{
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

func (m *modelServiceHTTPAdapter) UpdateModel(ctx context.Context, name string, req deliveryhttp.CreateModelRequest) (*deliveryhttp.ModelResponse, error) {
	// Find existing by name.
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
	// Only update API key if provided (empty means keep existing).
	if req.APIKey != "" {
		update.APIKeyEncrypted = req.APIKey
	}

	if err := m.repo.Update(ctx, existing.ID, update); err != nil {
		return nil, fmt.Errorf("update model: %w", err)
	}

	// Invalidate cached client so next access picks up changes.
	if m.modelCache != nil {
		m.modelCache.Invalidate(existing.ID)
	}

	// Re-read to get updated fields.
	hasKey := existing.APIKeyEncrypted != ""
	if req.APIKey != "" {
		hasKey = true
	}

	respName := req.Name
	if respName == "" {
		respName = existing.Name
	}

	return &deliveryhttp.ModelResponse{
		ID:         existing.ID,
		Name:       respName,
		Type:       req.Type,
		BaseURL:    req.BaseURL,
		ModelName:  req.ModelName,
		HasAPIKey:  hasKey,
		APIVersion: req.APIVersion,
		CreatedAt:  existing.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (m *modelServiceHTTPAdapter) DeleteModel(ctx context.Context, name string) error {
	// Find existing by name.
	providers, err := m.repo.List(ctx)
	if err != nil {
		return fmt.Errorf("list models for delete: %w", err)
	}

	for _, p := range providers {
		if p.Name == name {
			if err := m.repo.Delete(ctx, p.ID); err != nil {
				return err
			}
			// Invalidate cached client for the deleted model.
			if m.modelCache != nil {
				m.modelCache.Invalidate(p.ID)
			}
			return nil
		}
	}
	return pkgerrors.NotFound(fmt.Sprintf("model not found: %s", name))
}

func (m *modelServiceHTTPAdapter) VerifyModel(ctx context.Context, name string) (*deliveryhttp.ModelVerifyResult, error) {
	providers, err := m.repo.List(ctx)
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
		return nil, pkgerrors.NotFound(fmt.Sprintf("model not found: %s", name))
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

// taskServiceHTTPAdapter bridges task infrastructure to the http.TaskService interface.
type taskServiceHTTPAdapter struct {
	repo *config_repo.GORMTaskRepository
}

func (a *taskServiceHTTPAdapter) CreateTask(_ context.Context, _ deliveryhttp.CreateTaskRequest) (uint, error) {
	return 0, nil
}

func (a *taskServiceHTTPAdapter) buildRepoFilter(filter deliveryhttp.TaskListFilter) config_repo.TaskFilter {
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

func (a *taskServiceHTTPAdapter) ListTasks(ctx context.Context, filter deliveryhttp.TaskListFilter) ([]deliveryhttp.TaskResponse, error) {
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

func (a *taskServiceHTTPAdapter) CountTasks(ctx context.Context, filter deliveryhttp.TaskListFilter) (int64, error) {
	return a.repo.Count(ctx, a.buildRepoFilter(filter))
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

// knowledgeStatsHTTPAdapter bridges GORMKnowledgeRepository to the http.KnowledgeStats interface.
type knowledgeStatsHTTPAdapter struct {
	repo *config_repo.GORMKnowledgeRepository
}

func (a *knowledgeStatsHTTPAdapter) GetStats(ctx context.Context, agentName string) (int, int, *time.Time, error) {
	return a.repo.GetStats(ctx, agentName)
}

// knowledgeReindexerHTTPAdapter bridges knowledge.Indexer to the http.KnowledgeReindexer interface.
type knowledgeReindexerHTTPAdapter struct {
	indexer  knowledgeIndexer
	registry *agent_registry.AgentRegistry
}

// knowledgeIndexer is the consumer-side interface for indexing.
type knowledgeIndexer interface {
	IndexFolder(ctx context.Context, agentName string, folderPath string) error
}

func (a *knowledgeReindexerHTTPAdapter) Reindex(ctx context.Context, agentName string) error {
	agent, err := a.registry.Get(agentName)
	if err != nil {
		return fmt.Errorf("agent not found: %s", agentName)
	}
	if agent.Record.KnowledgePath == "" {
		return fmt.Errorf("agent %s has no knowledge_path configured", agentName)
	}
	slog.InfoContext(ctx, "starting knowledge reindex",
		"agent", agentName, "path", agent.Record.KnowledgePath)
	return a.indexer.IndexFolder(ctx, agentName, agent.Record.KnowledgePath)
}

// chatServiceHTTPAdapter bridges SessionRegistry + SessionProcessor to the
// deliveryhttp.ChatService interface for the REST chat endpoint.
type chatServiceHTTPAdapter struct {
	registry     *flow_registry.SessionRegistry
	processor    *session_processor.Processor
	agents       *agent_registry.AgentRegistry
	chatEnabled  bool // false when no LLM model configured
}

// Chat creates (or resumes) a session, enqueues the message, subscribes to
// events, and returns an SSEEvent channel that closes when processing stops.
func (a *chatServiceHTTPAdapter) Chat(ctx context.Context, agentName, message, userID, sessionID string) (<-chan deliveryhttp.SSEEvent, error) {
	if a.agents == nil {
		return nil, fmt.Errorf("no agents configured")
	}

	if _, err := a.agents.Get(agentName); err != nil {
		return nil, pkgerrors.NotFound(fmt.Sprintf("agent not found: %s", agentName))
	}

	// Create a new session if none provided.
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	// Create session in registry (idempotent — reuses existing if already present).
	if !a.registry.HasSession(sessionID) {
		a.registry.CreateSession(sessionID, "", userID, "", "", agentName)
	}

	// Subscribe BEFORE enqueueing so we don't miss events.
	eventCh, cleanup := a.registry.Subscribe(sessionID)

	// Enqueue the user message.
	if err := a.registry.EnqueueMessage(sessionID, message); err != nil {
		cleanup()
		return nil, fmt.Errorf("enqueue message: %w", err)
	}

	// Start processing with the enriched context (carries RequestContext for MCP header forwarding).
	a.processor.StartProcessing(ctx, sessionID)

	// Fan-out: read proto events, convert to SSE, close when processing stops.
	// Buffered channel prevents deadlock: PublishEvent holds entry.mu.Lock while
	// sending to subscriber channel. If sseCh is unbuffered and the HTTP handler
	// is slow to read/flush, the fan-out goroutine blocks on sseCh send, which
	// blocks the subscriber channel read, which blocks PublishEvent, which holds
	// the lock and blocks ALL subsequent events — causing stream truncation.
	sseCh := make(chan deliveryhttp.SSEEvent, 64)
	go func() {
		defer close(sseCh)
		defer cleanup()
		eventCount := 0

		for protoEvent := range eventCh {
			sseEvent := convertSessionEventToSSE(protoEvent, sessionID)
			if sseEvent == nil {
				continue
			}
			eventCount++
			sseCh <- *sseEvent

			if sseEvent.Type == "done" {
				return
			}
		}
	}()

	return sseCh, nil
}

// convertSessionEventToSSE maps a pb.SessionEvent to an SSEEvent.
// Returns nil for event types that should not be forwarded over SSE.
func convertSessionEventToSSE(event *pb.SessionEvent, sessionID string) *deliveryhttp.SSEEvent {
	switch event.GetType() {
	case pb.SessionEventType_SESSION_EVENT_REASONING:
		return sseEventJSON("thinking", map[string]interface{}{
			"content": event.GetContent(),
		})

	case pb.SessionEventType_SESSION_EVENT_ANSWER_CHUNK:
		return sseEventJSON("message_delta", map[string]interface{}{
			"content": event.GetContent(),
		})

	case pb.SessionEventType_SESSION_EVENT_ANSWER:
		return sseEventJSON("message", map[string]interface{}{
			"content": event.GetContent(),
		})

	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_START:
		data := map[string]interface{}{
			"tool":    event.GetToolName(),
			"call_id": event.GetCallId(),
		}
		if args := event.GetToolArguments(); len(args) > 0 {
			data["arguments"] = args
		}
		return sseEventJSON("tool_call", data)

	case pb.SessionEventType_SESSION_EVENT_TOOL_EXECUTION_END:
		return sseEventJSON("tool_result", map[string]interface{}{
			"tool":      event.GetToolName(),
			"call_id":   event.GetCallId(),
			"content":   event.GetToolResultSummary(),
			"has_error": event.GetToolHasError(),
		})

	case pb.SessionEventType_SESSION_EVENT_ASK_USER:
		return sseEventJSON("confirmation", map[string]interface{}{
			"content": event.GetContent(),
			"call_id": event.GetCallId(),
		})

	case pb.SessionEventType_SESSION_EVENT_PROCESSING_STOPPED:
		return sseEventJSON("done", map[string]interface{}{
			"session_id": sessionID,
		})

	case pb.SessionEventType_SESSION_EVENT_ERROR:
		data := map[string]interface{}{
			"content": event.GetContent(),
		}
		if detail := event.GetErrorDetail(); detail != nil {
			data["code"] = detail.GetCode()
			data["message"] = detail.GetMessage()
		}
		return sseEventJSON("error", data)

	default:
		// PROCESSING_STARTED, USER_MESSAGE, PLAN_UPDATE, UNSPECIFIED — skip.
		return nil
	}
}

// sseEventJSON creates an SSEEvent with JSON-encoded data.
func sseEventJSON(eventType string, data map[string]interface{}) *deliveryhttp.SSEEvent {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal SSE event data", "type", eventType, "error", err)
		return nil
	}
	return &deliveryhttp.SSEEvent{
		Type: eventType,
		Data: string(jsonBytes),
	}
}
