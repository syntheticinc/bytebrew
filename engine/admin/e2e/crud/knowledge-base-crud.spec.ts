// §1.7 CRUD — Knowledge Bases: create KB, upload file, assign to agent, verify M2M, remove
// TC: CRUD-06 | SCC-01

import { test, expect, apiFetch } from '../fixtures';

test.describe('Knowledge Base CRUD', () => {
  test('create knowledge base via API', async ({ request, adminToken }) => {
    const name = `test-kb-${Date.now()}`;
    const res = await apiFetch(request, '/knowledge-bases', {
      method: 'POST',
      token: adminToken,
      body: { name, description: 'Test KB' },
    });
    expect([200, 201]).toContain(res.status());
    const body = await res.json();
    const id = body.id;
    expect(id).toBeTruthy();

    await apiFetch(request, `/knowledge-bases/${id}`, { method: 'DELETE', token: adminToken });
  });

  test('KB appears in GET /knowledge-bases', async ({ request, adminToken }) => {
    const name = `list-kb-${Date.now()}`;
    const createRes = await apiFetch(request, '/knowledge-bases', {
      method: 'POST',
      token: adminToken,
      body: { name },
    });
    const created = await createRes.json();
    const id = created.id;

    const listRes = await apiFetch(request, '/knowledge-bases', { token: adminToken });
    expect(listRes.status()).toBe(200);
    const body = await listRes.json();
    const kbs = Array.isArray(body) ? body : (body.knowledge_bases ?? body.data ?? []);
    expect(kbs.some((k: { id: string }) => k.id === id)).toBe(true);

    await apiFetch(request, `/knowledge-bases/${id}`, { method: 'DELETE', token: adminToken });
  });

  test('assign KB to agent via M2M endpoint', async ({ request, adminToken }) => {
    const kbName = `m2m-kb-${Date.now()}`;
    const agentName = `m2m-agent-${Date.now()}`;

    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentName, system_prompt: 'Test' } });
    const kbRes = await apiFetch(request, '/knowledge-bases', { method: 'POST', token: adminToken, body: { name: kbName } });
    const kb = await kbRes.json();
    const kbId = kb.id;

    // Assign KB to agent
    const assignRes = await apiFetch(request, `/knowledge-bases/${kbId}/agents/${agentName}`, {
      method: 'POST',
      token: adminToken,
    });
    expect([200, 201, 204]).toContain(assignRes.status());

    // Teardown
    await apiFetch(request, `/knowledge-bases/${kbId}/agents/${agentName}`, { method: 'DELETE', token: adminToken });
    await apiFetch(request, `/knowledge-bases/${kbId}`, { method: 'DELETE', token: adminToken });
    await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });
  });
});
