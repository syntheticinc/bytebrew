# ByteBrew Engine -- Quick Start

## Prerequisites

- Docker and Docker Compose
- LLM API key (OpenRouter, OpenAI, or Anthropic)

## Setup

```bash
cp .env.example .env
```

Edit `.env` -- set `LLM_API_KEY` at minimum.

```bash
docker compose up -d
```

Open http://localhost:8443 -- Admin Dashboard.

Login: `admin` / `changeme` (change in `.env`).

## Local LLM (optional)

Uncomment the `ollama` service in `docker-compose.yml`, then:

```bash
docker compose up -d
docker exec bytebrew-ollama ollama pull llama3
```

Set `LLM_PROVIDER=ollama` and `LLM_MODEL=llama3` in `.env`.

## Configuration

All runtime configuration (agents, models, tools) is managed via the Admin Dashboard.
The `.env` file and `config.yaml` only control bootstrap settings: database, port, admin credentials.

## Volumes

| Volume | Purpose |
|--------|---------|
| `engine-data` | Engine data directory |
| `engine-logs` | Engine log files |
| `pg-data` | PostgreSQL data |

## Updating

To update the engine to the latest version:

```bash
docker compose pull engine
docker compose up -d engine
```

> **Note:** Always check the [changelog](https://github.com/syntheticinc/bytebrew/releases) before updating.
> Major versions may include database migrations or breaking changes.

## Troubleshooting

Check engine logs:

```bash
docker compose logs engine
```

Check database health:

```bash
docker compose exec db pg_isready -U bytebrew -d bytebrew
```

Rebuild after code changes:

```bash
docker compose build engine && docker compose up -d engine
```
