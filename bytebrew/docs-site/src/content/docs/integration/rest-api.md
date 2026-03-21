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
| `user_input_required` | `question`, `options`, `input_id` | Ask-user event. The agent called the `ask_user` tool and is waiting for a response. |
| `done` | `session_id`, `tokens` | Session completed. Contains the session ID for resuming and total token count. |
| `error` | `message`, `code` | Error occurred. The stream terminates after this event. |

### Example: full event stream

```
event: thinking
data: {"text":"Let me search for that information..."}

event: tool_call
data: {"tool":"search_products","input":{"query":"laptops under 1000"}}

event: tool_result
data: {"tool":"search_products","output":"[{\"name\":\"ProBook 450\",\"price\":849}]"}

event: message_delta
data: {"text":"I found "}

event: message_delta
data: {"text":"several options for you:\n\n"}

event: message_delta
data: {"text":"1. **ProBook 450** — $849"}

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
curl -X POST http://localhost:8080/api/v1/confirmations/conf_xyz \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"action": "approve"}'

# Reject
curl -X POST http://localhost:8080/api/v1/confirmations/conf_xyz \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"action": "reject", "reason": "Customer changed their mind"}'
```

### Handling user_input_required events

When an agent calls `ask_user`, the stream pauses:

```
event: user_input_required
data: {"question":"Which shipping method do you prefer?","options":["Standard","Express","Overnight"],"input_id":"inp_456"}
```

Provide the response:

```bash
curl -X POST http://localhost:8080/api/v1/inputs/inp_456 \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"response": "Express"}'
```

## Session management

### Creating a new session

Omit `session_id` to start a new conversation:

```bash
curl -N http://localhost:8080/api/v1/agents/my-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, I need help with my order"}'
```

The `done` event returns a `session_id`. Save it for continuations.

### Resuming a session

Pass `session_id` to continue a conversation with full history:

```bash
curl -N http://localhost:8080/api/v1/agents/my-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "Can you check order #12345?", "session_id": "sess_abc123"}'
```

### Listing sessions

```bash
curl "http://localhost:8080/api/v1/sessions?agent=my-agent&limit=20" \
  -H "Authorization: Bearer bb_your_token"
```

### Deleting a session

```bash
curl -X DELETE http://localhost:8080/api/v1/sessions/sess_abc123 \
  -H "Authorization: Bearer bb_your_token"
```

## Non-streaming mode

For clients that cannot handle SSE, set `stream: false` in the request body:

```bash
curl http://localhost:8080/api/v1/agents/my-agent/chat \
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

See [API Reference: Authentication](/getting-started/api-reference/#authentication) for details on scopes and token management.

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

## Rate limiting

The engine enforces rate limits per API token:

- Default: 60 requests per minute per token.
- Configurable in engine settings.
- Rate-limited responses return HTTP 429 with a `Retry-After` header.

---

## What's next

- [Multi-Agent Config](/integration/multi-agent/)
- [BYOK Integration](/integration/byok/)
- [API Reference](/getting-started/api-reference/)
