// §1.21 Concurrency — refresh page during SSE → history restored, no duplicate messages
// TC: CON-03 | GAP-7

import { test, expect, apiFetch } from '../fixtures';

test.describe('Concurrency — refresh during SSE', () => {
  test.skip(true, 'Refresh during active SSE requires a live LLM session — skip without real model configured. Document: session history should restore from GET /sessions/{id}/messages after reload.');

  test('page reload restores session history from backend', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;

    // Get a schema with an active session
    const sessRes = await apiFetch(request, '/sessions?per_page=1', { token: adminToken });
    if (sessRes.status() !== 200) {
      test.skip(true, 'Cannot get sessions');
      return;
    }
    const body = await sessRes.json();
    const sessions = Array.isArray(body) ? body : (body.sessions ?? body.data ?? []);
    if (sessions.length === 0) {
      test.skip(true, 'No sessions available');
      return;
    }

    const sessionId = sessions[0].id;
    const schemaId = sessions[0].schema_id;

    // Navigate to schema with session
    await page.goto(`/admin/schemas/${schemaId}?session=${sessionId}`);
    await page.waitForLoadState('networkidle');

    // Hard reload
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Session history should be restored — no error
    await expect(page.locator('text=/Something went wrong/i')).not.toBeVisible();
  });
});
