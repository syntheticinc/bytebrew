import type {
  HealthResponse,
  Model,
  MCPServer,
  Trigger,
  PaginatedTaskResponse,
  APIToken,
  Setting,
  AuditEntry,
  PaginatedResponse,
  WellKnownMCP,
} from '../types';

export const MOCK_HEALTH: HealthResponse = {
  status: 'ok',
  version: '2.0.0-prototype',
  uptime: '3h 42m',
  agents_count: 6,
};

export const MOCK_MODELS_LIST: Model[] = [
  { id: 1, name: 'claude-haiku-3', type: 'openai_compatible', model_name: 'claude-3-haiku-20240307', has_api_key: true, created_at: '2026-03-01T00:00:00Z' },
  { id: 2, name: 'claude-sonnet-3.7', type: 'openai_compatible', model_name: 'claude-3-5-sonnet-20241022', has_api_key: true, created_at: '2026-03-01T00:00:00Z' },
  { id: 3, name: 'claude-opus-4', type: 'openai_compatible', model_name: 'claude-opus-4-20260414', has_api_key: true, created_at: '2026-03-15T00:00:00Z' },
  { id: 4, name: 'gpt-4o', type: 'openai_compatible', model_name: 'gpt-4o', has_api_key: true, base_url: 'https://api.openai.com/v1', created_at: '2026-03-10T00:00:00Z' },
];

export const MOCK_MCP_SERVERS: MCPServer[] = [
  { id: 1, name: 'google-sheets', type: 'stdio', command: 'npx', args: ['-y', '@anthropic/mcp-google-sheets'], is_well_known: true, status: { status: 'connected', tools_count: 12, connected_at: '2026-04-05T10:00:00Z' }, agents: ['support-agent'] },
  { id: 2, name: 'web-search', type: 'stdio', command: 'npx', args: ['-y', '@anthropic/mcp-web-search'], is_well_known: true, status: { status: 'connected', tools_count: 3, connected_at: '2026-04-05T10:00:00Z' }, agents: ['classifier', 'support-agent'] },
  { id: 3, name: 'slack-notifications', type: 'http', url: 'https://mcp.example.com/slack', is_well_known: false, status: { status: 'disconnected', status_message: 'Auth expired', tools_count: 5 }, agents: [] },
];

export const MOCK_WELL_KNOWN: WellKnownMCP[] = [
  { name: 'google-sheets', display: 'Google Sheets', command: 'npx', args: ['-y', '@anthropic/mcp-google-sheets'], env: ['GOOGLE_API_KEY'] },
  { name: 'web-search', display: 'Web Search', command: 'npx', args: ['-y', '@anthropic/mcp-web-search'], env: ['TAVILY_API_KEY'] },
  { name: 'github', display: 'GitHub', command: 'npx', args: ['-y', '@anthropic/mcp-github'], env: ['GITHUB_TOKEN'] },
];

export const MOCK_TRIGGERS: Trigger[] = [
  { id: 1, type: 'webhook', title: 'user-message', agent_id: 1, agent_name: 'classifier', webhook_path: '/webhook/support', enabled: true, created_at: '2026-03-20T00:00:00Z' },
  { id: 2, type: 'cron', title: 'daily-report', agent_id: 2, agent_name: 'support-agent', schedule: '0 9 * * *', description: 'Daily summary', enabled: true, created_at: '2026-03-25T00:00:00Z' },
  { id: 3, type: 'webhook', title: 'escalation-hook', agent_id: 3, agent_name: 'escalation', webhook_path: '/webhook/escalate', enabled: false, created_at: '2026-04-01T00:00:00Z' },
];

export const MOCK_TASKS_PAGINATED: PaginatedTaskResponse = {
  data: [
    { id: 1, title: 'Process support ticket #4521', agent_name: 'support-agent', status: 'completed', source: 'webhook', created_at: '2026-04-05T14:30:00Z' },
    { id: 2, title: 'Analyze lead score batch', agent_name: 'lead-scorer', status: 'running', source: 'cron', created_at: '2026-04-05T14:00:00Z' },
    { id: 3, title: 'Code review PR #89', agent_name: 'review-agent', status: 'failed', source: 'webhook', created_at: '2026-04-05T13:15:00Z' },
    { id: 4, title: 'Outreach to prospect', agent_name: 'outreach-agent', status: 'completed', source: 'cron', created_at: '2026-04-05T12:00:00Z' },
  ],
  total: 4,
  page: 1,
  per_page: 20,
  total_pages: 1,
};

export const MOCK_TOKENS: APIToken[] = [
  { id: 1, name: 'Production API', scopes_mask: 7, created_at: '2026-03-01T00:00:00Z', last_used_at: '2026-04-05T14:00:00Z' },
  { id: 2, name: 'CI/CD Pipeline', scopes_mask: 3, created_at: '2026-03-15T00:00:00Z', last_used_at: '2026-04-04T22:00:00Z' },
  { id: 3, name: 'Monitoring', scopes_mask: 1, created_at: '2026-04-01T00:00:00Z' },
];

export const MOCK_SETTINGS: Setting[] = [
  { key: 'default_model', value: 'claude-sonnet-3.7', updated_at: '2026-04-01T00:00:00Z' },
  { key: 'max_concurrent_sessions', value: '10', updated_at: '2026-03-20T00:00:00Z' },
  { key: 'session_timeout_minutes', value: '30', updated_at: '2026-03-20T00:00:00Z' },
  { key: 'enable_audit_log', value: 'true', updated_at: '2026-03-25T00:00:00Z' },
  { key: 'prototype_mode_enabled', value: 'true', updated_at: '2026-04-05T00:00:00Z' },
];

export const MOCK_AUDIT_LOGS: PaginatedResponse<AuditEntry> = {
  data: [
    { id: 1, timestamp: '2026-04-05T14:30:00Z', actor_type: 'user', actor_id: 'admin', action: 'agent.create', resource: 'support-agent', details: 'Created agent with model claude-sonnet-3.7' },
    { id: 2, timestamp: '2026-04-05T14:25:00Z', actor_type: 'user', actor_id: 'admin', action: 'model.create', resource: 'claude-opus-4', details: 'Added new model' },
    { id: 3, timestamp: '2026-04-05T14:20:00Z', actor_type: 'system', actor_id: 'engine', action: 'trigger.fired', resource: 'daily-report', details: 'Cron trigger executed' },
    { id: 4, timestamp: '2026-04-05T14:15:00Z', actor_type: 'agent', actor_id: 'support-agent', action: 'tool.called', resource: 'search_knowledge', details: 'Query: billing FAQ' },
    { id: 5, timestamp: '2026-04-05T14:10:00Z', actor_type: 'user', actor_id: 'admin', action: 'mcp.connect', resource: 'google-sheets', details: 'MCP server connected' },
  ],
  total: 5,
  page: 1,
  per_page: 20,
  total_pages: 1,
};

export const MOCK_CONFIG_YAML = `# ByteBrew Engine Configuration
server:
  host: 0.0.0.0
  port: 8443

database:
  url: postgres://bytebrew:password@localhost:5432/bytebrew

auth:
  admin_username: admin
  jwt_secret: "***"

agents:
  max_steps: 50
  max_context_size: 16000
  default_model: claude-sonnet-3.7
`;
