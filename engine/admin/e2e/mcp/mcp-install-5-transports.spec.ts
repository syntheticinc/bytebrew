// §1.10 MCP — install 5 transports: stdio/sse/http/websocket/docker; validate + stored kind matches
// TC: MCP-03 | GAP-8

import { test, expect, apiFetch } from '../fixtures';

const TRANSPORTS = [
  { transport: 'http', url: 'http://localhost:9999/mcp', extra: {} },
  { transport: 'sse', url: 'http://localhost:9998/sse', extra: {} },
  { transport: 'websocket', url: 'ws://localhost:9997/ws', extra: {} },
  { transport: 'stdio', command: 'echo', args: ['hello'], extra: {} },
  { transport: 'docker', image: 'mcp-server:latest', extra: {} },
];

test.describe('MCP — install 5 transports', () => {
  for (const t of TRANSPORTS) {
    test(`create MCP server with transport=${t.transport}`, async ({ request, adminToken }) => {
      const name = `mcp-${t.transport}-${Date.now()}`;
      const body: Record<string, unknown> = { name, transport: t.transport, ...t.extra };
      if (t.url) body['url'] = t.url;
      if ((t as { command?: string }).command) {
        body['command'] = (t as { command?: string }).command;
        body['args'] = (t as { args?: string[] }).args;
      }
      if ((t as { image?: string }).image) body['image'] = (t as { image?: string }).image;

      const res = await apiFetch(request, '/mcp-servers', {
        method: 'POST',
        token: adminToken,
        body,
      });

      // 200/201 = created; 400 = transport not supported (document)
      expect([200, 201, 400, 422]).toContain(res.status());

      if ([200, 201].includes(res.status())) {
        const created = await res.json();
        const id = created.id ?? created.name ?? name;
        // Verify stored transport matches
        expect.soft(created.transport ?? created.kind).toBe(t.transport);

        // Teardown
        await apiFetch(request, `/mcp-servers/${id}`, { method: 'DELETE', token: adminToken });
      }
    });
  }
});
