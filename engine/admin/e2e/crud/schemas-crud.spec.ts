// §1.7 CRUD — Schemas: create, assign entry_agent, edit, delete
// TC: CRUD-03 | SCC-01

import { test, expect, apiFetch } from '../fixtures';

test.describe('Schemas CRUD', () => {
  test('create schema via API', async ({ request, adminToken }) => {
    const name = `test-schema-${Date.now()}`;
    const res = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name, chat_enabled: true },
    });
    expect([200, 201]).toContain(res.status());
    const body = await res.json();
    const id = body.id ?? body.name;
    expect(id).toBeTruthy();

    // Teardown
    await apiFetch(request, `/schemas/${id}`, { method: 'DELETE', token: adminToken });
  });

  test('schema appears in GET /schemas list', async ({ request, adminToken }) => {
    const name = `list-schema-${Date.now()}`;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name, chat_enabled: false },
    });
    const created = await createRes.json();
    const id = created.id ?? created.name;

    const listRes = await apiFetch(request, '/schemas', { token: adminToken });
    expect(listRes.status()).toBe(200);
    const body = await listRes.json();
    const schemas = Array.isArray(body) ? body : (body.schemas ?? body.data ?? []);
    expect(schemas.some((s: { id?: string; name?: string }) => s.id === id || s.name === name)).toBe(true);

    await apiFetch(request, `/schemas/${id}`, { method: 'DELETE', token: adminToken });
  });

  test('update schema name', async ({ request, adminToken }) => {
    const name = `upd-schema-${Date.now()}`;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name, chat_enabled: true },
    });
    const created = await createRes.json();
    const id = created.id ?? name;

    const updRes = await apiFetch(request, `/schemas/${id}`, {
      method: 'PUT',
      token: adminToken,
      body: { name: `${name}-upd`, chat_enabled: true },
    });
    expect([200, 204]).toContain(updRes.status());

    // Teardown
    await apiFetch(request, `/schemas/${id}`, { method: 'DELETE', token: adminToken });
  });

  test('delete schema removes it from list', async ({ request, adminToken }) => {
    const name = `del-schema-${Date.now()}`;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name },
    });
    const created = await createRes.json();
    const id = created.id ?? name;

    await apiFetch(request, `/schemas/${id}`, { method: 'DELETE', token: adminToken });

    const listRes = await apiFetch(request, '/schemas', { token: adminToken });
    const body = await listRes.json();
    const schemas = Array.isArray(body) ? body : (body.schemas ?? body.data ?? []);
    expect(schemas.some((s: { id?: string; name?: string }) => s.id === id || s.name === name)).toBe(false);
  });
});
