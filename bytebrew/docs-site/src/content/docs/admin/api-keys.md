---
title: "Admin Dashboard: API Keys"
description: Create, scope, and revoke API keys for programmatic access to ByteBrew Engine.
---

API keys authenticate programmatic access to the ByteBrew Engine. Each key can be scoped to specific capabilities, allowing you to follow the principle of least privilege. Keys are created through the dashboard and can be revoked at any time.

## Creating an API key

- Click "Create API Key" on the API Keys page.
- Give it a descriptive name (e.g., "chatbot-frontend", "ci-cd-pipeline").
- Select the scopes this key needs (see table below).
- Click "Create" -- the key is shown once. Copy it immediately.

:::caution[Copy immediately]
The full API key is shown only once at creation time. It is hashed before storage in the database and cannot be recovered. If you lose a key, revoke it and create a new one.
:::

## Available scopes

| Scope | Description |
|-------|-------------|
| `chat` | Send messages to agents (POST /agents/\{name\}/chat). The most common scope for client applications. |
| `tasks` | CRUD operations on /tasks. Create, list, cancel tasks and provide input. |
| `agents:read` | Read-only access to agent configurations (GET /agents). |
| `config` | Reload, export, and import configuration. Useful for CI/CD pipelines. |
| `admin` | Full access to all endpoints including API key management and settings. |

## Usage examples

```bash
# Use an API key in requests
curl http://localhost:8080/api/v1/agents \
  -H "Authorization: Bearer bb_your_api_token"

# Example: key with chat + tasks scopes
# Can call: POST /agents/{name}/chat, GET/POST/DELETE /tasks
# Cannot call: /config/reload, API key management, settings

# Example: key with config scope only (CI/CD)
curl -X POST http://localhost:8080/api/v1/config/reload \
  -H "Authorization: Bearer bb_cicd_deploy_token"
```

## Revoking a key

Click the "Revoke" button next to any key in the list. Revocation is immediate -- any request using that key will receive a `401 Unauthorized` response. Revocation is logged in the Audit Log.

---

## What's next

- [API Reference: Authentication](/docs/getting-started/api-reference/#authentication)
- [Settings](/docs/admin/settings/)
- [Audit Log](/docs/admin/audit-log/)
