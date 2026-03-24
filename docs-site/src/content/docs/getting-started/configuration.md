---
title: Configuration Reference
description: Complete reference for configuring ByteBrew Engine agents, models, tools, MCP servers, and triggers via YAML or the Admin Dashboard.
---

ByteBrew Engine is configured through YAML files or the Admin Dashboard. Both methods write to the same PostgreSQL database -- YAML is just a convenient bootstrap format. This reference covers every configuration option in detail.

:::note[Two ways to configure]
You can define everything in a single `agents.yaml` file (great for version control and GitOps), or use the Admin Dashboard for a visual editor. Changes made in the dashboard are persisted to the database immediately. Use `POST /api/v1/config/import` to sync YAML into the database, or `GET /api/v1/config/export` to export the current state as YAML.
:::

## Agent Configuration

Agents are the core building blocks of ByteBrew. Each agent is an LLM-powered entity with its own identity, behavior, tools, and memory. You define agents under the `agents:` key, where each key is the agent's unique name.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `model` * | -- | References a model defined in the models: section. Determines which LLM the agent uses for reasoning. |
| `system` | -- | Inline system prompt string that defines the agent's personality, role, and behavior rules. |
| `system_file` | -- | Path to a text file containing the system prompt. Mutually exclusive with system:. Useful for long prompts. |
| `lifecycle` | `persistent` | `persistent` keeps context across sessions. `spawn` creates a fresh instance per invocation and terminates after. |
| `kit` | `none` | Preset tool bundle. `developer` adds code-related tools (read_file, edit_file, bash, etc.). |
| `tool_execution` | `sequential` | `sequential` runs tool calls one at a time. `parallel` runs independent tool calls concurrently. |
| `max_steps` | `50` | Maximum number of reasoning iterations (1-500). Prevents infinite loops in complex tasks. |
| `max_context_size` | `16000` | Maximum context window in tokens (1,000-200,000). Older messages are compressed when exceeded. |
| `tools` | `[]` | List of built-in tools and custom tool names available to this agent. |
| `knowledge` | -- | Path to a folder of documents for RAG. The engine auto-indexes files at startup. |
| `mcp_servers` | `[]` | List of MCP server names (defined in mcp_servers: section) available to this agent. |
| `can_spawn` | `[]` | List of agent names this agent can create at runtime. The engine auto-generates spawn_&lt;name&gt; tools. |
| `confirm_before` | `[]` | List of tool names that require user confirmation before execution. |

```yaml
agents:
  sales-agent:
    model: glm-5                       # Required: model from models: section
    system: |                          # Multi-line system prompt
      You are a sales consultant for Acme Corp.
      Always be professional and helpful.
      Never discuss competitor products.
    lifecycle: persistent              # Keep conversation history
    tool_execution: parallel           # Run independent tools concurrently
    max_steps: 100                     # Allow complex multi-step tasks
    max_context_size: 32000            # Larger context for long conversations
    tools:
      - knowledge_search               # Search product docs
      - web_search                     # Search the internet
      - create_order                   # Custom HTTP tool
    knowledge: "./docs/products/"      # Auto-indexed product catalog
    mcp_servers:
      - crm-api                        # CRM integration via MCP
    can_spawn:
      - researcher                     # Can delegate research tasks
    confirm_before:
      - create_order                   # Ask user before placing orders
```

## System Prompts: Best Practices

The system prompt is the most important configuration for an agent. It defines personality, capabilities, constraints, and output format. A well-written prompt dramatically improves agent reliability.

### Structure of an effective prompt

- **Role definition** -- who the agent is and what organization it belongs to.
- **Capabilities** -- what tools are available and when to use each one.
- **Constraints** -- what the agent must never do (guardrails).
- **Output format** -- how to structure responses (markdown, JSON, bullet points).
- **Escalation rules** -- when to ask the user vs. act autonomously.

