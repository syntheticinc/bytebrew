// ============================================================================
// Agent types
// ============================================================================

export interface AgentInfo {
  name: string;
  description?: string;
  tools_count: number;
  kit?: string;
  has_knowledge: boolean;
  is_system?: boolean;
}

export interface AgentDetail extends AgentInfo {
  model_id?: string;
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
  model_id?: string;
  system_prompt: string;
  kit?: string;
  lifecycle?: string;
  tool_execution?: string;
  max_steps?: number;
  max_context_size?: number;
  max_turn_duration?: number;
  temperature?: number;
  top_p?: number;
  max_tokens?: number;
  stop_sequences?: string[];
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
  id: string;
  name: string;
  type: string;
  base_url?: string;
  model_name: string;
  has_api_key: boolean;
  api_version?: string;
  embedding_dim?: number; // >0 for embedding models
  created_at: string;
}

export interface CreateModelRequest {
  name: string;
  type: string;
  base_url?: string;
  model_name: string;
  api_key?: string;
  api_version?: string;
  embedding_dim?: number; // required when type=embedding
}

// ============================================================================
// MCP types
// ============================================================================

export interface MCPServer {
  id: string;
  name: string;
  type: 'stdio' | 'http' | 'sse' | 'streamable-http';
  command?: string;
  args?: string[];
  url?: string;
  env_vars?: Record<string, string>;
  forward_headers?: string[];
  is_well_known: boolean;
  auth_type?: string;
  auth_key_env?: string;
  auth_token_env?: string;
  auth_client_id?: string;
  status?: MCPServerStatus;
  agents: string[];
}

export interface MCPServerStatus {
  status: 'connected' | 'error' | 'connecting' | 'disconnected';
  status_message?: string;
  tools_count: number;
  connected_at?: string;
}

export type MCPCatalogCategory = 'search' | 'data' | 'communication' | 'dev-tools' | 'productivity' | 'payments' | 'generic';

export interface MCPCatalogEnvVar {
  name: string;
  description?: string;
  required: boolean;
  secret?: boolean;
}

export interface MCPCatalogTool {
  name: string;
  description: string;
}

export interface MCPCatalogPackage {
  type: 'stdio' | 'remote' | 'docker';
  transport?: string;
  command?: string;
  args?: string[];
  image?: string;
  url_template?: string;
  env_vars?: MCPCatalogEnvVar[];
}

export interface MCPCatalogEntry {
  name: string;
  display: string;
  description?: string;
  category?: MCPCatalogCategory;
  verified?: boolean;
  packages: MCPCatalogPackage[];
  provided_tools?: MCPCatalogTool[];
}

export interface MCPCatalogResponse {
  version: string;
  servers: MCPCatalogEntry[];
}

export interface CreateMCPServerRequest {
  name: string;
  type: string;
  command?: string;
  args?: string[];
  url?: string;
  env_vars?: Record<string, string>;
  forward_headers?: string[];
  auth_type?: string;
  auth_key_env?: string;
  auth_token_env?: string;
  auth_client_id?: string;
}

// ============================================================================
// Trigger types
// ============================================================================

export interface Trigger {
  id: string;
  type: 'cron' | 'webhook' | 'chat';
  title: string;
  agent_id: string;
  agent_name?: string;
  schema_id?: string;
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
  agent_id?: string;
  agent_name?: string;
  schema_id?: string;
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
  id: string;
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
  id: string;
  name: string;
  token: string;
}

// ============================================================================
// Circuit Breaker types
// ============================================================================

export interface CircuitBreakerState {
  name: string;
  state: 'closed' | 'open' | 'half_open';
  failure_count: number;
  last_failure?: string | null;
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
  id: string;
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
  id: string;
  name: string;
  description?: string;
  agents?: string[];
  agents_count: number;
  is_system?: boolean;
  created_at: string;
}

// ============================================================================
// V2: Capability types
// ============================================================================

export type CapabilityType =
  | 'memory'
  | 'knowledge'
  | 'guardrail'
  | 'escalation'
  | 'recovery'
  | 'policies';

export interface CapabilityConfig {
  id?: string;
  agent_name?: string;
  type: CapabilityType;
  enabled: boolean;
  config: Record<string, unknown>;
}

// ============================================================================
// V2: Capability CRUD types
// ============================================================================

export interface Capability {
  id: string;
  agent_name: string;
  type: string;
  config: Record<string, unknown>;
  enabled: boolean;
}

export interface CreateCapabilityRequest {
  type: string;
  config: Record<string, unknown>;
  enabled: boolean;
}

export interface UpdateCapabilityRequest {
  config?: Record<string, unknown>;
  enabled?: boolean;
}

// ============================================================================
// V2: Memory types
// ============================================================================

export interface MemoryEntry {
  id: string;
  schema_id: string;
  user_id?: string;
  content: string;
  metadata?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export const CAPABILITY_META: Record<CapabilityType, { label: string; icon: string; description: string }> = {
  memory:        { label: 'Memory',           icon: 'brain',          description: 'Per-schema cross-session persistence' },
  knowledge:     { label: 'Knowledge',        icon: 'book-open',      description: 'RAG sources (PDF, DOCX, TXT, MD, CSV)' },
  guardrail:     { label: 'Output Guardrail', icon: 'shield-check',   description: 'JSON Schema, LLM judge, webhook validation' },

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
  id: string;
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
  custom_headers?: Record<string, string>;
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
// V2: Knowledge Base types (many-to-many)
// ============================================================================

export interface KnowledgeBase {
  id: string;
  name: string;
  description?: string;
  embedding_model_id?: string;
  file_count: number;
  linked_agents: string[];
  created_at: string;
  updated_at: string;
}

export interface CreateKnowledgeBaseRequest {
  name: string;
  description?: string;
  embedding_model_id: string;
}

export type KnowledgeFileStatus = 'uploading' | 'indexing' | 'ready' | 'error';

export interface KnowledgeFile {
  id?: string;
  knowledge_base_id?: string;
  name: string;
  type: string;
  size: string;
  uploaded_at: string;
  status: KnowledgeFileStatus;
  error?: string;
  chunk_count?: number;
}

export interface KnowledgeStatus {
  agent_name: string;
  total_files: number;
  indexed_files: number;
  status: 'ready' | 'indexing' | 'empty';
}

// ============================================================================
// Session message types (for chat history restore)
// ============================================================================

/** @deprecated Use EventResponse instead */
export interface MessageResponse {
  id: string;
  role: 'user' | 'assistant' | 'tool' | 'system';
  content: string;
  tool_name?: string;
  created_at: string;
}

// EventResponse represents a runtime event from the session timeline.
export interface EventResponse {
  id: string;
  event_type: 'user_message' | 'assistant_message' | 'tool_call' | 'tool_result' | 'reasoning' | 'system';
  agent_id?: string;
  call_id?: string;
  payload: Record<string, unknown>;
  created_at: string;
}
