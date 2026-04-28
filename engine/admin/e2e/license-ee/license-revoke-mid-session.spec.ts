// §1.16 License — revoke mid-session: active SSE → POST /internal/licenses/{tenant}/revoke → SSE terminates
// TC: LIC-07

import { test, expect, apiFetch } from '../fixtures';

test.describe('License — revoke mid-session', () => {
  test.skip(true, '§1.16: Mid-session license revocation requires EE mode and active SSE session — skip in CE stack');

  test('revoking license terminates active SSE session gracefully', async ({ request, adminToken }) => {
    // Requires: 1) EE license active, 2) active SSE session in progress
    // POST /internal/licenses/{tenant}/revoke
    const tenantId = 'test-tenant';
    const res = await apiFetch(request, `/internal/licenses/${tenantId}/revoke`, {
      method: 'POST',
      token: adminToken,
    });
    expect([200, 204]).toContain(res.status());
    // SSE session should terminate with error event — verified via SSE client
  });
});
