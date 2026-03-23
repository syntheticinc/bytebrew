---
title: API Reference
description: Complete REST API reference for ByteBrew Engine including chat, agents, sessions, tasks, config, and health endpoints.
---

Complete REST API reference for the ByteBrew Engine. All endpoints return JSON (except SSE streams) and accept JSON request bodies.

## Authentication

All API requests must include a valid API token in the `Authorization` header. Tokens are created through the Admin Dashboard and are scoped to specific capabilities.

### Login (get a JWT token)

Use the login endpoint to obtain a JWT token for API access:

```bash
curl -X POST http://localhost:8443/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "changeme"}'
```

```json
{"token": "eyJhbG...", "expires_at": "2026-03-23T07:00:00Z"}
```

The returned JWT can be used in the `Authorization: Bearer <token>` header for all subsequent requests. JWTs expire after 24 hours.

### Creating a persistent API token

For long-lived integrations, create API tokens through the Admin Dashboard:

- Navigate to **Admin Dashboard** -> **API Keys**
- Click "Create API Key" and select the scopes you need
- Copy the token immediately -- it is shown only once and cannot be recovered
- Tokens are prefixed with `bb_` for easy identification in logs and config

### Using the token

```bash
# curl
curl http://localhost:8443/api/v1/agents \
  -H "Authorization: Bearer bb_your_api_token"
```

```javascript
// JavaScript (fetch)
const response = await fetch('http://localhost:8443/api/v1/agents', {
  headers: { 'Authorization': 'Bearer bb_your_api_token' },
});
```

```python
# Python (requests)
import requests
response = requests.get(
    'http://localhost:8443/api/v1/agents',
    headers={'Authorization': 'Bearer bb_your_api_token'},
)
```

### Token scopes

| Scope | Description |
|-------|-------------|
| `chat` | Send messages to agents (POST /agents/\{name\}/chat) |
| `tasks` | Create, list, cancel tasks and provide input |
| `agents:read` | List and inspect agent configurations |
| `config` | Reload, export, and import configuration |
| `admin` | Full access to all endpoints including API key management |

:::tip[Least privilege]
Create separate tokens for different integrations. A chatbot frontend only needs `chat` scope. A CI/CD pipeline might need `config` for hot-reload. Use `admin` scope only for the Admin Dashboard and management scripts.
:::

### Error responses

```json
// 401 Unauthorized — missing or invalid token
{"error": "unauthorized", "message": "Invalid or expired API token"}

// 403 Forbidden — token lacks required scope
{"error": "forbidden", "message": "Token does not have 'config' scope"}
```

**Base URL:** `http://localhost:8443/api/v1`

**Content-Type:** `application/json`

## Chat (SSE Streaming)

Send a message to an agent and receive a stream of Server-Sent Events. This is the primary endpoint for building conversational interfaces.

```
POST /api/v1/agents/{name}/chat
```

### Request body

| Parameter | Default | Description |
|-----------|---------|-------------|
| `message` * | -- | The user message to send to the agent. |
| `session_id` | auto-generated | Session ID for continuing a conversation. Omit to start a new session. |

### Full example

```bash
# Start a new conversation
curl -N http://localhost:8443/api/v1/agents/sales-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "What laptops do you have under $1000?"}'
```

### SSE event types

| Event | Description |
|-------|-------------|
| `message_delta` | Text chunk from the agent. Concatenate all deltas for the full response. |
| `message` | Complete message (sent after all deltas). |
| `tool_call` | Agent is calling a tool. Contains tool name and input parameters. |
| `tool_result` | Result returned from the tool execution. |
| `agent_spawn` | A sub-agent was spawned by the current agent. |
| `agent_result` | Result from a completed sub-agent. |
| `user_input_required` | Agent is asking the user a question (ask_user tool). |
| `error` | An error occurred during processing. |
| `done` | Stream is complete. Contains `session_id`. |

### Example response stream

```
event: message_delta
data: {"content":"I found several laptops under $1000. "}

event: tool_call
data: {"tool":"search_products","input":{"query":"laptops under 1000","limit":"5"}}

event: tool_result
data: {"tool":"search_products","output":"[{\"name\":\"ProBook 450\",\"price\":849}...]"}

event: message_delta
data: {"content":"Here are the top options:\n\n1. **ProBook 450** — $849..."}

event: message
data: {"content":"I found several laptops under $1000. Here are the top options:\n\n1. **ProBook 450** — $849..."}

event: done
data: {"session_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890"}
```

### Continue the conversation

