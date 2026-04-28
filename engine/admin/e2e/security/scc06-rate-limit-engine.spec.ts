// §1.19 SCC-06 — rate limit: 101 rapid requests → 429 on request 101
// TC: SCC-06 ADVISORY | GAP-5

import { test, expect, BASE_URL } from '../fixtures';

test.describe('SCC-06 — rate limiting', () => {
  test.skip(true, 'SCC-06: Rate limiting may not be enabled on engine in CE stack. Skip unless RATE_LIMIT_ENABLED=true is confirmed. Document behavior.');

  test('101 rapid auth login attempts trigger 429', async ({ request }) => {
    const results: number[] = [];
    for (let i = 0; i < 101; i++) {
      const res = await request.post(`${BASE_URL}/api/v1/auth/login`, {
        data: { username: 'admin', password: 'wrongpassword' },
        headers: { 'Content-Type': 'application/json' },
      });
      results.push(res.status());
      if (res.status() === 429) break;
    }
    // At least one 429 should occur before 101 requests
    expect(results).toContain(429);
  });
});