```yaml
# Good: specific role, clear boundaries, actionable instructions
system: |
  You are a customer support agent for ByteStore, an online electronics retailer.

  ## Your capabilities
  - Search the knowledge base for product information and policies
  - Look up order status by order ID
  - Create support tickets for issues you cannot resolve

  ## Rules
  - Always greet the customer by name if available
  - Never share internal pricing or margin data
  - If asked about a competitor, redirect to our product advantages
  - For refund requests over $500, escalate to a human agent

  ## Response format
  - Keep responses concise (2-3 paragraphs max)
  - Use bullet points for lists of options
  - Always end with a follow-up question or next step
```

:::caution[Common mistakes]
Avoid vague prompts like "You are a helpful assistant." The more specific your prompt, the more consistent the agent's behavior. Always tell the agent what it should NOT do -- LLMs are eager to please and will attempt tasks outside their scope unless explicitly told not to.
:::

For long prompts, use `system_file` to load from an external file. This keeps your YAML clean and lets you version-control prompts separately:

```yaml
agents:
  support-bot:
    model: glm-5
    system_file: "./prompts/support-bot.txt"   # Loaded at startup
```

## Security Zones Explained

Every tool in ByteBrew is assigned a security zone that indicates its risk level. This helps operators understand what an agent can do and enforce appropriate safeguards.

| Zone | Description |
|------|-------------|
| `Safe` | Read-only operations with no side effects. Examples: knowledge_search, web_search, list_files. No confirmation needed. |
| `Caution` | Operations that modify state but are reversible. Examples: edit_file, create_ticket, send_email. Consider adding to confirm_before. |
| `Dangerous` | Operations with irreversible side effects. Examples: bash, delete_file, create_order. Strongly recommended for confirm_before. |

:::tip[Defense in depth]
Use `confirm_before` for any Caution or Dangerous tool in production. This pauses execution and returns a `needs_input` event to the client, allowing a human to approve or reject the action before it executes.
:::

```yaml
agents:
  devops-bot:
    model: glm-5
    tools:
      - web_search              # Safe: read-only
      - edit_file               # Caution: modifies files
      - bash                    # Dangerous: arbitrary commands
    confirm_before:
      - bash                    # Require human approval
      - edit_file               # Require human approval
```

## Tool Confirmation (`confirm_before`)

The `confirm_before` list specifies tools that require user approval before execution. When the agent calls a confirmed tool, the engine pauses execution and sends a `confirmation` SSE event to the client. The client then approves or rejects the action.

```yaml
agents:
  sales-agent:
    model: glm-5
    tools:
      - web_search
      - create_order
      - send_email
    confirm_before:
      - create_order     # Pause before placing orders
      - send_email       # Pause before sending emails
```

### SSE event flow

1. Agent decides to call `create_order`.
2. Engine detects the tool is in `confirm_before`.
3. Engine sends a `confirmation` SSE event with the tool name, input, and a `confirmation_id`.
4. Client displays the pending action to the user.
5. User approves or rejects via `POST /api/v1/confirmations/{confirmation_id}`.
6. If approved, the tool executes and the stream continues.
7. If rejected, the agent receives the rejection reason and adapts.

```
event: confirmation
data: {"tool":"create_order","input":{"customer_id":"cust_123","items":"ProBook x1"},"confirmation_id":"conf_abc"}
```

```bash
# Approve
curl -X POST http://localhost:8443/api/v1/confirmations/conf_abc \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"action": "approve"}'

# Reject
curl -X POST http://localhost:8443/api/v1/confirmations/conf_abc \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"action": "reject", "reason": "Customer changed their mind"}'
```

## Environment Variables

ByteBrew supports `${VAR_NAME}` syntax for referencing environment variables anywhere in your YAML configuration. Variables are expanded at engine startup, so the YAML file never contains actual secrets.

### How it works

- The engine reads the YAML file and replaces every `${VAR_NAME}` with the value of that environment variable.
- If a referenced variable is not set, the engine logs a warning and leaves the placeholder empty.
- You can use variables in any string value: URLs, API keys, file paths, even system prompts.
- Variables are expanded once at startup (or on hot-reload). They are not re-evaluated per-request.

