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

## Documentation

https://bytebrew.ai/docs/
