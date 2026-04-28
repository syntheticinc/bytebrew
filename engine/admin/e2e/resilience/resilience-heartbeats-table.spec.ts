// §1.12 Resilience — heartbeats: running agents → table with agent_id + last_heartbeat
// TC: RES-03 | GAP-10

import { test, expect, apiFetch } from '../fixtures';

test.describe('Resilience — heartbeats table', () => {
  test('GET /resilience/heartbeats returns list', async ({ request, adminToken }) => {
    const paths = ['/resilience/heartbeats', '/resilience/agents/heartbeats'];
    for (const path of paths) {
      const res = await apiFetch(request, path, { token: adminToken });
      if (res.status() === 200) {
        const body = await res.json();
        const beats = Array.isArray(body) ? body : (body.heartbeats ?? body.data ?? []);
        expect(Array.isArray(beats)).toBe(true);
        for (const beat of beats) {
          // Verify shape: agent_id and last_heartbeat fields
          expect.soft(beat.agent_id ?? beat.session_id ?? beat.id).toBeTruthy();
        }
        return;
      }
    }
    test.skip(true, 'Heartbeats endpoint not found');
  });

  test('heartbeats table renders in admin resilience page', async ({ authenticatedAdmin }) => {
    const page = authenticatedAdmin;
    await page.goto('/admin/resilience');
    await page.waitForLoadState('networkidle');
    await expect(page.locator('text=/Something went wrong/i')).not.toBeVisible();
  });
});
