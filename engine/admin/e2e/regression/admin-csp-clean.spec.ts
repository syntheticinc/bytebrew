// CSP cleanliness across the admin nav. Catches F11 + F14:
//
//   F11 — external resources blocked (buttons.github.io / Google Fonts /
//         data: SVG inline backgrounds).
//   F14 — SPA's own bundle injects inline styles that fail under
//         `default-src 'self'` because style-src is missing
//         `'unsafe-inline'` (or per-bundle hashes/nonces).
//
// Both surface as "violates the following Content Security Policy
// directive" console errors. The admin SPA bundle and the Caddy CSP
// header are out of sync — either fix the headers (allow inline styles
// + vendor external resources) or remove the offending dependencies.

import { test, expect } from '../fixtures';

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
];

const CSP_VIOLATION_RE = /violates the following Content Security Policy/i;

test.describe('Regression — CSP clean on admin nav', () => {
  test('no CSP violations across admin pages', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    const violations: { url: string; msg: string }[] = [];

    page.on('console', m => {
      if (m.type() !== 'error') return;
      const txt = m.text();
      if (!CSP_VIOLATION_RE.test(txt)) return;
      violations.push({ url: page.url(), msg: txt });
    });

    for (const path of ADMIN_PAGES) {
      await page.goto(path);
      await page.waitForLoadState('networkidle');
    }

    const summary = violations
      .map(({ url, msg }) => `  - [${url}] ${msg.slice(0, 200)}`)
      .join('\n');
    expect(
      violations.length,
      `CSP violations recorded — F11 (external resources) and/or F14 (own inline styles):\n${summary}`,
    ).toBe(0);
  });
});
