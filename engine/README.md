# ByteBrew Engine

[![CI](https://github.com/syntheticinc/bytebrew/actions/workflows/ci.yml/badge.svg)](https://github.com/syntheticinc/bytebrew/actions/workflows/ci.yml)
[![License: BSL 1.1](https://img.shields.io/badge/License-BSL%201.1-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/syntheticinc/bytebrew)](go.mod)

**Open-source AI agent runtime.** Build, deploy, and orchestrate autonomous AI agents with multi-agent coordination, MCP tool integration, and a visual admin dashboard.

> Not another AI chatbot. ByteBrew is the agent brewery.

## Features

- **Multi-Agent Orchestration** — agents spawn and coordinate with each other via ReAct framework
- **MCP Tool Ecosystem** — connect any Model Context Protocol server (stdio, SSE, HTTP, Docker)
- **Visual Admin Dashboard** — configure agents, models, tools, and triggers from a web UI
- **Task System** — async background tasks with priorities, dependencies, approval gates, and webhooks
- **Cron & Webhook Triggers** — schedule agents or trigger them from external events
- **Knowledge Base / RAG** — vector search over uploaded documents with pgvector
- **Agent Memory** — cross-session persistent memory per agent
- **Multiple Clients** — REST API + SSE, gRPC, WebSocket (via bridge)
- **BYOK** — bring your own keys for any OpenAI-compatible LLM provider
- **Self-Hosted** — deploy on your infrastructure with Docker, Kubernetes, or bare metal

## Quick Start

```bash
# Start with Docker Compose
curl -fsSL https://bytebrew.ai/releases/docker-compose.yml -o docker-compose.yml
docker compose up -d

# Open admin dashboard
open http://localhost:8443/admin/
# Default credentials: admin / changeme
```

Or build from source:

```bash
go build -o bytebrew ./cmd/ce
./bytebrew
```

## Configuration

ByteBrew can be configured via:

| Method | Use Case |
|--------|----------|
| **Environment variables** | Docker, Kubernetes, CI/CD |
| **config.yaml** | Local development, bare metal |
| **Admin Dashboard** | Visual configuration at `/admin/` |

Key environment variables:

```bash
DATABASE_URL=postgresql://user:pass@host:5432/bytebrew
ADMIN_USER=admin
ADMIN_PASSWORD=changeme
```

LLM provider, model and API key are configured through the onboarding
wizard on first launch (or later via Admin → Models). Engine does not
read LLM credentials from env or config files.

## Architecture

ByteBrew follows Clean Architecture with strict layer separation:

```
cmd/ce/              Community Edition entry point
internal/
  domain/            Pure domain entities
  usecase/           Business logic + consumer-side interfaces
  service/           Task worker, scheduler, completion hooks
  infrastructure/    DB, LLM, MCP, agents, tools
  delivery/          HTTP & gRPC handlers
  app/               Application bootstrap
admin/               React/TypeScript admin dashboard
```

## Deployment

| Method | Guide |
|--------|-------|
| **Docker Compose** | See [Quick Start](#quick-start) above |
| **Kubernetes** | Helm chart in [`deploy/helm/`](deploy/helm/) |
| **Bare Metal** | Binary + systemd + PostgreSQL + Caddy/nginx |

## Editions

| Feature | Community (CE) | Enterprise (EE) |
|---------|:-:|:-:|
| Unlimited agents, models, MCP servers | :white_check_mark: | :white_check_mark: |
| Multi-agent spawn orchestration | :white_check_mark: | :white_check_mark: |
| Cron triggers, webhooks, background tasks | :white_check_mark: | :white_check_mark: |
| Knowledge Base / RAG | :white_check_mark: | :white_check_mark: |
| REST API + SSE + WebSocket | :white_check_mark: | :white_check_mark: |
| Admin Dashboard | :white_check_mark: | :white_check_mark: |
| API tokens with scopes | :white_check_mark: | :white_check_mark: |
| Session Explorer | | :white_check_mark: |
| Cost Analytics | | :white_check_mark: |
| Audit Log Export | | :white_check_mark: |
| SSO / SAML | | :white_check_mark: |

## Documentation

- **Website:** https://bytebrew.ai
- **Docs:** https://bytebrew.ai/docs/
- **API Reference:** https://bytebrew.ai/docs/api/

## Contributing

We welcome contributions! Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a PR.

- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security Policy](SECURITY.md)

## License

ByteBrew Engine is licensed under the [Business Source License 1.1](LICENSE).

- **Self-hosting** for internal use: allowed
- **Embedding** in your product via API: allowed
- **Managed Service** (reselling ByteBrew as a service): not allowed
- **Change Date:** 2030-04-06 (converts to Apache 2.0)

For alternative licensing arrangements, contact [info@bytebrew.ai](mailto:info@bytebrew.ai).
