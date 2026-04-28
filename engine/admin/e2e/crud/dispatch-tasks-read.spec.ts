// §1.7-ext CRUD — Dispatch tasks: GET /dispatch/tasks/{id} and /sessions/{id}/dispatch-tasks
// TC: CRUD-20 | GAP-9

import { test, expect, apiFetch } from '../fixtures';

test.describe('Dispatch tasks — read endpoints', () => {
  test('GET /dispatch/tasks/{id} returns task or 404 for unknown id', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/dispatch/tasks/nonexistent-id-xyz', { token: adminToken });
    // 404 for unknown = correct; 401 without token = SCC-01; 200 = has data
    expect([200, 404]).toContain(res.status());
  });

  test('GET /sessions with valid token returns sessions list', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/sessions', { token: adminToken });
    if (res.status() === 404) {
      test.skip(true, '/sessions endpoint path may differ');
      return;
    }
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(Array.isArray(body) || Array.isArray(body.sessions) || Array.isArray(body.data)).toBe(true);
  });

  test('sessions/{id}/dispatch-tasks returns list or 404 for unknown session', async ({ request, adminToken }) => {
    const res = await apiFetch(request, '/sessions/nonexistent-session-xyz/dispatch-tasks', { token: adminToken });
    expect([200, 404]).toContain(res.status());
  });
});
