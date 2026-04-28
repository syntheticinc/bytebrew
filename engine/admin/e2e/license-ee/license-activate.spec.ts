// §1.16 License — activate: POST /license/activate with valid key → status=active
// TC: LIC-01 | SCC skip if EE disabled

import { test, expect, apiFetch } from '../fixtures';

test.describe('License — activate', () => {
  test.skip(true, '§1.16: EE license tests require VITE_EE_LICENSE_ENABLED=true and a valid license key — skip in CE stack');

  test('POST /license/activate with valid key returns status=active', async ({ request, adminToken }) => {
    const licenseKey = process.env.TEST_LICENSE_KEY ?? 'test-license-jwt';
    const res = await apiFetch(request, '/license/activate', {
      method: 'POST',
      token: adminToken,
      body: { license_key: licenseKey },
    });
    expect([200, 201]).toContain(res.status());
    const body = await res.json();
    expect(body.status).toBe('active');
  });
});
