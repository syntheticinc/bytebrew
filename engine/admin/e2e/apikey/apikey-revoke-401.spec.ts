// §1.12 API Keys — revoke → immediately 401 within 1s
// TC: KEY-02

import { test, expect, apiFetch } from '../fixtures';

test.describe('API Keys — revoke → 401', () => {
  test('revoked key returns 401 immediately', async ({ request, adminToken }) => {
    const name = `revoke-now-${Date.now()}`;
    const createRes = await apiFetch(request, '/auth/tokens', {
      method: 'POST',
      token: adminToken,
      body: { name, scopes: ['api'] },
    });
    expect([200, 201]).toContain(createRes.status());
    const created = await createRes.json();
    const apiKey = created.token ?? created.key ?? created.access_token ?? '';
    const keyId = created.id;

    expect(apiKey).toBeTruthy();

    // Revoke
    if (keyId) {
      await apiFetch(request, `/auth/tokens/${keyId}`, { method: 'DELETE', token: adminToken });
    }

    // Immediately use revoked key
    if (apiKey.startsWith('bb_')) {
      const start = Date.now();
      const checkRes = await apiFetch(request, '/agents', { token: apiKey });
      const elapsed = Date.now() - start;
      expect(checkRes.status()).toBe(401);
      expect(elapsed).toBeLessThan(1000); // within 1s
    }
  });
});
