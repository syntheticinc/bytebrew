---
title: REST API Chat Integration
description: Complete guide to integrating with ByteBrew Engine SSE streaming API — event types, session management, error handling, and authentication.
---

This guide covers everything you need to build a client that communicates with ByteBrew Engine over the REST API with SSE streaming.

## SSE event types reference

When you send a message to `POST /api/v1/agents/{name}/chat`, the engine responds with a stream of Server-Sent Events. Each event has a `type` field in the event name and a JSON `data` payload.

| Event | Data fields | Description |
|-------|------------|-------------|
| `message_delta` | `text` | Streaming token. A partial text chunk from the agent. Concatenate all `message_delta` events for the full response. |
| `message` | `text`, `role` | Complete message. Sent when a full message is available (non-streaming mode or final assembly). |
| `thinking` | `text` | Reasoning started. The agent is processing internally. Contains partial reasoning text (if the model supports it). |
| `tool_call` | `tool`, `input` | Tool execution started. Contains the tool name and the input parameters the agent provided. |
| `tool_result` | `tool`, `output`, `error` | Tool execution completed. Contains the tool output or error message. |
| `confirmation` | `tool`, `input`, `confirmation_id` | Requires user approval. A tool with `confirm_before` is about to execute. Send approval via the confirmation endpoint. |
| `agent_spawn` | `agent`, `task` | Sub-agent created. The supervisor spawned a child agent to handle a subtask. |
| `agent_result` | `agent`, `result`, `status` | Sub-agent completed or failed. Contains the summary returned by the sub-agent. |
| `user_input_required` | `question`, `options`, `input_type`, `input_id` | Ask-user event. The agent called the `ask_user` tool and is waiting for a response. See [Extended options format](#extended-user-input-options) below. |
| `structured_output` | `output_type`, `title`, `rows`, `actions` | Structured data display (summary tables, action buttons). See [Structured output events](#structured-output-events) below. |
| `done` | `session_id`, `tokens` | Session completed. Contains the session ID for resuming and total token count. |
| `error` | `message`, `code` | Error occurred. The stream terminates after this event. |

### Example: full event stream

```
event: thinking
data: {"content":"Let me search for that information..."}

event: tool_call
data: {"tool":"search_products","input":{"query":"laptops under 1000"}}

event: tool_result
data: {"tool":"search_products","output":"[{\"name\":\"ProBook 450\",\"price\":849}]"}

event: message_delta
data: {"content":"I found "}

event: message_delta
data: {"content":"several options for you:\n\n"}

event: message_delta
data: {"content":"1. **ProBook 450** — $849"}

event: done
data: {"session_id":"sess_abc123","tokens":156}
```

### Handling confirmation events

When a tool has `confirm_before` configured, the stream pauses with a `confirmation` event:

```
event: confirmation
data: {"tool":"create_order","input":{"customer_id":"cust_123","items":"ProBook 450 x1"},"confirmation_id":"conf_xyz"}
```

To approve or reject:

```bash
# Approve
curl -X POST http://localhost:8443/api/v1/confirmations/conf_xyz \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"action": "approve"}'

# Reject
curl -X POST http://localhost:8443/api/v1/confirmations/conf_xyz \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"action": "reject", "reason": "Customer changed their mind"}'
```

### Handling user_input_required events

When an agent calls `ask_user`, the stream pauses. The event now supports extended options with `input_type`, structured `value`/`description` fields, and grid layout:

```
event: user_input_required
data: {"question":"Which shipping method do you prefer?","input_type":"single_select","options":[{"label":"Standard","value":"standard","description":"5-7 business days"},{"label":"Express","value":"express","description":"2-3 business days"},{"label":"Overnight","value":"overnight","description":"Next business day"}],"input_id":"inp_456"}
```

#### Input types

| `input_type` | Description |
|--------------|-------------|
| `text` | Free-text input (default). No options required. |
| `single_select` | Pick one option from the list. |
| `multi_select` | Pick one or more options from the list. |
| `confirm` | Yes/No confirmation prompt. |

#### Option fields

Each option in the `options` array supports:

| Field | Required | Description |
|-------|----------|-------------|
| `label` | Yes | Display text shown to the user. |
| `value` | No | Machine-readable value sent back (defaults to `label` if omitted). |
| `description` | No | Additional context shown below the label. |

The event may also include a `columns` field (integer) to hint at grid layout for card-style UIs.

Provide the response:

```bash
curl -X POST http://localhost:8443/api/v1/inputs/inp_456 \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"response": "express"}'
```

### Structured output events

The `structured_output` event delivers rich data blocks (tables, action buttons) to the client for display:

```
event: structured_output
data: {"output_type":"summary_table","title":"Project Summary","rows":[{"label":"Name","value":"MyApp"},{"label":"Status","value":"Active"},{"label":"Users","value":"1,234"}],"actions":[{"label":"Deploy","type":"primary","value":"deploy"},{"label":"Cancel","type":"secondary","value":"cancel"}]}
```

| Field | Description |
|-------|-------------|
| `output_type` | Type of structured output (e.g. `summary_table`). |
| `title` | Optional title for the output block. |
| `description` | Optional description text. |
| `rows` | Array of `{label, value}` pairs for table display. |
| `actions` | Array of `{label, type, value}` action buttons (`type`: `primary` or `secondary`). |

Structured output is display-only -- the client renders it but does not need to respond. If action buttons are present, the client can send the `value` back as a regular chat message.

## Session management

### Creating a new session

Omit `session_id` to start a new conversation:

```bash
curl -N http://localhost:8443/api/v1/agents/my-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, I need help with my order"}'
```

The `done` event returns a `session_id`. Save it for continuations.

### Resuming a session

Pass `session_id` to continue a conversation with full history:

```bash
curl -N http://localhost:8443/api/v1/agents/my-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "Can you check order #12345?", "session_id": "sess_abc123"}'
```

### Listing sessions

```bash
curl "http://localhost:8443/api/v1/sessions?agent=my-agent&limit=20" \
  -H "Authorization: Bearer bb_your_token"
```

### Deleting a session

```bash
curl -X DELETE http://localhost:8443/api/v1/sessions/sess_abc123 \
  -H "Authorization: Bearer bb_your_token"
```

## Non-streaming mode

For clients that cannot handle SSE, set `stream: false` in the request body:

```bash
curl http://localhost:8443/api/v1/agents/my-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello", "stream": false}'
```

Response (standard JSON, not SSE):

```json
{
  "response": "Hello! How can I help you today?",
  "session_id": "sess_abc123",
  "tokens": 42,
  "tool_calls": []
}
```

## Authentication

All endpoints require a Bearer token in the `Authorization` header:

```
Authorization: Bearer bb_your_api_token
```

Tokens are created in **Admin Dashboard** -> **API Keys**. Each token has scopes that limit what it can access. For chat integrations, the `chat` scope is sufficient.

See [API Reference: Authentication](/docs/getting-started/api-reference/#authentication) for details on scopes and token management.

## Error handling

### HTTP errors

| Status | Meaning |
|--------|---------|
| `400` | Bad request. Invalid JSON or missing required fields. |
| `401` | Unauthorized. Missing or invalid API token. |
| `403` | Forbidden. Token lacks the required scope. |
| `404` | Agent not found. Check the agent name in the URL. |
| `429` | Rate limited. Too many requests. Retry after the `Retry-After` header value. |
| `500` | Internal server error. Check engine logs. |

### SSE error events

Errors during streaming are sent as `error` events:

```
event: error
data: {"message":"Model returned an error: context length exceeded","code":"model_error"}
```

The stream closes after an error event. Your client should reconnect or show the error to the user.

### Retry strategy

For transient errors (429, 500), implement exponential backoff:

```javascript
async function chatWithRetry(message, maxRetries = 3) {
  for (let attempt = 0; attempt < maxRetries; attempt++) {
    try {
      return await sendMessage(message);
    } catch (error) {
      if (attempt === maxRetries - 1) throw error;
      const delay = Math.pow(2, attempt) * 1000;
      await new Promise(resolve => setTimeout(resolve, delay));
    }
  }
}
```

## JavaScript SSE client example

Do NOT use `EventSource` -- it only supports GET requests. Use `fetch` + `ReadableStream` for POST-based SSE:

```javascript
const response = await fetch('http://localhost:8443/api/v1/agents/my-agent/chat', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer bb_your_token',
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({ message: 'Hello', session_id: null }),
});

const reader = response.body.getReader();
const decoder = new TextDecoder();
let buffer = '';

while (true) {
  const { done, value } = await reader.read();
  if (done) break;
  buffer += decoder.decode(value, { stream: true });
  const lines = buffer.split('\n');
  buffer = lines.pop() || '';
  let currentEvent = '';
  for (const line of lines) {
    if (line.startsWith('event: ')) currentEvent = line.slice(7);
    if (line.startsWith('data: ')) {
      const data = JSON.parse(line.slice(6));
      if (currentEvent === 'message_delta') console.log(data.content);
      if (currentEvent === 'done') console.log('Session:', data.session_id);
    }
  }
}
```

## Rate limiting

The engine enforces rate limits per API token:

- Default: 60 requests per minute per token.
- Configurable in engine settings.
- Rate-limited responses return HTTP 429 with a `Retry-After` header.

### Rate limit headers

Every API response includes rate limit headers when configurable rate limiting is enabled (EE):

| Header | Description |
|--------|-------------|
| `X-RateLimit-Limit` | Maximum number of requests allowed in the current window. |
| `X-RateLimit-Remaining` | Number of requests remaining in the current window. |
| `X-RateLimit-Reset` | Unix timestamp (seconds) when the current window resets. |

```
HTTP/1.1 200 OK
X-RateLimit-Limit: 500
X-RateLimit-Remaining: 487
X-RateLimit-Reset: 1711929600
```

See [Configuration: Rate Limits](/docs/getting-started/configuration/#rate-limits-ee) for setup.

## Additional API endpoints

### Tool call audit log (EE)

Query tool call history for auditing and debugging. Requires `admin` scope.

```bash
curl "http://localhost:8443/api/v1/audit/tool-calls?agent=sales-agent&tool=create_order&page=1&per_page=20" \
  -H "Authorization: Bearer bb_your_token"
```

#### Query parameters

| Parameter | Description |
|-----------|-------------|
| `session_id` | Filter by session ID. |
| `agent` | Filter by agent name. |
| `tool` | Filter by tool name. |
| `status` | Filter by status: `completed` or `failed`. |
| `user_id` | Filter by user ID. |
| `from` | Start date (RFC3339 or YYYY-MM-DD). |
| `to` | End date (RFC3339 or YYYY-MM-DD). |
| `page` | Page number (default: 1). |
| `per_page` | Results per page (default: 50, max: 100). |

#### Response

```json
{
  "data": [
    {
      "id": 42,
      "session_id": "sess_abc123",
      "agent_name": "sales-agent",
      "tool_name": "create_order",
      "input": "{\"customer_id\":\"cust_123\"}",
      "output": "{\"order_id\":\"ord_456\"}",
      "status": "completed",
      "duration_ms": 340,
      "user_id": "user_789",
      "created_at": "2026-03-20T14:30:00Z"
    }
  ],
  "total": 156,
  "page": 1,
  "per_page": 20,
  "total_pages": 8
}
```

### Model registry

Browse the built-in catalog of known models and providers. No authentication required.

```bash
# List all models
curl http://localhost:8443/api/v1/models/registry

# Filter by provider
curl "http://localhost:8443/api/v1/models/registry?provider=anthropic"

# Filter by tier
curl "http://localhost:8443/api/v1/models/registry?tier=1"

# Filter by tool support
curl "http://localhost:8443/api/v1/models/registry?supports_tools=true"

# List all providers
curl http://localhost:8443/api/v1/models/registry/providers
```

See [Model Registry](/docs/deployment/model-registry/) for full details.

### Rate limit usage (EE)

Check current rate limit usage for a specific key. Requires `admin` scope.

```bash
curl "http://localhost:8443/api/v1/rate-limits/usage?key_header=X-Org-Id&key_value=org-123" \
  -H "Authorization: Bearer bb_your_token"
```

```json
{
  "rule": "per-org",
  "key": "org-123",
  "tier": "pro",
  "used": 42,
  "limit": 500,
  "window": "24h0m0s",
  "resets_at": "2026-03-25T00:00:00Z"
}
```

### Prometheus metrics (EE)

The engine exposes Prometheus-compatible metrics at `/metrics`. No authentication required.

```bash
curl http://localhost:8443/metrics
```

See [Production: Prometheus Metrics](/docs/deployment/production/#prometheus-metrics-ee) for available metrics and Kubernetes integration.

---

## What's next

- [Multi-Agent Config](/docs/integration/multi-agent/)
- [BYOK Integration](/docs/integration/byok/)
- [API Reference](/docs/getting-started/api-reference/)
