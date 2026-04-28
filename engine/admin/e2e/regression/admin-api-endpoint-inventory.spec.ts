// Admin SPA → engine API contract test (catches F9, F10).
//
// Walks every page in the admin nav and asserts that NO /api/v1/* call the
// SPA makes returns 404 (or 5xx). The SPA is built against an assumed
// engine route inventory; if engine drops or never implements a route,
// this test surfaces it loudly with the offending URL.
//
// Plus: explicit per-bug endpoint checks for routes the SPA references
// only conditionally (e.g. when a button is clicked) — those wouldn't
// show up on a passive nav-walk.
//
// Currently catches:
//   F9  GET /api/v1/mcp/catalog              → 404 (no engine handler)
//   F10 GET /api/v1/resilience/circuit-breakers → 404 (no engine handler)
//
// Re-running after engine team adds the missing routes should turn this
// test green automatically.

import { test, expect, apiFetch } from '../fixtures';

// All admin nav targets that have side-effects loading from /api/v1/*.
// Add to this list when new pages are introduced — the test naturally
// expands its surface as the SPA grows.
const ADMIN_PAGES = [
  '/admin/overview',
  '/admin/schemas',
  '/admin/agents',
  '/admin/mcp',
  '/admin/models',
  '/admin/knowledge',
  '/admin/widget',
  '/admin/tasks',
  '/admin/api-keys',
  '/admin/settings',
  '/admin/audit',
  '/admin/resilience',
  '/admin/tool-call-log',
];

test.describe('Regression — admin SPA <> engine API contract', () => {
  test('no /api/v1/* request returns 404 across the admin nav', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    const fourOhFours: { url: string; from: string }[] = [];

    page.on('response', res => {
      const url = res.url();
      const status = res.status();
      if (status !== 404) return;
      if (!url.includes('/api/v1/')) return;
      // /api/v1/agents/<unknown-name> 404 is expected for some cleanup
      // paths; filter them out by suffix lookup is fragile, so capture
      // and let the assertion log show context for triage.
      fourOhFours.push({ url, from: page.url() });
    });

    for (const path of ADMIN_PAGES) {
      await page.goto(path);
      await page.waitForLoadState('networkidle');
    }

    // List both URL and the page that triggered it. Comma-joined so the
    // assertion failure message stays readable even with several misses.
    const summary = fourOhFours
      .map(({ url, from }) => `  - ${url}  (from ${from})`)
      .join('\n');

    expect(
      fourOhFours.length,
      `Admin SPA called these /api/v1/* endpoints that engine returned 404:\n${summary}\n` +
        `Each entry is a contract gap — either the engine handler is missing or the SPA is using a stale path.\n` +
        `Known: F9 (/api/v1/mcp/catalog), F10 (/api/v1/resilience/circuit-breakers).`,
    ).toBe(0);
  });

  test('no /api/v1/* request returns 5xx across the admin nav', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    const fiveXxs: { url: string; status: number }[] = [];

    page.on('response', res => {
      const url = res.url();
      const status = res.status();
      if (status < 500) return;
      if (!url.includes('/api/v1/')) return;
      fiveXxs.push({ url, status });
    });

    for (const path of ADMIN_PAGES) {
      await page.goto(path);
      await page.waitForLoadState('networkidle');
    }

    const summary = fiveXxs.map(({ url, status }) => `  - [${status}] ${url}`).join('\n');
    expect(
      fiveXxs.length,
      `Engine returned 5xx on a normal admin nav walk-through:\n${summary}\n` +
        `Any 5xx on a happy-path navigation is a regression — engine handlers must always 4xx (auth/validation) or 2xx.`,
    ).toBe(0);
  });

  // Routes the admin SPA references conditionally (only when a specific
  // user action fires them). A passive nav-walk wouldn't trigger these,
  // so check them explicitly with the same admin token used by the SPA.
  test('explicit endpoint check — known SPA-referenced routes resolve (catches F9/F10)', async ({ request, adminToken }) => {
    const KNOWN_ENDPOINTS = [
      // F9 — MCP catalog (used by "Add from Catalog" modal)
      { path: '/mcp/catalog', method: 'GET', knownBug: 'F9' },
      // F10 — Circuit breakers list (used by MCP table "Circuit" column refresh)
      { path: '/resilience/circuit-breakers', method: 'GET', knownBug: 'F10' },
    ];

    const broken: { path: string; status: number; knownBug: string }[] = [];
    for (const ep of KNOWN_ENDPOINTS) {
      const res = await apiFetch(request, ep.path, { method: ep.method, token: adminToken });
      if (res.status() === 404) {
        broken.push({ path: ep.path, status: 404, knownBug: ep.knownBug });
      }
    }

    const summary = broken
      .map(({ path, status, knownBug }) => `  - ${knownBug}: GET /api/v1${path} → ${status}`)
      .join('\n');

    expect(
      broken.length,
      `Admin SPA references endpoints that the engine does not route:\n${summary}\n` +
        `These endpoints are dialed conditionally (e.g. when the user clicks a button), ` +
        `so a passive nav-walk doesn't surface them. The contract is broken even if no UI ` +
        `flow triggered the call during a test session.`,
    ).toBe(0);
  });
});
