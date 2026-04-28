// §1.7 CRUD — API Keys: create with scopes → bb_* token shown once; revoke → 401
// TC: CRUD-07 | SCC-01

import { test, expect, ENGINE_API, apiFetch } from '../fixtures';

test.describe('API Keys CRUD', () => {
  test('create API key returns bb_* token', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/auth/tokens', {
      method: 'POST',
      token: adminToken,
      body: { name: `test-key-${Date.now()}`, scopes: ['api'] },
    });
    expect([200, 201]).toContain(res.status());
    const body = await res.json();
    const token = body.token ?? body.key ?? body.access_token ?? body.api_key ?? '';
    expect(token).toMatch(/^bb_/);
  });

  test('revoked API key returns 401 on subsequent use', async ({ request, adminToken }) => {
    const name = `revoke-key-${Date.now()}`;
    const createRes = await apiFetch(request, '/auth/tokens', {
      method: 'POST',
      token: adminToken,
      body: { name, scopes: ['api'] },
    });
    const created = await createRes.json();
    const apiKey = created.token ?? created.key ?? created.access_token ?? created.api_key ?? '';
    const keyId = created.id;

    expect(apiKey).toBeTruthy();

    // Revoke
    if (keyId) {
      const revokeRes = await apiFetch(request, `/auth/tokens/${keyId}`, {
        method: 'DELETE',
        token: adminToken,
      });
      expect([200, 204]).toContain(revokeRes.status());
    }

    // Use revoked key — should 401
    if (apiKey.startsWith('bb_')) {
      const checkRes = await apiFetch(request, '/agents', { token: apiKey });
      expect(checkRes.status()).toBe(401);
    }
  });

  test('GET /auth/tokens lists existing keys without exposing secrets', async ({ request, adminToken }) => {
    const listRes = await apiFetch(request, '/auth/tokens', { token: adminToken });
    expect(listRes.status()).toBe(200);
    const body = await listRes.json();
    const tokens = Array.isArray(body) ? body : (body.tokens ?? body.data ?? []);
    expect(Array.isArray(tokens)).toBe(true);
    // Secrets must not be in the list response
    for (const t of tokens) {
      const hasSecret = typeof t.token === 'string' && t.token.length > 10;
      expect.soft(hasSecret).toBe(false);
    }
  });
});
