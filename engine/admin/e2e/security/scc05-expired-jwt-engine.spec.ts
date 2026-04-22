// §1.19 SCC-05 — expired JWT → 401 + WWW-Authenticate header
// TC: SCC-05 ADVISORY | GAP-5

import { test, expect, BASE_URL } from '../fixtures';

test.describe('SCC-05 — expired JWT returns 401 with WWW-Authenticate', () => {
  test('expired JWT (exp=1) returns 401', async ({ request }) => {
    // Craft a JWT-shaped token with exp in the past
    // header: {"alg":"EdDSA","typ":"JWT"} — base64url
    // payload: {"sub":"admin","exp":1} — exp=1 (Unix epoch 1970-01-01)
    const expiredToken = 'eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbiIsImV4cCI6MX0.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';

    const res = await request.get(`${BASE_URL}/api/v1/agents`, {
      headers: { Authorization: `Bearer ${expiredToken}` },
    });
    expect(res.status()).toBe(401);
  });

  test('expired JWT response includes WWW-Authenticate header', async ({ request }) => {
    // REAL BUG: engine returns 401 without WWW-Authenticate header (RFC 7235 §3.1 requires it on 401)
    test.fail(true, 'REAL BUG: engine /api/v1/agents returns 401 without WWW-Authenticate header (RFC 7235)');
    const expiredToken = 'eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbiIsImV4cCI6MX0.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';

    const res = await request.get(`${BASE_URL}/api/v1/agents`, {
      headers: { Authorization: `Bearer ${expiredToken}` },
    });
    // WWW-Authenticate is required by RFC 7235 §3.1 on any 401 response
    const wwwAuth = res.headers()['www-authenticate'];
    expect(wwwAuth).toBeTruthy();
  });

  test('alg:none JWT rejected with 401', async ({ request }) => {
    // header: {"alg":"none","typ":"JWT"} — known attack vector
    const algNoneToken = 'eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJhZG1pbiIsImV4cCI6OTk5OTk5OTk5OX0.';
    const res = await request.get(`${BASE_URL}/api/v1/agents`, {
      headers: { Authorization: `Bearer ${algNoneToken}` },
    });
    expect(res.status()).toBe(401);
  });

  test('HS256 JWT rejected with 401 (only EdDSA accepted)', async ({ request }) => {
    // header: {"alg":"HS256","typ":"JWT"}
    const hs256Token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbiIsImV4cCI6OTk5OTk5OTk5OX0.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c';
    const res = await request.get(`${BASE_URL}/api/v1/agents`, {
      headers: { Authorization: `Bearer ${hs256Token}` },
    });
    expect(res.status()).toBe(401);
  });
});