```yaml
# .env file (loaded by Docker Compose automatically)
OPENAI_API_KEY=sk-proj-abc123
CATALOG_API=https://api.mystore.com/v2
WEBHOOK_SECRET=whsec_xyz789
CRM_API_KEY=crm_live_456

# agents.yaml — references variables, never contains secrets
models:
  glm-5:
    provider: openai
    api_key: ${OPENAI_API_KEY}

tools:
  search_products:
    type: http
    url: "${CATALOG_API}/products/search"

triggers:
  order-webhook:
    secret: ${WEBHOOK_SECRET}
```

:::caution[Never hardcode secrets]
If your YAML file is checked into version control (recommended for GitOps), all secrets must use `${VAR}` syntax. The engine will refuse to start if it detects bare API keys in the configuration file.
:::

## Model Configuration

Models define the LLM backends your agents use. ByteBrew supports any OpenAI-compatible API, Anthropic, and local models via Ollama. You can configure multiple models and assign different ones to different agents.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `provider` * | -- | LLM provider type: `ollama`, `openai_compatible`, `anthropic`, `azure_openai`, `google`, `openrouter`, `deepseek`, `mistral`, `xai`, or `zai`. |
| `model` | -- | Model name as expected by the provider API (e.g., gpt-4o, claude-sonnet-4-20250514, llama3.2). |
| `base_url` | Provider default | Custom API endpoint. Required for Ollama and third-party OpenAI-compatible providers. |
| `api_key` | -- | API key for the provider. Use `${VAR}` syntax. Not required for Ollama. |

### Ollama (local models)

Run models locally with zero API costs. Install Ollama, pull a model, and point ByteBrew at it:

```bash
# 1. Install Ollama (https://ollama.com)
curl -fsSL https://ollama.com/install.sh | sh

# 2. Pull a model
ollama pull llama3.2
ollama pull qwen2.5-coder:32b
```

```yaml
# 3. Configure in ByteBrew
models:
  llama-local:
    provider: ollama
    model: llama3.2
    base_url: "http://localhost:11434/v1"
    api_key: "ollama"              # Ollama ignores the key, but the field is required

  qwen-coder:
    provider: ollama
    model: qwen2.5-coder:32b
    base_url: "http://localhost:11434/v1"
    api_key: "ollama"
```

:::tip[GPU acceleration]
Ollama uses GPU automatically if available. For 32B+ parameter models, you need at least 24 GB VRAM (RTX 4090 or A100). Smaller models like llama3.2 (3B) run on 4 GB VRAM or even CPU.
:::

### OpenAI-compatible providers

Any API that follows the OpenAI chat completions format works out of the box. Just change the `base_url`:

| Provider | base_url |
|----------|----------|
| OpenAI | `https://api.openai.com/v1` (default, can be omitted) |
| DeepInfra | `https://api.deepinfra.com/v1/openai` |
| Together AI | `https://api.together.xyz/v1` |
| Groq | `https://api.groq.com/openai/v1` |
| vLLM | `http://localhost:8000/v1` (self-hosted) |
| LiteLLM | `http://localhost:4000/v1` (proxy) |

```yaml
models:
  # DeepInfra — pay-per-token cloud inference
  qwen-3-32b:
    provider: openai
    model: Qwen/Qwen3-32B
    base_url: "https://api.deepinfra.com/v1/openai"
    api_key: ${DEEPINFRA_API_KEY}

  # Groq — ultra-fast inference
  llama-groq:
    provider: openai
    model: llama-3.3-70b-versatile
    base_url: "https://api.groq.com/openai/v1"
    api_key: ${GROQ_API_KEY}

  # Self-hosted vLLM
  local-vllm:
    provider: openai
    model: meta-llama/Llama-3.2-8B-Instruct
    base_url: "http://gpu-server:8000/v1"
    api_key: "not-needed"
```

### Anthropic

Native Anthropic API support with automatic message formatting:

