// §1.16 License — tampered signature: modify JWT sig → engine rejects 401
// TC: LIC-06

import { test, expect, apiFetch } from '../fixtures';

test.describe('License — tampered signature rejected', () => {
  test.skip(true, '§1.16: EE license signature validation requires EE-enabled engine — skip in CE stack');

  test('license JWT with tampered signature returns 401 on activate', async ({ request, adminToken }) => {
    // Take a valid JWT structure but corrupt the signature
    const tamperedJwt = 'eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIiwidGllciI6InBlcnNvbmFsIiwiZXhwIjo5OTk5OTk5OTk5fQ.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';
    const res = await apiFetch(request, '/license/activate', {
      method: 'POST',
      token: adminToken,
      body: { license_key: tamperedJwt },
    });
    expect([401, 400, 422]).toContain(res.status());
  });
});
