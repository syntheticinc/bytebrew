// §1.19 SCC-03 — invalid input: POST empty body to write endpoints → 400 not 500 GATE
// TC: SCC-03 GATE | GAP-5

import { test, expect, apiFetch } from '../fixtures';

const WRITE_ENDPOINTS = [
  '/agents',
  '/schemas',
  '/models',
  '/mcp-servers',
  '/knowledge-bases',
  '/auth/tokens',
];

test.describe('SCC-03 — invalid input returns 400 not 500', () => {
  for (const path of WRITE_ENDPOINTS) {
    test(`⛔ GATE SCC-03: POST ${path} empty body → 400 not 500`, async ({ request, adminToken }) => {
      const res = await apiFetch(request, path, {
        method: 'POST',
        token: adminToken,
        body: {},
      });
      // 500 is never acceptable — should be 400/422
      expect(res.status()).not.toBe(500);
      expect([400, 422]).toContain(res.status());
    });

    test(`SCC-03: POST ${path} malformed JSON → 400 not 500`, async ({ request, adminToken }) => {
      const res = await request.fetch(`/api/v1${path}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${adminToken}`,
        },
        data: '{invalid json[[[',
      });
      expect(res.status()).not.toBe(500);
      expect([400, 422]).toContain(res.status());
    });
  }
});
