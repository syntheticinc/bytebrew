// §1.23 Headers — GET /vite.svg returns 200 (not 404); regression for known bug
// TC: HDR-02 | Known gap: /vite.svg was 404 in admin build

import { test, expect, BASE_URL } from '../fixtures';

test.describe('Headers — /vite.svg 404 regression', () => {
  test('GET /admin/vite.svg does not 404 if referenced in admin HTML', async ({ request, page }) => {
    // First check if vite.svg is referenced in the admin page
    const adminPage = await request.get(`${BASE_URL}/admin/`);
    const html = await adminPage.text();

    if (!html.includes('vite.svg')) {
      // Not referenced — no regression to test
      test.skip(true, '/vite.svg is not referenced in admin HTML — no regression');
      return;
    }

    // If referenced, it should not 404
    const res = await request.get(`${BASE_URL}/admin/vite.svg`);
    expect.soft(res.status()).not.toBe(404);
    expect([200, 301, 302]).toContain(res.status());
  });

  test('admin page loads without broken asset requests (no 404 in network)', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    const failed404s: string[] = [];

    page.on('response', res => {
      if (res.status() === 404 && (res.url().includes('.js') || res.url().includes('.css') || res.url().includes('.svg'))) {
        failed404s.push(res.url());
      }
    });

    await page.goto('/admin/');
    await page.waitForLoadState('networkidle');

    // Known bug: /vite.svg was 404. After fix should be empty.
    expect.soft(failed404s).toHaveLength(0);
  });
});
