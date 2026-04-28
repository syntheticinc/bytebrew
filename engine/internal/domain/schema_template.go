package domain

import "time"

// SchemaTemplateCategory is the enum of supported template categories.
//
// V2 ships a curated set — user-created templates are out of scope (§2.2).
type SchemaTemplateCategory string

const (
	SchemaTemplateCategorySupport  SchemaTemplateCategory = "support"
	SchemaTemplateCategorySales    SchemaTemplateCategory = "sales"
	SchemaTemplateCategoryInternal SchemaTemplateCategory = "internal"
	SchemaTemplateCategoryGeneric  SchemaTemplateCategory = "generic"
)

// IsValid reports whether c is one of the four supported template categories.
func (c SchemaTemplateCategory) IsValid() bool {
	switch c {
	case SchemaTemplateCategorySupport,
		SchemaTemplateCategorySales,
		SchemaTemplateCategoryInternal,
		SchemaTemplateCategoryGeneric:
		return true
	}
	return false
}

// SchemaTemplate is the system-wide catalog entry for a schema starter
// template. Same hybrid pattern as MCPCatalogRecord (§5.5): rows are seeded
// from `schema-templates.yaml` at engine startup via seedSchemaTemplates,
// and the "Use template" action forks the `Definition` graph into
// tenant-owned rows in schemas + agents + agent_relations. Forked
// data has no FK back — catalog updates never touch existing forks (§2.2).
//
// This is a pure domain entity (no GORM tags). Definition is stored as
// jsonb at rest and round-trips through Scan/Value as a JSON blob.
type SchemaTemplate struct {
	ID          string
	Name        string // stable catalog key, unique (e.g. "customer-support-basic")
	Display     string
	Description string
	Category    SchemaTemplateCategory
	Icon        string
	Version     string
	Definition  SchemaTemplateDefinition
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SchemaTemplateDefinition is the full template graph: agents, relations, and
// an entry-agent reference. Agent references inside Relations use logical
// names (the `Name` field on SchemaTemplateAgent) — the fork service resolves
// them to freshly-created UUIDs at fork time.
type SchemaTemplateDefinition struct {
	EntryAgentName string                    `json:"entry_agent_name" yaml:"entry_agent_name"`
	Agents         []SchemaTemplateAgent     `json:"agents"           yaml:"agents"`
	Relations      []SchemaTemplateRelation  `json:"relations"        yaml:"relations"`
}

// SchemaTemplateAgent describes one agent in the template graph.
// Capabilities are attached to the agent after creation by the fork service.
type SchemaTemplateAgent struct {
	Name         string                         `json:"name"          yaml:"name"`
	SystemPrompt string                         `json:"system_prompt" yaml:"system_prompt"`
	Model        string                         `json:"model"         yaml:"model"` // optional — empty falls back to default model at fork time
	Capabilities []SchemaTemplateCapability     `json:"capabilities"  yaml:"capabilities"`
}

// SchemaTemplateCapability describes one capability to attach to an agent
// after it is created. Mirrors capabilities.type + capabilities.config.
type SchemaTemplateCapability struct {
	Type   string                 `json:"type"             yaml:"type"`
	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
}

// SchemaTemplateRelation is a single source → target delegation edge.
// Source and Target are logical agent names that must match an entry in
// SchemaTemplateDefinition.Agents.
type SchemaTemplateRelation struct {
	Source string `json:"source" yaml:"source"`
	Target string `json:"target" yaml:"target"`
}