```bash
curl -N http://localhost:8443/api/v1/agents/sales-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "Tell me more about the ProBook 450", "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}'
```

## Agents

List and inspect configured agents. Requires `agents:read` or `admin` scope.

```bash
# List all agents
curl http://localhost:8443/api/v1/agents \
  -H "Authorization: Bearer bb_your_token"
```

```json
[
  {
    "name": "sales-agent",
    "model": "glm-5",
    "lifecycle": "persistent",
    "tools_count": 5,
    "has_knowledge": true
  }
]
```

```bash
# Get agent details
curl http://localhost:8443/api/v1/agents/sales-agent \
  -H "Authorization: Bearer bb_your_token"
```

```json
{
  "name": "sales-agent",
  "model": "glm-5",
  "lifecycle": "persistent",
  "tools": ["knowledge_search", "search_products", "create_order"],
  "mcp_servers": ["crm-api"],
  "can_spawn": ["researcher"],
  "max_steps": 50,
  "max_context_size": 16000
}
```

### Create Agent

```bash
curl -X POST http://localhost:8443/api/v1/agents \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-agent",
    "system_prompt": "You are a helpful assistant.",
    "model_id": 1,
    "tools": ["web_search", "knowledge_search"],
    "can_spawn": ["researcher"],
    "lifecycle": "persistent",
    "max_steps": 50
  }'
```

### Update Agent

```bash
curl -X PUT http://localhost:8443/api/v1/agents/my-agent \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"system_prompt": "Updated prompt.", "max_steps": 100}'
```

### Delete Agent

```bash
curl -X DELETE http://localhost:8443/api/v1/agents/my-agent \
  -H "Authorization: Bearer bb_your_token"
# Returns 204 No Content
```

## Models

CRUD for LLM model providers. Requires `admin` scope.

### List Models

```bash
curl http://localhost:8443/api/v1/models \
  -H "Authorization: Bearer bb_your_token"
```

```json
[
  {
    "id": 1,
    "name": "qwen3",
    "type": "ollama",
    "base_url": "http://localhost:11434",
    "model_name": "qwen3:30b-a3b",
    "has_api_key": false,
    "created_at": "2026-03-20T12:00:00Z"
  }
]
```

### Create Model

```bash
curl -X POST http://localhost:8443/api/v1/models \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-model",
    "type": "ollama",
    "model_name": "llama3.2",
    "base_url": "http://localhost:11434",
    "api_key": "ollama"
  }'
```

Supported types: `ollama`, `openai_compatible`, `anthropic`.

:::note
Models are resolved dynamically — no Engine restart needed after adding a model. The next chat request will use the new model immediately.
:::

### Update Model

```bash
curl -X PUT http://localhost:8443/api/v1/models/my-model \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"api_key": "new-key", "model_name": "qwen3:70b"}'
```

API key is only updated if a non-empty value is provided.

### Delete Model

```bash
curl -X DELETE http://localhost:8443/api/v1/models/my-model \
  -H "Authorization: Bearer bb_your_token"
# Returns 204 No Content
```

### Verify Model Connectivity

Test that a model is accessible and supports tool calling before using it.

```bash
curl -X POST http://localhost:8443/api/v1/models/my-model/verify \
  -H "Authorization: Bearer bb_your_token"
```

```json
{
  "connectivity": "ok",
  "tool_calling": "supported",
  "response_time_ms": 1240,
  "model_name": "llama3.2",
  "provider": "ollama",
  "error": null
}
```

| Field | Values | Description |
|-------|--------|-------------|
| connectivity | `ok`, `error` | Whether the API endpoint is accessible |
| tool_calling | `supported`, `not_detected`, `skipped` | Whether the model generates tool calls |
| response_time_ms | number | Latency of the ping request |
| error | string or null | Error details if connectivity failed |

:::tip
For known providers (OpenAI, Anthropic, Google, Mistral), tool calling probe is skipped — all their models support it. Probe runs only for Ollama and custom providers.
:::

## Session Respond

When an agent calls `ask_user` with structured options, the client receives a `user_input_required` SSE event. Use this endpoint to send the user's answer back to the agent.

```bash
curl -X POST http://localhost:8443/api/v1/sessions/{session_id}/respond \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{
    "call_id": "server-ask_user-3",
    "answers": ["Email"]
  }'
```

The `call_id` comes from the SSE event's metadata. The agent receives the answer and continues execution.

## Sessions

Manage conversation sessions. Sessions store the full message history between a user and an agent. Requires `chat` or `admin` scope.

