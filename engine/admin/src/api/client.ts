import type {
  AgentInfo,
  AgentDetail,
  CreateAgentRequest,
  Model,
  CreateModelRequest,
  MCPServer,
  MCPCatalogEntry,
  MCPCatalogResponse,
  CreateMCPServerRequest,
  TaskResponse,
  TaskDetailResponse,
  PaginatedTaskResponse,
  Trigger,
  CreateTriggerRequest,
  APIToken,
  CreateTokenRequest,
  CreateTokenResponse,
  HealthResponse,
  Setting,
  LoginResponse,
  ToolMetadata,
  AuditEntry,
  PaginatedResponse,
  ModelRegistryEntry,
  RegistryProviderInfo,
  Schema,
  PaginatedSessions,
  SessionSummary,
  SessionTrace,
  WidgetConfig,
  CreateWidgetRequest,
  UsageData,
  MemoryEntry,
  Capability,
  CreateCapabilityRequest,
  UpdateCapabilityRequest,
  KnowledgeBase,
  CreateKnowledgeBaseRequest,
  KnowledgeFile,
  KnowledgeStatus,
  CircuitBreakerState,
  MessageResponse,
  EventResponse,
} from '../types';
import {
  MOCK_HEALTH,
  MOCK_MODELS_LIST,
  MOCK_MCP_SERVERS,
  MOCK_CATALOG,
  MOCK_TRIGGERS,
  MOCK_TASKS_PAGINATED,
  MOCK_TOKENS,
  MOCK_SETTINGS,
  MOCK_AUDIT_LOGS,
  MOCK_CONFIG_YAML,
} from '../mocks/pages';
import { MOCK_AGENTS } from '../mocks/agents';
import { SCHEMA_NAMES } from '../mocks/canvas';
import { MOCK_SESSIONS_LIST, MOCK_TRACE, MOCK_TRACE_ERROR } from '../mocks/inspect';

const BASE_URL = '/api/v1';
const PROTOTYPE_KEY = 'bytebrew_prototype_mode';

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

