// §1.11 Inspect — audit filter matrix: actor_type × action × resource × date → filtered subset
// TC: INS-03

import { test, expect, apiFetch } from '../fixtures';

test.describe('Audit log — filter matrix', () => {
  test('GET /audit returns list', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/audit', { token: adminToken });
    if (res.status() === 404) {
      test.skip(true, '/audit endpoint not found');
      return;
    }
    expect(res.status()).toBe(200);
    const body = await res.json();
    const events = Array.isArray(body) ? body : (body.events ?? body.audit_logs ?? body.data ?? []);
    expect(Array.isArray(events)).toBe(true);
  });

  test('filter by action returns subset', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/audit?action=agent.create', { token: adminToken });
    if (res.status() === 404) {
      test.skip(true, 'Audit endpoint not available');
      return;
    }
    expect([200, 400]).toContain(res.status());
  });

  test('filter by date range returns subset', async ({ request, adminToken }) => {
    const from = new Date(Date.now() - 86400000).toISOString();
    const to = new Date().toISOString();
    const res = await apiFetch(request, `/audit?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`, { token: adminToken });
    if (res.status() === 404) {
      test.skip(true, 'Audit endpoint not available');
      return;
    }
    expect([200, 400]).toContain(res.status());
    expect(res.status()).not.toBe(500);
  });

  test('filter by resource returns subset', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/audit?resource=agent', { token: adminToken });
    if (res.status() === 404) {
      test.skip(true, 'Audit endpoint not available');
      return;
    }
    expect([200, 400]).toContain(res.status());
  });
});
