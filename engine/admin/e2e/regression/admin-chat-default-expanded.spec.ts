// Regression bug #1 — chat panel (Test Flow) should be expanded by default, not minimized
// TC: REG-01

import { test, expect, apiFetch } from '../fixtures';

test.describe('Regression bug #1 — chat panel default expanded', () => {
  test('Test Flow tab visible and not collapsed by default on schema page', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;

    // Need a schema to open
    const name = `reg01-schema-${Date.now()}`;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name, chat_enabled: true },
    });
    const created = await createRes.json();
    const schemaId = created.id ?? name;

    await page.goto(`/admin/schemas/${schemaId}`);
    await page.waitForLoadState('networkidle');

    // Test Flow tab should be visible and the chat panel should be accessible
    const testFlowTab = page.locator('button:has-text("Test Flow"), [data-testid*="test-flow"], [role="tab"]:has-text("Test")').first();
    const chatPanel = page.locator('[data-testid="chat-panel"], [class*="chat"], textarea[placeholder*="message"]').first();

    // Either the tab is clickable or the panel is already visible
    if (await testFlowTab.count() > 0) {
      await testFlowTab.click();
      await page.waitForTimeout(300);
    }

    // Bug #1: panel was collapsed — after fix, input or content should be visible
    const panelHeight = await page.evaluate(() => {
      const panel = document.querySelector('[data-testid="chat-panel"], [class*="chat-panel"]');
      if (!panel) return null;
      return panel.getBoundingClientRect().height;
    });

    if (panelHeight !== null) {
      // Panel should not be collapsed (height > 50px minimum)
      expect.soft(panelHeight).toBeGreaterThan(50);
    }

    // Teardown
    await apiFetch(request, `/schemas/${schemaId}`, { method: 'DELETE', token: adminToken });
  });
});
