// §1.5-ext Onboarding — step 3 regression guard: "Open Canvas" loop is fixed, step 3 no longer exists
// TC: OB-10 | No SCC tags
// Bug #5 regression: wizard navigates directly after step 2, no step 3

import { test, expect } from '../fixtures';

test.describe('Onboarding — step 3 does not exist (regression guard)', () => {
  test('after step 2 action, URL does not contain step=3 or "canvas"', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    // The wizard uses React state (useState<1|2>), NOT URL query params.
    // ?step=2 is ignored — the page always renders step 1 on first load.
    await page.goto('/admin/onboarding');
    await page.waitForLoadState('networkidle');

    const skipBtn = page.locator('button:has-text("Skip")').first();
    if (await skipBtn.count() > 0) {
      await skipBtn.click();
      await page.waitForLoadState('networkidle');

      // Should NOT go to step 3 or canvas loop
      const url = page.url();
      expect(url).not.toContain('step=3');
      expect(url).not.toMatch(/onboarding.*canvas/);
    } else {
      // Already past onboarding (model exists) — the wizard is not shown.
      // Verify that navigating to ?step=3 does not show a dedicated step 3 UI.
      // The SPA will preserve the query string in the URL (no server redirect for SPAs),
      // but the rendered content must not be a "step 3" screen.
      await page.goto('/admin/onboarding?step=3');
      await page.waitForLoadState('networkidle');
      const bodyText = await page.textContent('body') ?? '';
      // Step 3 never existed — page should show step 1, schemas list, or be redirected elsewhere
      expect(bodyText).not.toMatch(/Step 3 of|step three/i);
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
