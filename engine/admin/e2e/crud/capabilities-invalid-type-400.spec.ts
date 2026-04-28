// §1.7-ext CRUD — Capabilities: POST with type="foo" → 400 (BUG-001 regression)
// TC: CRUD-11 | GAP-2 | SCC-01

import { test, expect, apiFetch } from '../fixtures';

test.describe('Capabilities — invalid type rejected (BUG-001 regression)', () => {
  let agentName: string;

  test.beforeAll(async ({ request, adminToken }: { request: import('@playwright/test').APIRequestContext; adminToken: string }) => {
    agentName = `bug001-agent-${Date.now()}`;
    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentName, system_prompt: 'Test' } });
  });

  test.afterAll(async ({ request, adminToken }: { request: import('@playwright/test').APIRequestContext; adminToken: string }) => {
    await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });
  });

  test('POST capability with unknown type returns 400 not 500', async ({ request, adminToken }) => {
    const res = await apiFetch(request, `/agents/${agentName}/capabilities`, {
      method: 'POST',
      token: adminToken,
      body: { type: 'totally_invalid_type_xyz', config: {} },
    });
    // BUG-001: was returning 500 — should be 400
    expect(res.status()).toBe(400);
  });

  test('POST capability with empty type returns 400', async ({ request, adminToken }) => {
    const res = await apiFetch(request, `/agents/${agentName}/capabilities`, {
      method: 'POST',
      token: adminToken,
      body: { type: '', config: {} },
    });
    expect([400, 422]).toContain(res.status());
  });

  test('POST capability with null type returns 400', async ({ request, adminToken }) => {
    const res = await apiFetch(request, `/agents/${agentName}/capabilities`, {
      method: 'POST',
      token: adminToken,
      body: { type: null, config: {} },
    });
    expect([400, 422]).toContain(res.status());
  });
});
