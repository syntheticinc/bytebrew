package admin

import "context"

// NOTE: Admin tools operate without tenant scoping (CE = single-tenant by design).
// Cloud deployments MUST NOT expose admin tools to non-admin agents.

// AdminToolDependencies holds repositories and callbacks for admin tools.
// Captured in closure at registration time via RegisterAdminTools.
type AdminToolDependencies struct {
	AgentRepo         AgentRepository
	SchemaRepo        SchemaRepository
	MCPServerRepo     MCPServerRepository
	ModelRepo         ModelRepository
	AgentRelationRepo AgentRelationRepository
	SessionRepo       SessionRepository
	CapabilityRepo    CapabilityRepository
	Reloader          func() // AgentRegistry reload callback
}

// Consumer-side interfaces (defined here, implemented by GORM repo adapters):

// AgentRepository provides agent CRUD for admin tools.
type AgentRepository interface {
	List(ctx context.Context) ([]AgentRecord, error)
	GetByName(ctx context.Context, name string) (*AgentRecord, error)
	Create(ctx context.Context, record *AgentRecord) error
	Update(ctx context.Context, name string, record *AgentRecord) error
	Delete(ctx context.Context, name string) error
}

// SchemaRepository provides schema CRUD for admin tools.
//
// V2: schema membership is derived from `agent_relations` (see
// docs/architecture/agent-first-runtime.md §2.1) — there is no separate
// AddAgent / RemoveAgent surface. Adding an agent to a schema is done by
// creating a delegation relation through AgentRelationRepository.
type SchemaRepository interface {
	List(ctx context.Context) ([]SchemaRecord, error)
	GetByID(ctx context.Context, id string) (*SchemaRecord, error)
	Create(ctx context.Context, record *SchemaRecord) error
	Update(ctx context.Context, id string, record *SchemaRecord) error
	Delete(ctx context.Context, id string) error
}

// MCPServerRepository provides MCP server CRUD for admin tools.
type MCPServerRepository interface {
	List(ctx context.Context) ([]MCPServerRecord, error)
	GetByID(ctx context.Context, id string) (*MCPServerRecord, error)
	Create(ctx context.Context, record *MCPServerRecord) error
	Update(ctx context.Context, id string, record *MCPServerRecord) error
	Delete(ctx context.Context, id string) error
}

// ModelRepository provides model CRUD for admin tools.
type ModelRepository interface {
	List(ctx context.Context) ([]ModelRecord, error)
	GetByID(ctx context.Context, id string) (*ModelRecord, error)
	Create(ctx context.Context, record *ModelRecord) error
	Update(ctx context.Context, id string, record *ModelRecord) error
	Delete(ctx context.Context, id string) error
}

// AgentRelationRepository provides agent-relation CRUD for admin tools.
type AgentRelationRepository interface {
	List(ctx context.Context, schemaID string) ([]AgentRelationRecord, error)
	Create(ctx context.Context, record *AgentRelationRecord) error
	Delete(ctx context.Context, id string) error
}

// SessionRepository provides read-only session access for admin tools.
type SessionRepository interface {
	List(ctx context.Context) ([]SessionRecord, error)
	GetByID(ctx context.Context, id string) (*SessionRecord, error)
}

// CapabilityRepository provides capability CRUD for admin tools.
type CapabilityRepository interface {
	ListByAgent(ctx context.Context, agentName string) ([]CapabilityRecord, error)
	Create(ctx context.Context, record *CapabilityRecord) error
	Update(ctx context.Context, id string, record *CapabilityRecord) error
	Delete(ctx context.Context, id string) error
}

// AgentRecord mirrors configrepo.AgentRecord fields needed by admin tools.
type AgentRecord struct {
	Name          string
	SystemPrompt  string
	ModelName     string
	Lifecycle     string
	ToolExecution string
	MaxSteps      int
	BuiltinTools  []string
	MCPServers    []string
	CanSpawn      []string
	IsSystem      bool
}

// SchemaRecord represents a schema for admin tools.
type SchemaRecord struct {
	ID          string
	Name        string
	Description string
	AgentNames  []string
}

// MCPServerRecord represents an MCP server for admin tools.
type MCPServerRecord struct {
	ID      string
	Name    string
	Type    string
	Command string
	URL     string
	Args    []string
	EnvVars map[string]string
}

// ModelRecord represents an LLM model configuration for admin tools.
type ModelRecord struct {
	ID        string
	Name      string
	Type      string
	BaseURL   string
	ModelName string
	APIKey    string // write-only, masked on read
}

// AgentRelationRecord represents a delegation relation between agents in a
// schema. V2 has a single implicit DELEGATION type — no per-row Type field
// (see docs/architecture/agent-first-runtime.md §3.1).
type AgentRelationRecord struct {
	ID        string
	SchemaID  string
	FromAgent string
	ToAgent   string
	Label     string
}

// SessionRecord represents a session for admin tools.
// Q.5: AgentName dropped — session belongs to schema.
type SessionRecord struct {
	ID        string
	UserID    string
	StartedAt string
	Status    string
}

// CapabilityRecord represents an agent capability for admin tools.
type CapabilityRecord struct {
	ID        string
	AgentName string
	Type      string
	Config    map[string]interface{}
	Enabled   bool
}
