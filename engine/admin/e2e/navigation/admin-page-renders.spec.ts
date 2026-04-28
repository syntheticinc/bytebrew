// §1.6 Navigation — each page renders without React error boundary or 500
// TC: NAV-02 | No SCC tags

import { test, expect } from '../fixtures';

const ADMIN_PAGES = [
  '/admin/',
  '/admin/agents',
  '/admin/schemas',
  '/admin/models',
  '/admin/mcp',
  '/admin/knowledge',
  '/admin/settings',
  '/admin/widget',
  '/admin/tasks',
  '/admin/resilience',
];

test.describe('Admin pages — render without errors', () => {
  for (const pagePath of ADMIN_PAGES) {
    test(`${pagePath} renders without error boundary`, async ({ authenticatedAdmin }) => {
      const page = authenticatedAdmin;
      const consoleErrors: string[] = [];
      page.on('console', msg => {
        if (msg.type() === 'error') consoleErrors.push(msg.text());
      });

      const responses: number[] = [];
      page.on('response', res => {
        if (res.status() >= 500) responses.push(res.status());
      });

      await page.goto(pagePath);
      await page.waitForLoadState('networkidle');

      // No React error boundary
      const errorBoundary = page.locator('text=/Something went wrong|Unexpected error|Error:/i').first();
      await expect(errorBoundary).not.toBeVisible();

      // No 500s in network
      expect(responses).toHaveLength(0);
    });
  }
});
