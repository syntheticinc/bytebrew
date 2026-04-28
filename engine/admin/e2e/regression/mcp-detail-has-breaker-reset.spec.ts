// MCP detail panel must expose the circuit-breaker state and a reset
// action. Catches F12: detail panel currently shows only "Edit" and
// "Remove" — when an MCP server's breaker opens, the user has no UI
// path to recover (must hit the API or restart engine).

import { test, expect, BASE_URL, apiFetch } from '../fixtures';

test.describe('Regression — MCP detail panel has reset-breaker action', () => {
  test('clicking an MCP row reveals a "Reset" / breaker control', async ({ authenticatedAdmin, request, adminToken }) => {
    const page = authenticatedAdmin;

    // Bypass OnboardingGate. The fixture's tenant is fresh (zero models), so
    // /admin/* normally redirects to /onboarding. This is unrelated to the
    // F12 surface we're testing — the gate trusts a sticky session flag, so
    // setting it lets MCP page render.
    await page.addInitScript(() => {
      try { sessionStorage.setItem('bb_onboarded', '1'); } catch { /* no-op */ }
    });

    // Seed an MCP server so the table is populated. Bogus URL keeps
    // anything off our hands.
    const name = `regression-f12-${Date.now()}`;
    const create = await apiFetch(request, '/mcp-servers', {
      method: 'POST', token: adminToken,
      body: { name, type: 'http', url: 'http://nonexistent.invalid:9999/mcp' },
    });
    expect(
      [200, 201],
      `seed MCP must succeed: status=${create.status()} body=${await create.text().catch(() => '<unreadable>')}`,
    ).toContain(create.status());

    await page.goto(`${BASE_URL}/admin/mcp`);
    await page.waitForLoadState('networkidle');
    // SPA caches the listing; one reload after the seed picks it up.
    await page.reload();
    await page.waitForLoadState('networkidle');

    // DataTable now stamps each row with data-testid={`row-${keyField}`}.
    // MCPPage uses keyField="name", so the seeded row's testid is `row-${name}`.
    const row = page.getByTestId(`row-${name}`);
    await expect(
      row,
      `Seeded MCP "${name}" must appear in /admin/mcp table after reload. ` +
        `Either DataTable lost the data-testid contract or the SPA failed to refetch the list.`,
    ).toBeVisible({ timeout: 10_000 });
    await row.click();

    // Detail panel must expose a Reset Breaker / Reset Circuit button.
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
