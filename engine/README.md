# ByteBrew Engine

AI agent runtime with multi-agent orchestration, MCP tools, and REST API.

## Build

```bash
go build ./cmd/ce                           # Community Edition binary
docker build -t bytebrew/engine:latest .    # Docker image
```

## Entry Points

- `cmd/ce` — Production entry point (Community Edition)
- `cmd/testserver` — Test server with MockChatModel (for integration tests)

## CE vs EE

Single binary. Community Edition includes all features. Enterprise Edition (coming soon) adds observability and compliance tools, gated by license key.

### CE Features (free forever)
- Unlimited agents, models, MCP servers
- Multi-agent spawn orchestration
- Cron triggers, webhooks, background tasks
- Knowledge Base / RAG
- REST API + SSE + WebSocket
- Admin Dashboard
- BYOK (bring your own key)
- API tokens with scopes

### EE Features (coming soon)
- Session Explorer
- Cost Analytics
- Quality Metrics
- Audit Log Export
- PII Redaction

## Configuration

Engine can be configured via:
1. **Environment variables** (recommended for Docker): `DATABASE_URL`, `ADMIN_USER`, `ADMIN_PASSWORD`, `LLM_API_KEY`
2. **YAML config file**: `config.yaml`
3. **Admin Dashboard**: visual configuration at `/admin/`

## Docker

```bash
# Quick start
curl -fsSL https://bytebrew.ai/releases/docker-compose.yml -o docker-compose.yml
docker compose up -d

# Verify
curl http://localhost:8443/api/v1/health
```

Default credentials: `admin` / `changeme`

## Tasks: Async Work Management

EngineTask is ByteBrew's unified system for managing background work, scheduled jobs, and complex multi-step workflows. Tasks can be created by agents, cron triggers, webhooks, or the REST API.

Key features:
- **State machine** — draft → approved → pending → in_progress → completed/failed
- **Priority queue** — 0=normal, 1=high, 2=critical
- **Hierarchical** — Parent tasks with subtasks
- **Dependencies** — BlockedBy logic for workflow sequencing
- **Agent integration** — `manage_tasks` tool for agents to self-organize work
- **Completion webhooks** — Async notification when tasks finish
- **Auto-execution** — Cron and webhook triggers automatically run agents

Example:

```bash
# Create task with approval gate
curl -X POST http://localhost:8443/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Generate daily report",
    "agent_name": "reporter",
    "require_approval": true,
    "priority": 1
  }'

# Agent can also create subtasks
{
  "action": "create_subtask",
  "parent_task_id": "task-123",
  "title": "Extract data",
  "priority": 2,
  "blocked_by": []
}
```

**Documentation:**
- [Task Concepts](../../docs/concepts/tasks.md) — Architecture, lifecycle, state machine
- [REST API](../../docs/api/tasks.md) — HTTP endpoints
- [manage_tasks Tool](../../docs/agent-tools/manage-tasks.md) — Agent integration
- [Cron Automation Example](../../docs/examples/cron-task-automation.md) — Full workflow

## Documentation

https://bytebrew.ai/docs/
