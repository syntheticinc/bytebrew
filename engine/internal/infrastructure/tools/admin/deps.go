package admin

import "context"

// NOTE: Admin tools operate without tenant scoping (CE = single-tenant by design).
// Cloud deployments MUST NOT expose admin tools to non-admin agents.

// AdminToolDependencies holds repositories and callbacks for admin tools.
// Captured in closure at registration time via RegisterAdminTools.
type AdminToolDependencies struct {
	AgentRepo      AgentRepository
	SchemaRepo     SchemaRepository
	TriggerRepo    TriggerRepository
	MCPServerRepo  MCPServerRepository
	ModelRepo      ModelRepository
	EdgeRepo       EdgeRepository
	SessionRepo    SessionRepository
	CapabilityRepo CapabilityRepository
	Reloader       func() // AgentRegistry reload callback
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
type SchemaRepository interface {
	List(ctx context.Context) ([]SchemaRecord, error)
	GetByID(ctx context.Context, id string) (*SchemaRecord, error)
	Create(ctx context.Context, record *SchemaRecord) error
	Update(ctx context.Context, id string, record *SchemaRecord) error
	Delete(ctx context.Context, id string) error
	AddAgent(ctx context.Context, schemaID string, agentName string) error
	RemoveAgent(ctx context.Context, schemaID string, agentName string) error
}

// TriggerRepository provides trigger CRUD for admin tools.
type TriggerRepository interface {
	List(ctx context.Context) ([]TriggerRecord, error)
	GetByID(ctx context.Context, id string) (*TriggerRecord, error)
	Create(ctx context.Context, record *TriggerRecord) error
	Update(ctx context.Context, id string, record *TriggerRecord) error
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

// EdgeRepository provides edge CRUD for admin tools.
type EdgeRepository interface {
	List(ctx context.Context, schemaID string) ([]EdgeRecord, error)
	Create(ctx context.Context, record *EdgeRecord) error
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

// AgentRecord mirrors config_repo.AgentRecord fields needed by admin tools.
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

// TriggerRecord represents a trigger for admin tools.
type TriggerRecord struct {
	ID          string
	Type        string
	Title       string
	AgentName   string
	AgentID     string
	SchemaID    *string
	Schedule    string
	WebhookPath string
	Description string
	Enabled     bool
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

// EdgeRecord represents an edge between agents in a schema.
type EdgeRecord struct {
	ID        string
	SchemaID  string
	FromAgent string
	ToAgent   string
	Type      string // flow, transfer, loop, spawn
	Label     string
}

// SessionRecord represents a session for admin tools.
type SessionRecord struct {
	ID        string
	AgentName string
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
