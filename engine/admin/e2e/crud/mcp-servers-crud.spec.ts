// §1.7 CRUD — MCP Servers: add http MCP server, verify, delete
// TC: CRUD-05 | SCC-01

import { test, expect, apiFetch } from '../fixtures';

test.describe('MCP Servers CRUD', () => {
  test('create http MCP server via API', async ({ request, adminToken }) => {
    const name = `test-mcp-${Date.now()}`;
    const res = await apiFetch(request, '/mcp-servers', {
      method: 'POST',
      token: adminToken,
      body: {
        name,
        transport: 'http',
        url: 'http://localhost:9999/mcp',
        description: 'Test MCP server',
      },
    });
    expect([200, 201]).toContain(res.status());
    const body = await res.json();
    expect(body.name ?? body.id).toBeTruthy();

    // Teardown
    const id = body.id ?? body.name ?? name;
    await apiFetch(request, `/mcp-servers/${id}`, { method: 'DELETE', token: adminToken });
  });

  test('MCP server appears in GET /mcp-servers', async ({ request, adminToken }) => {
    const name = `list-mcp-${Date.now()}`;
    const createRes = await apiFetch(request, '/mcp-servers', {
      method: 'POST',
      token: adminToken,
      body: { name, transport: 'http', url: 'http://localhost:9999/mcp' },
    });
    const created = await createRes.json();
    const id = created.id ?? created.name ?? name;

    const listRes = await apiFetch(request, '/mcp-servers', { token: adminToken });
    expect(listRes.status()).toBe(200);
    const body = await listRes.json();
    const servers = Array.isArray(body) ? body : (body.servers ?? body.mcp_servers ?? body.data ?? []);
    expect(servers.some((s: { name?: string; id?: string }) => s.name === name || s.id === id)).toBe(true);

    await apiFetch(request, `/mcp-servers/${id}`, { method: 'DELETE', token: adminToken });
  });

  test('delete MCP server removes it', async ({ request, adminToken }) => {
    const name = `del-mcp-${Date.now()}`;
    const createRes = await apiFetch(request, '/mcp-servers', {
      method: 'POST',
      token: adminToken,
      body: { name, transport: 'http', url: 'http://localhost:9999/mcp' },
    });
    const created = await createRes.json();
    const id = created.id ?? created.name ?? name;

    await apiFetch(request, `/mcp-servers/${id}`, { method: 'DELETE', token: adminToken });

    const listRes = await apiFetch(request, '/mcp-servers', { token: adminToken });
    const body = await listRes.json();
    const servers = Array.isArray(body) ? body : (body.servers ?? body.data ?? []);
    expect(servers.some((s: { name?: string; id?: string }) => s.name === name || s.id === id)).toBe(false);
  });
});
