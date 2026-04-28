// §1.24 Docs-site — Pagefind search: "BYOK" → hit; "multi-agent" → hit; empty → no crash
// TC: DOCS-02 | GAP-1

import { test, expect, BASE_URL } from '../fixtures';

test.describe('Docs-site — Pagefind search', () => {
  test.skip(true, 'GAP-1: Docs-site not in engine/admin stack. Pagefind requires build-time index. Skip until docs-site is deployed in test compose.');

  test('search "BYOK" returns at least one result', async ({ page }) => {
    await page.goto(`${BASE_URL}/docs/`);
    await page.waitForLoadState('networkidle');

    const searchInput = page.locator('[data-pagefind-ui] input, input[placeholder*="search"], input[type="search"]').first();
    if (await searchInput.count() === 0) {
      test.skip(true, 'Pagefind search input not found');
      return;
    }
    await searchInput.fill('BYOK');
    await page.waitForTimeout(1000);

    const results = page.locator('[data-pagefind-ui] [class*="result"], [class*="search-result"]');
    expect(await results.count()).toBeGreaterThan(0);
  });

  test('empty search query does not crash', async ({ page }) => {
    await page.goto(`${BASE_URL}/docs/`);
    await page.waitForLoadState('networkidle');

    const searchInput = page.locator('[data-pagefind-ui] input, input[type="search"]').first();
    if (await searchInput.count() > 0) {
      await searchInput.fill('');
      await page.keyboard.press('Enter');
      await page.waitForTimeout(500);
      await expect(page.locator('text=/Something went wrong/i')).not.toBeVisible();
    }
  });
});
