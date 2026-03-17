import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// We need to test the APIClient class behavior.
// Since it's a singleton export, we'll test via module re-import.

describe('APIClient', () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('stores token in localStorage on setToken', async () => {
    const { api } = await import('./client');
    api.setToken('test-jwt-token');
    expect(localStorage.getItem('jwt')).toBe('test-jwt-token');
    expect(api.isAuthenticated()).toBe(true);
  });

  it('clears token on clearToken', async () => {
    const { api } = await import('./client');
    api.setToken('test-jwt-token');
    api.clearToken();
    expect(localStorage.getItem('jwt')).toBeNull();
    expect(api.isAuthenticated()).toBe(false);
  });

  it('sends Authorization header when token is set', async () => {
    const { api } = await import('./client');
    api.setToken('my-jwt');

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      headers: new Headers({ 'Content-Type': 'application/json' }),
      json: () => Promise.resolve([]),
    });
    vi.stubGlobal('fetch', mockFetch);

    await api.listAgents();

    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/agents',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer my-jwt',
        }),
      }),
    );
  });

  it('redirects to /login on 401', async () => {
    const { api } = await import('./client');
    api.setToken('expired-token');

    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 401,
      headers: new Headers(),
      text: () => Promise.resolve('Unauthorized'),
    });
    vi.stubGlobal('fetch', mockFetch);

    // Mock window.location
    const locationMock = { href: '' };
    Object.defineProperty(window, 'location', {
      value: locationMock,
      writable: true,
    });

    await expect(api.listAgents()).rejects.toThrow('Unauthorized');
    expect(locationMock.href).toBe('/login');
    expect(localStorage.getItem('jwt')).toBeNull();
  });

  it('throws on non-OK responses', async () => {
    const { api } = await import('./client');
    api.setToken('valid');

    const mockFetch = vi.fn().mockResolvedValue({
      ok: false,
      status: 500,
      headers: new Headers(),
      text: () => Promise.resolve('{"error":"internal server error"}'),
    });
    vi.stubGlobal('fetch', mockFetch);

    await expect(api.health()).rejects.toThrow('internal server error');
  });
});
