// §1.19 SCC-02 — cross-tenant: Tenant A resource; Tenant B access → 403/404 GATE
// TC: SCC-02 GATE | GAP-5

import { test, expect, apiFetch } from '../fixtures';

test.describe('SCC-02 — cross-tenant isolation', () => {
  test.skip(true, 'SCC-02 full sweep requires two separate tenant tokens. Current test stack has single admin tenant. Run with multi-tenant compose or add second tenant token to env.');

  test('⛔ GATE SCC-02: agent created by tenant A not visible to tenant B', async ({ request, adminToken }) => {
    // Tenant A: create agent
    const agentName = `scc02-agent-${Date.now()}`;
    await apiFetch(request, '/agents', {
      method: 'POST',
      token: adminToken,
      body: { name: agentName, system_prompt: 'Tenant A agent' },
    });

    // Tenant B: attempt to GET (using a different token from env or fixture)
    const tenantBToken = process.env.TENANT_B_TOKEN ?? '';
    if (!tenantBToken) {
      test.skip(true, 'TENANT_B_TOKEN not set in environment');
      return;
    }

    const res = await apiFetch(request, `/agents/${agentName}`, { token: tenantBToken });
    expect([403, 404]).toContain(res.status());

    // Cleanup
    await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });
  });

  test('⛔ GATE SCC-02: schema created by tenant A not listed by tenant B', async ({ request, adminToken }) => {
    const schemaName = `scc02-schema-${Date.now()}`;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name: schemaName },
    });
    const created = await createRes.json();
    const schemaId = created.id;

    const tenantBToken = process.env.TENANT_B_TOKEN ?? '';
    if (!tenantBToken) {
      test.skip(true, 'TENANT_B_TOKEN not set');
      if (schemaId) await apiFetch(request, `/schemas/${schemaId}`, { method: 'DELETE', token: adminToken });
      return;
    }

    const listRes = await apiFetch(request, '/schemas', { token: tenantBToken });
    const body = await listRes.json();
    const schemas = Array.isArray(body) ? body : (body.schemas ?? body.data ?? []);
    expect(schemas.some((s: { id?: string }) => s.id === schemaId)).toBe(false);

    if (schemaId) await apiFetch(request, `/schemas/${schemaId}`, { method: 'DELETE', token: adminToken });
  });
});
