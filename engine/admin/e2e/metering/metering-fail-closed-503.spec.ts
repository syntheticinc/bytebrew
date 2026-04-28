// §1.17 Metering — fail-closed: metering service down → chat returns 503
// TC: MET-02

import { test, expect, apiFetch } from '../fixtures';

test.describe('Metering — fail-closed 503', () => {
  test.skip(true, '§1.17: Fail-closed test requires docker stop cloud-full-landing-1 — cannot automate in shared stack without infra manipulation');

  test('chat returns 503 when metering service is unavailable', async ({ request, adminToken }) => {
    // Requires: docker stop metering container before this test
    const res = await apiFetch(request, '/schemas/test-schema/chat', {
      method: 'POST',
      token: adminToken,
      body: { message: 'Hello' },
    });
    expect(res.status()).toBe(503);
    const body = await res.json();
    expect(JSON.stringify(body)).toMatch(/metering|unavailable/i);
  });
});
