// CloudApiClient — HTTP client for Vector Cloud API
// Base URL: BYTEBREW_CLOUD_URL env or http://localhost:60402

import type { AuthTokens } from '../auth/AuthStorage.js';
export type { AuthTokens };

export interface LicenseInfo {
  jwt: string;
}

export interface UsageInfo {
  tier: string;
  proxyStepsUsed: number;
  proxyStepsLimit: number;
  proxyStepsRemaining: number;
  byokEnabled: boolean;
  currentPeriodEnd?: string;
}

export class CloudApiError extends Error {
  constructor(
    public readonly code: string,
    message: string,
    public readonly statusCode: number,
  ) {
    super(message);
    this.name = 'CloudApiError';
  }
}

interface AuthRegisterResponse {
  access_token: string;
  refresh_token: string;
  user_id: string;
  email?: string;
}

interface UsageResponse {
  tier: string;
  proxy_steps_used: number;
  proxy_steps_limit: number;
  proxy_steps_remaining: number;
  byok_enabled: boolean;
  current_period_end?: string;
}

export class CloudApiClient {
  private readonly baseUrl: string;
  private accessToken: string | null = null;
  private refreshToken: string | null = null;
  private email: string = '';
  private userId: string = '';
  private readonly onTokenRefreshed?: (tokens: AuthTokens) => void;

  constructor(options?: {
    baseUrl?: string;
    accessToken?: string;
    refreshToken?: string;
    email?: string;
    userId?: string;
    onTokenRefreshed?: (tokens: AuthTokens) => void;
  }) {
    this.baseUrl =
      options?.baseUrl ??
      process.env.BYTEBREW_CLOUD_URL ??
      'http://localhost:60402';
    this.accessToken = options?.accessToken ?? null;
    this.refreshToken = options?.refreshToken ?? null;
    this.email = options?.email ?? '';
    this.userId = options?.userId ?? '';
    this.onTokenRefreshed = options?.onTokenRefreshed;
  }

  setTokens(accessToken: string, refreshToken: string): void {
    this.accessToken = accessToken;
    this.refreshToken = refreshToken;
  }

  // Public endpoints

  async register(email: string, password: string): Promise<AuthTokens> {
    const data = await this.request<AuthRegisterResponse>(
      'POST',
      '/api/v1/auth/register',
      { email, password },
      false,
    );
    const tokens: AuthTokens = {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      email: data.email ?? email,
      userId: data.user_id,
    };
    this.setTokens(tokens.accessToken, tokens.refreshToken);
    this.email = tokens.email;
    this.userId = tokens.userId;
    return tokens;
  }

  async login(email: string, password: string): Promise<AuthTokens> {
    const data = await this.request<AuthRegisterResponse>(
      'POST',
      '/api/v1/auth/login',
      { email, password },
      false,
    );
    const tokens: AuthTokens = {
      accessToken: data.access_token,
      refreshToken: data.refresh_token,
      email: data.email ?? email,
      userId: data.user_id,
    };
    this.setTokens(tokens.accessToken, tokens.refreshToken);
    this.email = tokens.email;
    this.userId = tokens.userId;
    return tokens;
  }

  // Protected endpoints

  async activateLicense(): Promise<string> {
    const data = await this.authenticatedRequest<{ license: string }>(
      'POST',
      '/api/v1/license/activate',
      {},
    );
    return data.license;
  }

  async refreshLicense(currentJwt: string): Promise<string> {
    const data = await this.authenticatedRequest<{ license: string }>(
      'POST',
      '/api/v1/license/refresh',
      { current_license: currentJwt },
    );
    return data.license;
  }

  async getLicenseStatus(jwt: string): Promise<Record<string, unknown>> {
    const data = await this.authenticatedRequest<Record<string, unknown>>(
      'GET',
      `/api/v1/license/status?license=${encodeURIComponent(jwt)}`,
    );
    return data;
  }

  async getUsage(): Promise<UsageInfo> {
    const data = await this.authenticatedRequest<UsageResponse>(
      'GET',
      '/api/v1/subscription/usage',
    );
    return {
      tier: data.tier,
      proxyStepsUsed: data.proxy_steps_used,
      proxyStepsLimit: data.proxy_steps_limit,
      proxyStepsRemaining: data.proxy_steps_remaining,
      byokEnabled: data.byok_enabled,
      currentPeriodEnd: data.current_period_end,
    };
  }

  // Billing

  async createCheckout(plan: string, period: string): Promise<string> {
    const data = await this.authenticatedRequest<{ checkout_url: string }>(
      'POST',
      '/api/v1/billing/checkout',
      { plan, period },
    );
    return data.checkout_url;
  }

  async createPortal(): Promise<string> {
    const data = await this.authenticatedRequest<{ portal_url: string }>(
      'POST',
      '/api/v1/billing/portal',
      {},
    );
    return data.portal_url;
  }

  // Private helpers

  private async request<T>(
    method: string,
    path: string,
    body?: unknown,
    auth = false,
  ): Promise<T> {
    const url = `${this.baseUrl}${path}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (auth && this.accessToken) {
      headers['Authorization'] = `Bearer ${this.accessToken}`;
    }

    const response = await fetch(url, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
    });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const json = (await response.json()) as any;

    if (!response.ok) {
      const err = json?.error;
      throw new CloudApiError(
        err?.code ?? 'UNKNOWN',
        err?.message ?? `HTTP ${response.status}`,
        response.status,
      );
    }

    return json.data as T;
  }

  private async authenticatedRequest<T>(
    method: string,
    path: string,
    body?: unknown,
  ): Promise<T> {
    try {
      return await this.request<T>(method, path, body, true);
    } catch (err) {
      if (
        err instanceof CloudApiError &&
        err.statusCode === 401 &&
        this.refreshToken
      ) {
        await this.doRefreshToken();
        return await this.request<T>(method, path, body, true);
      }
      throw err;
    }
  }

  private async doRefreshToken(): Promise<void> {
    if (!this.refreshToken) {
      throw new CloudApiError('TOKEN_EXPIRED', 'Session expired, please login again', 401);
    }

    const data = await this.request<{ access_token: string }>(
      'POST',
      '/api/v1/auth/refresh',
      { refresh_token: this.refreshToken },
      false,
    );

    this.accessToken = data.access_token;

    if (this.onTokenRefreshed) {
      this.onTokenRefreshed({
        accessToken: data.access_token,
        refreshToken: this.refreshToken,
        email: this.email,
        userId: this.userId,
      });
    }
  }
}
