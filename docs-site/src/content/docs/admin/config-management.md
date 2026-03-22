---
title: "Admin Dashboard: Config Management"
description: Hot reload, export, and import engine configuration for zero-downtime changes and GitOps workflows.
---

The Config Management page provides three operations for managing engine configuration at runtime: hot reload, export, and import. These enable zero-downtime configuration changes and GitOps workflows.

## Hot Reload

Apply configuration changes from the database without restarting the engine. This is triggered automatically when you save changes in the dashboard, but you can also trigger it manually or via API after a database import.

- Agents are re-initialized with updated prompts, tools, and settings.
- Active sessions are preserved -- only future messages use the new config.
- MCP servers are reconnected if their configuration changed.
- Failed reloads are rolled back -- the previous config remains active.

```bash
# Hot reload via API
curl -X POST http://localhost:8443/api/v1/config/reload \
  -H "Authorization: Bearer bb_admin_token"

# Response
# {"status":"ok","agents_loaded":4,"models_loaded":3}
```

## Export

Download the current configuration as a YAML file. Useful for backups, version control, and migrating between environments. Secrets (API keys) are excluded from the export.

```bash
# Export via API
curl http://localhost:8443/api/v1/config/export \
  -H "Authorization: Bearer bb_admin_token" \
  -o config-backup.yaml
```

:::note[Secrets handling]
Exported YAML replaces API keys with `${VAR_NAME}` placeholders. When importing into another environment, set the corresponding environment variables.
:::

## Import

Upload a YAML file to merge with or replace the current configuration. This is the recommended way to deploy configuration changes in CI/CD pipelines.

```bash
# Import via API
curl -X POST http://localhost:8443/api/v1/config/import \
  -H "Authorization: Bearer bb_admin_token" \
  -H "Content-Type: application/x-yaml" \
  --data-binary @new-config.yaml

# Response
# {"status":"ok","agents_imported":2,"models_imported":1,"tools_imported":3}
```

## GitOps workflow

A common pattern is to store your `agents.yaml` in Git and deploy changes via CI/CD:

- Developer edits `agents.yaml` in a feature branch.
- Pull request is reviewed and merged to main.
- CI/CD pipeline runs `config/import` followed by `config/reload`.
- Agents are updated with zero downtime.

---

## What's next

- [Audit Log](/docs/admin/audit-log/)
- [API Reference: Config](/docs/getting-started/api-reference/#config)
