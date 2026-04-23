# ByteBrew Engine -- Quick Start

Two deployment paths are supported:

- **Docker Compose** — single-host (VPS, on-prem, local dev). See "Docker Compose" below.
- **Kubernetes (Helm)** — clustered (dev/staging/prod). See "Kubernetes (Helm)" below.

Both ship the same engine binary + admin SPA bundle. Pick based on your infra.

---

## Docker Compose

### Prerequisites

- Docker and Docker Compose
- LLM API key (OpenRouter, OpenAI, or Anthropic)

### Setup

```bash
cp .env.example .env
```

Edit `.env` -- set `LLM_API_KEY` at minimum.

```bash
docker compose up -d
```

Open http://localhost:8443 -- Admin Dashboard.

Create the first admin user:

```bash
docker compose exec engine /usr/local/bin/bytebrew-ce admin create --username admin --password <your-password>
```

Then log in with those credentials.

## Local LLM (optional)

Uncomment the `ollama` service in `docker-compose.yml`, then:

```bash
docker compose up -d
docker exec bytebrew-ollama ollama pull llama3
```

Set `LLM_PROVIDER=ollama` and `LLM_MODEL=llama3` in `.env`.

## Configuration

All runtime configuration (agents, models, tools) is managed via the Admin Dashboard.
The `.env` file and `config.yaml` only control bootstrap settings: database, port, JWT secret.

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

---

## Kubernetes (Helm)

### Prerequisites

- Kubernetes 1.24+
- `kubectl` + `helm` v3/v4 installed and authenticated to your cluster
- Ingress controller (nginx-ingress recommended) and optionally cert-manager for TLS
- External PostgreSQL 15+ with `pgvector` extension (managed or in-cluster)
- LLM API key

### Setup

```bash
cd helm

# Copy the example, fill fields marked <REQUIRED>
cp bytebrew/values.example.yaml values.yaml
$EDITOR values.yaml

# Validate template rendering
helm lint ./bytebrew
helm template bytebrew ./bytebrew -f values.yaml

# Install
helm install bytebrew ./bytebrew -f values.yaml
```

Required fields in `values.yaml`:

- `ingress.hosts[0].host` — your public hostname
- `postgresql.external.host` / `username` / `password` — managed PG endpoint
- `secrets.llmAPIKeys.openai` (or `anthropic` / `openrouter`) — at least one

After pods are Ready, the engine is reachable at `https://<your-host>/admin/`.

### Upgrade

```bash
helm upgrade bytebrew ./bytebrew -f values.yaml
```

### Rollback

```bash
helm history bytebrew
helm rollback bytebrew <revision>
```

### Single-replica constraint

`config.auth.mode: "local"` requires `replicaCount: 1` (Ed25519 keypair on a single PVC). For HA, switch to `auth.mode: "external"` and provide a pre-generated Ed25519 public key via ConfigMap/Secret.

See [docs: production deployment](https://bytebrew.ai/docs/deployment/production) for the full walkthrough including TLS via cert-manager, Prometheus scraping, and operational checklist.
