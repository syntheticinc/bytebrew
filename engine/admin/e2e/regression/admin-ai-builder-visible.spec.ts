// Regression bug #2 — AI Builder tab must be accessible from schema canvas
// TC: REG-02

import { test, expect, apiFetch } from '../fixtures';

test.describe('Regression bug #2 — AI Builder visible on schema page', () => {
  test('AI Builder tab accessible from schema detail page', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;

    const name = `reg02-schema-${Date.now()}`;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name, chat_enabled: true },
    });
    const created = await createRes.json();
    const schemaId = created.id ?? name;

    await page.goto(`/admin/schemas/${schemaId}`);
    await page.waitForLoadState('networkidle');

    // Bug #2: AI Builder tab was missing after prototype→production migration
    const aiBuilderTab = page.locator(
      'button:has-text("AI Builder"), button:has-text("Builder"), [data-testid*="ai-builder"], [role="tab"]:has-text("AI")'
    ).first();

    // Document current state — soft assertion since bug may still be open
    const isVisible = await aiBuilderTab.count() > 0;
    expect.soft(isVisible).toBe(true);

    if (!isVisible) {
      // Explicitly document that bug #2 is still present
      console.log('BUG #2: AI Builder tab not found on schema page');
    }

    // Teardown
    await apiFetch(request, `/schemas/${schemaId}`, { method: 'DELETE', token: adminToken });
  });
});
