// MCP detail panel must expose the circuit-breaker state and a reset
// action. Catches F12: detail panel currently shows only "Edit" and
// "Remove" — when an MCP server's breaker opens, the user has no UI
// path to recover (must hit the API or restart engine).

import { test, expect, BASE_URL, apiFetch } from '../fixtures';

test.describe('Regression — MCP detail panel has reset-breaker action', () => {
  test('clicking an MCP row reveals a "Reset" / breaker control', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;

    // Seed an MCP server so the table is populated. Bogus URL keeps
    // anything off our hands.
    const name = `regression-f12-${Date.now()}`;
    const create = await apiFetch(request, '/mcp-servers', {
      method: 'POST', token: adminToken,
      body: { name, type: 'http', url: 'http://nonexistent.invalid:9999/mcp' },
    });
    if (![200, 201].includes(create.status())) {
      test.skip(true, `cannot seed MCP: ${create.status()} ${await create.text()}`);
      return;
    }

    await page.goto(`${BASE_URL}/admin/mcp`);
    await page.waitForLoadState('networkidle');

    // Admin SPA caches the table; reload to pick up the new MCP server.
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Open the detail panel by clicking the row. Try the role-based
    // selector first; fall back to a text-based locator if the row isn't
    // wired as ARIA "row".
    let opened = false;
    try {
      await page.getByRole('row', { name: new RegExp(name) }).click({ timeout: 5_000 });
      opened = true;
    } catch {
      try {
        await page.getByText(name, { exact: false }).first().click({ timeout: 5_000 });
        opened = true;
      } catch { /* fallthrough — opened stays false */ }
    }
    if (!opened) {
      test.skip(true, `seeded MCP "${name}" not visible in /admin/mcp table — UI cache or different shape`);
      return;
    }

    // Expected: a button referencing reset/circuit-breaker.
    const resetCandidates = [
      page.getByRole('button', { name: /reset.*breaker/i }),
      page.getByRole('button', { name: /breaker.*reset/i }),
      page.getByRole('button', { name: /^reset$/i }),
      page.getByRole('button', { name: /reset circuit/i }),
    ];

    let found = false;
    for (const c of resetCandidates) {
      if (await c.count() > 0) {
        found = true;
        break;
      }
    }

    expect(
      found,
      `F12: MCP detail panel for "${name}" exposes no Reset Circuit Breaker action; ` +
        `only Edit/Remove are visible. Users have no UI path to recover an open breaker.`,
    ).toBe(true);

    // Cleanup
    await apiFetch(request, `/mcp-servers/${name}`, { method: 'DELETE', token: adminToken });
  });
});
