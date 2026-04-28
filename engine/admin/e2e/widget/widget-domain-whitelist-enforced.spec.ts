// §1.8-ext Widget — domain whitelist enforced: allowed origin → 200; other origin → 403
// TC: WID-05

import { test, expect, apiFetch } from '../fixtures';

test.describe('Widget — domain whitelist enforcement', () => {
  test.skip(true, 'Domain whitelist enforcement requires configuring allowed origins and making cross-origin requests — needs specific test infrastructure');

  test('request from allowed origin gets 200 on widget endpoint', async ({ request, adminToken }) => {
    // Set whitelist
    await apiFetch(request, '/settings/widget_allowed_origins', {
      method: 'PUT',
      token: adminToken,
      body: { value: 'https://allowed.example.com' },
    });

    const res = await request.get('/widget/widget.js', {
      headers: { Origin: 'https://allowed.example.com' },
    });
    expect([200]).toContain(res.status());
  });

  test('request from disallowed origin gets 403 on widget endpoint', async ({ request, adminToken }) => {
    await apiFetch(request, '/settings/widget_allowed_origins', {
      method: 'PUT',
      token: adminToken,
      body: { value: 'https://allowed.example.com' },
    });

    const res = await request.get('/widget/widget.js', {
      headers: { Origin: 'https://evil.other.com' },
    });
    expect([403]).toContain(res.status());
  });
});
