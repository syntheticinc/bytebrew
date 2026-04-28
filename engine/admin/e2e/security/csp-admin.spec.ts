// §1.9 Security — CSP headers on admin responses
// TC: SEC-CSP-01 | Known gap: CSP cleanup needed (Google Fonts, inline SVG, /vite.svg 404)

import { test, expect, BASE_URL } from '../fixtures';

test.describe('Admin — CSP and security headers', () => {
  test('admin page includes X-Content-Type-Options: nosniff', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/admin/`);
    const headers = res.headers();
    expect.soft(headers['x-content-type-options']).toMatch(/nosniff/i);
  });

  test('admin page includes X-Frame-Options or frame-ancestors CSP', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/admin/`);
    const headers = res.headers();
    const xFrameOptions = headers['x-frame-options'] ?? '';
    const csp = headers['content-security-policy'] ?? '';
    const hasFrameProtection = /DENY|SAMEORIGIN/i.test(xFrameOptions) || /frame-ancestors/i.test(csp);
    expect.soft(hasFrameProtection).toBe(true);
  });

  test('admin page response has Content-Security-Policy header', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/admin/`);
    const headers = res.headers();
    const csp = headers['content-security-policy'] ?? '';
    // Known gap: CSP may be missing or incomplete
    // Document current state — soft assertion
    expect.soft(csp.length).toBeGreaterThan(0);
  });

  test('API endpoints return JSON content-type on errors', async ({ request }) => {
    const res = await request.get(`${BASE_URL}/api/v1/agents`);
    // 401 response should be JSON, not HTML
    if (res.status() === 401) {
      const ct = res.headers()['content-type'] ?? '';
      expect.soft(ct).toMatch(/application\/json/i);
    }
  });
});
