import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ApiClient, ApiError } from '../client';

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

describe('ApiClient', () => {
  let client: ApiClient;
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    client = new ApiClient();
    vi.restoreAllMocks();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it('returns unwrapped data from successful response', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      jsonResponse(200, { data: { id: '1', name: 'test' } }),
    );

    const result = await client.request<{ id: string }>('GET', '/api/v1/users/1');

    expect(result).toEqual({ id: '1', name: 'test' });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      '/api/v1/users/1',
      expect.objectContaining({ method: 'GET' }),
    );
  });

  it('setToken / getToken stores and retrieves token', async () => {
    expect(client.getToken()).toBeNull();
    client.setToken('tok-abc');
    expect(client.getToken()).toBe('tok-abc');

    globalThis.fetch = vi.fn().mockResolvedValue(jsonResponse(200, { data: {} }));
    await client.request('GET', '/api/v1/me');

    const headers = (globalThis.fetch as ReturnType<typeof vi.fn>).mock.calls[0][1].headers;
    expect(headers['Authorization']).toBe('Bearer tok-abc');
  });

  it('handles 204 No Content', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));

    const result = await client.request('DELETE', '/api/v1/sessions/1');

    expect(result).toBeUndefined();
  });

  it('throws ApiError on non-ok response', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      jsonResponse(422, { error: { code: 'VALIDATION', message: 'email required' } }),
    );

    try {
      await client.request('POST', '/api/v1/auth/register');
      expect.unreachable('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      const apiErr = err as ApiError;
      expect(apiErr.code).toBe('VALIDATION');
      expect(apiErr.message).toBe('email required');
      expect(apiErr.status).toBe(422);
    }
  });

  it('refreshes token on 401 and retries the request', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse(401, { error: { code: 'UNAUTHORIZED', message: 'expired' } }))
      .mockResolvedValueOnce(jsonResponse(200, { data: { ok: true } }));
    globalThis.fetch = fetchMock;

    const refresher = vi.fn().mockResolvedValue('new-token');
    client.setRefresher(refresher);
    client.setToken('old-token');

    const result = await client.request<{ ok: boolean }>('GET', '/api/v1/me');

    expect(result).toEqual({ ok: true });
    expect(refresher).toHaveBeenCalledOnce();
    expect(client.getToken()).toBe('new-token');
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it('deduplicates concurrent refresh calls on parallel 401s', async () => {
    const fetchMock = vi.fn()
      // Both first calls return 401
      .mockResolvedValueOnce(jsonResponse(401, { error: { code: 'UNAUTHORIZED', message: 'expired' } }))
      .mockResolvedValueOnce(jsonResponse(401, { error: { code: 'UNAUTHORIZED', message: 'expired' } }))
      // Retries succeed
      .mockResolvedValueOnce(jsonResponse(200, { data: { v: 'a' } }))
      .mockResolvedValueOnce(jsonResponse(200, { data: { v: 'b' } }));
    globalThis.fetch = fetchMock;

    let resolveRefresh!: (token: string) => void;
    const refresher = vi.fn(
      () => new Promise<string>((resolve) => { resolveRefresh = resolve; }),
    );
    client.setRefresher(refresher);
    client.setToken('old');

    const p1 = client.request<{ v: string }>('GET', '/api/v1/a');
    const p2 = client.request<{ v: string }>('GET', '/api/v1/b');

    // Wait until refresher is called, then resolve
    await vi.waitFor(() => expect(refresher).toHaveBeenCalled());
    resolveRefresh('fresh-token');

    const [r1, r2] = await Promise.all([p1, p2]);
    expect(r1.v).toBe('a');
    expect(r2.v).toBe('b');

    // Only ONE refresh call despite two concurrent 401s
    expect(refresher).toHaveBeenCalledOnce();
  });

  it('throws ApiError when refresher returns null', async () => {
    globalThis.fetch = vi.fn().mockResolvedValue(
      jsonResponse(401, { error: { code: 'UNAUTHORIZED', message: 'expired' } }),
    );

    const refresher = vi.fn().mockResolvedValue(null);
    client.setRefresher(refresher);
    client.setToken('old');

    try {
      await client.request('GET', '/api/v1/me');
      expect.unreachable('should have thrown');
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError);
      expect((err as ApiError).status).toBe(401);
    }
    expect(refresher).toHaveBeenCalledOnce();
  });
});
