// Regression bug #2 — Builder chat must be accessible from any admin page
// TC: REG-02
// BUG-13 update: sidebar "AI Builder" shortcut was removed as legacy
// prototype artifact. The builder remains accessible via BottomPanel's
// "AI Assistant" tab — the single global entry point.

import { test, expect, apiFetch } from '../fixtures';

test.describe('Regression bug #2 — Builder chat accessible from schema page', () => {
  test('AI Assistant tab visible in bottom panel on schema detail page', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;

    const name = `reg02-schema-${Date.now()}`;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name, chat_enabled: true },
    });
    if (![200, 201].includes(createRes.status())) {
      test.skip(true, `schema create: ${createRes.status()}`);
      return;
    }
    const created = await createRes.json();
    const schemaId = (created.id ?? created.data?.id) ?? name;

    await page.goto(`/admin/schemas/${schemaId}`);
    await page.waitForLoadState('networkidle');

    const aiAssistantTab = page.getByRole('button', { name: /ai assistant|builder/i }).first();
    await expect(aiAssistantTab).toBeVisible({ timeout: 10_000 });

    await apiFetch(request, `/schemas/${schemaId}`, { method: 'DELETE', token: adminToken });
  });

  test('sidebar does NOT expose an "AI Builder" shortcut (BUG-13 removed)', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/');
    await page.waitForLoadState('networkidle');
    const sidebarButton = page.locator('aside').getByRole('button', { name: /open ai builder chat/i });
    expect(await sidebarButton.count()).toBe(0);
  });
});
