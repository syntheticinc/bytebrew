// §1.22 A11y — Esc closes admin modals (create, delete confirm)
// TC: A11Y-02 | GAP-17

import { test, expect, apiFetch } from '../fixtures';

test.describe('Accessibility — Esc closes modals', () => {
  test('Esc closes create agent modal', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/agents');
    await page.waitForLoadState('networkidle');

    // Open create modal
    const createBtn = page.locator('button:has-text("Create"), button:has-text("New agent"), button:has-text("Add agent")').first();
    if (await createBtn.count() === 0) {
      test.skip(true, 'No create button found on agents page');
      return;
    }
    await createBtn.click();

    // Modal should open
    const modal = page.locator('[role="dialog"], .modal, [data-testid*="modal"]').first();
    if (await modal.count() === 0) {
      test.skip(true, 'No modal opened — may be a page-navigation create flow');
      return;
    }
    await expect(modal).toBeVisible({ timeout: 3000 });

    // Press Esc
    await page.keyboard.press('Escape');
    await page.waitForTimeout(300);

    // Modal should be closed
    await expect(modal).not.toBeVisible({ timeout: 3000 });
  });

  test('Esc closes delete confirmation dialog', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;

    // Create a test agent to delete
    const agentName = `esc-test-${Date.now()}`;
    await apiFetch(request, '/agents', {
      method: 'POST',
      token: adminToken,
      body: { name: agentName, system_prompt: 'Esc test' },
    });

    await page.goto('/admin/agents');
    await page.waitForLoadState('networkidle');

    // Find delete button for the test agent
    const deleteBtn = page.locator(`[data-testid*="delete-${agentName}"], tr:has-text("${agentName}") button:has-text("Delete"), tr:has-text("${agentName}") [aria-label*="delete"]`).first();
    if (await deleteBtn.count() > 0) {
      await deleteBtn.click();

      const confirmModal = page.locator('[role="dialog"], .modal, [data-testid*="confirm"]').first();
      if (await confirmModal.count() > 0) {
        await expect(confirmModal).toBeVisible({ timeout: 3000 });
        await page.keyboard.press('Escape');
        await page.waitForTimeout(300);
        await expect(confirmModal).not.toBeVisible({ timeout: 3000 });
      }
    }

    // Teardown
    await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });
  });
});
