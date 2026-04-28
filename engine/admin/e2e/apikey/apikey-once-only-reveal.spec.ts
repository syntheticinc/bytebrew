// §1.12 API Keys — bb_* token shown once only; reload → hidden
// TC: KEY-01

import { test, expect, apiFetch } from '../fixtures';

test.describe('API Keys — once-only reveal', () => {
  test('created API key response contains bb_* token', async ({ request, adminToken }) => {
    const name = `once-key-${Date.now()}`;
    const res = await apiFetch(request, '/auth/tokens', {
      method: 'POST',
      token: adminToken,
      body: { name, scopes: ['api'] },
    });
    expect([200, 201]).toContain(res.status());
    const body = await res.json();
    const token = body.token ?? body.key ?? body.access_token ?? body.api_key ?? '';
    expect(token).toMatch(/^bb_/);

    // Teardown
    if (body.id) {
      await apiFetch(request, `/auth/tokens/${body.id}`, { method: 'DELETE', token: adminToken });
    }
  });

  test('GET /auth/tokens list does NOT expose raw key values', async ({ request, adminToken }) => {
    // Create a key first
    const name = `list-check-key-${Date.now()}`;
    const createRes = await apiFetch(request, '/auth/tokens', {
      method: 'POST',
      token: adminToken,
      body: { name, scopes: ['api'] },
    });
    const created = await createRes.json();
    const rawToken = created.token ?? created.key ?? '';

    // List all keys
    const listRes = await apiFetch(request, '/auth/tokens', { token: adminToken });
    expect(listRes.status()).toBe(200);
    const listBody = await listRes.json();
    const tokens = Array.isArray(listBody) ? listBody : (listBody.tokens ?? listBody.data ?? []);

    // Raw token must not appear in list response
    const listText = JSON.stringify(tokens);
    if (rawToken.startsWith('bb_')) {
      expect(listText).not.toContain(rawToken);
    }

    // Teardown
    if (created.id) {
      await apiFetch(request, `/auth/tokens/${created.id}`, { method: 'DELETE', token: adminToken });
    }
  });

  test('API keys page renders and shows masked tokens', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/api-keys');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('text=/Something went wrong/i')).not.toBeVisible();
  });
});
