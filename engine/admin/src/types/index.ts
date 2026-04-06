// ============================================================================
// Agent types
// ============================================================================

export interface AgentInfo {
  name: string;
  description?: string;
  tools_count: number;
  kit?: string;
  has_knowledge: boolean;
}

export interface AgentDetail extends AgentInfo {
  model_id?: number;
  system_prompt: string;
  tools: string[];
  can_spawn: string[];
  lifecycle: 'persistent' | 'spawn';
  tool_execution: 'sequential' | 'parallel';
  max_steps: number;
  max_context_size: number;
  max_turn_duration: number;
  confirm_before: string[];
  mcp_servers: string[];
  escalation?: AgentEscalation;
}

export interface AgentEscalation {
  action: 'transfer_to_human' | 'notify';
  webhook_url?: string;
  triggers: string[];
}

export interface CreateAgentRequest {
  name: string;
  model_id?: number;
  system_prompt: string;
  kit?: string;
  lifecycle?: string;
  tool_execution?: string;
  max_steps?: number;
  max_context_size?: number;
  max_turn_duration?: number;
  confirm_before?: string[];
  tools?: string[];
  can_spawn?: string[];
  mcp_servers?: string[];
  escalation?: AgentEscalation;
}

// ============================================================================
// Model types
// ============================================================================

export interface Model {
  id: number;
  name: string;
  type: string;
  base_url?: string;
  model_name: string;
  has_api_key: boolean;
  api_version?: string;
  created_at: string;
}

export interface CreateModelRequest {
  name: string;
  type: string;
  base_url?: string;
  model_name: string;
  api_key?: string;
  api_version?: string;
}

// ============================================================================
// MCP types
// ============================================================================

export interface MCPServer {
  id: number;
  name: string;
  type: 'stdio' | 'http' | 'sse' | 'streamable-http';
  command?: string;
  args?: string[];
  url?: string;
  env_vars?: Record<string, string>;
  is_well_known: boolean;
  status?: MCPServerStatus;
  agents: string[];
}

export interface MCPServerStatus {
  status: 'connected' | 'error' | 'connecting' | 'disconnected';
  status_message?: string;
  tools_count: number;
  connected_at?: string;
}

export interface WellKnownMCP {
  name: string;
  display: string;
  command: string;
  args: string[];
  env: string[];
}

export interface CreateMCPServerRequest {
  name: string;
  type: string;
  command?: string;
  args?: string[];
  url?: string;
  env_vars?: Record<string, string>;
}

// ============================================================================
// Task types
// ============================================================================

export interface TaskResponse {
  id: number;
  title: string;
  agent_name: string;
  status: string;
  source: string;
  created_at: string;
}

export interface TaskDetailResponse extends TaskResponse {
  description?: string;
  mode: string;
  result?: string;
  error?: string;
  started_at?: string;
  completed_at?: string;
}

export interface PaginatedTaskResponse {
  data: TaskResponse[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

// ============================================================================
// Trigger types
// ============================================================================

export interface Trigger {
  id: number;
  type: 'cron' | 'webhook';
  title: string;
  agent_id: number;
  agent_name?: string;
  schedule?: string;
  webhook_path?: string;
  description?: string;
  enabled: boolean;
  on_complete_url?: string;
  on_complete_headers?: Record<string, string>;
  last_fired_at?: string;
  created_at: string;
}

export interface CreateTriggerRequest {
  type: string;
  title: string;
  agent_id?: number;
  agent_name?: string;
  schedule?: string;
  webhook_path?: string;
  description?: string;
  enabled?: boolean;
  on_complete_url?: string;
  on_complete_headers?: Record<string, string>;
}

// ============================================================================
// Token types
// ============================================================================

export interface APIToken {
  id: number;
  name: string;
  scopes_mask: number;
  created_at: string;
  last_used_at?: string;
}

export interface CreateTokenRequest {
  name: string;
  scopes_mask: number;
}

export interface CreateTokenResponse {
  id: number;
  name: string;
  token: string;
}

// ============================================================================
// Health types
// ============================================================================

export interface HealthResponse {
  status: string;
  version: string;
  uptime: string;
  agents_count: number;
  update_available?: string;
}

// ============================================================================
// Settings types
// ============================================================================

export interface Setting {
  key: string;
  value: string;
  updated_at: string;
}

// ============================================================================
// Audit types
// ============================================================================

export interface AuditEntry {
  id: number;
  timestamp: string;
  actor_type: string;
  actor_id: string;
  action: string;
  resource: string;
  details: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

// ============================================================================
// Auth types
// ============================================================================

export interface LoginResponse {
  token: string;
  expires_at: string;
}

// ============================================================================
// Tool metadata types
// ============================================================================

export type SecurityZone = 'safe' | 'caution' | 'dangerous';

export interface ToolMetadata {
  name: string;
  description: string;
  security_zone: SecurityZone;
  risk_warning?: string;
  hint?: string;
  companion?: string;
}

// ============================================================================
// V2: Schema types
// ============================================================================

export interface Schema {
  id: number;
  name: string;
  description?: string;
  agents_count: number;
  created_at: string;
}

// ============================================================================
// V2: Capability types
// ============================================================================

export type CapabilityType =
  | 'memory'
  | 'knowledge'
  | 'guardrail'
  | 'output_schema'
  | 'escalation'
  | 'recovery'
  | 'policies';

export interface CapabilityConfig {
  type: CapabilityType;
  enabled: boolean;
  config: Record<string, unknown>;
}

export const CAPABILITY_META: Record<CapabilityType, { label: string; abbr: string; description: string }> = {
  memory:        { label: 'Memory',           abbr: 'MEM', description: 'Cross-session persistence, per-user isolation' },
  knowledge:     { label: 'Knowledge',        abbr: 'KB',  description: 'RAG sources (PDF, URL, text)' },
  guardrail:     { label: 'Output Guardrail', abbr: 'GRD', description: 'JSON Schema, LLM check, webhook validation' },
  output_schema: { label: 'Output Schema',    abbr: 'SCH', description: 'Structured JSON output format' },
  escalation:    { label: 'Escalation',       abbr: 'ESC', description: 'transfer_to_human, notify, webhook' },
  recovery:      { label: 'Recovery Policy',  abbr: 'REC', description: 'Retry rules per failure type' },
  policies:      { label: 'Agent Policies',   abbr: 'POL', description: 'When [condition] → Do [action] rules' },
};

// ============================================================================
// V2: Inspect types
// ============================================================================

export type InspectStepKind = 'reasoning' | 'tool_call' | 'memory_recall' | 'final_answer';

export interface InspectStep {
  id: number;
  kind: InspectStepKind;
  label: string;
  input?: string;
  output?: string;
  duration_ms: number;
  tokens?: number;
}

export interface SessionTrace {
  session_id: string;
  agent_name: string;
  status: 'running' | 'completed' | 'failed';
  steps: InspectStep[];
  total_duration_ms: number;
  total_tokens: number;
  created_at: string;
}

// ============================================================================
// Model Registry types
// ============================================================================

export interface ModelRegistryEntry {
  id: string;
  display_name: string;
  provider: string;
  tier: number; // 1 = Orchestrator, 2 = Sub-agent, 3 = Utility
  context_window: number;
  supports_tools: boolean;
  pricing_input: number;
  pricing_output: number;
  description: string;
  recommended_for: string[];
}

export interface RegistryProviderInfo {
  id: string;
  display_name: string;
  auth_type: string;
  website: string;
}
