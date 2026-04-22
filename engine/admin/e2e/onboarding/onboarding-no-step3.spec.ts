// §1.5-ext Onboarding — step 3 regression guard: "Open Canvas" loop is fixed, step 3 no longer exists
// TC: OB-10 | No SCC tags
// Bug #5 regression: wizard navigates directly after step 2, no step 3

import { test, expect } from '../fixtures';

test.describe('Onboarding — step 3 does not exist (regression guard)', () => {
  test('after step 2 action, URL does not contain step=3 or "canvas"', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/onboarding?step=2');

    const skipBtn = page.locator('button:has-text("Skip")').first();
    if (await skipBtn.count() > 0) {
      await skipBtn.click();
      await page.waitForLoadState('networkidle');

      // Should NOT go to step 3 or canvas loop
      const url = page.url();
      expect(url).not.toContain('step=3');
      expect(url).not.toMatch(/onboarding.*canvas/);
    } else {
      // Already past onboarding — verify no step 3 route exists
      const res = await page.request.get('/admin/onboarding?step=3');
      // Should redirect away or show admin, not a dedicated step 3 UI
      expect(res.url()).not.toContain('step=3');
    }
  });

  test('no "Open Canvas" button exists in onboarding flow', async ({ page }) => {
    await page.goto('/admin/onboarding');
    const openCanvasBtn = page.locator('button:has-text("Open Canvas"), a:has-text("Open Canvas")');
    // Should not be present — was removed in fix for bug #5
    const count = await openCanvasBtn.count();
    expect(count).toBe(0);
  });
});
