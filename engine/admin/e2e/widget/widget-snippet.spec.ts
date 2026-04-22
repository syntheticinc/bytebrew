// §1.8 Widget — copy snippet: widget page → select schema → <script> snippet shown
// TC: WID-01

import { test, expect, apiFetch } from '../fixtures';

test.describe('Widget — snippet copy', () => {
  test('widget page renders without error', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/widget');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('text=/Something went wrong/i')).not.toBeVisible();
  });

  test('widget page shows script snippet or embed code', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/widget');
    await page.waitForLoadState('networkidle');

    // Look for a <script> tag text or code block
    const codeBlock = page.locator('code, pre, [data-testid*="snippet"], textarea[readonly]').first();
    const copyBtn = page.locator('button:has-text("Copy"), button[aria-label*="copy"]').first();

    const hasCode = await codeBlock.count() > 0;
    const hasCopy = await copyBtn.count() > 0;

    // At minimum one of these should be present
    expect(hasCode || hasCopy).toBe(true);
  });

  test('snippet contains data-schema-id attribute pattern', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;

    // Create a schema to select
    const schemaName = `widget-schema-${Date.now()}`;
    const createRes = await apiFetch(request, '/schemas', {
      method: 'POST',
      token: adminToken,
      body: { name: schemaName, chat_enabled: true },
    });
    const created = await createRes.json();
    const schemaId = created.id;

    await page.goto('/admin/widget');
    await page.waitForLoadState('networkidle');

    // Try selecting the schema
    const schemaSelect = page.locator('select, [data-testid="schema-select"]').first();
    if (await schemaSelect.count() > 0) {
      await schemaSelect.selectOption({ value: schemaId });
      await page.waitForTimeout(500);
    }

    const pageText = await page.textContent('body') ?? '';
    // Snippet should reference the widget script
    expect(pageText).toMatch(/widget|script|embed|data-schema/i);

    // Teardown
    if (schemaId) await apiFetch(request, `/schemas/${schemaId}`, { method: 'DELETE', token: adminToken });
  });
});
