---
title: API Reference
description: Complete REST API reference for ByteBrew Engine including chat, agents, sessions, tasks, config, and health endpoints.
---

Complete REST API reference for the ByteBrew Engine. All endpoints return JSON (except SSE streams) and accept JSON request bodies.

## Authentication

All API requests must include a valid API token in the `Authorization` header. Tokens are created through the Admin Dashboard and are scoped to specific capabilities.

### Creating an API token

- Navigate to **Admin Dashboard** -> **API Keys**
- Click "Create API Key" and select the scopes you need
- Copy the token immediately -- it is shown only once and cannot be recovered
- Tokens are prefixed with `bb_` for easy identification in logs and config

### Using the token

```bash
# curl
curl http://localhost:8080/api/v1/agents \
  -H "Authorization: Bearer bb_your_api_token"
```

```javascript
// JavaScript (fetch)
const response = await fetch('http://localhost:8080/api/v1/agents', {
  headers: { 'Authorization': 'Bearer bb_your_api_token' },
});
```

```python
# Python (requests)
import requests
response = requests.get(
    'http://localhost:8080/api/v1/agents',
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

**Base URL:** `http://localhost:8080/api/v1`

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
curl -N http://localhost:8080/api/v1/agents/sales-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "What laptops do you have under $1000?"}'
```

### SSE event types

| Event | Description |
|-------|-------------|
| `content` | Text chunk from the agent. Concatenate all content events for the full response. |
| `tool_call` | Agent is calling a tool. Contains tool name and input parameters. |
| `tool_result` | Result returned from the tool execution. |
| `error` | An error occurred during processing. |
| `done` | Stream is complete. Contains session_id and token count. |

### Example response stream

```
event: content
data: {"text":"I found several laptops under $1000. "}

event: tool_call
data: {"tool":"search_products","input":{"query":"laptops under 1000","limit":"5"}}

event: tool_result
data: {"tool":"search_products","output":"[{\"name\":\"ProBook 450\",\"price\":849}...]"}

event: content
data: {"text":"Here are the top options:\n\n1. **ProBook 450** — $849..."}

event: done
data: {"session_id":"a1b2c3d4-e5f6-7890-abcd-ef1234567890","tokens":234}
```

### Continue the conversation

```bash
curl -N http://localhost:8080/api/v1/agents/sales-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "Tell me more about the ProBook 450", "session_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}'
```

## Agents

List and inspect configured agents. Requires `agents:read` or `admin` scope.

```bash
# List all agents
curl http://localhost:8080/api/v1/agents \
  -H "Authorization: Bearer bb_your_token"
```

```json
{
  "agents": [
    {
      "name": "sales-agent",
      "model": "glm-5",
      "lifecycle": "persistent",
      "tools_count": 5,
      "has_knowledge": true
    }
  ]
}
```

```bash
# Get agent details
curl http://localhost:8080/api/v1/agents/sales-agent \
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

## Sessions

Manage conversation sessions. Sessions store the full message history between a user and an agent. Requires `chat` or `admin` scope.

```bash
# List sessions (with optional filters)
curl "http://localhost:8080/api/v1/sessions?agent=sales-agent&limit=10" \
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
curl http://localhost:8080/api/v1/sessions/a1b2c3d4 \
  -H "Authorization: Bearer bb_your_token"

# Delete session
curl -X DELETE http://localhost:8080/api/v1/sessions/a1b2c3d4 \
  -H "Authorization: Bearer bb_your_token"
```

## Tasks

Create and manage agent tasks. Tasks are units of work that agents process asynchronously -- they can be created by users, triggers, or other agents. Requires `tasks` or `admin` scope.

```bash
# Create a task
curl -X POST http://localhost:8080/api/v1/tasks \
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
curl "http://localhost:8080/api/v1/tasks?status=pending&agent=researcher" \
  -H "Authorization: Bearer bb_your_token"

# Get task details
curl http://localhost:8080/api/v1/tasks/task_abc123 \
  -H "Authorization: Bearer bb_your_token"

# Cancel a task (pending or in_progress only)
curl -X DELETE http://localhost:8080/api/v1/tasks/task_abc123 \
  -H "Authorization: Bearer bb_your_token"

# Provide input to a waiting task (status: needs_input)
curl -X POST http://localhost:8080/api/v1/tasks/task_abc123/input \
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
curl -X POST http://localhost:8080/api/v1/config/reload \
  -H "Authorization: Bearer bb_your_token"

# Response
# {"status":"ok","agents_loaded":4,"models_loaded":3}

# Export current config as YAML (secrets are excluded)
curl http://localhost:8080/api/v1/config/export \
  -H "Authorization: Bearer bb_your_token" \
  -o config-backup.yaml

# Import YAML config (merges with existing)
curl -X POST http://localhost:8080/api/v1/config/import \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/x-yaml" \
  --data-binary @new-config.yaml

# Response
# {"status":"ok","agents_imported":2,"models_imported":1,"tools_imported":3}
```

## Health

Check engine status. No authentication required -- useful for load balancer health checks.

```bash
curl http://localhost:8080/api/v1/health
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
curl -N http://localhost:8080/api/v1/agents/my-agent/chat \
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

- [Admin Dashboard: API Keys](/admin/api-keys/)
- [Core Concepts: Tasks](/concepts/tasks/)
- [Example: Sales Agent](/examples/sales-agent/)
