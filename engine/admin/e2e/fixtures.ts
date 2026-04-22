import { test as base, expect, Page, APIRequestContext } from '@playwright/test';

export { expect };

export const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? 'http://localhost:18082';
export const ENGINE_API = `${BASE_URL}/api/v1`;

export type AdminToken = {
  token: string;
  userId?: string;
};

async function adminLocalSession(request: APIRequestContext): Promise<string> {
  // AUTH_MODE=local — engine issues its own session
  const res = await request.post(`${ENGINE_API}/auth/local-session`);
  if (res.status() === 404) {
    // legacy HS256 admin login fallback
    const legacy = await request.post(`${ENGINE_API}/auth/login`, {
      data: { username: 'admin', password: 'admin123' },
    });
    if (!legacy.ok()) throw new Error(`admin auth failed (both local-session and legacy): ${legacy.status()}`);
    const body = await legacy.json();
    return body.token ?? body.access_token;
  }
  if (!res.ok()) throw new Error(`local-session failed: ${res.status()}`);
  const body = await res.json();
  return body.access_token ?? body.token;
}

type Fixtures = {
  adminToken: string;
  authenticatedAdmin: Page;
};

export const test = base.extend<Fixtures>({
  adminToken: async ({ request }, use) => {
    const token = await adminLocalSession(request);
    await use(token);
  },
  authenticatedAdmin: async ({ page, adminToken }, use) => {
    await page.addInitScript((token: string) => {
      window.localStorage.setItem('jwt', token);
      window.localStorage.setItem('access_token', token);
    }, adminToken);
    await page.goto('/admin/');
    await use(page);
  },
});

export async function apiFetch(
  request: APIRequestContext,
  path: string,
  options: { method?: string; token?: string; body?: unknown; headers?: Record<string, string> } = {}
) {
  const url = path.startsWith('http') ? path : `${ENGINE_API}${path}`;
  const init: Parameters<APIRequestContext['fetch']>[1] = {
    method: options.method ?? 'GET',
    headers: {
      'Content-Type': 'application/json',
      ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}),
      ...options.headers,
    },
    data: options.body,
  };
  return await request.fetch(url, init);
}
