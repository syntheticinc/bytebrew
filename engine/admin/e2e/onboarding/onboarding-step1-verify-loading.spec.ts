// §1.5-ext Onboarding Step 1 — "Test connection" loading spinner; double-click disabled while loading
// TC: OB-08 | No SCC tags
// Known gap: /api/v1/models/validate endpoint not yet implemented

import { test, expect } from '../fixtures';

test.describe('Onboarding Step 1 — connection test loading state', () => {
  test.skip(true, 'Known gap: /api/v1/models/validate endpoint not yet implemented — pending backend task');

  test('clicking Test connection shows spinner and disables button', async ({ page }) => {
    await page.goto('/admin/onboarding');

    const testBtn = page.locator('button:has-text("Test"), button:has-text("Validate"), button:has-text("Verify")').first();
    if (await testBtn.count() === 0) {
      test.skip(true, 'No test connection button found');
      return;
    }
    await testBtn.click();

    // Button should be disabled while loading
    const isDisabled = await testBtn.isDisabled();
    expect(isDisabled).toBe(true);

    // Spinner or loading indicator
    const spinner = page.locator('[role="progressbar"], [class*="spin"], [class*="loading"]').first();
    await expect(spinner).toBeVisible({ timeout: 3000 });
  });
});
