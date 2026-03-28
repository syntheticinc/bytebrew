package config_repo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// AgentRecord is an intermediate struct for DB <-> domain mapping.
// Contains all agent config from DB (agent + tools + spawn + escalation + MCP).
type AgentRecord struct {
	Name           string
	ModelID        *uint
	ModelName      string
	SystemPrompt   string
	Kit            string
	KnowledgePath  string
	Lifecycle      string
	ToolExecution  string
	MaxSteps       int
	MaxContextSize int
	ConfirmBefore  []string
	BuiltinTools   []string
	CustomTools    []CustomToolRecord
	MCPServers     []string
	CanSpawn       []string
	Escalation     *EscalationRecord
}

// CustomToolRecord holds a custom tool name and its JSON config.
type CustomToolRecord struct {
	Name   string
	Config string
}

// EscalationRecord holds escalation settings for an agent.
type EscalationRecord struct {
	Action     string
	WebhookURL string
	Triggers   []string
}

// GORMAgentRepository implements AgentReader and AgentWriter using GORM.
type GORMAgentRepository struct {
	db *gorm.DB
}

// NewGORMAgentRepository creates a new GORMAgentRepository.
func NewGORMAgentRepository(db *gorm.DB) *GORMAgentRepository {
	return &GORMAgentRepository{db: db}
}

// List returns all agent records with all associations preloaded.
func (r *GORMAgentRepository) List(ctx context.Context) ([]AgentRecord, error) {
	var agents []models.AgentModel
	err := r.db.WithContext(ctx).
		Preload("Tools").
		Preload("SpawnTargets").
		Preload("SpawnTargets.TargetAgent").
		Preload("Escalation").
		Preload("Escalation.Triggers").
		Preload("Model").
		Find(&agents).Error
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	// Load MCP server names for all agents in one query
	mcpByAgent, err := r.loadAllAgentMCPServers(ctx)
	if err != nil {
		return nil, fmt.Errorf("load mcp servers: %w", err)
	}

	records := make([]AgentRecord, 0, len(agents))
	for _, a := range agents {
		rec, err := toAgentRecord(a)
		if err != nil {
			return nil, fmt.Errorf("convert agent %q: %w", a.Name, err)
		}
		rec.MCPServers = mcpByAgent[a.ID]
		records = append(records, rec)
	}
	return records, nil
}

// GetByName returns a single agent record by name.
func (r *GORMAgentRepository) GetByName(ctx context.Context, name string) (*AgentRecord, error) {
	var agent models.AgentModel
	err := r.db.WithContext(ctx).
		Preload("Tools").
		Preload("SpawnTargets").
		Preload("SpawnTargets.TargetAgent").
		Preload("Escalation").
		Preload("Escalation.Triggers").
		Preload("Model").
		Where("name = ?", name).
		First(&agent).Error
	if err != nil {
		return nil, fmt.Errorf("get agent %q: %w", name, err)
	}

	rec, err := toAgentRecord(agent)
	if err != nil {
		return nil, fmt.Errorf("convert agent %q: %w", name, err)
	}

	// Load MCP server names separately (GORM many2many infers wrong column names)
	mcpNames, err := r.loadMCPServersForAgent(ctx, agent.ID)
	if err != nil {
		return nil, fmt.Errorf("load mcp servers for agent %q: %w", name, err)
	}
	rec.MCPServers = mcpNames

	return &rec, nil
}

// Count returns the number of agents in the database.
func (r *GORMAgentRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.AgentModel{}).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count agents: %w", err)
	}
	return count, nil
}

// Create inserts a new agent with all associations.
func (r *GORMAgentRepository) Create(ctx context.Context, record *AgentRecord) error {
	agent, err := r.toAgentModel(ctx, record)
	if err != nil {
		return fmt.Errorf("build agent model: %w", err)
	}

	if err := r.db.WithContext(ctx).Create(&agent).Error; err != nil {
		return fmt.Errorf("create agent %q: %w", record.Name, err)
	}

	if err := r.createSpawnTargets(ctx, agent.ID, record.CanSpawn); err != nil {
		return fmt.Errorf("create spawn targets: %w", err)
	}

	if err := r.createMCPAssociations(ctx, agent.ID, record.MCPServers); err != nil {
		return fmt.Errorf("create mcp associations: %w", err)
	}

	return nil
}

