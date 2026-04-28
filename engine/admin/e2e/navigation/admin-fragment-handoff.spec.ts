// §1.6 Navigation — /admin/#at=<token> → stored in localStorage, fragment scrubbed from URL
// TC: NAV-04 | No SCC tags

import { test, expect } from '../fixtures';

test.describe('Admin — fragment token handoff', () => {
  test('#at=<token> is stored in localStorage and fragment is scrubbed', async ({ page, adminToken }) => {
    // Navigate with fragment token
    await page.goto(`/admin/#at=${adminToken}`);
    await page.waitForLoadState('networkidle');

    // Fragment should be removed from URL
    const url = page.url();
    expect(url).not.toContain('#at=');
    expect(url).not.toContain(adminToken);

    // Token should be stored in localStorage
    const stored = await page.evaluate(() => {
      return localStorage.getItem('jwt') ?? localStorage.getItem('access_token') ?? localStorage.getItem('token');
    });
    // Token stored (may be same value as adminToken or a derived session token)
    expect(stored).toBeTruthy();
  });

  test('page still loads correctly after fragment handoff', async ({ page, adminToken }) => {
    await page.goto(`/admin/#at=${adminToken}`);
    await page.waitForLoadState('networkidle');

    // Should be on admin page, not redirected to login
    const url = page.url();
    expect(url).toContain('/admin');
    expect(url).not.toContain('/login');
  });
});
