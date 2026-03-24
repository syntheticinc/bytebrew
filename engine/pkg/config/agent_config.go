package config

// AgentDefinition describes a single agent in the universal engine.
// These definitions are stored in the database (source of truth) and can be
// imported/exported via YAML for portability.
type AgentDefinition struct {
	Name             string              `mapstructure:"name"`
	Model            string              `mapstructure:"model"`
	SystemPrompt     string              `mapstructure:"system_prompt"`
	SystemPromptFile string              `mapstructure:"system_prompt_file"`
	Tools            AgentToolsConfig    `mapstructure:"tools"`
	Kit              string              `mapstructure:"kit"`
	Knowledge        string              `mapstructure:"knowledge"`
	ConfirmBefore    []string            `mapstructure:"confirm_before"`
	CanSpawn         []string            `mapstructure:"can_spawn"`
	Escalation       *EscalationConfig   `mapstructure:"escalation"`
	Lifecycle        string              `mapstructure:"lifecycle"`
	ToolExecution    string              `mapstructure:"tool_execution"`
	MaxSteps         int                 `mapstructure:"max_steps"`
	MaxContextSize   int                 `mapstructure:"max_context_size"`
	Flow             *FlowConfig         `mapstructure:"flow"`
	Triggers         []TriggerDefinition `mapstructure:"triggers"`
}

// FlowConfig describes optional workflow steps for an agent.
type FlowConfig struct {
	Steps []string `mapstructure:"steps"`
}

// AgentToolsConfig describes which tools an agent has access to.
type AgentToolsConfig struct {
	Builtin    []string                   `mapstructure:"builtin"`
	MCPServers map[string]MCPServerConfig `mapstructure:"mcp_servers"`
	Custom     []CustomToolConfig         `mapstructure:"custom"`
}

// MCPServerConfig describes a Model Context Protocol server connection.
type MCPServerConfig struct {
	Type           string            `mapstructure:"type"` // "stdio" | "sse"
	Command        string            `mapstructure:"command"`
	Args           []string          `mapstructure:"args"`
	URL            string            `mapstructure:"url"`
	Env            map[string]string `mapstructure:"env"`
	ForwardHeaders []string          `mapstructure:"forward_headers"` // HTTP headers to forward from chat request to MCP server
}

// CustomToolConfig describes a custom HTTP-based tool.
type CustomToolConfig struct {
	Name        string            `mapstructure:"name"`
	Description string            `mapstructure:"description"`
	Endpoint    string            `mapstructure:"endpoint"`
	Method      string            `mapstructure:"method"`
	Headers     map[string]string `mapstructure:"headers"`
	Params      []CustomToolParam `mapstructure:"params"`
	Auth        *ToolAuthConfig   `mapstructure:"auth"`
}

// ToolAuthConfig describes authentication for a custom tool.
type ToolAuthConfig struct {
	Type     string `mapstructure:"type"`      // "bearer"
	TokenEnv string `mapstructure:"token_env"` // env var name holding the token
}

// CustomToolParam describes a parameter for a custom tool.
type CustomToolParam struct {
	Name        string `mapstructure:"name"`
	Type        string `mapstructure:"type"`
	Description string `mapstructure:"description"`
	Required    bool   `mapstructure:"required"`
}

// EscalationConfig describes when and how an agent should escalate.
type EscalationConfig struct {
	Triggers []string `mapstructure:"triggers"`
	Action   string   `mapstructure:"action"` // "notify" | "webhook"
	Webhook  string   `mapstructure:"webhook"`
}

// TriggerDefinition describes an automated trigger for an agent.
type TriggerDefinition struct {
	Type        string `mapstructure:"type"` // "manual" | "schedule" | "webhook" | "file_watch"
	Title       string `mapstructure:"title"`
	Schedule    string `mapstructure:"schedule"`    // cron expression (for type=schedule)
	Path        string `mapstructure:"path"`        // file/dir path (for type=file_watch) or webhook path
	Description string `mapstructure:"description"`
	Agent       string `mapstructure:"agent"` // agent name to invoke
}

// ModelDefinition describes an LLM provider configuration.
type ModelDefinition struct {
	Name    string `mapstructure:"name"`
	Type    string `mapstructure:"type"` // "openai" | "anthropic" | "ollama"
	BaseURL string `mapstructure:"base_url"`
	Model   string `mapstructure:"model"`
	APIKey  string `mapstructure:"api_key"`
}

// ModelsConfig holds all model provider definitions.
type ModelsConfig struct {
	Providers []ModelDefinition `mapstructure:"providers"`
	BYOK      BYOKConfig        `mapstructure:"byok"`
}

// BYOKConfig controls Bring Your Own Key settings.
type BYOKConfig struct {
	Enabled          bool     `mapstructure:"enabled"`
	AllowedProviders []string `mapstructure:"allowed_providers"`
}

// RuntimeBridgeConfig holds bridge relay settings for the universal engine.
// Named differently from legacy BridgeConfig to avoid conflicts during migration.
type RuntimeBridgeConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	URL     string `mapstructure:"url"`
}
