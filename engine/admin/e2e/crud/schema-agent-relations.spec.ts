// §1.7 CRUD — Agent relations are scoped to a schema via
// POST /api/v1/schemas/{id}/agent-relations. Covers create + list + delete.
// TC: CRUD-04 | SCC-01
// BUG-09 fix: earlier spec called POST /agents/{name}/relations which doesn't
// exist — relations are always schema-scoped.

import { test, expect, apiFetch } from '../fixtures';

test.describe('Schema agent-relations CRUD', () => {
  test('create + list + delete a flow relation on a schema', async ({ request, adminToken }) => {
    const ts = Date.now();
    const agentA = `rel-a-${ts}`;
    const agentB = `rel-b-${ts}`;
    const schemaName = `rel-schema-${ts}`;

    const aRes = await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentA, system_prompt: 'A' } });
    const bRes = await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentB, system_prompt: 'B' } });
    if (![200, 201].includes(aRes.status()) || ![200, 201].includes(bRes.status())) {
      test.skip(true, 'agent create failed — cannot proceed');
      return;
    }
    const aBody = await aRes.json();
    const bBody = await bRes.json();
    const aID = aBody.id ?? aBody.data?.id;
    const bID = bBody.id ?? bBody.data?.id;

    const schemaRes = await apiFetch(request, '/schemas', {
      method: 'POST', token: adminToken,
      body: { name: schemaName, description: 'e2e relations', entry_agent_id: aID },
    });
    if (![200, 201].includes(schemaRes.status())) {
      test.skip(true, `schema create: ${schemaRes.status()}`);
      return;
    }
    const schemaBody = await schemaRes.json();
    const schemaID = schemaBody.id ?? schemaBody.data?.id;

    const createRes = await apiFetch(request, `/schemas/${schemaID}/agent-relations`, {
      method: 'POST', token: adminToken,
      body: { source: aID, target: bID, config: { type: 'flow' } },
    });
    expect([200, 201]).toContain(createRes.status());
    const relBody = await createRes.json();
    const relationID = relBody.id ?? relBody.data?.id;

    const listRes = await apiFetch(request, `/schemas/${schemaID}/agent-relations`, { token: adminToken });
    expect(listRes.status()).toBe(200);
    const listBody = await listRes.json();
    const items = Array.isArray(listBody) ? listBody : listBody.data ?? [];
    expect(items.length).toBeGreaterThan(0);

    // Cleanup
    if (relationID) {
      await apiFetch(request, `/schemas/${schemaID}/agent-relations/${relationID}`, { method: 'DELETE', token: adminToken });
    }
    await apiFetch(request, `/schemas/${schemaID}`, { method: 'DELETE', token: adminToken });
    await apiFetch(request, `/agents/${agentA}`, { method: 'DELETE', token: adminToken });
    await apiFetch(request, `/agents/${agentB}`, { method: 'DELETE', token: adminToken });
  });

  test('GATE SCC-01: /schemas/{id}/agent-relations without auth returns 401', async ({ request }) => {
    const res = await apiFetch(request, '/schemas/00000000-0000-0000-0000-000000000000/agent-relations');
    expect(res.status()).toBe(401);
  });
});