// Update replaces the agent record identified by name.
func (r *GORMAgentRepository) Update(ctx context.Context, name string, record *AgentRecord) error {
	var existing models.AgentModel
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&existing).Error; err != nil {
		return fmt.Errorf("find agent %q: %w", name, err)
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete old associations
		if err := tx.Where("agent_id = ?", existing.ID).Delete(&models.AgentToolModel{}).Error; err != nil {
			return fmt.Errorf("delete old tools: %w", err)
		}
		if err := tx.Where("agent_id = ?", existing.ID).Delete(&models.AgentSpawnTarget{}).Error; err != nil {
			return fmt.Errorf("delete old spawn targets: %w", err)
		}
		if err := r.deleteEscalation(tx, existing.ID); err != nil {
			return fmt.Errorf("delete old escalation: %w", err)
		}
		if err := tx.Exec("DELETE FROM agent_mcp_servers WHERE agent_id = ?", existing.ID).Error; err != nil {
			return fmt.Errorf("delete old mcp associations: %w", err)
		}

		// Build updated model
		agent, err := r.toAgentModelWithTx(tx, record)
		if err != nil {
			return fmt.Errorf("build agent model: %w", err)
		}

		// Update scalar fields
		updates := map[string]interface{}{
			"name":             agent.Name,
			"model_id":         agent.ModelID,
			"system_prompt":    agent.SystemPrompt,
			"kit":              agent.Kit,
			"knowledge_path":   agent.KnowledgePath,
			"lifecycle":        agent.Lifecycle,
			"tool_execution":   agent.ToolExecution,
			"max_steps":        agent.MaxSteps,
			"max_context_size": agent.MaxContextSize,
			"confirm_before":   agent.ConfirmBefore,
		}
		if err := tx.Model(&models.AgentModel{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("update agent %q: %w", name, err)
		}

		// Recreate associations with existing ID
		for i := range agent.Tools {
			agent.Tools[i].AgentID = existing.ID
		}
		if len(agent.Tools) > 0 {
			if err := tx.Create(&agent.Tools).Error; err != nil {
				return fmt.Errorf("create tools: %w", err)
			}
		}

		if agent.Escalation != nil {
			agent.Escalation.AgentID = existing.ID
			if err := tx.Create(agent.Escalation).Error; err != nil {
				return fmt.Errorf("create escalation: %w", err)
			}
		}

		if err := r.createSpawnTargetsWithTx(tx, existing.ID, record.CanSpawn); err != nil {
			return fmt.Errorf("create spawn targets: %w", err)
		}

		if err := r.createMCPAssociationsWithTx(tx, existing.ID, record.MCPServers); err != nil {
			return fmt.Errorf("create mcp associations: %w", err)
		}

		return nil
	})
}

// Delete removes an agent and all its associations by name.
func (r *GORMAgentRepository) Delete(ctx context.Context, name string) error {
	var agent models.AgentModel
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&agent).Error; err != nil {
		return fmt.Errorf("find agent %q: %w", name, err)
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("agent_id = ?", agent.ID).Delete(&models.AgentToolModel{}).Error; err != nil {
			return fmt.Errorf("delete tools: %w", err)
		}
		if err := tx.Where("agent_id = ?", agent.ID).Delete(&models.AgentSpawnTarget{}).Error; err != nil {
			return fmt.Errorf("delete spawn targets: %w", err)
		}
		if err := r.deleteEscalation(tx, agent.ID); err != nil {
			return fmt.Errorf("delete escalation: %w", err)
		}
		if err := tx.Exec("DELETE FROM agent_mcp_servers WHERE agent_id = ?", agent.ID).Error; err != nil {
			return fmt.Errorf("delete mcp associations: %w", err)
		}
		if err := tx.Delete(&agent).Error; err != nil {
			return fmt.Errorf("delete agent %q: %w", name, err)
		}
		return nil
	})
}

