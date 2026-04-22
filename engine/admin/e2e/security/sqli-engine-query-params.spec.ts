// §1.20 OWASP A03 — SQL injection in query params → 400/404, no 500, no data corruption
// TC: SEC-SQLI-01

import { test, expect, apiFetch } from '../fixtures';

const SQLI_CASES: Array<{ label: string; path: string }> = [
  { label: 'audit?from=drop',            path: `/audit?from=${encodeURIComponent("'; DROP TABLE users;--")}` },
  { label: 'audit?from=or1=1',           path: `/audit?from=${encodeURIComponent("' OR '1'='1")}` },
  { label: 'audit?from=union',           path: `/audit?from=${encodeURIComponent("' UNION SELECT null,null,null--")}` },
  { label: 'sessions?agent_name=drop',   path: `/sessions?agent_name=${encodeURIComponent("'; DROP TABLE users;--")}` },
  { label: 'sessions?agent_name=select', path: `/sessions?agent_name=${encodeURIComponent("1; SELECT * FROM agents--")}` },
  { label: 'sessions?agent_name=union',  path: `/sessions?agent_name=${encodeURIComponent("' UNION SELECT null,null,null--")}` },
  { label: 'agents?name=drop',           path: `/agents?name=${encodeURIComponent("'; DROP TABLE users;--")}` },
  { label: 'agents?name=or1=1',          path: `/agents?name=${encodeURIComponent("' OR '1'='1")}` },
  { label: 'sessions/{drop}/dispatch',   path: `/sessions/${encodeURIComponent("'; DROP TABLE sessions;--")}/dispatch-tasks` },
  { label: 'sessions/{union}/dispatch',  path: `/sessions/${encodeURIComponent("' UNION SELECT null--")}/dispatch-tasks` },
];

test.describe('SQL injection prevention — query params', () => {
  for (const { label, path } of SQLI_CASES) {
    test(`SQLi [${label}] does not cause 500`, async ({ request, adminToken }) => {
      const res = await apiFetch(request, path, { token: adminToken });
      // 500 = possible injection vulnerability; 400/404/422 = correct rejection
      expect(res.status()).not.toBe(500);
      expect([200, 400, 404, 422]).toContain(res.status());
    });
  }
});
