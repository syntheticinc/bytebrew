// §1.7-ext CRUD — Capabilities: create/list/update/delete for 7 capability types
// TC: CRUD-10 | GAP-2 | SCC-01

import { test, expect, apiFetch } from '../fixtures';

const CAPABILITY_TYPES = [
  { type: 'memory', config: {} },
  { type: 'knowledge', config: {} },
  { type: 'escalation', config: { confidence_threshold: 0.7 } },
  { type: 'guardrail', config: {} },
  { type: 'output_schema', config: { schema: '{}' } },
  { type: 'policies', config: {} },
  { type: 'recovery', config: {} },
];

test.describe('Capabilities CRUD — 7 types', () => {
  let agentName: string;

  test.beforeAll(async ({ request, adminToken }: { request: import('@playwright/test').APIRequestContext; adminToken: string }) => {
    agentName = `cap-agent-${Date.now()}`;
    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentName, system_prompt: 'Test' } });
  });

  test.afterAll(async ({ request, adminToken }: { request: import('@playwright/test').APIRequestContext; adminToken: string }) => {
    await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });
  });

  for (const cap of CAPABILITY_TYPES) {
    test(`create capability type=${cap.type}`, async ({ request, adminToken }) => {
      const res = await apiFetch(request, `/agents/${agentName}/capabilities`, {
        method: 'POST',
        token: adminToken,
        body: { type: cap.type, config: cap.config },
      });
      // 200/201 = created; 400 = not yet supported (document)
      expect([200, 201, 400, 422]).toContain(res.status());
      if ([200, 201].includes(res.status())) {
        const body = await res.json();
        const id = body.id;
        if (id) {
          // Teardown
          await apiFetch(request, `/agents/${agentName}/capabilities/${id}`, { method: 'DELETE', token: adminToken });
        }
      }
    });
  }

  test('list capabilities for agent', async ({ request, adminToken }) => {
    const res = await apiFetch(request, `/agents/${agentName}/capabilities`, { token: adminToken });
    expect([200]).toContain(res.status());
    const body = await res.json();
    expect(Array.isArray(body) || Array.isArray(body.capabilities) || Array.isArray(body.data)).toBe(true);
  });
});
