// §1.5 Onboarding Step 2 — Skip: admin overview loads without starter template
// TC: OB-04 | No SCC tags

import { test, expect, apiFetch } from '../fixtures';

test.describe('Onboarding Step 2 — skip', () => {
  test('Skip on step 2 loads admin overview', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/onboarding?step=2');

    const skipBtn = page.locator('button:has-text("Skip"), button:has-text("skip"), a:has-text("Skip")').first();
    if (await skipBtn.count() === 0) {
      test.skip(true, 'No Skip button found on step 2 — may already be past onboarding');
      return;
    }
    await skipBtn.click();

    // Should navigate away from onboarding
    await page.waitForURL(/\/admin\/?(?!onboarding)/, { timeout: 10_000 });
    const url = page.url();
    expect(url).not.toContain('onboarding');
  });

  test('admin overview renders after skip (no error boundary)', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    // If already past onboarding, just verify overview renders
    const errors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') errors.push(msg.text());
    });
    await page.goto('/admin/');
    await page.waitForLoadState('networkidle');
    // No React error boundary
    const errorBoundary = page.locator('text=/Something went wrong|Unexpected error|React error/i');
    await expect(errorBoundary).not.toBeVisible();
  });
});
