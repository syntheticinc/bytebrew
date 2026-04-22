// §1.6 Navigation — stale token → 401 → redirect to login
// TC: NAV-05 | SCC-05

import { test, expect } from '../fixtures';

test.describe('Admin — stale/invalid token handling', () => {
  test('stale JWT in localStorage causes 401 and redirect away from admin', async ({ page }) => {
    // Inject an obviously expired/invalid token
    await page.addInitScript(() => {
      // Fake JWT: header.payload.sig — expired
      const fakeToken = 'eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhZG1pbiIsImV4cCI6MX0.AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA';
      localStorage.setItem('jwt', fakeToken);
      localStorage.setItem('access_token', fakeToken);
    });

    await page.goto('/admin/');
    await page.waitForLoadState('networkidle');

    // Should redirect to login or show auth error
    const url = page.url();
    const loginRedirected = url.includes('login') || url.includes('onboarding') || url.includes('auth');
    const authError = await page.locator('text=/sign in|log in|unauthorized|session expired/i').count() > 0;
    expect(loginRedirected || authError).toBe(true);
  });

  test('no valid token → admin API returns 401', async ({ request }) => {
    const res = await request.get(`/api/v1/agents`);
    expect(res.status()).toBe(401);
  });
});
