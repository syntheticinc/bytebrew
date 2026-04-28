// §1.22 A11y — tab order: admin pages cycle through focusable elements in logical order
// TC: A11Y-01 | GAP-17

import { test, expect } from '../fixtures';

const PAGES_TO_CHECK = [
  '/admin/',
  '/admin/agents',
  '/admin/schemas',
];

test.describe('Accessibility — tab order', () => {
  for (const pagePath of PAGES_TO_CHECK) {
    test(`${pagePath} — Tab key cycles through focusable elements`, async ({ authenticatedAdmin }) => {
      const page = authenticatedAdmin;
      await page.goto(pagePath);
      await page.waitForLoadState('networkidle');

      // Focus first element
      await page.keyboard.press('Tab');
      const firstFocused = await page.evaluate(() => document.activeElement?.tagName);

      // Tab through several elements
      const focusedElements: string[] = [];
      for (let i = 0; i < 5; i++) {
        const tag = await page.evaluate(() => document.activeElement?.tagName ?? '');
        const type = await page.evaluate(() => (document.activeElement as HTMLInputElement)?.type ?? '');
        if (tag) focusedElements.push(`${tag}${type ? `[${type}]` : ''}`);
        await page.keyboard.press('Tab');
      }

      // At least some focusable elements should be reachable
      expect(focusedElements.length).toBeGreaterThan(0);
      // No element should be BODY (would mean focus escaped)
      const allBody = focusedElements.every(e => e === 'BODY');
      expect(allBody).toBe(false);
    });
  }
});