class APIClient {
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('jwt');
  }

  private get isPrototype(): boolean {
    return localStorage.getItem(PROTOTYPE_KEY) === 'true';
  }

  private mock<T>(data: T): Promise<T> {
    return Promise.resolve(data);
  }

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('jwt', token);
  }

  clearToken() {
    this.token = null;
    localStorage.removeItem('jwt');
  }

  isAuthenticated(): boolean {
    return this.token !== null;
  }

  private async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const headers: Record<string, string> = { 'Content-Type': 'application/json' };
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const res = await fetch(`${BASE_URL}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });

    if (res.status === 401 && path !== '/auth/login') {
      this.clearToken();
      window.location.href = import.meta.env.BASE_URL + 'login';
      throw new Error('Unauthorized');
    }

    if (!res.ok) {
      const text = await res.text();
      let message = text;
      try {
        const json = JSON.parse(text) as { error?: string };
        if (json.error) message = json.error;
      } catch {
        // use raw text
      }
      throw new Error(message);
    }

    const contentType = res.headers.get('Content-Type') ?? '';
    if (contentType.includes('application/json')) {
      return (await res.json()) as T;
    }
    return (await res.text()) as unknown as T;
  }

  // ---- Auth ----
  login(username: string, password: string) {
    return this.request<LoginResponse>('POST', '/auth/login', { username, password });
  }

  // ---- Agents ----
  listAgents() {
    if (this.isPrototype) {
      const agents: AgentInfo[] = Object.values(MOCK_AGENTS)
        .filter((a) => a.name !== 'builder-assistant')
        .map((a) => ({
          name: a.name,
          description: a.description,
          tools_count: a.tools_count,
          has_knowledge: a.has_knowledge,
        }));
      return this.mock(agents);
    }
    return this.request<AgentInfo[]>('GET', '/agents');
  }
  getAgent(name: string) {
    if (this.isPrototype) {
      const agent = MOCK_AGENTS[name] ?? Object.values(MOCK_AGENTS)[0]!;
      return this.mock<AgentDetail>(agent);
    }
    return this.request<AgentDetail>('GET', `/agents/${encodeURIComponent(name)}`);
  }
  createAgent(data: CreateAgentRequest) {
    if (this.isPrototype) return this.mock({ ...data, tools_count: 0, has_knowledge: false } as AgentDetail);
    return this.request<AgentDetail>('POST', '/agents', data);
  }
  updateAgent(name: string, data: Partial<CreateAgentRequest>) {
    if (this.isPrototype) return this.mock({ name, ...data } as AgentDetail);
    return this.request<AgentDetail>('PUT', `/agents/${encodeURIComponent(name)}`, data);
  }
  deleteAgent(name: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/agents/${encodeURIComponent(name)}`);
  }

  // ---- Models ----
  listModels(typeFilter?: string) {
    if (this.isPrototype) {
      const models = MOCK_MODELS_LIST;
      if (typeFilter === 'embedding') return this.mock(models.filter(m => m.type === 'embedding'));
      if (typeFilter === '!embedding') return this.mock(models.filter(m => m.type !== 'embedding'));
      return this.mock(models);
    }
    const query = typeFilter ? `?type=${encodeURIComponent(typeFilter)}` : '';
    return this.request<Model[]>('GET', `/models${query}`);
  }
  createModel(data: CreateModelRequest) {
    if (this.isPrototype) return this.mock({ id: crypto.randomUUID(), ...data, has_api_key: !!data.api_key, created_at: new Date().toISOString() } as Model);
    return this.request<Model>('POST', '/models', data);
  }
  updateModel(name: string, data: CreateModelRequest) {
    if (this.isPrototype) return this.mock({ ...data, name } as Model);
    return this.request<Model>('PUT', `/models/${encodeURIComponent(name)}`, data);
  }
  deleteModel(name: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/models/${encodeURIComponent(name)}`);
  }

  // ---- MCP Servers ----
  listMCPServers() {
    if (this.isPrototype) return this.mock(MOCK_MCP_SERVERS);
    return this.request<MCPServer[]>('GET', '/mcp-servers');
  }
  createMCPServer(data: CreateMCPServerRequest) {
    if (this.isPrototype) return this.mock({ id: crypto.randomUUID(), ...data, status: { status: 'connected', tools_count: 0 }, is_well_known: false, agents: [] } as MCPServer);
    return this.request<MCPServer>('POST', '/mcp-servers', data);
  }
  updateMCPServer(name: string, data: CreateMCPServerRequest) {
    if (this.isPrototype) return this.mock({ id: '', ...data, name, is_well_known: false, agents: [] } as MCPServer);
    return this.request<MCPServer>('PUT', `/mcp-servers/${encodeURIComponent(name)}`, data);
  }
  deleteMCPServer(name: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/mcp-servers/${encodeURIComponent(name)}`);
  }

  // ---- Triggers ----
  listTriggers(schemaId?: string) {
    if (this.isPrototype) return this.mock(MOCK_TRIGGERS);
    const q = schemaId != null ? `?schema_id=${schemaId}` : '';
    return this.request<Trigger[]>('GET', `/triggers${q}`);
  }
  createTrigger(data: CreateTriggerRequest) {
    if (this.isPrototype) return this.mock({ id: crypto.randomUUID(), ...data, created_at: new Date().toISOString() } as Trigger);
    return this.request<Trigger>('POST', '/triggers', data);
  }
  updateTrigger(id: string, data: CreateTriggerRequest) {
    if (this.isPrototype) return this.mock({ id, ...data } as Trigger);
    return this.request<Trigger>('PUT', `/triggers/${id}`, data);
  }
  deleteTrigger(id: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/triggers/${id}`);
  }
  setTriggerTarget(id: string, agentName: string) {
    if (this.isPrototype) return this.mock({} as Trigger);
    return this.request<Trigger>('PATCH', `/triggers/${id}/target`, { agent_name: agentName });
  }
  clearTriggerTarget(id: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/triggers/${id}/target`);
  }

  // ---- Tasks ----
  listTasks(params?: Record<string, string>) {
    if (this.isPrototype) return this.mock([] as TaskResponse[]);
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.request<TaskResponse[]>('GET', `/tasks${qs}`);
  }
  listTasksPaginated(params: Record<string, string>) {
    if (this.isPrototype) return this.mock(MOCK_TASKS_PAGINATED);
    const qs = '?' + new URLSearchParams(params).toString();
    return this.request<PaginatedTaskResponse>('GET', `/tasks${qs}`);
  }
  getTask(id: string) {
    if (this.isPrototype) return this.mock({ id, title: 'Mock Task', agent_name: 'assistant', status: 'completed', source: 'api', created_at: new Date().toISOString(), mode: 'chat' } as TaskDetailResponse);
    return this.request<TaskDetailResponse>('GET', `/tasks/${id}`);
  }
  cancelTask(id: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/tasks/${id}`);
  }

  // ---- Health ----
  health() {
    if (this.isPrototype) return this.mock(MOCK_HEALTH);
    return this.request<HealthResponse>('GET', '/health');
  }

  // ---- Tokens ----
  listTokens() {
    if (this.isPrototype) return this.mock(MOCK_TOKENS);
    return this.request<APIToken[]>('GET', '/auth/tokens');
  }
  createToken(data: CreateTokenRequest) {
    if (this.isPrototype) return this.mock({ id: crypto.randomUUID(), name: data.name, token: 'bb_proto_' + Math.random().toString(36).slice(2) } as CreateTokenResponse);
    return this.request<CreateTokenResponse>('POST', '/auth/tokens', data);
  }
  deleteToken(id: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/auth/tokens/${id}`);
  }

  // ---- Settings ----
  listSettings() {
    if (this.isPrototype) return this.mock(MOCK_SETTINGS as Setting[] | Record<string, unknown>);
    // API may return Setting[] or flat object depending on backend implementation
    return this.request<Setting[] | Record<string, unknown>>('GET', '/settings');
  }
  updateSetting(key: string, value: string) {
    if (this.isPrototype) return this.mock({ key, value } as Setting);
    return this.request<Setting>('PUT', `/settings/${encodeURIComponent(key)}`, { value });
  }

  // ---- Tools ----
  listToolMetadata() {
    if (this.isPrototype) return this.mock([] as ToolMetadata[]);
    return this.request<ToolMetadata[]>('GET', '/tools/metadata');
  }

  // ---- Config ----
  reloadConfig() {
    if (this.isPrototype) return this.mock({ reloaded: true, agents_count: 6 });
    return this.request<{ reloaded: boolean; agents_count: number }>('POST', '/config/reload');
  }
  exportConfig() {
    if (this.isPrototype) return this.mock(MOCK_CONFIG_YAML);
    return this.request<string>('GET', '/config/export');
  }
  importConfig(yamlContent: string) {
    if (this.isPrototype) return this.mock({ imported: true, agents_count: 3 });
    return this.requestRaw<{ imported: boolean; agents_count: number }>('POST', '/config/import', yamlContent, 'text/yaml');
  }

  // ---- Audit ----
  listAuditLogs(params: Record<string, string> = {}) {
    if (this.isPrototype) return this.mock(MOCK_AUDIT_LOGS);
    const qs = Object.keys(params).length ? '?' + new URLSearchParams(params).toString() : '';
    return this.request<PaginatedResponse<AuditEntry>>('GET', `/audit${qs}`);
  }

  // ---- Model Registry ----
  getModelRegistry(filters?: { provider?: string; tier?: number }) {
    const params = new URLSearchParams();
    if (filters?.provider) params.set('provider', filters.provider);
    if (filters?.tier) params.set('tier', String(filters.tier));
    const qs = params.toString() ? '?' + params.toString() : '';
    return this.request<ModelRegistryEntry[]>('GET', `/models/registry${qs}`);
  }

  getRegistryProviders() {
    return this.request<RegistryProviderInfo[]>('GET', `/models/registry/providers`);
  }

  // ─── Schemas ─────────────────────────────────────────────────────────────────

  listSchemas() {
    if (this.isPrototype) {
      return this.mock<Schema[]>(
        SCHEMA_NAMES.map((name, i) => ({
          id: String(i + 1),
          name,
          agents_count: 3,
          created_at: new Date().toISOString(),
        })),
      );
    }
    return this.request<Schema[]>('GET', '/schemas');
  }

  createSchema(data: { name: string; description?: string }) {
    if (this.isPrototype) return this.mock({ id: String(Date.now()), name: data.name, description: data.description, agents_count: 0, created_at: new Date().toISOString() } as Schema);
    return this.request<Schema>('POST', '/schemas', data);
  }

  deleteSchema(schemaId: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/schemas/${schemaId}`);
  }

  listSchemaAgents(schemaId: string) {
    if (this.isPrototype) return this.mock<string[]>([]);
    return this.request<string[]>('GET', `/schemas/${schemaId}/agents`);
  }

  addAgentToSchema(schemaId: string, agentName: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('POST', `/schemas/${schemaId}/agents`, { agent_name: agentName });
  }

  removeAgentFromSchema(schemaId: string, agentName: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/schemas/${schemaId}/agents/${encodeURIComponent(agentName)}`);
  }

  // ─── Sessions / Inspect ──────────────────────────────────────────────────────

  async listSessions(params?: {
    page?: number;
    per_page?: number;
    search?: string;
    status?: string[];
    sort_by?: string;
    sort_dir?: 'asc' | 'desc';
    from?: string;
    to?: string;
    agent_name?: string;
  }): Promise<PaginatedSessions> {
    if (this.isPrototype) {
      const page = params?.page ?? 1;
      const perPage = params?.per_page ?? 20;
      let filtered = [...MOCK_SESSIONS_LIST];

      if (params?.agent_name) {
        filtered = filtered.filter((s) => s.entry_agent === params.agent_name);
      }
      if (params?.search) {
        const q = params.search.toLowerCase();
        filtered = filtered.filter(
          (s) => s.session_id.toLowerCase().includes(q) || s.entry_agent.toLowerCase().includes(q),
        );
      }
      if (params?.status && params.status.length > 0) {
        filtered = filtered.filter((s) => params.status!.includes(s.status));
      }

      const total = filtered.length;
      const start = (page - 1) * perPage;
      const sessions = filtered.slice(start, start + perPage);
      return this.mock<PaginatedSessions>({ sessions, total, page, per_page: perPage });
    }

    const qs = new URLSearchParams();
    if (params?.page) qs.set('page', String(params.page));
    if (params?.per_page) qs.set('per_page', String(params.per_page));
    if (params?.search) qs.set('search', params.search);
    if (params?.status) qs.set('status', params.status.join(','));
    if (params?.sort_by) qs.set('sort_by', params.sort_by);
    if (params?.sort_dir) qs.set('sort_dir', params.sort_dir);
    if (params?.from) qs.set('from', params.from);
    if (params?.to) qs.set('to', params.to);
    if (params?.agent_name) qs.set('agent_name', params.agent_name);
    const q = qs.toString() ? '?' + qs.toString() : '';
    // Backend returns { data: [...], total, page, per_page } — map to PaginatedSessions
    const raw = await this.request<{ data?: SessionSummary[]; sessions?: SessionSummary[]; total: number; page: number; per_page: number }>('GET', `/sessions${q}`);
    return { sessions: raw.data ?? raw.sessions ?? [], total: raw.total, page: raw.page, per_page: raw.per_page };
  }

  deleteSession(sessionId: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/sessions/${sessionId}`);
  }

  getSessionMessages(sessionId: string): Promise<MessageResponse[]> {
    if (this.isPrototype) return this.mock<MessageResponse[]>([]);
    return this.request<MessageResponse[]>('GET', `/sessions/${sessionId}/messages`);
  }

  getSessionEvents(sessionId: string): Promise<EventResponse[]> {
    if (this.isPrototype) return this.mock<EventResponse[]>([]);
    return this.request<EventResponse[]>('GET', `/sessions/${sessionId}/messages`);
  }

  getSessionTrace(sessionId: string): Promise<SessionTrace> {
    if (this.isPrototype) {
      if (sessionId === MOCK_TRACE_ERROR.session_id) {
        return this.mock(MOCK_TRACE_ERROR);
      }
      return this.mock({ ...MOCK_TRACE, session_id: sessionId });
    }
    return this.request<SessionTrace>('GET', `/sessions/${sessionId}`);
  }

  // ─── Widgets ─────────────────────────────────────────────────────────────────

  private static MOCK_WIDGETS: WidgetConfig[] = [
    {
      id: 'wid_abc123',
      name: 'Support Chat',
      schema: 'Support Schema',
      status: 'active',
      primary_color: '#D7513E',
      position: 'bottom-right',
      size: 'standard',
      welcome_message: 'Hi! How can we help you today?',
      placeholder_text: 'Type your message...',
      avatar_url: '',
      domain_whitelist: 'example.com, app.example.com',
      created_at: '2026-04-01T10:00:00Z',
    },
    {
      id: 'wid_def456',
      name: 'Sales Bot',
      schema: 'Sales Schema',
      status: 'disabled',
      primary_color: '#3B82F6',
      position: 'bottom-left',
      size: 'compact',
      welcome_message: 'Welcome! Looking for a demo?',
      placeholder_text: 'Ask about our plans...',
      avatar_url: '',
      domain_whitelist: '',
      created_at: '2026-04-02T14:30:00Z',
    },
  ];

  listWidgets(): Promise<WidgetConfig[]> {
    if (this.isPrototype) return this.mock(APIClient.MOCK_WIDGETS);
    return this.request<WidgetConfig[]>('GET', '/widgets');
  }

  createWidget(data: CreateWidgetRequest): Promise<WidgetConfig> {
    if (this.isPrototype) {
      const widget: WidgetConfig = {
        ...data,
        id: `wid_${Math.random().toString(36).slice(2, 8)}`,
        created_at: new Date().toISOString(),
      };
      APIClient.MOCK_WIDGETS.push(widget);
      return this.mock(widget);
    }
    return this.request<WidgetConfig>('POST', '/widgets', data);
  }

  updateWidget(id: string, data: Partial<CreateWidgetRequest>): Promise<WidgetConfig> {
    if (this.isPrototype) {
      const idx = APIClient.MOCK_WIDGETS.findIndex((w) => w.id === id);
      if (idx >= 0) {
        APIClient.MOCK_WIDGETS[idx] = { ...APIClient.MOCK_WIDGETS[idx]!, ...data };
        return this.mock(APIClient.MOCK_WIDGETS[idx]!);
      }
      return Promise.reject(new Error('Widget not found'));
    }
    return this.request<WidgetConfig>('PUT', `/widgets/${id}`, data);
  }

  deleteWidget(id: string): Promise<void> {
    if (this.isPrototype) {
      APIClient.MOCK_WIDGETS = APIClient.MOCK_WIDGETS.filter((w) => w.id !== id);
      return this.mock(undefined as unknown as void);
    }
    return this.request<void>('DELETE', `/widgets/${id}`);
  }

  // ─── Usage / Quota ───────────────────────────────────────────────────────────

  getUsage(): Promise<UsageData> {
    if (this.isPrototype) {
      return this.mock<UsageData>({
        plan: 'Pro',
        billing_cycle_start: '2026-04-01T00:00:00Z',
        billing_cycle_end: '2026-05-01T00:00:00Z',
        metrics: [
          { name: 'api_calls', label: 'API Calls', used: 8500, limit: 10000, unit: 'calls' },
          { name: 'storage', label: 'Storage', used: 3.2, limit: 5, unit: 'GB' },
          { name: 'schemas', label: 'Schemas', used: 2, limit: 5, unit: '' },
          { name: 'agents', label: 'Agents per Schema', used: 7, limit: 20, unit: '' },
        ],
        stripe_portal_url: 'https://billing.stripe.com/p/session/test',
      });
    }
    return this.request<UsageData>('GET', '/usage');
  }

  // ─── Memory ──────────────────────────────────────────────────────────────────

  async listMemories(schemaId: string): Promise<MemoryEntry[]> {
    if (this.isPrototype) {
      return this.mock<MemoryEntry[]>([
        { id: 'mem_1', schema_id: schemaId, content: 'User prefers concise responses', metadata: { source: 'conversation' }, created_at: '2026-04-01T10:00:00Z', updated_at: '2026-04-01T10:00:00Z' },
        { id: 'mem_2', schema_id: schemaId, user_id: 'user_42', content: 'Customer is on Enterprise plan, prefers email communication', metadata: { source: 'crm_sync' }, created_at: '2026-04-02T14:30:00Z', updated_at: '2026-04-02T14:30:00Z' },
        { id: 'mem_3', schema_id: schemaId, content: 'Product FAQ: refund policy is 30 days', metadata: { source: 'knowledge_base' }, created_at: '2026-04-03T09:15:00Z', updated_at: '2026-04-03T09:15:00Z' },
        { id: 'mem_4', schema_id: schemaId, user_id: 'user_99', content: 'Reported bug with checkout flow — escalated to engineering', metadata: { source: 'conversation', priority: 'high' }, created_at: '2026-04-04T16:45:00Z', updated_at: '2026-04-04T16:45:00Z' },
      ]);
    }
    return this.request<MemoryEntry[]>('GET', `/schemas/${encodeURIComponent(schemaId)}/memory`);
  }

  async clearMemories(schemaId: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/schemas/${encodeURIComponent(schemaId)}/memory`);
  }

  async deleteMemory(schemaId: string, entryId: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/schemas/${encodeURIComponent(schemaId)}/memory/${encodeURIComponent(entryId)}`);
  }

  // ─── MCP Catalog ───────────────────────────────────────────────────────────────

  async listCatalog(category?: string, query?: string): Promise<MCPCatalogEntry[]> {
    if (this.isPrototype) {
      let results = [...MOCK_CATALOG];
      if (category) results = results.filter((e) => e.category === category);
      if (query) {
        const q = query.toLowerCase();
        results = results.filter((e) => e.display.toLowerCase().includes(q) || e.name.toLowerCase().includes(q));
      }
      return this.mock(results);
    }
    const params = new URLSearchParams();
    if (category) params.set('category', category);
    if (query) params.set('q', query);
    const qs = params.toString() ? '?' + params.toString() : '';
    const resp = await this.request<MCPCatalogResponse>('GET', `/mcp/catalog${qs}`);
    return resp.servers ?? [];
  }

  // ─── Capabilities ──────────────────────────────────────────────────────────────

  async listCapabilities(agentName: string): Promise<Capability[]> {
    if (this.isPrototype) {
      return this.mock<Capability[]>([
        { id: '1', agent_name: agentName, type: 'memory', config: { unlimited_retention: true, max_entries: 500 }, enabled: true },
        { id: '2', agent_name: agentName, type: 'knowledge', config: { sources: ['support-docs.pdf'], top_k: 5 }, enabled: true },
      ]);
    }
    return this.request<Capability[]>('GET', `/agents/${encodeURIComponent(agentName)}/capabilities`);
  }

  async addCapability(agentName: string, data: CreateCapabilityRequest): Promise<Capability> {
    if (this.isPrototype) {
      return this.mock<Capability>({ id: String(Date.now()), agent_name: agentName, ...data });
    }
    return this.request<Capability>('POST', `/agents/${encodeURIComponent(agentName)}/capabilities`, data);
  }

  async updateCapability(agentName: string, capId: string, data: UpdateCapabilityRequest): Promise<Capability> {
    if (this.isPrototype) {
      return this.mock<Capability>({ id: capId, agent_name: agentName, type: 'memory', config: {}, enabled: true, ...data });
    }
    return this.request<Capability>('PUT', `/agents/${encodeURIComponent(agentName)}/capabilities/${capId}`, data);
  }

  async removeCapability(agentName: string, capId: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/agents/${encodeURIComponent(agentName)}/capabilities/${capId}`);
  }

  // ─── Knowledge ──────────────────────────────────────────────────────────────

  async getKnowledgeStatus(agentName: string): Promise<KnowledgeStatus> {
    if (this.isPrototype) return this.mock<KnowledgeStatus>({ agent_name: agentName, total_files: 2, indexed_files: 2, status: 'ready' });
    return this.request<KnowledgeStatus>('GET', `/agents/${encodeURIComponent(agentName)}/knowledge/status`);
  }

  async listKnowledgeFiles(agentName: string): Promise<KnowledgeFile[]> {
    if (this.isPrototype) return this.mock<KnowledgeFile[]>([]);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await this.request<any[]>('GET', `/agents/${encodeURIComponent(agentName)}/knowledge/files`);
    return (raw ?? []).map((r) => ({
      id: r.id,
      name: r.file_name ?? r.name ?? '',
      type: (r.file_type ?? r.type ?? '').toUpperCase(),
      size: r.file_size != null ? formatBytes(r.file_size) : (r.size ?? ''),
      uploaded_at: r.created_at ?? r.uploaded_at ?? '',
      status: r.status ?? 'ready',
      error: r.status_message,
      chunk_count: r.chunk_count,
    } as KnowledgeFile));
  }

  async deleteKnowledgeFile(agentName: string, fileId: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/agents/${encodeURIComponent(agentName)}/knowledge/files/${encodeURIComponent(fileId)}`);
  }

  async reindexKnowledge(agentName: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('POST', `/agents/${encodeURIComponent(agentName)}/knowledge/reindex`);
  }

  async reindexKnowledgeFile(agentName: string, fileId: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('POST', `/agents/${encodeURIComponent(agentName)}/knowledge/files/${encodeURIComponent(fileId)}/reindex`);
  }

  async uploadKnowledgeFile(agentName: string, file: File): Promise<KnowledgeFile> {
    if (this.isPrototype) {
      return this.mock<KnowledgeFile>({
        name: file.name,
        type: file.name.split('.').pop() ?? '',
        size: `${(file.size / 1024).toFixed(1)} KB`,
        status: 'ready',
        uploaded_at: new Date().toISOString(),
      });
    }
    const formData = new FormData();
    formData.append('file', file);
    const headers: Record<string, string> = {};
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }
    const res = await fetch(`${BASE_URL}/agents/${encodeURIComponent(agentName)}/knowledge/files`, {
      method: 'POST',
      headers,
      body: formData,
    });
    if (res.status === 401) {
      this.clearToken();
      window.location.href = import.meta.env.BASE_URL + 'login';
      throw new Error('Unauthorized');
    }
    if (!res.ok) {
      const text = await res.text();
      let message = text;
      try {
        const json = JSON.parse(text) as { error?: string };
        if (json.error) message = json.error;
      } catch { /* use raw text */ }
      throw new Error(message);
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const r = (await res.json()) as any;
    return {
      id: r.id,
      name: r.file_name ?? r.name ?? '',
      type: (r.file_type ?? r.type ?? '').toUpperCase(),
      size: r.file_size != null ? formatBytes(r.file_size) : (r.size ?? ''),
      uploaded_at: r.created_at ?? r.uploaded_at ?? '',
      status: r.status ?? 'indexing',
      error: r.status_message,
      chunk_count: r.chunk_count,
    } as KnowledgeFile;
  }

  // ─── Knowledge Bases (many-to-many) ──────────────────────────────────────────

  async listKnowledgeBases(): Promise<KnowledgeBase[]> {
    if (this.isPrototype) return this.mock<KnowledgeBase[]>([
      { id: 'kb-1', name: 'Support Docs', description: 'Customer support documentation', embedding_model_id: '', file_count: 3, linked_agents: [], created_at: '2026-04-10T10:00:00Z', updated_at: '2026-04-10T10:00:00Z' },
    ]);
    return this.request<KnowledgeBase[]>('GET', '/knowledge-bases');
  }

  async getKnowledgeBase(id: string): Promise<KnowledgeBase> {
    if (this.isPrototype) return this.mock<KnowledgeBase>({ id, name: 'Mock KB', file_count: 0, linked_agents: [], created_at: '', updated_at: '' });
    return this.request<KnowledgeBase>('GET', `/knowledge-bases/${encodeURIComponent(id)}`);
  }

  async createKnowledgeBase(data: CreateKnowledgeBaseRequest): Promise<KnowledgeBase> {
    if (this.isPrototype) return this.mock<KnowledgeBase>({ id: 'kb-new', ...data, file_count: 0, linked_agents: [], created_at: new Date().toISOString(), updated_at: new Date().toISOString() });
    return this.request<KnowledgeBase>('POST', '/knowledge-bases', data);
  }

  async updateKnowledgeBase(id: string, data: CreateKnowledgeBaseRequest): Promise<KnowledgeBase> {
    if (this.isPrototype) return this.mock<KnowledgeBase>({ id, ...data, file_count: 0, linked_agents: [], created_at: '', updated_at: new Date().toISOString() });
    return this.request<KnowledgeBase>('PUT', `/knowledge-bases/${encodeURIComponent(id)}`, data);
  }

  async deleteKnowledgeBase(id: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/knowledge-bases/${encodeURIComponent(id)}`);
  }

  async linkAgentToKB(kbId: string, agentName: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('POST', `/knowledge-bases/${encodeURIComponent(kbId)}/agents/${encodeURIComponent(agentName)}`);
  }

  async unlinkAgentFromKB(kbId: string, agentName: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/knowledge-bases/${encodeURIComponent(kbId)}/agents/${encodeURIComponent(agentName)}`);
  }

  async listKBFiles(kbId: string): Promise<KnowledgeFile[]> {
    if (this.isPrototype) return this.mock<KnowledgeFile[]>([]);
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const raw = await this.request<any[]>('GET', `/knowledge-bases/${encodeURIComponent(kbId)}/files`);
    return (raw ?? []).map((r) => ({
      id: r.id,
      knowledge_base_id: r.knowledge_base_id,
      name: r.file_name ?? r.name ?? '',
      type: (r.file_type ?? r.type ?? '').toUpperCase(),
      size: r.file_size != null ? formatBytes(r.file_size) : (r.size ?? ''),
      uploaded_at: r.created_at ?? r.uploaded_at ?? '',
      status: r.status ?? 'ready',
      error: r.status_message,
      chunk_count: r.chunk_count,
    } as KnowledgeFile));
  }

  async uploadKBFile(kbId: string, file: File): Promise<KnowledgeFile> {
    if (this.isPrototype) {
      return this.mock<KnowledgeFile>({
        name: file.name,
        type: file.name.split('.').pop() ?? '',
        size: `${(file.size / 1024).toFixed(1)} KB`,
        status: 'ready',
        uploaded_at: new Date().toISOString(),
      });
    }
    const formData = new FormData();
    formData.append('file', file);
    const headers: Record<string, string> = {};
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }
    const res = await fetch(`${BASE_URL}/knowledge-bases/${encodeURIComponent(kbId)}/files`, {
      method: 'POST',
      headers,
      body: formData,
    });
    if (res.status === 401) {
      this.clearToken();
      window.location.href = import.meta.env.BASE_URL + 'login';
      throw new Error('Unauthorized');
    }
    if (!res.ok) {
      const text = await res.text();
      let message = text;
      try {
        const json = JSON.parse(text) as { error?: string };
        if (json.error) message = json.error;
      } catch { /* use raw text */ }
      throw new Error(message);
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const r = (await res.json()) as any;
    return {
      id: r.id,
      name: r.file_name ?? r.name ?? '',
      type: (r.file_type ?? r.type ?? '').toUpperCase(),
      size: r.file_size != null ? formatBytes(r.file_size) : (r.size ?? ''),
      uploaded_at: r.created_at ?? r.uploaded_at ?? '',
      status: r.status ?? 'indexing',
      error: r.status_message,
      chunk_count: r.chunk_count,
    } as KnowledgeFile;
  }

  async deleteKBFile(kbId: string, fileId: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/knowledge-bases/${encodeURIComponent(kbId)}/files/${encodeURIComponent(fileId)}`);
  }

  async reindexKBFile(kbId: string, fileId: string): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('POST', `/knowledge-bases/${encodeURIComponent(kbId)}/files/${encodeURIComponent(fileId)}/reindex`);
  }

  // ─── Circuit Breakers ────────────────────────────────────────────────────────

  async listCircuitBreakers(): Promise<CircuitBreakerState[]> {
    if (this.isPrototype) return [];
    const res = await fetch(`${BASE_URL}/admin/resilience/circuit-breakers`, {
      headers: this.token ? { Authorization: `Bearer ${this.token}` } : {},
    });
    if (!res.ok) return [];
    return res.json();
  }

  // ─── Builder Assistant ───────────────────────────────────────────────────────

  async restoreBuilderAssistant(): Promise<void> {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    await this.request<void>('POST', '/admin/builder-assistant/restore', undefined);
  }

  /**
   * Send a request with a raw (non-JSON) body.
   */
  private async requestRaw<T>(method: string, path: string, body: string, contentType: string): Promise<T> {
    const headers: Record<string, string> = { 'Content-Type': contentType };
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const res = await fetch(`${BASE_URL}${path}`, {
      method,
      headers,
      body,
    });

    if (res.status === 401) {
      this.clearToken();
      window.location.href = import.meta.env.BASE_URL + 'login';
      throw new Error('Unauthorized');
    }

    if (!res.ok) {
      const text = await res.text();
      let message = text;
      try {
        const json = JSON.parse(text) as { error?: string };
        if (json.error) message = json.error;
      } catch {
        // use raw text
      }
      throw new Error(message);
    }

    const ct = res.headers.get('Content-Type') ?? '';
    if (ct.includes('application/json')) {
      return (await res.json()) as T;
    }
    return (await res.text()) as unknown as T;
  }
}

export const api = new APIClient();