```bash
# List sessions (with optional filters)
curl "http://localhost:8443/api/v1/sessions?agent=sales-agent&limit=10" \
  -H "Authorization: Bearer bb_your_token"
```

```json
{
  "sessions": [
    {
      "id": "a1b2c3d4",
      "agent": "sales-agent",
      "created_at": "2025-03-19T10:00:00Z",
      "message_count": 12
    }
  ]
}
```

```bash
# Get session with messages
curl http://localhost:8443/api/v1/sessions/a1b2c3d4 \
  -H "Authorization: Bearer bb_your_token"

# Delete session
curl -X DELETE http://localhost:8443/api/v1/sessions/a1b2c3d4 \
  -H "Authorization: Bearer bb_your_token"
```

## Tasks

Create and manage agent tasks. Tasks are units of work that agents process asynchronously -- they can be created by users, triggers, or other agents. Requires `tasks` or `admin` scope.

```bash
# Create a task
curl -X POST http://localhost:8443/api/v1/tasks \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "researcher",
    "title": "Market analysis Q1",
    "description": "Analyze Q1 2025 market trends for the SaaS sector"
  }'
```

```json
{
  "id": "task_abc123",
  "agent": "researcher",
  "title": "Market analysis Q1",
  "status": "pending",
  "created_at": "2025-03-19T14:30:00Z"
}
```

```bash
# List tasks with filters
curl "http://localhost:8443/api/v1/tasks?status=pending&agent=researcher" \
  -H "Authorization: Bearer bb_your_token"

# Get task details
curl http://localhost:8443/api/v1/tasks/task_abc123 \
  -H "Authorization: Bearer bb_your_token"

# Cancel a task (pending or in_progress only)
curl -X DELETE http://localhost:8443/api/v1/tasks/task_abc123 \
  -H "Authorization: Bearer bb_your_token"

# Provide input to a waiting task (status: needs_input)
curl -X POST http://localhost:8443/api/v1/tasks/task_abc123/input \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"input": "Focus on enterprise segment and include competitor analysis"}'
```

### Task statuses

| Status | Description |
|--------|-------------|
| `pending` | Task created, waiting to be picked up by the agent. |
| `in_progress` | Agent is actively working on the task. |
| `needs_input` | Agent paused and waiting for user input (e.g., confirmation). |
| `completed` | Task finished successfully. |
| `failed` | Task failed due to an error. |
| `cancelled` | Task was cancelled by a user or API call. |
| `escalated` | Agent escalated the task to a human operator. |

## Config

Manage engine configuration at runtime. Hot-reload applies changes without restarting the engine. Export/import enable GitOps workflows. Requires `config` or `admin` scope.

```bash
# Hot-reload configuration from the database
curl -X POST http://localhost:8443/api/v1/config/reload \
  -H "Authorization: Bearer bb_your_token"

# Response
# {"status":"ok","agents_loaded":4,"models_loaded":3}

# Export current config as YAML (secrets are excluded)
curl http://localhost:8443/api/v1/config/export \
  -H "Authorization: Bearer bb_your_token" \
  -o config-backup.yaml

# Import YAML config (merges with existing)
curl -X POST http://localhost:8443/api/v1/config/import \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/x-yaml" \
  --data-binary @new-config.yaml

# Response
# {"status":"ok","agents_imported":2,"models_imported":1,"tools_imported":3}
```

## Health

Check engine status. No authentication required -- useful for load balancer health checks.

```bash
curl http://localhost:8443/api/v1/health
```

```json
{
  "status": "ok",
  "version": "1.0.0",
  "agents_count": 4,
  "uptime": "2h34m12s"
}
```

## BYOK Headers (per-request model override)

Bring Your Own Key lets API consumers override the model for a single request. This must be enabled in Settings for each provider. Useful for multi-tenant deployments where each customer provides their own API key.

```bash
curl -N http://localhost:8443/api/v1/agents/my-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -H "X-Model-Provider: anthropic" \
  -H "X-Model-API-Key: sk-ant-user-provided-key" \
  -H "X-Model-Name: claude-sonnet-4-20250514" \
  -d '{"message": "Hello"}'
```

:::caution[Security consideration]
BYOK headers are only accepted when the corresponding provider is explicitly enabled in Settings. By default, all providers are disabled for BYOK. The user-provided key is used for that single request only and is never stored.
:::

---

## What's next

- [Admin Dashboard: API Keys](/docs/admin/api-keys/)
- [Core Concepts: Tasks](/docs/concepts/tasks/)
- [Example: Sales Agent](/docs/examples/sales-agent/)
