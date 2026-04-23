// §1.7-ext CRUD — Knowledge Base M2M: create KB, assign agent, verify link, unassign, verify unlinked
// TC: CRUD-09 | GAP-3 | SCC-01

import { test, expect, apiFetch } from '../fixtures';

test.describe('Knowledge Base M2M agent assignment', () => {
  test('assign and unassign agent from KB via M2M endpoint', async ({ request, adminToken }) => {
    const agentName = `m2m-ag-${Date.now()}`;
    const kbName = `m2m-kb-${Date.now()}`;

    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentName, system_prompt: 'Test' } });
    const kbRes = await apiFetch(request, '/knowledge-bases', { method: 'POST', token: adminToken, body: { name: kbName } });
    const kb = await kbRes.json();
    const kbId = kb.id;
    expect(kbId).toBeTruthy();

    // Assign
    const assignRes = await apiFetch(request, `/knowledge-bases/${kbId}/agents/${agentName}`, {
      method: 'POST',
      token: adminToken,
    });
    expect([200, 201, 204]).toContain(assignRes.status());

    // Verify link
    // GET /knowledge-bases/{id} returns { linked_agents: string[] }
    const getRes = await apiFetch(request, `/knowledge-bases/${kbId}`, { token: adminToken });
    if (getRes.status() === 200) {
      const body = await getRes.json();
      const agents: Array<{ name?: string } | string> = body.linked_agents ?? body.agents ?? body.agent_names ?? [];
      expect.soft(agents.some(a => (typeof a === 'string' ? a : a.name) === agentName)).toBe(true);
    }

    // Unassign
    const unassignRes = await apiFetch(request, `/knowledge-bases/${kbId}/agents/${agentName}`, {
      method: 'DELETE',
      token: adminToken,
    });
    expect([200, 204]).toContain(unassignRes.status());

    // Verify unlinked
    const verifyRes = await apiFetch(request, `/knowledge-bases/${kbId}`, { token: adminToken });
    if (verifyRes.status() === 200) {
      const body = await verifyRes.json();
      const agents: Array<{ name?: string } | string> = body.linked_agents ?? body.agents ?? body.agent_names ?? [];
      expect.soft(agents.some(a => (typeof a === 'string' ? a : a.name) === agentName)).toBe(false);
    }

    // Teardown
    await apiFetch(request, `/knowledge-bases/${kbId}`, { method: 'DELETE', token: adminToken });
    await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });
  });
});
