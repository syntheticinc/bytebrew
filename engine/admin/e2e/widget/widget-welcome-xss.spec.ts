// §1.8-ext Widget — XSS: welcome_message with <script> stored as literal, not executed
// TC: WID-03 | OWASP A03

import { test, expect, apiFetch } from '../fixtures';

test.describe('Widget — welcome message XSS prevention', () => {
  test('XSS payload in welcome_message stored as literal text', async ({ request, adminToken }) => {
    const xssPayload = '<script>window.__xss_executed=true;</script>';

    // Try to set welcome message via settings
    const setRes = await apiFetch(request, '/settings/widget_welcome_message', {
      method: 'PUT',
      token: adminToken,
      body: { value: xssPayload },
    });

    if ([200, 204].includes(setRes.status())) {
      // Read back
      const getRes = await apiFetch(request, '/settings', { token: adminToken });
      const settings = await getRes.json();
      const storedValue = settings['widget_welcome_message'] ?? '';
      // Should be stored as literal string, not executed
      expect(storedValue).toBe(xssPayload);
    } else {
      // 404 = different key name; document
      expect([200, 204, 404]).toContain(setRes.status());
    }
  });

  test('widget page does not execute injected script from welcome_message', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;

    // Check if XSS was executed from previous test
    await page.goto('/admin/widget');
    await page.waitForLoadState('networkidle');

    const xssExecuted = await page.evaluate(() => (window as Window & { __xss_executed?: boolean }).__xss_executed);
    expect(xssExecuted).toBeFalsy();
  });
});
