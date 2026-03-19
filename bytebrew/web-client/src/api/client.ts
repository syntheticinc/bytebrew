import type {
  AgentInfo,
  AgentDetail,
  ChatEvent,
  ChatEventType,
  HealthResponse,
  LoginResponse,
  PaginatedSessionResponse,
  PaginatedTaskResponse,
  SessionResponse,
  TaskDetailResponse,
  TaskResponse,
} from '../types';

const BASE_URL = '/api/v1';

class ByteBrewClient {
  private token: string | null = null;

  constructor() {
    this.token = localStorage.getItem('bytebrew_token');
  }

  setToken(token: string) {
    this.token = token;
    localStorage.setItem('bytebrew_token', token);
  }

  clearToken() {
    this.token = null;
    localStorage.removeItem('bytebrew_token');
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

  // Auth
  login(username: string, password: string) {
    return this.request<LoginResponse>('POST', '/auth/login', { username, password });
  }

  // Agents
  listAgents() {
    return this.request<AgentInfo[]>('GET', '/agents');
  }

  getAgent(name: string) {
    return this.request<AgentDetail>('GET', `/agents/${encodeURIComponent(name)}`);
  }

  // Tasks
  listTasks(params: Record<string, string> = {}) {
    const qs = Object.keys(params).length ? '?' + new URLSearchParams(params).toString() : '';
    return this.request<PaginatedTaskResponse>('GET', `/tasks${qs}`);
  }

  getTask(id: number) {
    return this.request<TaskDetailResponse>('GET', `/tasks/${id}`);
  }

  createTask(data: { title: string; description?: string; agent_name: string }) {
    return this.request<TaskResponse>('POST', '/tasks', data);
  }

  cancelTask(id: number) {
    return this.request<void>('DELETE', `/tasks/${id}`);
  }

  // Sessions
  listSessions(agentName?: string) {
    const qs = agentName ? `?agent=${encodeURIComponent(agentName)}` : '';
    return this.request<PaginatedSessionResponse>('GET', `/sessions${qs}`);
  }

  createSession(data: { title?: string; agent_name: string }) {
    return this.request<SessionResponse>('POST', '/sessions', { ...data, user_id: 'web-client' });
  }

  updateSession(id: string, data: { title?: string; status?: string }) {
    return this.request<SessionResponse>('PUT', `/sessions/${encodeURIComponent(id)}`, data);
  }

  deleteSession(id: string) {
    return this.request<void>('DELETE', `/sessions/${encodeURIComponent(id)}`);
  }

  // Health
  health() {
    return this.request<HealthResponse>('GET', '/health');
  }

  // SSE Chat
  chat(agent: string, message: string, onEvent: (event: ChatEvent) => void, sessionId?: string): AbortController {
    const controller = new AbortController();
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      Accept: 'text/event-stream',
    };
    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    fetch(`${BASE_URL}/agents/${encodeURIComponent(agent)}/chat`, {
      method: 'POST',
      headers,
      body: JSON.stringify({ message, user_id: 'web-client', ...(sessionId && { session_id: sessionId }) }),
      signal: controller.signal,
    })
      .then(async (res) => {
        if (!res.ok) {
          onEvent({ type: 'error', data: { message: `HTTP ${res.status}` } });
          return;
        }
        const reader = res.body?.getReader();
        if (!reader) return;
        const decoder = new TextDecoder();
        let buffer = '';
        let eventType = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) {
            onEvent({ type: 'done', data: { status: 'stream_ended' } });
            break;
          }
          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split('\n');
          buffer = lines.pop() ?? '';
          for (const line of lines) {
            if (line.startsWith('event: ')) {
              eventType = line.slice(7).trim();
            } else if (line.startsWith('data: ')) {
              try {
                const data = JSON.parse(line.slice(6)) as Record<string, unknown>;
                onEvent({ type: eventType as ChatEventType, data });
              } catch {
                // ignore parse errors
              }
            }
          }
        }
      })
      .catch((err: Error) => {
        if (err.name !== 'AbortError') {
          onEvent({ type: 'error', data: { message: err.message } });
        }
      });

    return controller;
  }
}

export const api = new ByteBrewClient();
