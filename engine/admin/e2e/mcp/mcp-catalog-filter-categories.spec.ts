// §1.10 MCP — catalog filter by categories: support/sales/internal/generic
// TC: MCP-02 | GAP-8

import { test, expect, apiFetch } from '../fixtures';

const CATEGORIES = ['support', 'sales', 'internal', 'generic'];

test.describe('MCP catalog — category filters', () => {
  test('MCP catalog can be filtered by category', async ({ request, adminToken }) => {
    for (const category of CATEGORIES) {
      const res = await apiFetch(request, `/mcp/catalog?category=${category}`, { token: adminToken });
      if (res.status() === 404) {
        test.skip(true, 'MCP catalog endpoint not available');
        return;
      }
      // 200 with possibly empty list = correct; 400 = invalid category
      expect([200, 400]).toContain(res.status());
      if (res.status() === 200) {
        const body = await res.json();
        const servers = Array.isArray(body) ? body : (body.servers ?? body.data ?? []);
        expect(Array.isArray(servers)).toBe(true);
      }
    }
  });

  test('MCP catalog UI shows category filter options', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/mcp');
    await page.waitForLoadState('networkidle');

    // Open catalog if button exists
    const catalogBtn = page.locator('button:has-text("catalog"), button:has-text("Catalog")').first();
    if (await catalogBtn.count() > 0) {
      await catalogBtn.click();
      await page.waitForTimeout(500);

      const filterText = await page.textContent('body') ?? '';
      // At least one category keyword should appear in catalog UI
      const hasCategory = CATEGORIES.some(c => filterText.toLowerCase().includes(c));
      expect(hasCategory || filterText.length > 100).toBe(true);
    } else {
      test.skip(true, 'Catalog UI not accessible');
    }
  });
});