```yaml
models:
  claude-sonnet-4:
    provider: anthropic
    model: claude-sonnet-4-20250514
    api_key: ${ANTHROPIC_API_KEY}
```

### Azure OpenAI

Azure-hosted OpenAI models use deployment-based URLs and require an `api_version` field:

```yaml
models:
  gpt4-azure:
    provider: azure_openai
    base_url: "https://my-company.openai.azure.com"
    model: "gpt-4o-deploy"              # Your deployment name
    api_version: "2024-10-21"
    api_key: ${AZURE_OPENAI_KEY}
```

The engine constructs the full Azure URL automatically: `{base_url}/openai/deployments/{model}/chat/completions?api-version={api_version}`. Authentication uses the `api-key` header instead of `Authorization: Bearer`.

### Google Gemini

Native Google Gemini API support via the `generateContent` endpoint:

```yaml
models:
  gemini-pro:
    provider: google
    model: "gemini-3.1-pro"
    api_key: ${GOOGLE_API_KEY}
```

Authentication uses the `x-goog-api-key` header. No `base_url` needed -- the engine uses the default Google AI API endpoint.

### Preset providers

Several providers have preset `base_url` values, so you only need to specify `provider`, `model`, and `api_key`:

| Provider | Preset base_url |
|----------|----------------|
| `openrouter` | `https://openrouter.ai/api/v1` |
| `deepseek` | `https://api.deepseek.com/v1` |
| `mistral` | `https://api.mistral.ai/v1` |
| `xai` | `https://api.x.ai/v1` |
| `zai` | `https://open.bigmodel.cn/api/paas/v4` |

```yaml
models:
  # OpenRouter — access 100+ models via one API key
  openrouter-claude:
    provider: openrouter
    model: "anthropic/claude-sonnet-4-20250514"
    api_key: ${OPENROUTER_API_KEY}

  # DeepSeek — cost-effective coding model
  deepseek-v3:
    provider: deepseek
    model: "deepseek-chat"
    api_key: ${DEEPSEEK_API_KEY}

  # Mistral
  mistral-medium:
    provider: mistral
    model: "mistral-medium-3"
    api_key: ${MISTRAL_API_KEY}

  # xAI Grok
  grok:
    provider: xai
    model: "grok-4.1"
    api_key: ${XAI_API_KEY}

  # Z.ai GLM
  glm-5:
    provider: zai
    model: "glm-5"
    api_key: ${ZAI_API_KEY}
```

## Tool Configuration (Declarative YAML)

Declarative HTTP tools let you connect agents to any REST API without writing code. You define the endpoint, parameters, and authentication in YAML -- the engine handles the HTTP request and passes the result back to the LLM.

| Parameter | Default | Description |
|-----------|---------|-------------|
| `type` * | -- | Tool type. Currently only `http` is supported for declarative tools. |
| `method` * | -- | HTTP method: GET, POST, PUT, PATCH, DELETE. |
| `url` * | -- | Endpoint URL. Supports `${VAR}` for env vars and `{{param}}` for LLM-provided values. |
| `params` | -- | Query parameters as key-value pairs. Values can use `{{param}}` placeholders. |
| `body` | -- | Request body (POST/PUT/PATCH). Keys and values can use `{{param}}` placeholders. |
| `headers` | -- | Additional HTTP headers as key-value pairs. |
| `auth` | -- | Authentication block: type (bearer, basic, header), token/username/password/name/value. |
| `confirmation_required` | `false` | When true, pauses execution and asks the user before making the request. |
| `description` | -- | Human-readable description shown to the LLM. Helps the model decide when to use this tool. |

