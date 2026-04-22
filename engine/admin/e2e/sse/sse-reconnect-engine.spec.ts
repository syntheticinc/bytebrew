// §1.26 SSE — reconnect with Last-Event-ID: receive events, disconnect, reconnect → only new events delivered
// TC: SSE-01 | GAP-18

import { test, expect, apiFetch } from '../fixtures';

test.describe('SSE — reconnect with Last-Event-ID', () => {
  test.skip(true, 'GAP-18: HTTP SSE reconnect with Last-Event-ID replay requires active LLM session. Cannot test deterministically without real model. Document: gRPC SubscribeSession supports last_event_id; HTTP SSE support TBD.');

  test('SSE reconnect delivers only events after Last-Event-ID', async ({ request, adminToken }) => {
    // This test requires:
    // 1. An active SSE chat session
    // 2. Ability to simulate disconnect and reconnect with Last-Event-ID header
    // 3. Verify no duplicate events delivered

    // Get a schema to chat with
    const schemasRes = await apiFetch(request, '/schemas', { token: adminToken });
    const body = await schemasRes.json();
    const schemas = Array.isArray(body) ? body : (body.schemas ?? body.data ?? []);
    if (schemas.length === 0) {
      test.skip(true, 'No schemas available for SSE test');
      return;
    }

    const schemaId = schemas[0].id;
    // Would need EventSource or fetch with streaming to test properly
    // Placeholder assertion:
    expect(schemaId).toBeTruthy();
  });
});
