// §1.16 License — expired: tamper expiry to past → chat returns 503 "license expired"
// TC: LIC-05

import { test, expect, apiFetch } from '../fixtures';

test.describe('License — expired blocks chat', () => {
  test.skip(true, '§1.16: Requires EE mode with ability to tamper license expiry — skip in CE stack');

  test('expired license causes chat endpoint to return 503', async ({ request, adminToken }) => {
    // Activate a license with past expiry (requires test tooling)
    // Then attempt chat
    const res = await apiFetch(request, '/schemas/test-schema/chat', {
      method: 'POST',
      token: adminToken,
      body: { message: 'Hello' },
    });
    expect(res.status()).toBe(503);
    const body = await res.json();
    expect(JSON.stringify(body)).toMatch(/license|expired/i);
  });
});