```yaml
tools:
  # GET with query parameters
  search_products:
    type: http
    method: GET
    url: "${CATALOG_API}/products/search"
    description: "Search the product catalog by keyword"
    params:
      query: "{{search_term}}"
      limit: "10"
    auth:
      type: bearer
      token: ${API_TOKEN}

  # POST with JSON body
  create_order:
    type: http
    method: POST
    url: "${ORDER_API}/orders"
    description: "Create a new order for a customer"
    body:
      customer_id: "{{customer_id}}"
      items: "{{items}}"
      notes: "{{notes}}"
    confirmation_required: true       # Human approval before execution
    auth:
      type: bearer
      token: ${ORDER_API_TOKEN}

  # Basic auth example
  legacy_erp:
    type: http
    method: GET
    url: "${ERP_URL}/api/inventory/{{sku}}"
    auth:
      type: basic
      username: ${ERP_USER}
      password: ${ERP_PASSWORD}

  # Custom header auth
  internal_api:
    type: http
    method: GET
    url: "http://internal:3000/data"
    auth:
      type: header
      name: "X-Internal-Key"
      value: ${INTERNAL_KEY}
```

:::tip[Placeholders vs environment variables]
`${VAR}` is expanded at startup from environment variables (static). `{{param}}` is filled by the LLM at runtime (dynamic). Use `${}` for secrets and base URLs, `{{}}` for user-specific values like search queries and IDs.
:::

## MCP Server Configuration

Model Context Protocol (MCP) servers extend agent capabilities with external tools. ByteBrew supports two transport types: **stdio** (the engine spawns a local process) and **SSE** (the engine connects to a remote HTTP server via Server-Sent Events).

| Parameter | Default | Description |
|-----------|---------|-------------|
| `command` | -- | For stdio transport: the command to run (e.g., npx, python, node). |
| `args` | `[]` | Command-line arguments for the stdio process. |
| `env` | `{}` | Environment variables passed to the stdio process. Supports `${VAR}` syntax. |
| `type` | `stdio` | Transport type: `sse` for HTTP-based MCP servers. Omit for stdio (default). |
| `url` | -- | For HTTP/SSE transport: the server URL to connect to. |
| `forward_headers` | `[]` | List of HTTP header names to forward from the incoming chat request to the MCP server. Useful for passing tenant/user context to multi-tenant MCP backends. |

```yaml
mcp_servers:
  # Stdio: Engine spawns the process and communicates over stdin/stdout
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: ${GITHUB_TOKEN}

  # Stdio: Python-based MCP server
  database:
    command: python
    args: ["-m", "mcp_server_postgres"]
    env:
      DATABASE_URL: ${DATABASE_URL}

  # SSE: Engine connects to a running server via Server-Sent Events
  analytics:
    type: sse
    url: "http://analytics-service:3000/mcp"

  # SSE: Another remote MCP server
  realtime-data:
    type: sse
    url: "http://localhost:4000/sse"

  # SSE with forwarded headers — pass tenant context to multi-tenant MCP backends
  my-platform:
    type: sse
    url: "http://mcp-server:8087/mcp"
    forward_headers:
      - "X-Org-Id"
      - "X-User-Id"
```

:::tip[Forward headers for multi-tenant MCP]
When your MCP server needs to know which organization or user is making the request, use `forward_headers`. The engine extracts these headers from the incoming chat request and includes them in every HTTP request to the MCP server. This way, the MCP server can apply tenant-specific logic without requiring separate server instances per tenant.
:::

:::note[Tool discovery]
When an MCP server connects, the engine discovers its available tools automatically. These tools appear in the agent's tool palette alongside built-in tools. You can see discovered tools and their descriptions in the Admin Dashboard under MCP Servers.
:::

## Trigger Configuration

Triggers let agents run autonomously without user interaction. Cron triggers execute on a schedule; webhook triggers fire when an external service sends an HTTP request. Both types create background tasks that the agent processes independently.

### Cron expression reference

| Expression | Description |
|------------|-------------|
| `* * * * *` | Every minute |
| `*/5 * * * *` | Every 5 minutes |
| `0 */2 * * *` | Every 2 hours |
| `0 9 * * 1-5` | Weekdays at 9:00 AM |
| `0 9,17 * * *` | Daily at 9:00 AM and 5:00 PM |
| `0 0 * * *` | Every day at midnight |
| `0 0 * * 0` | Every Sunday at midnight |
| `0 0 1 * *` | First day of each month at midnight |
| `0 0 1 1 *` | January 1st at midnight (yearly) |

