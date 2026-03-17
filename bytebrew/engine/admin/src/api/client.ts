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
  Trigger,
  CreateTriggerRequest,
  APIToken,
  CreateTokenRequest,
  CreateTokenResponse,
  HealthResponse,
  Setting,
  LoginResponse,
} from '../types';

const BASE_URL = '/api/v1';

class APIClient {
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('jwt');
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

    if (res.status === 401) {
      this.clearToken();
      window.location.href = '/login';
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
    return this.request<AgentInfo[]>('GET', '/agents');
  }
  getAgent(name: string) {
    return this.request<AgentDetail>('GET', `/agents/${encodeURIComponent(name)}`);
  }
  createAgent(data: CreateAgentRequest) {
    return this.request<AgentDetail>('POST', '/agents', data);
  }
  updateAgent(name: string, data: Partial<CreateAgentRequest>) {
    return this.request<AgentDetail>('PUT', `/agents/${encodeURIComponent(name)}`, data);
  }
  deleteAgent(name: string) {
    return this.request<void>('DELETE', `/agents/${encodeURIComponent(name)}`);
  }

  // ---- Models ----
  listModels() {
    return this.request<Model[]>('GET', '/models');
  }
  createModel(data: CreateModelRequest) {
    return this.request<Model>('POST', '/models', data);
  }
  deleteModel(name: string) {
    return this.request<void>('DELETE', `/models/${encodeURIComponent(name)}`);
  }

  // ---- MCP Servers ----
  listMCPServers() {
    return this.request<MCPServer[]>('GET', '/mcp-servers');
  }
  getWellKnownMCP() {
    return this.request<WellKnownMCP[]>('GET', '/mcp/well-known');
  }
  createMCPServer(data: CreateMCPServerRequest) {
    return this.request<MCPServer>('POST', '/mcp-servers', data);
  }
  deleteMCPServer(name: string) {
    return this.request<void>('DELETE', `/mcp-servers/${encodeURIComponent(name)}`);
  }

  // ---- Triggers ----
  listTriggers() {
    return this.request<Trigger[]>('GET', '/triggers');
  }
  createTrigger(data: CreateTriggerRequest) {
    return this.request<Trigger>('POST', '/triggers', data);
  }
  deleteTrigger(id: number) {
    return this.request<void>('DELETE', `/triggers/${id}`);
  }

  // ---- Tasks ----
  listTasks(params?: Record<string, string>) {
    const qs = params ? '?' + new URLSearchParams(params).toString() : '';
    return this.request<TaskResponse[]>('GET', `/tasks${qs}`);
  }
  getTask(id: number) {
    return this.request<TaskDetailResponse>('GET', `/tasks/${id}`);
  }
  cancelTask(id: number) {
    return this.request<void>('DELETE', `/tasks/${id}`);
  }

  // ---- Health ----
  health() {
    return this.request<HealthResponse>('GET', '/health');
  }

  // ---- Tokens ----
  listTokens() {
    return this.request<APIToken[]>('GET', '/auth/tokens');
  }
  createToken(data: CreateTokenRequest) {
    return this.request<CreateTokenResponse>('POST', '/auth/tokens', data);
  }
  deleteToken(id: number) {
    return this.request<void>('DELETE', `/auth/tokens/${id}`);
  }

  // ---- Settings ----
  listSettings() {
    return this.request<Setting[]>('GET', '/settings');
  }
  updateSetting(key: string, value: string) {
    return this.request<Setting>('PUT', `/settings/${encodeURIComponent(key)}`, { value });
  }

  // ---- Config ----
  reloadConfig() {
    return this.request<{ reloaded: boolean; agents_count: number }>('POST', '/config/reload');
  }
  exportConfig() {
    return this.request<string>('GET', '/config/export');
  }
  importConfig(yamlContent: string) {
    return this.request<{ imported: boolean; agents_count: number }>('POST', '/config/import', yamlContent);
  }
}

export const api = new APIClient();
