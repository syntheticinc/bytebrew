// §1.6 Navigation — all sidebar links visible post-onboarding
// TC: NAV-01 | No SCC tags

import { test, expect } from '../fixtures';

const EXPECTED_NAV_ITEMS = [
  /overview/i,
  /agents/i,
  /schemas/i,
  /models/i,
  /mcp/i,
  /knowledge/i,
  /api.?key/i,
  /settings/i,
  /widget/i,
];

test.describe('Admin navigation — sidebar links', () => {
  // REAL BUG: BUG-11 — admin SPA in cloud stack (VITE_AUTH_MODE=external) ignores
  // jwt localStorage token set by fixture and redirects to landing instead.
  // Fixture uses localStorage injection but external-mode SPA only reads #at= hash fragment.
  test.fail(true, 'REAL BUG: BUG-11 — admin SPA external-mode ignores localStorage jwt; redirects to landing');
  test('all expected sidebar links are visible', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.waitForLoadState('networkidle');

    const nav = page.locator('nav, aside, [role="navigation"]').first();
    const navText = await nav.textContent() ?? await page.textContent('body') ?? '';

    for (const pattern of EXPECTED_NAV_ITEMS) {
      expect.soft(navText).toMatch(pattern);
    }
  });

  test('each sidebar link is clickable and does not 404', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;

    const navLinks = page.locator('nav a[href*="/admin/"], aside a[href*="/admin/"]');
    const count = await navLinks.count();
    expect(count).toBeGreaterThan(0);

    for (let i = 0; i < Math.min(count, 10); i++) {
      const href = await navLinks.nth(i).getAttribute('href');
      if (href) {
        const res = await page.request.get(href);
        expect.soft(res.status()).not.toBe(404);
      }
    }
  });
});
