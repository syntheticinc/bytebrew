// §1.12 API Keys — 5 parallel POST /auth/tokens with same name → document behavior (unique or one wins)
// TC: KEY-04

import { test, expect, apiFetch } from '../fixtures';

test.describe('API Keys — concurrent create', () => {
  test('5 parallel POST /auth/tokens succeed or return consistent errors', async ({ request, adminToken }) => {
    const name = `concurrent-key-${Date.now()}`;

    // Fire 5 parallel requests
    const results = await Promise.all(
      Array.from({ length: 5 }).map(() =>
        apiFetch(request, '/auth/tokens', {
          method: 'POST',
          token: adminToken,
          body: { name, scopes: ['api'] },
        })
      )
    );

    const statuses = await Promise.all(results.map(r => r.status()));
    const bodies = await Promise.all(results.map(r => r.json().catch(() => null)));

    // Acceptable: all succeed (201) or some fail (409 duplicate) — never 500
    for (const status of statuses) {
      expect(status).not.toBe(500);
      expect([200, 201, 409, 422]).toContain(status);
    }

    // Cleanup: delete all successfully created keys
    for (const body of bodies) {
      if (body?.id) {
        await apiFetch(request, `/auth/tokens/${body.id}`, { method: 'DELETE', token: adminToken });
      }
    }
  });
});