```yaml
triggers:
  # Cron trigger — agent runs on a schedule
  morning-report:
    cron: "0 9 * * 1-5"               # Weekdays at 9 AM
    agent: supervisor
    message: "Generate the daily report from all data sources."

  # Webhook trigger — agent responds to external events
  order-webhook:
    type: webhook
    path: /webhooks/orders             # Exposed at POST /api/v1/webhooks/orders
    agent: sales-agent
    secret: ${WEBHOOK_SECRET}          # HMAC-SHA256 signature verification

  # Webhook without signature verification (not recommended for production)
  internal-events:
    type: webhook
    path: /webhooks/internal
    agent: ops-bot
```

### Trigger `on_complete` webhook

Triggers can fire a webhook callback when the agent completes its task. This is useful for notifying external systems about task results (e.g., CI pipelines, monitoring dashboards, or downstream workflows).

```yaml
triggers:
  telemetry-check:
    cron: "*/10 * * * *"
    agent: monitor
    message: "Check telemetry dashboards for anomalies"
    on_complete:
      webhook: "https://my-app.com/callback"
      headers:
        Authorization: "Bearer ${CALLBACK_TOKEN}"
```

The engine sends a POST request to the `webhook` URL when the agent finishes, with the task result in the request body. Custom `headers` are included in the callback request and support `${VAR}` syntax for secrets.

### Webhook security

When a `secret` is configured, the engine verifies incoming requests using HMAC-SHA256 signature verification. The external service must include the signature in the `X-Webhook-Secret` header:

```bash
# Sending a verified webhook request
curl -X POST http://localhost:8443/api/v1/webhooks/orders \
  -H "X-Webhook-Secret: whsec_your_secret_here" \
  -H "Content-Type: application/json" \
  -d '{"order_id": "12345", "event": "created", "total": 99.99}'
```

:::caution[Production webhooks]
Always configure a `secret` for production webhook triggers. Without signature verification, anyone who knows the URL can trigger your agent.
:::

## Rate Limits (EE)

Enterprise Edition supports configurable, header-based rate limiting with tiered access levels. Rate limit rules are defined at the top level of the configuration and apply to all chat API endpoints.

Each rule identifies requests by a header value (e.g., `X-Org-Id`) and assigns a tier based on another header (e.g., `X-Subscription-Tier`). Each tier defines its own request quota and time window.

```yaml
rate_limits:
  - name: "per-org"
    key_header: "X-Org-Id"
    tier_header: "X-Subscription-Tier"
    tiers:
      free:
        requests: 50
        window: "24h"
      pro:
        requests: 500
        window: "24h"
      enterprise:
        unlimited: true
    default_tier: "free"
```

| Parameter | Description |
|-----------|-------------|
| `name` | Unique name for this rate limit rule. |
| `key_header` | HTTP header used to identify the requester (e.g., org ID, user ID). |
| `tier_header` | HTTP header that specifies the requester's tier (e.g., subscription level). |
| `tiers` | Map of tier names to rate limit parameters. |
| `tiers.<name>.requests` | Maximum number of requests allowed within the window. |
| `tiers.<name>.window` | Time window as a Go duration string (e.g., `1h`, `24h`, `30m`). |
| `tiers.<name>.unlimited` | Set to `true` to allow unlimited requests for this tier. |
| `default_tier` | Tier used when the `tier_header` is missing or contains an unknown value. |

When rate limiting is active, every response includes `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` headers. Requests that exceed the limit receive HTTP 429.

:::note[Enterprise Edition]
Configurable rate limits are an Enterprise Edition feature. Community Edition uses a simple per-token rate limiter configured in Settings.
:::

---

## What's next

- [API Reference](/docs/getting-started/api-reference/)
- [Admin Dashboard: Agents](/docs/admin/agents/)
- [Core Concepts: Tools](/docs/concepts/tools/)
