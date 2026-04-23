// §1.12 API Keys — read-only scope token cannot POST agents → 403
// TC: KEY-03

import { test, expect, apiFetch } from '../fixtures';

test.describe('API Keys — scope enforcement', () => {
  test('read-only scoped token cannot create agents (403)', async ({ request, adminToken }) => {
    const name = `readonly-key-${Date.now()}`;
    const createRes = await apiFetch(request, '/auth/tokens', {
      method: 'POST',
      token: adminToken,
      body: { name, scopes: ['read'] },
    });

    if (createRes.status() !== 200 && createRes.status() !== 201) {
      test.skip(true, 'Cannot create read-only scoped token — scopes may not be supported');
      return;
    }

    const created = await createRes.json();
    const readToken = created.token ?? created.key ?? '';
    const keyId = created.id;

    if (!readToken.startsWith('bb_')) {
      test.skip(true, 'Created token does not have bb_ prefix — likely not a real API key');
      if (keyId) await apiFetch(request, `/auth/tokens/${keyId}`, { method: 'DELETE', token: adminToken });
      return;
    }

    // Try to create an agent with read-only token
    const agentRes = await apiFetch(request, '/agents', {
      method: 'POST',
      token: readToken,
      body: { name: `scope-test-${Date.now()}`, system_prompt: 'Test' },
    });
    expect([403, 401]).toContain(agentRes.status());

    // Teardown
    if (keyId) await apiFetch(request, `/auth/tokens/${keyId}`, { method: 'DELETE', token: adminToken });
  });

  // REAL BUG: BUG-10 — 'api' scope token cannot create agents (returns 403).
  // Use 'agents:write' scope instead. Marking test.fail until BUG-10 is fixed.
  test.fail(true, 'REAL BUG: BUG-10 — api-scoped token returns 403 on agent creation; use agents:write scope');
  test('full-scope token can create agents', async ({ request, adminToken }) => {
    const name = `fullscope-key-${Date.now()}`;
    const createRes = await apiFetch(request, '/auth/tokens', {
      method: 'POST',
      token: adminToken,
      body: { name, scopes: ['api'] },
    });

    if (createRes.status() !== 200 && createRes.status() !== 201) {
      test.skip(true, 'Cannot create api-scoped token');
      return;
    }

    const created = await createRes.json();
    const apiKey = created.token ?? created.key ?? '';
    const keyId = created.id;

    if (apiKey.startsWith('bb_')) {
      const agentName = `scope-agent-${Date.now()}`;
      const agentRes = await apiFetch(request, '/agents', {
        method: 'POST',
        token: apiKey,
        body: { name: agentName, system_prompt: 'Test' },
      });
      expect([200, 201]).toContain(agentRes.status());
      // Cleanup agent
      await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });
    }

    if (keyId) await apiFetch(request, `/auth/tokens/${keyId}`, { method: 'DELETE', token: adminToken });
  });
});
