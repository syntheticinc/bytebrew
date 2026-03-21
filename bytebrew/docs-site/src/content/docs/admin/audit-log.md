---
title: "Admin Dashboard: Audit Log"
description: View the immutable record of all administrative actions performed on ByteBrew Engine.
---

The Audit Log provides a complete, immutable record of all administrative actions performed on the engine. Every configuration change, authentication event, and API key lifecycle event is captured with full context.

## What is logged

- **Configuration changes** -- creating, updating, or deleting agents, models, tools, triggers, and MCP servers. Includes before/after state.
- **Authentication events** -- admin login attempts (successful and failed).
- **API key lifecycle** -- key creation (with scopes) and revocation.
- **Config operations** -- hot reload, import, and export events.
- **Settings changes** -- BYOK toggles, logging level changes.

## Filtering and search

The audit log provides several filters to find specific events:

- **Actor type** -- filter by who performed the action (admin user, API key, system).
- **Action** -- create, update, delete, login, reload, import, export.
- **Resource** -- agent, model, tool, trigger, mcp_server, api_key, config, settings.
- **Date range** -- select a start and end date to narrow results.

## Audit entry structure

Click any entry to expand the detail view with the full JSON payload:

```json
{
  "id": "audit_abc123",
  "actor": "admin",
  "actor_type": "user",
  "action": "update",
  "resource_type": "agent",
  "resource_id": "sales-bot",
  "timestamp": "2025-03-19T14:30:00Z",
  "details": {
    "changes": [
      {
        "field": "max_steps",
        "old_value": 50,
        "new_value": 100
      },
      {
        "field": "tools",
        "old_value": ["web_search"],
        "new_value": ["web_search", "create_order"]
      }
    ]
  }
}
```

:::note[Retention]
Audit log entries are stored in PostgreSQL and retained indefinitely by default. For high-volume deployments, consider setting up a retention policy to archive or delete entries older than your compliance requirements.
:::

---

## What's next

- [API Keys](/docs/admin/api-keys/)
- [Settings](/docs/admin/settings/)
