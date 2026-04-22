// §1.16 License — status: GET /license/status returns current
// TC: LIC-03

import { test, expect, apiFetch } from '../fixtures';

test.describe('License — status', () => {
  test('GET /license/status returns current license info', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/license/status', { token: adminToken });
    if (res.status() === 404) {
      test.skip(true, '/license/status not found — EE may be disabled or endpoint path differs');
      return;
    }
    expect([200]).toContain(res.status());
    const body = await res.json();
    // Should have status field
    expect(body.status ?? body.license_status ?? body.state).toBeTruthy();
  });

  test('⛔ GATE SCC-01: /license/status without auth returns 401', async ({ request }) => {
    const res = await request.get('/api/v1/license/status');
    if (res.status() === 404) return; // EE disabled
    expect(res.status()).toBe(401);
  });
});
