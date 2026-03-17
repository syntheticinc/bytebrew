import { describe, it, expect, beforeEach, mock, afterEach } from 'bun:test';
import {
  CloudApiClient,
  CloudApiError,
  type AuthTokens,
  type UsageInfo,
} from '../CloudApiClient';

// ─── helpers ──────────────────────────────────────────────────────────────────

function makeResponse(data: unknown, status = 200): Response {
  return new Response(JSON.stringify({ data }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

function makeErrorResponse(code: string, message: string, status: number): Response {
  return new Response(JSON.stringify({ error: { code, message } }), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}

// ─── tests ────────────────────────────────────────────────────────────────────

describe('CloudApiClient', () => {
  let client: CloudApiClient;
  let fetchMock: ReturnType<typeof mock>;

  beforeEach(() => {
    fetchMock = mock(() => Promise.resolve(makeResponse({})));
    globalThis.fetch = fetchMock as unknown as typeof fetch;
    client = new CloudApiClient({ baseUrl: 'http://test.local' });
  });

  afterEach(() => {
    fetchMock.mockRestore?.();
  });

  // ── register ──────────────────────────────────────────────────────────────

  describe('register', () => {
    it('sends correct body and returns AuthTokens', async () => {
      const responseData = {
        access_token: 'acc-tok',
        refresh_token: 'ref-tok',
        user_id: 'uid-123',
        email: 'user@test.com',
      };
      fetchMock.mockReturnValueOnce(Promise.resolve(makeResponse(responseData, 201)));

      const tokens: AuthTokens = await client.register('user@test.com', 'secret');

      expect(fetchMock).toHaveBeenCalledTimes(1);
      const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toBe('http://test.local/api/v1/auth/register');
      expect(init.method).toBe('POST');
      expect(JSON.parse(init.body as string)).toEqual({
        email: 'user@test.com',
        password: 'secret',
      });

      expect(tokens).toEqual({
        accessToken: 'acc-tok',
        refreshToken: 'ref-tok',
        email: 'user@test.com',
        userId: 'uid-123',
      });
    });

    it('falls back to provided email when response omits email field', async () => {
      const responseData = {
        access_token: 'a',
        refresh_token: 'r',
        user_id: 'u',
      };
      fetchMock.mockReturnValueOnce(Promise.resolve(makeResponse(responseData, 201)));

      const tokens = await client.register('fallback@test.com', 'pass');
      expect(tokens.email).toBe('fallback@test.com');
    });
  });

  // ── login ─────────────────────────────────────────────────────────────────

  describe('login', () => {
    it('sends correct body and returns AuthTokens', async () => {
      const responseData = {
        access_token: 'acc',
        refresh_token: 'ref',
        user_id: 'u1',
        email: 'me@test.com',
      };
      fetchMock.mockReturnValueOnce(Promise.resolve(makeResponse(responseData)));

      const tokens = await client.login('me@test.com', 'pw');

      const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toBe('http://test.local/api/v1/auth/login');
      expect(init.method).toBe('POST');
      expect(JSON.parse(init.body as string)).toEqual({
        email: 'me@test.com',
        password: 'pw',
      });
      expect(tokens.accessToken).toBe('acc');
      expect(tokens.refreshToken).toBe('ref');
    });
  });

  // ── activateLicense ───────────────────────────────────────────────────────

  describe('activateLicense', () => {
    it('sends auth header and returns JWT string', async () => {
      client.setTokens('my-access-token', 'my-refresh');
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeResponse({ license: 'eyJhbG.jwt.value' })),
      );

      const jwt = await client.activateLicense();

      const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toBe('http://test.local/api/v1/license/activate');
      expect((init.headers as Record<string, string>)['Authorization']).toBe(
        'Bearer my-access-token',
      );
      expect(jwt).toBe('eyJhbG.jwt.value');
    });
  });

  // ── refreshLicense ────────────────────────────────────────────────────────

  describe('refreshLicense', () => {
    it('sends body with current_license and returns new JWT', async () => {
      client.setTokens('token', 'refresh');
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeResponse({ license: 'new.jwt' })),
      );

      const newJwt = await client.refreshLicense('old.jwt');

      const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(JSON.parse(init.body as string)).toEqual({
        current_license: 'old.jwt',
      });
      expect(newJwt).toBe('new.jwt');
    });
  });

  // ── getUsage ──────────────────────────────────────────────────────────────

  describe('getUsage', () => {
    it('maps snake_case response to camelCase UsageInfo', async () => {
      client.setTokens('tok', 'ref');
      const responseData = {
        tier: 'personal',
        proxy_steps_used: 47,
        proxy_steps_limit: 300,
        proxy_steps_remaining: 253,
        byok_enabled: true,
        current_period_end: '2026-04-01T00:00:00Z',
      };
      fetchMock.mockReturnValueOnce(Promise.resolve(makeResponse(responseData)));

      const usage: UsageInfo = await client.getUsage();

      expect(usage).toEqual({
        tier: 'personal',
        proxyStepsUsed: 47,
        proxyStepsLimit: 300,
        proxyStepsRemaining: 253,
        byokEnabled: true,
        currentPeriodEnd: '2026-04-01T00:00:00Z',
      });
    });

    it('handles missing optional current_period_end', async () => {
      client.setTokens('tok', 'ref');
      const responseData = {
        tier: 'trial',
        proxy_steps_used: 0,
        proxy_steps_limit: 0,
        proxy_steps_remaining: 0,
        byok_enabled: false,
      };
      fetchMock.mockReturnValueOnce(Promise.resolve(makeResponse(responseData)));

      const usage = await client.getUsage();
      expect(usage.currentPeriodEnd).toBeUndefined();
    });
  });

  // ── error handling ────────────────────────────────────────────────────────

  describe('error handling', () => {
    it('throws CloudApiError with code and message on non-200 response', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(
          makeErrorResponse('INVALID_CREDENTIALS', 'Wrong password', 401),
        ),
      );

      let thrown: unknown;
      try {
        await client.login('x@y.com', 'wrong');
      } catch (e) {
        thrown = e;
      }

      expect(thrown).toBeInstanceOf(CloudApiError);
      const err = thrown as CloudApiError;
      expect(err.code).toBe('INVALID_CREDENTIALS');
      expect(err.message).toBe('Wrong password');
      expect(err.statusCode).toBe(401);
      expect(err.name).toBe('CloudApiError');
    });

    it('throws CloudApiError with UNKNOWN code when error field is missing', async () => {
      fetchMock.mockReturnValueOnce(
        Promise.resolve(
          new Response(JSON.stringify({}), { status: 500 }),
        ),
      );

      let thrown: unknown;
      try {
        await client.login('x@y.com', 'pw');
      } catch (e) {
        thrown = e;
      }

      expect(thrown).toBeInstanceOf(CloudApiError);
      const err = thrown as CloudApiError;
      expect(err.code).toBe('UNKNOWN');
      expect(err.statusCode).toBe(500);
    });
  });

  // ── createCheckout ────────────────────────────────────────────────────────

  describe('createCheckout', () => {
    it('sends plan and period, returns checkout URL', async () => {
      client.setTokens('tok', 'ref');
      fetchMock.mockReturnValueOnce(
        Promise.resolve(
          makeResponse({ checkout_url: 'https://checkout.stripe.com/sess_abc' }),
        ),
      );

      const url = await client.createCheckout('personal', 'monthly');

      const [reqUrl, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(reqUrl).toBe('http://test.local/api/v1/billing/checkout');
      expect(JSON.parse(init.body as string)).toEqual({
        plan: 'personal',
        period: 'monthly',
      });
      expect(url).toBe('https://checkout.stripe.com/sess_abc');
    });
  });

  // ── createPortal ──────────────────────────────────────────────────────────

  describe('createPortal', () => {
    it('sends empty body and returns portal URL', async () => {
      client.setTokens('tok', 'ref');
      fetchMock.mockReturnValueOnce(
        Promise.resolve(
          makeResponse({ portal_url: 'https://billing.stripe.com/portal_xyz' }),
        ),
      );

      const url = await client.createPortal();

      const [reqUrl] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(reqUrl).toBe('http://test.local/api/v1/billing/portal');
      expect(url).toBe('https://billing.stripe.com/portal_xyz');
    });
  });

  // ── auto-refresh on 401 ───────────────────────────────────────────────────

  describe('auto-refresh on 401', () => {
    it('retries with new access token after successful refresh on 401', async () => {
      client.setTokens('expired-token', 'some-refresh-token');

      // First call: 401 (triggers refresh)
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeErrorResponse('UNAUTHORIZED', 'Token expired', 401)),
      );
      // Second call: POST /auth/refresh → new access_token
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeResponse({ access_token: 'new-access-token' })),
      );
      // Third call: retry original request with new token → success
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeResponse({ license: 'renewed.jwt' })),
      );

      const jwt = await client.activateLicense();

      expect(jwt).toBe('renewed.jwt');
      expect(fetchMock).toHaveBeenCalledTimes(3);
      // Verify refresh was called
      const [refreshUrl] = fetchMock.mock.calls[1] as [string, RequestInit];
      expect(refreshUrl).toBe('http://test.local/api/v1/auth/refresh');
    });

    it('rethrows error when refresh endpoint also fails', async () => {
      client.setTokens('expired-token', 'some-refresh-token');

      // First call: 401
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeErrorResponse('UNAUTHORIZED', 'Token expired', 401)),
      );
      // Refresh call also fails with 401
      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeErrorResponse('INVALID_REFRESH', 'Refresh token expired', 401)),
      );

      let thrown: unknown;
      try {
        await client.activateLicense();
      } catch (e) {
        thrown = e;
      }

      expect(thrown).toBeInstanceOf(CloudApiError);
      const err = thrown as CloudApiError;
      expect(err.code).toBe('INVALID_REFRESH');
      expect(err.statusCode).toBe(401);
    });

    it('does not attempt refresh when no refresh token is set', async () => {
      // No tokens set at all → auth header absent, but let's also test 401 without refresh token
      client.setTokens('expired-token', '');
      // Override: remove refreshToken by creating client without refreshToken
      const clientNoRefresh = new CloudApiClient({ baseUrl: 'http://test.local' });
      clientNoRefresh.setTokens('expired', '');

      fetchMock.mockReturnValueOnce(
        Promise.resolve(makeErrorResponse('UNAUTHORIZED', 'No auth', 401)),
      );

      let thrown: unknown;
      try {
        // Directly call request via activateLicense — no refresh token, should throw original error
        await clientNoRefresh.activateLicense();
      } catch (e) {
        thrown = e;
      }

      // Without refreshToken, the 401 error propagates directly (no retry)
      expect(thrown).toBeInstanceOf(CloudApiError);
      const err = thrown as CloudApiError;
      expect(err.code).toBe('UNAUTHORIZED');
      // fetch called only once (no retry)
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });
  });
});
