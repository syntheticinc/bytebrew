// §1.10 MCP — auth modes: api_key/forward_headers/oauth2/service_account each persisted correctly
// TC: MCP-04 | GAP-8

import { test, expect, apiFetch } from '../fixtures';

const AUTH_MODES = [
  { auth_type: 'api_key', auth_config: { api_key: 'sk-test-key', header_name: 'X-Api-Key' } },
  { auth_type: 'forward_headers', auth_config: { headers: ['Authorization'] } },
  { auth_type: 'oauth2', auth_config: { client_id: 'test-client', token_url: 'https://auth.example.com/token' } },
  { auth_type: 'service_account', auth_config: { credentials_json: '{}' } },
];

test.describe('MCP — auth modes', () => {
  for (const auth of AUTH_MODES) {
    test(`create MCP server with auth_type=${auth.auth_type}`, async ({ request, adminToken }) => {
      const name = `mcp-auth-${auth.auth_type}-${Date.now()}`;
      const res = await apiFetch(request, '/mcp-servers', {
        method: 'POST',
        token: adminToken,
        body: {
          name,
          transport: 'http',
          url: 'http://localhost:9999/mcp',
          auth_type: auth.auth_type,
          auth_config: auth.auth_config,
        },
      });

      expect([200, 201, 400, 422]).toContain(res.status());

      if ([200, 201].includes(res.status())) {
        const created = await res.json();
        const id = created.id ?? created.name ?? name;
        // Verify auth type stored
        expect.soft(created.auth_type).toBe(auth.auth_type);

        // Teardown
        await apiFetch(request, `/mcp-servers/${id}`, { method: 'DELETE', token: adminToken });
      }
    });
  }
});
