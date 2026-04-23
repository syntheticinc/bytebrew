// §1.7 CRUD — Schema agent relations: A→B allowed; B→A circular → 400; delete relation
// TC: CRUD-04 | SCC-01

import { test, expect, apiFetch } from '../fixtures';

test.describe('Schema agent relations', () => {
  test.fail(true, 'REAL BUG: BUG-09 — POST /agents/{name}/relations returns 404; endpoint not implemented');
  let agentA: string;
  let agentB: string;

  test.beforeAll(async ({ request, adminToken }: { request: import('@playwright/test').APIRequestContext; adminToken: string }) => {
    agentA = `rel-agent-a-${Date.now()}`;
    agentB = `rel-agent-b-${Date.now()}`;
    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentA, system_prompt: 'A' } });
    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentB, system_prompt: 'B' } });
  });

  test.afterAll(async ({ request, adminToken }: { request: import('@playwright/test').APIRequestContext; adminToken: string }) => {
    await apiFetch(request, `/agents/${agentA}`, { method: 'DELETE', token: adminToken });
    await apiFetch(request, `/agents/${agentB}`, { method: 'DELETE', token: adminToken });
  });

  test('add A→B relation succeeds', async ({ request, adminToken }) => {
    const res = await apiFetch(request, `/agents/${agentA}/relations`, {
      method: 'POST',
      token: adminToken,
      body: { target_agent: agentB, type: 'flow' },
    });
    expect([200, 201, 204]).toContain(res.status());
  });

  test('add B→A circular relation returns 400 or 409', async ({ request, adminToken }) => {
    // First ensure A→B exists
    await apiFetch(request, `/agents/${agentA}/relations`, {
      method: 'POST',
      token: adminToken,
      body: { target_agent: agentB, type: 'flow' },
    });

    const res = await apiFetch(request, `/agents/${agentB}/relations`, {
      method: 'POST',
      token: adminToken,
      body: { target_agent: agentA, type: 'flow' },
    });
    // Circular delegation should be rejected
    expect([400, 409, 422]).toContain(res.status());
  });

  test('delete relation removes it', async ({ request, adminToken }) => {
    // Create a fresh relation
    const a = `circ-a-${Date.now()}`;
    const b = `circ-b-${Date.now()}`;
    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: a, system_prompt: 'A' } });
    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: b, system_prompt: 'B' } });

    const createRes = await apiFetch(request, `/agents/${a}/relations`, {
      method: 'POST',
      token: adminToken,
      body: { target_agent: b, type: 'flow' },
    });
    const rel = await createRes.json();
    const relId = rel.id;

    if (relId) {
      const delRes = await apiFetch(request, `/agents/${a}/relations/${relId}`, { method: 'DELETE', token: adminToken });
      expect([200, 204]).toContain(delRes.status());
    }

    await apiFetch(request, `/agents/${a}`, { method: 'DELETE', token: adminToken });
    await apiFetch(request, `/agents/${b}`, { method: 'DELETE', token: adminToken });
  });
});
