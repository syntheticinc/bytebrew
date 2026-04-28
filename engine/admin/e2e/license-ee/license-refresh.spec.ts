// §1.16 License — refresh: POST /license/refresh → new expiry
// TC: LIC-02

import { test, expect, apiFetch } from '../fixtures';

test.describe('License — refresh', () => {
  test.skip(true, '§1.16: EE license tests require VITE_EE_LICENSE_ENABLED=true — skip in CE stack');

  test('POST /license/refresh returns new expiry timestamp', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/license/refresh', {
      method: 'POST',
      token: adminToken,
    });
    expect([200, 201]).toContain(res.status());
    const body = await res.json();
    expect(body.expires_at ?? body.expiry ?? body.valid_until).toBeTruthy();
  });
});
