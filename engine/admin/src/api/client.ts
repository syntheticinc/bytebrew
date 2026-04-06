import type {
  AgentInfo,
  AgentDetail,
  CreateAgentRequest,
  Model,
  CreateModelRequest,
  MCPServer,
  WellKnownMCP,
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
} from '../types';
import {
  MOCK_HEALTH,
  MOCK_MODELS_LIST,
  MOCK_MCP_SERVERS,
  MOCK_WELL_KNOWN,
  MOCK_TRIGGERS,
  MOCK_TASKS_PAGINATED,
  MOCK_TOKENS,
  MOCK_SETTINGS,
  MOCK_AUDIT_LOGS,
  MOCK_CONFIG_YAML,
} from '../mocks/pages';
import { MOCK_AGENTS } from '../mocks/agents';

const BASE_URL = '/api/v1';
const PROTOTYPE_KEY = 'bytebrew_prototype_mode';

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
  listModels() {
    if (this.isPrototype) return this.mock(MOCK_MODELS_LIST);
    return this.request<Model[]>('GET', '/models');
  }
  createModel(data: CreateModelRequest) {
    if (this.isPrototype) return this.mock({ id: Date.now(), ...data, has_api_key: !!data.api_key, created_at: new Date().toISOString() } as Model);
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
  getWellKnownMCP() {
    if (this.isPrototype) return this.mock(MOCK_WELL_KNOWN);
    return this.request<WellKnownMCP[]>('GET', '/mcp/well-known');
  }
  createMCPServer(data: CreateMCPServerRequest) {
    if (this.isPrototype) return this.mock({ id: Date.now(), ...data, status: { status: 'connected', tools_count: 0 }, is_well_known: false, agents: [] } as MCPServer);
    return this.request<MCPServer>('POST', '/mcp-servers', data);
  }
  updateMCPServer(name: string, data: CreateMCPServerRequest) {
    if (this.isPrototype) return this.mock({ id: 0, ...data, name, is_well_known: false, agents: [] } as MCPServer);
    return this.request<MCPServer>('PUT', `/mcp-servers/${encodeURIComponent(name)}`, data);
  }
  deleteMCPServer(name: string) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/mcp-servers/${encodeURIComponent(name)}`);
  }

  // ---- Triggers ----
  listTriggers() {
    if (this.isPrototype) return this.mock(MOCK_TRIGGERS);
    return this.request<Trigger[]>('GET', '/triggers');
  }
  createTrigger(data: CreateTriggerRequest) {
    if (this.isPrototype) return this.mock({ id: Date.now(), ...data, created_at: new Date().toISOString() } as Trigger);
    return this.request<Trigger>('POST', '/triggers', data);
  }
  updateTrigger(id: number, data: CreateTriggerRequest) {
    if (this.isPrototype) return this.mock({ id, ...data } as Trigger);
    return this.request<Trigger>('PUT', `/triggers/${id}`, data);
  }
  deleteTrigger(id: number) {
    if (this.isPrototype) return this.mock(undefined as unknown as void);
    return this.request<void>('DELETE', `/triggers/${id}`);
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
  getTask(id: number) {
    if (this.isPrototype) return this.mock({ id, title: 'Mock Task', agent_name: 'assistant', status: 'completed', source: 'api', created_at: new Date().toISOString(), mode: 'chat' } as TaskDetailResponse);
    return this.request<TaskDetailResponse>('GET', `/tasks/${id}`);
  }
  cancelTask(id: number) {
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
    if (this.isPrototype) return this.mock({ id: Date.now(), name: data.name, token: 'bb_proto_' + Math.random().toString(36).slice(2) } as CreateTokenResponse);
    return this.request<CreateTokenResponse>('POST', '/auth/tokens', data);
  }
  deleteToken(id: number) {
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
