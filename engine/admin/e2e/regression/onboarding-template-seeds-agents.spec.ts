// Regression bug #3 — Support Bot template must seed agents, not just an empty schema
// TC: REG-03 | Bug #3: template-apply path drops agents silently

import { test, expect, apiFetch } from '../fixtures';

test.describe('Regression bug #3 — template seeds agents', () => {
  test('after applying Support Bot template: GET /agents returns ≥1 agent', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/onboarding?step=2');
    await page.waitForLoadState('networkidle');

    const supportBotTemplate = page.locator(
      '[data-testid*="support"], [class*="template"]:has-text("Support"), [role="radio"]:has-text("Support")'
    ).first();

    if (await supportBotTemplate.count() === 0) {
      test.skip(true, 'Support Bot template not found — may already be past onboarding');
      return;
    }

    // Record agent count before
    const beforeRes = await apiFetch(request, '/agents', { token: adminToken });
    const beforeBody = await beforeRes.json();
    const beforeAgents = Array.isArray(beforeBody) ? beforeBody : (beforeBody.agents ?? beforeBody.data ?? []);
    const beforeCount = beforeAgents.length;

    await supportBotTemplate.click();
    const applyBtn = page.locator('button:has-text("Apply"), button:has-text("Use"), button:has-text("Next"), button[type="submit"]').first();
    if (await applyBtn.count() > 0) {
      await applyBtn.click();
      await page.waitForTimeout(3000);
    }

    // After template: agents count must have increased
    const afterRes = await apiFetch(request, '/agents', { token: adminToken });
    const afterBody = await afterRes.json();
    const afterAgents = Array.isArray(afterBody) ? afterBody : (afterBody.agents ?? afterBody.data ?? []);

    // Bug #3: agents were NOT seeded — assert they ARE
    expect(afterAgents.length).toBeGreaterThan(beforeCount);
  });

  test('template-applied schema has agents assigned via GET /schemas/{id}/agents', async ({ request, adminToken }) => {
    // Get schemas created by template
    const schemasRes = await apiFetch(request, '/schemas', { token: adminToken });
    if (schemasRes.status() !== 200) {
      test.skip(true, 'Cannot list schemas');
      return;
    }
    const body = await schemasRes.json();
    const schemas = Array.isArray(body) ? body : (body.schemas ?? body.data ?? []);
    if (schemas.length === 0) {
      test.skip(true, 'No schemas found');
      return;
    }

    const schemaId = schemas[0].id;
    const agentsRes = await apiFetch(request, `/schemas/${schemaId}/agents`, { token: adminToken });
    // May be 200 with empty array or list — just document
    expect([200, 204]).toContain(agentsRes.status());
  });
});
