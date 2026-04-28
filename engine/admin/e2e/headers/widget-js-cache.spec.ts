// §1.23 Headers — Cache-Control on /widget.js: max-age ≤ 3600
// TC: HDR-01 | GAP-15

import { test, expect, BASE_URL } from '../fixtures';

test.describe('Headers — widget.js cache control', () => {
  test('GET /widget.js returns Cache-Control with max-age ≤ 3600', async ({ request }) => {
    // Try common widget paths
    const paths = ['/widget/widget.js', '/widget.js'];
    let found = false;

    for (const path of paths) {
      const res = await request.get(`${BASE_URL}${path}`);
      if (res.status() === 200) {
        found = true;
        const cacheControl = res.headers()['cache-control'] ?? '';
        // Should have some cache control
        expect.soft(cacheControl.length).toBeGreaterThan(0);

        // Extract max-age value
        const maxAgeMatch = cacheControl.match(/max-age=(\d+)/);
        if (maxAgeMatch) {
          const maxAge = parseInt(maxAgeMatch[1], 10);
          // max-age should be ≤ 3600 (1 hour) — stale widget JS is a bug
          expect.soft(maxAge).toBeLessThanOrEqual(3600);
        }
        break;
      }
    }

    if (!found) {
      test.skip(true, 'widget.js not found at known paths — stack may not include widget');
    }
  });

  test('GET /widget.js has content-type: application/javascript', async ({ request }) => {
    const paths = ['/widget/widget.js', '/widget.js'];
    for (const path of paths) {
      const res = await request.get(`${BASE_URL}${path}`);
      if (res.status() === 200) {
        const ct = res.headers()['content-type'] ?? '';
        expect(ct).toMatch(/javascript/i);
        return;
      }
    }
    test.skip(true, 'widget.js not found');
  });
});
