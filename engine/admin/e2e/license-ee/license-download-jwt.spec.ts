// §1.16 License — download: GET /license/download returns valid JWT with EdDSA signature
// TC: LIC-04

import { test, expect, apiFetch } from '../fixtures';

test.describe('License — download JWT', () => {
  test.skip(true, '§1.16: EE license download requires active EE license — skip in CE stack');

  test('GET /license/download returns JWT with 3-part structure', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/license/download', { token: adminToken });
    expect(res.status()).toBe(200);
    const body = await res.text();
    // JWT has 3 parts separated by dots
    const parts = body.trim().split('.');
    expect(parts.length).toBe(3);
    // Header should decode to EdDSA algorithm
    const header = JSON.parse(Buffer.from(parts[0], 'base64url').toString());
    expect(header.alg).toMatch(/EdDSA|Ed25519/i);
  });
});
