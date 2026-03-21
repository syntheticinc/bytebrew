---
title: Docker Deployment
description: Step-by-step Docker Compose setup for ByteBrew Engine with PostgreSQL, environment variables, volumes, and troubleshooting.
---

ByteBrew Engine ships as a single Docker image that pairs with PostgreSQL. This guide covers the complete Docker Compose setup from scratch.

## Docker Compose setup

Create a `docker-compose.yml` file:

```yaml
version: "3.8"

services:
  engine:
    image: ghcr.io/syntheticinc/bytebrew-engine:latest
    ports:
      - "8080:8080"    # REST API
      - "8443:8443"    # Admin Dashboard
    environment:
      - DATABASE_URL=postgres://bytebrew:bytebrew@postgres:5432/bytebrew?sslmode=disable
      - ADMIN_USER=${ADMIN_USER:-admin}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-changeme}
      - LLM_API_KEY=${LLM_API_KEY}
      - ENGINE_PORT=8080
    volumes:
      - ./agents.yaml:/app/agents.yaml:ro
      - ./knowledge:/app/knowledge:ro
      - engine-data:/app/data
    depends_on:
      postgres:
        condition: service_healthy
    restart: unless-stopped

  postgres:
    image: pgvector/pgvector:pg16
    environment:
      - POSTGRES_USER=bytebrew
      - POSTGRES_PASSWORD=bytebrew
      - POSTGRES_DB=bytebrew
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bytebrew"]
      interval: 5s
      timeout: 3s
      retries: 5
    restart: unless-stopped

volumes:
  pgdata:
  engine-data:
```

Create a `.env` file alongside it:

```bash
ADMIN_USER=admin
ADMIN_PASSWORD=your-secure-password
LLM_API_KEY=sk-your-openai-key
```

Start everything:

```bash
docker compose up -d
```

Verify:

```bash
curl http://localhost:8080/api/v1/health
# {"status":"ok","version":"1.0.0","agents_count":0}
```

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string. The compose file sets this automatically. |
| `ADMIN_USER` | Yes | Username for the Admin Dashboard login. |
| `ADMIN_PASSWORD` | Yes | Password for the Admin Dashboard login. |
| `LLM_API_KEY` | No | Default API key referenced as `${LLM_API_KEY}` in agents.yaml. You can use any variable name. |
| `ENGINE_PORT` | No | HTTP port for the REST API. Default: `8080`. |

## Understanding host.docker.internal

When the engine runs inside Docker but your LLM (e.g., Ollama) runs on the host machine, the engine cannot reach `localhost` because `localhost` inside a container refers to the container itself.

Use `host.docker.internal` instead:

```yaml
models:
  llama-local:
    provider: ollama
    model: llama3.2
    base_url: "http://host.docker.internal:11434/v1"
    api_key: "ollama"
```

| Platform | `host.docker.internal` support |
|----------|-------------------------------|
| Docker Desktop (macOS, Windows) | Built-in, works automatically. |
| Linux (Docker Engine) | Add `extra_hosts: ["host.docker.internal:host-gateway"]` to the engine service in docker-compose.yml. |

Linux example:

```yaml
services:
  engine:
    image: ghcr.io/syntheticinc/bytebrew-engine:latest
    extra_hosts:
      - "host.docker.internal:host-gateway"
    # ... rest of config
```

## Volumes and data persistence

| Volume | Purpose |
|--------|---------|
| `pgdata` | PostgreSQL database files. All agent configs, sessions, tasks, and audit logs. |
| `engine-data` | Engine runtime data: knowledge base vector indexes, cached embeddings. |
| `./agents.yaml` (bind mount) | Your YAML configuration file. Mounted read-only. |
| `./knowledge` (bind mount) | Knowledge base documents for RAG. Mounted read-only. |

:::caution[Backup strategy]
Back up the `pgdata` volume regularly. It contains all your configuration and conversation history. Use `docker compose exec postgres pg_dump -U bytebrew bytebrew > backup.sql` for a quick SQL dump.
:::

## Port mapping

| Port | Service |
|------|---------|
| `8080` | REST API (chat, agents, tasks, config, health) |
| `8443` | Admin Dashboard (web UI) |

Both ports serve HTTP. For production, put a reverse proxy (Caddy, nginx) in front for TLS. See [Production deployment](/docs/deployment/production/).

## Troubleshooting

### "model requires more memory"

The LLM provider returned an out-of-memory error. This happens with large models on Ollama:

- Use a smaller model (llama3.2 3B instead of 70B).
- Increase Docker memory limits in Docker Desktop settings.
- If using Ollama, ensure the host machine has enough VRAM. Check with `nvidia-smi`.

### Port conflicts

If ports 8080 or 8443 are already in use:

```yaml
# Change the host-side port (left of the colon)
ports:
  - "9080:8080"    # API accessible at localhost:9080
  - "9443:8443"    # Dashboard at localhost:9443
```

### pgvector extension errors

ByteBrew requires the `pgvector` extension for knowledge base embeddings. The `pgvector/pgvector:pg16` image includes it pre-installed. If you use a plain PostgreSQL image, install pgvector manually:

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

### Engine cannot reach Ollama on the host

See the [host.docker.internal](#understanding-hostdockerinternal) section above. On Linux, you must add `extra_hosts` to the compose file.

### Container keeps restarting

Check logs:

```bash
docker compose logs engine --tail 50
```

Common causes:

- `DATABASE_URL` is wrong or PostgreSQL is not ready yet (the `depends_on` with healthcheck should handle this).
- Invalid `agents.yaml` syntax.
- Missing environment variables referenced in YAML (`${VAR}` that is not set).

## Upgrading

Pull the latest image and restart:

```bash
docker compose pull
docker compose up -d
```

The engine runs database migrations automatically on startup. Your data and configuration are preserved.

---

## What's next

- [Model Selection](/docs/deployment/model-selection/)
- [Production Deployment](/docs/deployment/production/)
- [Quick Start](/docs/getting-started/quick-start/)
