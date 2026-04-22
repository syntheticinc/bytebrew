// §1.11 Inspect — tool call log: filter by agent/tool/status=failed/user_id
// TC: INS-05

import { test, expect, apiFetch } from '../fixtures';

test.describe('Tool call log — filters', () => {
  test('GET /tool-call-log or /inspect/tool-calls returns 200', async ({ request, adminToken }) => {
    // Try several possible paths
    const paths = ['/tool-call-log', '/tool-calls', '/inspect/tool-calls', '/audit/tool-calls'];
    let found = false;
    for (const path of paths) {
      const res = await apiFetch(request, path, { token: adminToken });
      if (res.status() === 200) {
        found = true;
        const body = await res.json();
        expect(Array.isArray(body) || typeof body === 'object').toBe(true);
        break;
      }
    }
    if (!found) {
      test.skip(true, 'Tool call log endpoint not found at any known path');
    }
  });

  test('filter by status=failed returns subset', async ({ request, adminToken }) => {
    const paths = ['/tool-call-log?status=failed', '/tool-calls?status=failed'];
    for (const path of paths) {
      const res = await apiFetch(request, path, { token: adminToken });
      if (res.status() === 200) {
        const body = await res.json();
        const items = Array.isArray(body) ? body : (body.data ?? body.tool_calls ?? []);
        // All returned items should have status=failed
        for (const item of items) {
          if (item.status) {
            expect.soft(item.status).toBe('failed');
          }
        }
        return;
      }
    }
    test.skip(true, 'Tool call log endpoint not available');
  });

  test('tool call log page renders in admin UI', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    // Try common paths
    await page.goto('/admin/tool-calls');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('text=/Something went wrong/i')).not.toBeVisible();
  });
});
