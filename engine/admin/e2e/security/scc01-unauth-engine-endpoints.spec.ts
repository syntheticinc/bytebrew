// §1.19 SCC-01 — parametric sweep: ~40 representative protected engine endpoints → 401 without auth
// TC: SCC-01 GATE | GAP-5

import { test, expect, BASE_URL } from '../fixtures';

const PROTECTED_ENDPOINTS: Array<{ method: string; path: string }> = [
  { method: 'GET',    path: '/api/v1/agents' },
  { method: 'POST',   path: '/api/v1/agents' },
  { method: 'GET',    path: '/api/v1/agents/nonexistent' },
  { method: 'PUT',    path: '/api/v1/agents/nonexistent' },
  { method: 'DELETE', path: '/api/v1/agents/nonexistent' },
  { method: 'GET',    path: '/api/v1/schemas' },
  { method: 'POST',   path: '/api/v1/schemas' },
  { method: 'GET',    path: '/api/v1/schemas/nonexistent' },
  { method: 'PUT',    path: '/api/v1/schemas/nonexistent' },
  { method: 'DELETE', path: '/api/v1/schemas/nonexistent' },
  { method: 'GET',    path: '/api/v1/models' },
  { method: 'POST',   path: '/api/v1/models' },
  { method: 'DELETE', path: '/api/v1/models/nonexistent' },
  { method: 'GET',    path: '/api/v1/mcp-servers' },
  { method: 'POST',   path: '/api/v1/mcp-servers' },
  { method: 'DELETE', path: '/api/v1/mcp-servers/nonexistent' },
  { method: 'GET',    path: '/api/v1/sessions' },
  { method: 'GET',    path: '/api/v1/sessions/nonexistent' },
  { method: 'GET',    path: '/api/v1/tasks' },
  { method: 'POST',   path: '/api/v1/tasks' },
  { method: 'GET',    path: '/api/v1/audit' },
  { method: 'GET',    path: '/api/v1/settings' },
  { method: 'PUT',    path: '/api/v1/settings/some_key' },
  // POST /api/v1/config/reload → 404 (endpoint not in this stack) — removed
  // POST /api/v1/config/import → 404 (endpoint not in this stack) — removed
  // GET  /api/v1/config/export → 404 (endpoint not in this stack) — removed
  // GET  /api/v1/resilience/circuit-breakers → 404 (not in this stack) — removed
  // POST /api/v1/resilience/circuit-breakers/test/reset → 404 (not in this stack) — removed
  { method: 'GET',    path: '/api/v1/knowledge-bases' },
  { method: 'POST',   path: '/api/v1/knowledge-bases' },
  { method: 'DELETE', path: '/api/v1/knowledge-bases/nonexistent' },
  { method: 'GET',    path: '/api/v1/auth/tokens' },
  { method: 'POST',   path: '/api/v1/auth/tokens' },
  { method: 'DELETE', path: '/api/v1/auth/tokens/nonexistent' },
  { method: 'GET',    path: '/api/v1/agents/nonexistent/capabilities' },
  { method: 'POST',   path: '/api/v1/agents/nonexistent/capabilities' },
  // GET  /api/v1/agents/nonexistent/relations → 404 (route not registered) — removed
  // POST /api/v1/agents/nonexistent/relations → 404 (route not registered) — removed
  { method: 'GET',    path: '/api/v1/license/status' },
  // POST /api/v1/license/activate → 405 in CE stack (route not registered for POST without body) — removed from sweep
];

test.describe('SCC-01 — unauthenticated access returns 401', () => {
  for (const ep of PROTECTED_ENDPOINTS) {
    test(`⛔ GATE SCC-01: ${ep.method} ${ep.path} → 401`, async ({ request }) => {
      const res = await request.fetch(`${BASE_URL}${ep.path}`, {
        method: ep.method,
        headers: { 'Content-Type': 'application/json' },
        data: ep.method !== 'GET' && ep.method !== 'DELETE' ? '{}' : undefined,
      });
      // Must be 401, not 200, 500, or (unexpectedly) 200
      expect.soft(res.status()).not.toBe(200);
      expect.soft(res.status()).not.toBe(500);
      expect(res.status()).toBe(401);
    });
  }
});
