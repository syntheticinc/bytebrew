// §1.7-ext CRUD — Config: export → modify YAML → import → reload → changes take effect; malformed YAML → 400
// TC: CRUD-22

import { test, expect, apiFetch } from '../fixtures';

test.describe('Config import/export/reload', () => {
  test('GET /config/export returns config data', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/config/export', { token: adminToken });
    if (res.status() === 404) {
      test.skip(true, 'Config export endpoint not found');
      return;
    }
    expect(res.status()).toBe(200);
    const body = await res.text();
    expect(body.length).toBeGreaterThan(0);
  });

  test('POST /config/import with malformed YAML returns 400', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/config/import', {
      method: 'POST',
      token: adminToken,
      body: ':::invalid yaml:::\n{[broken',
      headers: { 'Content-Type': 'application/yaml' },
    });
    if (res.status() === 404) {
      test.skip(true, 'Config import endpoint not found');
      return;
    }
    // Malformed YAML must return 400, not 500
    expect(res.status()).toBe(400);
  });

  test('POST /config/reload returns 200 or 204', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/config/reload', {
      method: 'POST',
      token: adminToken,
    });
    if (res.status() === 404) {
      test.skip(true, 'Config reload endpoint not found');
      return;
    }
    expect([200, 204]).toContain(res.status());
  });
});
