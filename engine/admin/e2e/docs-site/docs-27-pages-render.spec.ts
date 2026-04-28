// §1.24 Docs-site — parametric: 27 doc pages each return 200, title present, no console errors
// TC: DOCS-01 | GAP-1

import { test, expect, BASE_URL } from '../fixtures';

// Known Starlight docs pages (adjust paths to match actual docs-site structure)
const DOC_PAGES = [
  '/docs/',
  '/docs/getting-started/',
  '/docs/getting-started/installation/',
  '/docs/getting-started/quick-start/',
  '/docs/concepts/',
  '/docs/concepts/agents/',
  '/docs/concepts/schemas/',
  '/docs/concepts/multi-agent/',
  '/docs/concepts/memory/',
  '/docs/concepts/knowledge-base/',
  '/docs/concepts/mcp/',
  '/docs/guides/',
  '/docs/guides/create-agent/',
  '/docs/guides/create-schema/',
  '/docs/guides/configure-mcp/',
  '/docs/guides/widget-embed/',
  '/docs/guides/api-keys/',
  '/docs/integration/',
  '/docs/integration/byok/',
  '/docs/integration/rest-api/',
  '/docs/integration/sse-streaming/',
  '/docs/deployment/',
  '/docs/deployment/docker/',
  '/docs/deployment/kubernetes/',
  '/docs/deployment/bare-metal/',
  '/docs/reference/',
  '/docs/reference/api/',
];

test.describe('Docs-site — 27 pages render', () => {
  test.skip(true, 'GAP-1: Docs-site (Starlight) not included in engine/admin test stack. Run against docs-site port directly or add to compose. Skip until docs-site is part of the stack.');

  for (const docPath of DOC_PAGES) {
    test(`${docPath} returns 200 with title`, async ({ page }) => {
      const consoleErrors: string[] = [];
      page.on('console', msg => {
        if (msg.type() === 'error') consoleErrors.push(msg.text());
      });

      const res = await page.goto(`${BASE_URL}${docPath}`);
      expect(res?.status()).toBe(200);

      // Title should be present
      const title = await page.title();
      expect(title.length).toBeGreaterThan(0);

      // No JS errors
      expect.soft(consoleErrors).toHaveLength(0);
    });
  }
});
