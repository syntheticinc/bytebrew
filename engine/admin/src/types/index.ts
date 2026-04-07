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
  temperature?: number;
  top_p?: number;
  max_tokens?: number;
  stop_sequences?: string[];
  confirm_before: string[];
  mcp_servers: string[];
  escalation?: AgentEscalation;
}

export interface AgentEscalation {
  action: 'transfer_to_user' | 'notify';
  webhook_url?: string;
  triggers: EscalationTrigger[];
}

export interface EscalationTrigger {
  condition: EscalationConditionType;
  threshold?: number;
  pattern?: string;
  prompt?: string;
}

export type EscalationConditionType =
  | 'confidence_below'
  | 'topic_matches'
  | 'user_sentiment'
  | 'max_turns_exceeded'
  | 'tool_failed'
  | 'custom';

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
  category?: MCPCatalogCategory;
  auth_types?: WebhookAuthType[];
}

export type MCPCatalogCategory = 'search' | 'data' | 'communication' | 'dev_tools' | 'productivity' | 'generic';

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

export const CAPABILITY_META: Record<CapabilityType, { label: string; icon: string; description: string }> = {
  memory:        { label: 'Memory',           icon: 'brain',          description: 'Per-schema cross-session persistence' },
  knowledge:     { label: 'Knowledge',        icon: 'book-open',      description: 'RAG sources (PDF, DOCX, URL, text)' },
  guardrail:     { label: 'Output Guardrail', icon: 'shield-check',   description: 'JSON Schema, LLM judge, webhook validation' },
  output_schema: { label: 'Output Schema',    icon: 'file-json',      description: 'Structured JSON output via response_format' },
  escalation:    { label: 'Escalation',       icon: 'arrow-up-right', description: 'transfer_to_user, notify, webhook' },
  recovery:      { label: 'Recovery Policy',  icon: 'refresh-cw',     description: 'Retry rules per failure type (per-session scope)' },
  policies:      { label: 'Agent Policies',   icon: 'settings',       description: 'When [condition] → Do [action] rules' },
};

// ============================================================================
// V2: Inspect types
// ============================================================================

export type InspectStepKind =
  | 'reasoning'
  | 'tool_call'
  | 'memory_recall'
  | 'knowledge_search'
  | 'guardrail_check'
  | 'final_answer'
  | 'error'
  | 'escalation'
  | 'task_dispatch'
  | 'task_timeout';

export type SessionStatus = 'running' | 'completed' | 'failed' | 'blocked' | 'timeout';

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
  status: SessionStatus;
  steps: InspectStep[];
  total_duration_ms: number;
  total_tokens: number;
  created_at: string;
}

export interface SessionSummary {
  session_id: string;
  entry_agent: string;
  status: SessionStatus;
  duration_ms: number;
  total_tokens: number;
  created_at: string;
}

export interface PaginatedSessions {
  sessions: SessionSummary[];
  total: number;
  page: number;
  per_page: number;
}

// ============================================================================
// V2: Widget types
// ============================================================================

export type WidgetPosition = 'bottom-right' | 'bottom-left';
export type WidgetSize = 'compact' | 'standard' | 'full';

export interface WidgetConfig {
  id: string;
  name: string;
  schema: string;
  status: 'active' | 'disabled';
  primary_color: string;
  position: WidgetPosition;
  size: WidgetSize;
  welcome_message: string;
  placeholder_text: string;
  avatar_url: string;
  domain_whitelist: string;
  created_at?: string;
}

export type CreateWidgetRequest = Omit<WidgetConfig, 'id' | 'created_at'>;

// ============================================================================
// V2: Usage / Quota types
// ============================================================================

export interface UsageMetric {
  name: string;
  label: string;
  used: number;
  limit: number;
  unit: string;
}

export interface UsageData {
  plan: string;
  billing_cycle_start: string;
  billing_cycle_end: string;
  metrics: UsageMetric[];
  stripe_portal_url?: string;
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

// ============================================================================
// V2: Webhook & Auth types
// ============================================================================

export type WebhookAuthType = 'none' | 'api_key' | 'forward_headers' | 'oauth2';

export interface WebhookConfig {
  url: string;
  auth_type: WebhookAuthType;
  token?: string;
  client_id?: string;
  client_secret?: string;
  timeout_ms?: number;
}

// ============================================================================
// V2: Policy types
// ============================================================================

export type PolicyConditionType =
  | 'before_tool_call'
  | 'after_tool_call'
  | 'tool_matches'
  | 'time_range'
  | 'error_occurred';

export type PolicyActionType =
  | 'block'
  | 'log_to_webhook'
  | 'notify'
  | 'inject_header'
  | 'write_audit';

export interface PolicyRule {
  condition: PolicyConditionType;
  action: PolicyActionType;
  tool_pattern?: string;
  time_start?: string;
  time_end?: string;
  webhook_url?: string;
  webhook_auth?: WebhookAuthType;
  header_name?: string;
  header_value?: string;
}

// ============================================================================
// V2: Knowledge file types
// ============================================================================

export type KnowledgeFileStatus = 'uploading' | 'indexing' | 'ready' | 'error';

export interface KnowledgeFile {
  name: string;
  type: string;
  size: string;
  uploaded_at: string;
  status: KnowledgeFileStatus;
}