// toAgentRecord converts AgentModel to AgentRecord.
func toAgentRecord(a models.AgentModel) (AgentRecord, error) {
	rec := AgentRecord{
		Name:           a.Name,
		SystemPrompt:   a.SystemPrompt,
		Kit:            a.Kit,
		KnowledgePath:  a.KnowledgePath,
		Lifecycle:      a.Lifecycle,
		ToolExecution:  a.ToolExecution,
		MaxSteps:       a.MaxSteps,
		MaxContextSize: a.MaxContextSize,
	}

	// Model ID and name
	rec.ModelID = a.ModelID
	if a.Model != nil {
		rec.ModelName = a.Model.Name
	}

	// ConfirmBefore: JSON array -> []string
	if a.ConfirmBefore != "" {
		if err := json.Unmarshal([]byte(a.ConfirmBefore), &rec.ConfirmBefore); err != nil {
			return AgentRecord{}, fmt.Errorf("parse confirm_before: %w", err)
		}
	}

	// Tools: split by type
	for _, t := range a.Tools {
		switch t.ToolType {
		case "builtin":
			rec.BuiltinTools = append(rec.BuiltinTools, t.ToolName)
		case "custom":
			rec.CustomTools = append(rec.CustomTools, CustomToolRecord{
				Name:   t.ToolName,
				Config: t.Config,
			})
		}
	}

	// SpawnTargets: extract target agent names
	for _, st := range a.SpawnTargets {
		rec.CanSpawn = append(rec.CanSpawn, st.TargetAgent.Name)
	}

	// MCP servers: skip loading (loaded separately if needed)

	// Escalation
	if a.Escalation != nil {
		esc := &EscalationRecord{
			Action:     a.Escalation.Action,
			WebhookURL: a.Escalation.WebhookURL,
		}
		for _, t := range a.Escalation.Triggers {
			esc.Triggers = append(esc.Triggers, t.Keyword)
		}
		rec.Escalation = esc
	}

	return rec, nil
}

// toAgentModel converts AgentRecord to AgentModel (for Create).
func (r *GORMAgentRepository) toAgentModel(ctx context.Context, rec *AgentRecord) (models.AgentModel, error) {
	return r.toAgentModelWithDB(r.db.WithContext(ctx), rec)
}

// toAgentModelWithTx converts AgentRecord to AgentModel using a transaction.
func (r *GORMAgentRepository) toAgentModelWithTx(tx *gorm.DB, rec *AgentRecord) (models.AgentModel, error) {
	return r.toAgentModelWithDB(tx, rec)
}

func (r *GORMAgentRepository) toAgentModelWithDB(db *gorm.DB, rec *AgentRecord) (models.AgentModel, error) {
	agent := models.AgentModel{
		Name:           rec.Name,
		SystemPrompt:   rec.SystemPrompt,
		Kit:            rec.Kit,
		KnowledgePath:  rec.KnowledgePath,
		Lifecycle:      rec.Lifecycle,
		ToolExecution:  rec.ToolExecution,
		MaxSteps:       rec.MaxSteps,
		MaxContextSize: rec.MaxContextSize,
	}

	// Resolve model name -> ID
	if rec.ModelName != "" {
		var model models.LLMProviderModel
		if err := db.Where("name = ?", rec.ModelName).First(&model).Error; err != nil {
			return models.AgentModel{}, fmt.Errorf("resolve model %q: %w", rec.ModelName, err)
		}
		agent.ModelID = &model.ID
	}

	// ConfirmBefore: []string -> JSON string
	if len(rec.ConfirmBefore) > 0 {
		data, err := json.Marshal(rec.ConfirmBefore)
		if err != nil {
			return models.AgentModel{}, fmt.Errorf("marshal confirm_before: %w", err)
		}
		agent.ConfirmBefore = string(data)
	}

	// Builtin tools
	for i, name := range rec.BuiltinTools {
		agent.Tools = append(agent.Tools, models.AgentToolModel{
			ToolType:  "builtin",
			ToolName:  name,
			SortOrder: i,
		})
	}

	// Custom tools
	for i, ct := range rec.CustomTools {
		agent.Tools = append(agent.Tools, models.AgentToolModel{
			ToolType:  "custom",
			ToolName:  ct.Name,
			Config:    ct.Config,
			SortOrder: len(rec.BuiltinTools) + i,
		})
	}

	// Escalation
	if rec.Escalation != nil {
		esc := &models.AgentEscalation{
			Action:     rec.Escalation.Action,
			WebhookURL: rec.Escalation.WebhookURL,
		}
		for _, keyword := range rec.Escalation.Triggers {
			esc.Triggers = append(esc.Triggers, models.AgentEscalationTrigger{
				Keyword: keyword,
			})
		}
		agent.Escalation = esc
	}

	return agent, nil
}

