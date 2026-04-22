// §1.7-ext CRUD — Agent delete with FK dependencies: cascade or friendly error, no 500 (BUG-004 regression)
// TC: CRUD-14 | GAP-2

import { test, expect, apiFetch } from '../fixtures';

test.describe('Agent delete with FK dependencies (BUG-004 regression)', () => {
  test('delete agent with capabilities returns 200/204 or friendly 409, never 500', async ({ request, adminToken }) => {
    const agentName = `bug004-agent-${Date.now()}`;

    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentName, system_prompt: 'Test' } });

    // Add a capability
    await apiFetch(request, `/agents/${agentName}/capabilities`, {
      method: 'POST',
      token: adminToken,
      body: { type: 'memory', config: {} },
    });

    // Attempt delete
    const delRes = await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });

    // BUG-004: was returning 500 — should be 200/204 (cascade) or 409 (friendly error)
    expect([200, 204, 409]).toContain(delRes.status());
    expect(delRes.status()).not.toBe(500);

    // Cleanup if still exists
    await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });
  });
});
