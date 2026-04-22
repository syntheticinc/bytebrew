// §1.17 Metering — both HMAC keys invalid → engine rejects request
// TC: MET-04

import { test, expect, apiFetch } from '../fixtures';

test.describe('Metering — both HMAC keys invalid', () => {
  test.skip(true, '§1.17: HMAC key validation requires Cloud metering stack — skip in CE stack');

  test('metering request with invalid HMAC is rejected', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/internal/metering/steps', {
      method: 'POST',
      token: adminToken,
      body: { tenant_id: 'test', steps: 1 },
      headers: {
        'X-Metering-Hmac': 'totally-invalid-hmac-signature-xyz',
      },
    });
    // Should be rejected — 401 or 403
    expect([401, 403]).toContain(res.status());
  });
});
