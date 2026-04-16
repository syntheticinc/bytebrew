package http

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"
)

// --- YAML DTOs for schema export/import ---

// SchemaYAML is the top-level YAML representation of a schema.
type SchemaYAML struct {
	Name           string               `yaml:"name"`
	Description    string               `yaml:"description,omitempty"`
	Agents         []AgentYAML          `yaml:"agents,omitempty"`
	AgentRelations []AgentRelationYAML  `yaml:"agent_relations,omitempty"`
}

// AgentYAML is the YAML representation of an agent within a schema export.
type AgentYAML struct {
	Name            string   `yaml:"name"`
	SystemPrompt    string   `yaml:"system_prompt,omitempty"`
	Lifecycle       string   `yaml:"lifecycle,omitempty"`
	ToolExecution   string   `yaml:"tool_execution,omitempty"`
	MaxSteps        int      `yaml:"max_steps,omitempty"`
	MaxContextSize  int      `yaml:"max_context_size,omitempty"`
	MaxTurnDuration int      `yaml:"max_turn_duration,omitempty"`
	Tools           []string `yaml:"tools,omitempty"`
	CanSpawn        []string `yaml:"can_spawn,omitempty"`
	ConfirmBefore   []string `yaml:"confirm_before,omitempty"`
	MCPServers      []string `yaml:"mcp_servers,omitempty"`
}

// AgentRelationYAML is the YAML representation of an agent relation.
//
// V2 has a single implicit DELEGATION relationship type (see
// docs/architecture/agent-first-runtime.md §3.1) — no `type` field is exported.
type AgentRelationYAML struct {
	Source string                 `yaml:"source"`
	Target string                 `yaml:"target"`
	Config map[string]interface{} `yaml:"config,omitempty"`
}

// --- Consumer-side interface for agent detail lookup ---

// SchemaAgentDetailer provides agent details for schema export.
type SchemaAgentDetailer interface {
	GetAgent(ctx context.Context, name string) (*AgentDetail, error)
}

// SetAgentDetailer sets the agent detailer used by export.
// This is optional; export will include agent names only if not set.
func (h *SchemaHandler) SetAgentDetailer(detailer SchemaAgentDetailer) {
	h.agentDetailer = detailer
}

// --- Export endpoint ---

// ExportSchema handles GET /api/v1/schemas/{id}/export.
// It returns the full schema as a YAML file.
func (h *SchemaHandler) ExportSchema(w http.ResponseWriter, r *http.Request) {
	id, err := parseStringParam(r, "id")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	schema, err := h.schemas.GetSchema(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	agents, err := h.schemas.ListSchemaAgents(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	relations, err := h.agentRelations.ListAgentRelations(ctx, id)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	export := SchemaYAML{
		Name:        schema.Name,
		Description: schema.Description,
	}

	// Build agent list with details if detailer is available
	for _, agentName := range agents {
		agentYAML := AgentYAML{Name: agentName}

		if h.agentDetailer != nil {
			detail, detailErr := h.agentDetailer.GetAgent(ctx, agentName)
			if detailErr != nil {
				slog.WarnContext(ctx, "failed to get agent detail for export, using name only",
					"agent", agentName, "error", detailErr)
			} else {
				agentYAML.SystemPrompt = detail.SystemPrompt
				agentYAML.Lifecycle = detail.Lifecycle
				agentYAML.ToolExecution = detail.ToolExecution
				agentYAML.MaxSteps = detail.MaxSteps
				agentYAML.MaxContextSize = detail.MaxContextSize
				agentYAML.MaxTurnDuration = detail.MaxTurnDuration
				agentYAML.Tools = detail.Tools
				agentYAML.CanSpawn = detail.CanSpawn
				agentYAML.ConfirmBefore = detail.ConfirmBefore
				agentYAML.MCPServers = detail.MCPServers
			}
		}

		export.Agents = append(export.Agents, agentYAML)
	}

	// Build agent relations
	for _, rel := range relations {
		export.AgentRelations = append(export.AgentRelations, AgentRelationYAML{
			Source: rel.SourceAgentID,
			Target: rel.TargetAgentID,
			Config: rel.Config,
		})
	}

	yamlData, err := yaml.Marshal(export)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("marshal yaml: %s", err.Error()))
		return
	}

	filename := sanitizeFilename(schema.Name) + ".yaml"
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	w.Write(yamlData)
}

// --- Import endpoint ---

// ImportSchema handles POST /api/v1/schemas/import.
// It accepts a YAML body and creates a schema with agents and agent relations.
func (h *SchemaHandler) ImportSchema(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("read body: %s", err.Error()))
		return
	}
	defer r.Body.Close()

	var input SchemaYAML
	if err := yaml.Unmarshal(body, &input); err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid yaml: %s", err.Error()))
		return
	}

	if input.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "name is required in yaml")
		return
	}

	ctx := r.Context()

	// 1. Create schema
	schema, err := h.schemas.CreateSchema(ctx, CreateSchemaRequest{
		Name:        input.Name,
		Description: input.Description,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	// 2. Create agent relations.
	//
	// V2: schema membership is derived from agent_relations (see
	// docs/architecture/agent-first-runtime.md §2.1). Agent names listed at
	// the top level of the YAML carry no membership information of their
	// own — an agent only counts as a schema member by participating in a
	// relation. Solo-agent imports therefore produce a schema with no
	// members; create at least one relation in the YAML to express the
	// delegation tree.
	for _, rel := range input.AgentRelations {
		if rel.Source == "" || rel.Target == "" {
			continue
		}
		_, err := h.agentRelations.CreateAgentRelation(ctx, schema.ID, CreateAgentRelationRequest{
			Source: rel.Source,
			Target: rel.Target,
			Config: rel.Config,
		})
		if err != nil {
			slog.WarnContext(ctx, "failed to create agent relation for imported schema",
				"schema", schema.Name, "relation_source", rel.Source, "relation_target", rel.Target, "error", err)
		}
	}

	// Re-fetch schema to include agents in response
	result, err := h.schemas.GetSchema(ctx, schema.ID)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

// sanitizeFilename replaces characters not safe for filenames.
func sanitizeFilename(name string) string {
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "-",
	)
	result := replacer.Replace(name)
	if result == "" {
		return "schema"
	}
	return result
}
