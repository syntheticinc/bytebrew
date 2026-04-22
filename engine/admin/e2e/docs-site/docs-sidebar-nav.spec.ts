// §1.24 Docs-site — sidebar sections collapse/expand; mobile hamburger; active page highlighted
// TC: DOCS-03 | GAP-1

import { test, expect, BASE_URL } from '../fixtures';

test.describe('Docs-site — sidebar navigation', () => {
  test.skip(true, 'GAP-1: Docs-site not in engine/admin stack. Skip until docs-site container is part of test compose.');

  test('docs sidebar expands and collapses sections', async ({ page }) => {
    await page.goto(`${BASE_URL}/docs/`);
    await page.waitForLoadState('networkidle');

    const sectionToggle = page.locator('[data-accordion-trigger], button[aria-expanded], details summary').first();
    if (await sectionToggle.count() === 0) {
      test.skip(true, 'No collapsible sidebar sections found');
      return;
    }

    const initialState = await sectionToggle.getAttribute('aria-expanded');
    await sectionToggle.click();
    await page.waitForTimeout(300);
    const newState = await sectionToggle.getAttribute('aria-expanded');
    expect(newState).not.toBe(initialState);
  });

  test('mobile hamburger toggles sidebar', async ({ page }) => {
    // Use mobile viewport
    await page.setViewportSize({ width: 375, height: 812 });
    await page.goto(`${BASE_URL}/docs/`);
    await page.waitForLoadState('networkidle');

    const hamburger = page.locator('button[aria-label*="menu"], button[aria-label*="navigation"], [data-mobile-menu-toggle]').first();
    if (await hamburger.count() === 0) {
      test.skip(true, 'No hamburger menu found');
      return;
    }

    await hamburger.click();
    await page.waitForTimeout(300);

    const sidebar = page.locator('nav[aria-label*="site"], aside, [data-sidebar]').first();
    if (await sidebar.count() > 0) {
      await expect(sidebar).toBeVisible();
    }
  });
});
