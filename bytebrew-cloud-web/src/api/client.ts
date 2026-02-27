const API_BASE = import.meta.env.VITE_API_URL || '';

export class ApiError extends Error {
  constructor(
    public code: string,
    message: string,
    public status: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

type TokenRefresher = () => Promise<string | null>;

export class ApiClient {
  private accessToken: string | null = null;
  private refresher: TokenRefresher | null = null;
  private refreshPromise: Promise<string | null> | null = null;

  setToken(token: string | null) {
    this.accessToken = token;
  }

  getToken(): string | null {
    return this.accessToken;
  }

  setRefresher(fn: TokenRefresher) {
    this.refresher = fn;
  }

  private async doRefresh(): Promise<string | null> {
    if (this.refreshPromise) return this.refreshPromise;
    this.refreshPromise = this.refresher!().finally(() => {
      this.refreshPromise = null;
    });
    return this.refreshPromise;
  }

  async request<T>(method: string, path: string, body?: unknown): Promise<T> {
    const response = await this.doFetch(method, path, body);

    if (response.status === 401 && this.refresher) {
      const newToken = await this.doRefresh();
      if (newToken) {
        this.accessToken = newToken;
        const retried = await this.doFetch(method, path, body);
        return this.unwrap<T>(retried);
      }
    }

    return this.unwrap<T>(response);
  }

  private async doFetch(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<Response> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (this.accessToken) {
      headers['Authorization'] = `Bearer ${this.accessToken}`;
    }

    return fetch(`${API_BASE}${path}`, {
      method,
      headers,
      body: body ? JSON.stringify(body) : undefined,
    });
  }

  private async unwrap<T>(res: Response): Promise<T> {
    if (res.status === 204) {
      return undefined as T;
    }

    const json = await res.json();

    if (!res.ok) {
      throw new ApiError(
        json.error?.code ?? 'UNKNOWN',
        json.error?.message ?? 'Unknown error',
        res.status,
      );
    }

    return json.data as T;
  }
}

export const api = new ApiClient();
