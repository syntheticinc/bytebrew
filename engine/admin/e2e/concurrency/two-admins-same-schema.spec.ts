// §1.21 Concurrency — two admins edit same schema simultaneously → last-write-wins or 409
// TC: CON-01 | GAP-7

import { test, expect, apiFetch } from '../fixtures';

test.describe('Concurrency — two admins same schema', () => {
  test('two concurrent PUT /schemas/{id} → one wins, no 500', async ({ request, adminToken }) => {
    // Create schema
    const name = `concurrent-schema-${Date.now()}`;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name, chat_enabled: true },
    });
    expect([200, 201]).toContain(createRes.status());
    const created = await createRes.json();
    const id = created.id ?? name;

    // Fire two concurrent updates
    const [res1, res2] = await Promise.all([
      apiFetch(request, `/schemas/${id}`, {
        method: 'PUT',
        token: adminToken,
        body: { name: `${name}-update-a`, chat_enabled: true },
      }),
      apiFetch(request, `/schemas/${id}`, {
        method: 'PUT',
        token: adminToken,
        body: { name: `${name}-update-b`, chat_enabled: false },
      }),
    ]);

    // Neither should 500 — one wins (200/204) or optimistic lock (409)
    expect.soft(res1.status()).not.toBe(500);
    expect.soft(res2.status()).not.toBe(500);
    expect([200, 204, 409]).toContain(res1.status());
    expect([200, 204, 409]).toContain(res2.status());

    // Cleanup
    await apiFetch(request, `/schemas/${id}`, { method: 'DELETE', token: adminToken });
  });

  test('two admins simultaneously view same schema — both get 200', async ({ request, adminToken }) => {
    const listRes = await apiFetch(request, '/schemas', { token: adminToken });
    if (listRes.status() !== 200) {
      test.skip(true, 'Schemas endpoint not available');
      return;
    }
    const body = await listRes.json();
    const schemas = Array.isArray(body) ? body : (body.schemas ?? body.data ?? []);
    if (schemas.length === 0) {
      test.skip(true, 'No schemas to test concurrent reads');
      return;
    }

    const schemaId = schemas[0].id;
    const [r1, r2] = await Promise.all([
      apiFetch(request, `/schemas/${schemaId}`, { token: adminToken }),
      apiFetch(request, `/schemas/${schemaId}`, { token: adminToken }),
    ]);

    expect(r1.status()).toBe(200);
    expect(r2.status()).toBe(200);
  });
});
