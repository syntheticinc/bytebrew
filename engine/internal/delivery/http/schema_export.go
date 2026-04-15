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
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Agents      []AgentYAML `yaml:"agents,omitempty"`
	Edges       []EdgeYAML  `yaml:"edges,omitempty"`
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

// EdgeYAML is the YAML representation of an edge.
type EdgeYAML struct {
	Source string                 `yaml:"source"`
	Target string                 `yaml:"target"`
	Type   string                 `yaml:"type"`
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

	edges, err := h.edges.ListEdges(ctx, id)
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

	// Build edges
	for _, e := range edges {
		export.Edges = append(export.Edges, EdgeYAML{
			Source: e.SourceAgentName,
			Target: e.TargetAgentName,
			Type:   e.Type,
			Config: e.Config,
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
// It accepts a YAML body and creates a schema with agents and edges.
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

	// 2. Add agents to schema (by name reference)
	for _, agent := range input.Agents {
		if agent.Name == "" {
			continue
		}
		if err := h.schemas.AddSchemaAgent(ctx, schema.ID, agent.Name); err != nil {
			slog.WarnContext(ctx, "failed to add agent to imported schema",
				"schema", schema.Name, "agent", agent.Name, "error", err)
		}
	}

	// 3. Create edges
	for _, e := range input.Edges {
		if e.Source == "" || e.Target == "" {
			continue
		}
		_, err := h.edges.CreateEdge(ctx, schema.ID, CreateEdgeRequest{
			Source: e.Source,
			Target: e.Target,
			Type:   e.Type,
			Config: e.Config,
		})
		if err != nil {
			slog.WarnContext(ctx, "failed to create edge for imported schema",
				"schema", schema.Name, "edge_source", e.Source, "edge_target", e.Target, "error", err)
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
