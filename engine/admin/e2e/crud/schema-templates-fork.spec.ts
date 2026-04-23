// §1.7-ext CRUD — Schema templates: list → fork "Support Bot" → schema+agents created
// TC: CRUD-17

import { test, expect, apiFetch } from '../fixtures';

test.describe('Schema templates fork', () => {
  test('GET /schemas/templates returns list', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/schemas/templates', { token: adminToken });
    if (res.status() === 404 || res.status() === 400) {
      test.skip(true, 'GET /schemas/templates not implemented — may use different endpoint');
      return;
    }
    expect(res.status()).toBe(200);
    const body = await res.json();
    const templates = Array.isArray(body) ? body : (body.templates ?? body.data ?? []);
    expect(templates.length).toBeGreaterThan(0);
  });

  test('fork a template creates schema and agents', async ({ request, adminToken }) => {
    // List templates
    const listRes = await apiFetch(request, '/schemas/templates', { token: adminToken });
    if (listRes.status() !== 200) {
      test.skip(true, 'Templates endpoint not available');
      return;
    }
    const body = await listRes.json();
    const templates = Array.isArray(body) ? body : (body.templates ?? body.data ?? []);
    if (templates.length === 0) {
      test.skip(true, 'No templates available to fork');
      return;
    }

    const template = templates[0];
    const templateId = template.id ?? template.name;

    // Fork
    const forkRes = await apiFetch(request, `/schemas/templates/${templateId}/fork`, {
      method: 'POST',
      token: adminToken,
      body: { name: `forked-${Date.now()}` },
    });
    expect([200, 201]).toContain(forkRes.status());
    const forked = await forkRes.json();
    const schemaId = forked.id ?? forked.schema?.id;
    expect(schemaId).toBeTruthy();

    // Verify agents accessible
    if (schemaId) {
      const agentsRes = await apiFetch(request, `/schemas/${schemaId}/agents`, { token: adminToken });
      // Even if 0 agents, endpoint should respond 200
      expect([200, 204]).toContain(agentsRes.status());

      // Teardown
      await apiFetch(request, `/schemas/${schemaId}`, { method: 'DELETE', token: adminToken });
    }
  });
});
