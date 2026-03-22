---
title: Quick Start
description: Get ByteBrew Engine running with Docker in under 5 minutes and send your first message to an AI agent.
---

Get ByteBrew Engine running with Docker in under 5 minutes. By the end of this guide, you will have a working AI agent that responds to messages over a REST API.

:::note[Prerequisites]
You need `docker` and `docker compose` installed. ByteBrew Engine runs on Linux, macOS, and Windows (WSL2). Minimum 2 GB RAM for the engine + PostgreSQL.
:::

## Step 1: Start the Engine

Download the Docker Compose file and start the engine. This spins up two containers: the ByteBrew Engine and a PostgreSQL database.

```bash
curl -fsSL https://bytebrew.ai/releases/docker-compose.yml -o docker-compose.yml
docker compose up -d
```

The engine starts on port `8443` — both the REST API and the Admin Dashboard are served on this single port. Verify it is running:

```bash
curl http://localhost:8443/api/v1/health
# {"status":"ok","version":"1.0.0","agents_count":0}
```

:::tip[Default credentials]
The Admin Dashboard login uses `admin` / `changeme` by default. Change these by setting `ADMIN_USER` and `ADMIN_PASSWORD` in a `.env` file next to your `docker-compose.yml`:

```bash
# .env
ADMIN_USER=myadmin
ADMIN_PASSWORD=s3cur3-pa$$w0rd
```
:::

## Step 2: Create your first agent

Create an `agents.yaml` file in the same directory as your `docker-compose.yml`. This file defines your agents, models, and tools:

```yaml
# agents.yaml
agents:
  my-agent:
    model: glm-5
    system: "You are a helpful assistant for our product."
    tools:
      - web_search

models:
  glm-5:
    provider: openai
    api_key: ${OPENAI_API_KEY}
```

:::note[Available built-in tools]
`web_search` works out of the box. `knowledge_search` requires a `knowledge:` path in the agent config — see [Knowledge / RAG](/docs/concepts/knowledge/). `manage_tasks` enables task tracking. `ask_user` pauses to ask the user a question.
:::

:::tip[Environment variables]
The `${OPENAI_API_KEY}` syntax references an environment variable. Set it in your shell (`export OPENAI_API_KEY=sk-...`) or in a `.env` file next to `docker-compose.yml`. Never hardcode secrets in YAML.
:::

:::tip[Prefer a visual editor?]
Skip the YAML file and use the Admin Dashboard instead. Open `http://localhost:8443/admin`, log in, and click **Create Agent**. The dashboard lets you configure everything visually -- model, system prompt, tools, security zones, spawn rules, and more.
:::

## Step 3: Send your first message

Use the REST API to talk to your agent. The response streams back as Server-Sent Events (SSE), so you see tokens as they are generated:

```bash
curl -N http://localhost:8443/api/v1/agents/my-agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello, what can you do?"}'
```

## Step 4: See the response

The engine returns a stream of SSE events. Each event has a `type` field that tells you what kind of data it contains:

```
event: message_delta
data: {"content":"Hello! I'm your product assistant. "}

event: message_delta
data: {"content":"I can help you with product questions, "}

event: message_delta
data: {"content":"documentation search, and more."}

event: message
data: {"content":"Hello! I'm your product assistant. I can help you with product questions, documentation search, and more."}

event: done
data: {"session_id":"a1b2c3d4"}
```

The `session_id` in the `done` event lets you continue the conversation. Pass it in subsequent requests to maintain context:

```bash
curl -N http://localhost:8443/api/v1/agents/my-agent/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Tell me more about that", "session_id": "a1b2c3d4"}'
```

## Step 5: Open the Admin Dashboard

Navigate to `http://localhost:8443/admin` in your browser. Log in with the default credentials (`admin` / `changeme`) or whatever you set in your `.env` file. From the dashboard you can manage agents, models, MCP servers, tools, triggers, and API keys — all without editing YAML.

---

## What's next

- [Configuration Reference](/docs/getting-started/configuration/)
- [API Reference](/docs/getting-started/api-reference/)
- [Core Concepts: Agents](/docs/concepts/agents/)
- [Example: Sales Agent](/docs/examples/sales-agent/)
