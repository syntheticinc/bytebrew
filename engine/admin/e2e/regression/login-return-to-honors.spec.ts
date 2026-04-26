// Login form must honour `?return_to=...` after a successful sign-in.
// Catches F7: cloud-web-spa post-login always redirects to /dashboard
// regardless of return_to, breaking deep-links to /admin/* and any other
// authenticated route reached from a fresh browser tab.

import { test, expect, BASE_URL } from '../fixtures';

test.describe('Regression — login honours return_to', () => {
  test('navigating to /admin/ unauthenticated then logging in lands at /admin/, not /dashboard', async ({ adminSession, page }) => {
    if (!adminSession.available) {
      test.skip(true, `cannot sign-in: ${adminSession.blockedReason ?? 'no session'}`);
      return;
    }

    // Cold-start: open /admin/ without any localStorage. Cloud-web-spa
    // should redirect to /login?return_to=/admin/.
    await page.goto(`${BASE_URL}/admin/`);
    await page.waitForLoadState('networkidle');

    expect(page.url()).toContain('/login');
    expect(page.url()).toContain('return_to');

    // Submit creds — this exercises the real login form, not an injected
    // localStorage path. The 30s wait covers Vite dev cold-compile of /login
    // when the spec is the first hit after a Tilt stack restart.
    const email = page.locator('input[type="email"], input[placeholder*="@" i]');
    await email.waitFor({ state: 'visible', timeout: 30_000 });
    await email.fill(adminSession.email);
    await page.locator('input[type="password"]').fill(adminSession.password);
    await page.getByRole('button', { name: /sign in/i }).click();

    // toHaveURL auto-retries — covers async login → mint → hard nav chain.
    await expect(
      page,
      'F7: login form ignored ?return_to=... — admin SPA never reached',
    ).toHaveURL(/\/admin\//);
  });
});
