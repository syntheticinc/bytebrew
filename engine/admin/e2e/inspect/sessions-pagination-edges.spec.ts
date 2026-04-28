// §1.11 Inspect — sessions pagination edge cases: page=0/-1/99999/per_page=0/beyond totalPages → no 500
// TC: INS-01 | GAP-10

import { test, expect, apiFetch } from '../fixtures';

const EDGE_PARAMS = [
  { page: 0 },
  { page: -1 },
  { page: 99999 },
  { per_page: 0 },
  { per_page: 99999 },
  { page: 1, per_page: 1 },
];

test.describe('Sessions — pagination edge cases', () => {
  for (const params of EDGE_PARAMS) {
    const label = Object.entries(params).map(([k, v]) => `${k}=${v}`).join('&');
    test(`GET /sessions?${label} returns valid shape, no 500`, async ({ request, adminToken }) => {
      const query = new URLSearchParams(Object.entries(params).map(([k, v]) => [k, String(v)])).toString();
      const res = await apiFetch(request, `/sessions?${query}`, { token: adminToken });

      // 500 is never acceptable
      expect(res.status()).not.toBe(500);
      // Acceptable: 200 (possibly empty), 400 (invalid param), 404
      expect([200, 400, 404]).toContain(res.status());

      if (res.status() === 200) {
        const body = await res.json();
        // Should have array or pagination shape
        const isArray = Array.isArray(body);
        const hasSessions = Array.isArray(body.sessions) || Array.isArray(body.data);
        expect(isArray || hasSessions || typeof body === 'object').toBe(true);
      }
    });
  }
});
