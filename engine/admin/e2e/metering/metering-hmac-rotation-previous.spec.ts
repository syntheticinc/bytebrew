// §1.17 Metering — HMAC rotation: request signed with METERING_HMAC_PREVIOUS still accepted
// TC: MET-03

import { test, expect, apiFetch } from '../fixtures';

test.describe('Metering — HMAC key rotation', () => {
  test.skip(true, '§1.17: HMAC rotation testing requires Cloud metering stack with METERING_HMAC_PREVIOUS configured — skip in CE stack');

  test('metering request signed with previous HMAC key is accepted', async ({ request, adminToken }) => {
    // This test verifies the metering endpoint accepts requests signed
    // with the rotated-out METERING_HMAC_PREVIOUS key
    const res = await apiFetch(request, '/internal/metering/steps', {
      method: 'POST',
      token: adminToken,
      body: { tenant_id: 'test', steps: 1 },
      headers: {
        'X-Metering-Hmac': 'previous-key-signature-placeholder',
      },
    });
    // 200 = accepted with previous key; 401/403 = rejected (bug)
    expect([200, 204]).toContain(res.status());
  });
});