// createSpawnTargets resolves target agent names to IDs and inserts spawn target rows.
func (r *GORMAgentRepository) createSpawnTargets(ctx context.Context, agentID uint, targets []string) error {
	return r.createSpawnTargetsWithTx(r.db.WithContext(ctx), agentID, targets)
}

func (r *GORMAgentRepository) createSpawnTargetsWithTx(tx *gorm.DB, agentID uint, targets []string) error {
	if len(targets) == 0 {
		return nil
	}

	var targetAgents []models.AgentModel
	if err := tx.Where("name IN ?", targets).Find(&targetAgents).Error; err != nil {
		return fmt.Errorf("resolve spawn targets: %w", err)
	}

	nameToID := make(map[string]uint, len(targetAgents))
	for _, a := range targetAgents {
		nameToID[a.Name] = a.ID
	}

	for _, name := range targets {
		targetID, ok := nameToID[name]
		if !ok {
			return fmt.Errorf("spawn target agent %q not found", name)
		}
		st := models.AgentSpawnTarget{
			AgentID:       agentID,
			TargetAgentID: targetID,
		}
		if err := tx.Create(&st).Error; err != nil {
			return fmt.Errorf("create spawn target %q: %w", name, err)
		}
	}
	return nil
}

// createMCPAssociations links agent to MCP servers via join table.
func (r *GORMAgentRepository) createMCPAssociations(ctx context.Context, agentID uint, serverNames []string) error {
	return r.createMCPAssociationsWithTx(r.db.WithContext(ctx), agentID, serverNames)
}

func (r *GORMAgentRepository) createMCPAssociationsWithTx(tx *gorm.DB, agentID uint, serverNames []string) error {
	if len(serverNames) == 0 {
		return nil
	}

	var servers []models.MCPServerModel
	if err := tx.Where("name IN ?", serverNames).Find(&servers).Error; err != nil {
		return fmt.Errorf("resolve mcp servers: %w", err)
	}

	for _, s := range servers {
		if err := tx.Exec(
			"INSERT INTO agent_mcp_servers (agent_id, mcp_server_id) VALUES (?, ?)",
			agentID, s.ID,
		).Error; err != nil {
			return fmt.Errorf("link mcp server %q: %w", s.Name, err)
		}
	}
	return nil
}

// loadAllAgentMCPServers loads MCP server names for all agents in a single query.
func (r *GORMAgentRepository) loadAllAgentMCPServers(ctx context.Context) (map[uint][]string, error) {
	var joins []models.AgentMCPServer
	if err := r.db.WithContext(ctx).Preload("MCPServer").Find(&joins).Error; err != nil {
		return nil, fmt.Errorf("load agent mcp servers: %w", err)
	}

	result := make(map[uint][]string)
	for _, j := range joins {
		result[j.AgentID] = append(result[j.AgentID], j.MCPServer.Name)
	}
	return result, nil
}

// loadMCPServersForAgent loads MCP server names for a single agent.
func (r *GORMAgentRepository) loadMCPServersForAgent(ctx context.Context, agentID uint) ([]string, error) {
	var joins []models.AgentMCPServer
	if err := r.db.WithContext(ctx).Preload("MCPServer").Where("agent_id = ?", agentID).Find(&joins).Error; err != nil {
		return nil, fmt.Errorf("load mcp servers: %w", err)
	}

	names := make([]string, 0, len(joins))
	for _, j := range joins {
		names = append(names, j.MCPServer.Name)
	}
	return names, nil
}

// deleteEscalation removes escalation and its triggers for an agent.
func (r *GORMAgentRepository) deleteEscalation(tx *gorm.DB, agentID uint) error {
	var esc models.AgentEscalation
	err := tx.Where("agent_id = ?", agentID).First(&esc).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return fmt.Errorf("find escalation: %w", err)
	}

	if err := tx.Where("escalation_id = ?", esc.ID).Delete(&models.AgentEscalationTrigger{}).Error; err != nil {
		return fmt.Errorf("delete escalation triggers: %w", err)
	}
	if err := tx.Delete(&esc).Error; err != nil {
		return fmt.Errorf("delete escalation: %w", err)
	}
	return nil
}
