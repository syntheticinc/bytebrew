// §1.7-ext CRUD — Capabilities UI: inject unknown type via API, open admin → friendly fallback, no TypeError
// TC: CRUD-12 | GAP-2 BUG-002 regression

import { test, expect, apiFetch } from '../fixtures';

test.describe('Capabilities UI — unknown type fallback (BUG-002 regression)', () => {
  test.skip(true, 'BUG-002: requires injecting unknown capability type directly in DB or via internal API — backend must allow it. Skip until injection endpoint available.');

  test('admin agent detail with unknown capability type shows friendly fallback, no TypeError', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;
    const agentName = `bug002-agent-${Date.now()}`;

    await apiFetch(request, '/agents', { method: 'POST', token: adminToken, body: { name: agentName, system_prompt: 'Test' } });

    // Inject unknown type via API (if endpoint permits)
    await apiFetch(request, `/agents/${agentName}/capabilities`, {
      method: 'POST',
      token: adminToken,
      body: { type: 'unknown_future_type', config: {} },
    });

    const consoleErrors: string[] = [];
    page.on('console', msg => {
      if (msg.type() === 'error') consoleErrors.push(msg.text());
    });

    await page.goto(`/admin/agents/${agentName}`);
    await page.waitForLoadState('networkidle');

    // No TypeError in console
    const typeErrors = consoleErrors.filter(e => e.includes('TypeError'));
    expect(typeErrors).toHaveLength(0);

    // No React error boundary
    await expect(page.locator('text=/Something went wrong/i')).not.toBeVisible();

    // Teardown
    await apiFetch(request, `/agents/${agentName}`, { method: 'DELETE', token: adminToken });
  });
});
