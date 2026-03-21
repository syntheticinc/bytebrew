---
title: "Example: Sales Agent"
description: Multi-agent sales team with supervisor, product search, inventory checks, order creation, and CRM integration.
---

A multi-agent sales team with a supervisor that coordinates product search, inventory checks, order creation, and customer support. This example demonstrates agent spawning, custom HTTP tools, MCP integration, and cron triggers.

## What this demonstrates

- **Multi-agent orchestration** -- a supervisor delegates to specialized sales and support agents.
- **Custom HTTP tools** -- product search, inventory check, and order creation via REST APIs.
- **MCP integration** -- CRM data access via an MCP server.
- **Cron trigger** -- automatic morning lead review on weekdays.
- **Mixed models** -- powerful model for the supervisor, cheaper models for specialists.

## Prerequisites

- A running ByteBrew Engine instance.
- API keys for your chosen LLM providers (OpenAI and/or Anthropic).
- A product catalog API (or a mock endpoint for testing).
- A CRM API key for the MCP CRM server (optional).

## Full configuration

```yaml
# Sales Agent — Full Configuration Example
agents:
  sales-supervisor:
    model: glm-5
    system: |
      You are a sales team supervisor. Route incoming customer
      queries to the appropriate sales or support agent.
      Prioritize high-intent buyers.
    can_spawn:
      - sales-agent
      - support-agent
    tools:
      - customer_lookup

  sales-agent:
    model: qwen-3-32b
    lifecycle: spawn
    system: |
      You are a sales consultant. Interview the buyer to
      understand their needs, recommend products, check
      inventory, and create orders when ready.
    tools:
      - product_search
      - check_inventory
      - create_order
      - apply_discount
    mcp_servers:
      - crm-api

  support-agent:
    model: claude-sonnet-4
    lifecycle: spawn
    system: |
      You are a customer support agent. Answer product
      questions using the knowledge base, create tickets
      for issues you cannot resolve.
    tools:
      - knowledge_search
      - create_ticket
      - order_status

tools:
  product_search:
    type: http
    method: GET
    url: "${CATALOG_API}/products/search"
    params:
      query: "{{input}}"

  check_inventory:
    type: http
    method: GET
    url: "${CATALOG_API}/inventory/{{product_id}}"

  create_order:
    type: http
    method: POST
    url: "${ORDER_API}/orders"
    body:
      customer_id: "{{customer_id}}"
      items: "{{items}}"

mcp_servers:
  crm-api:
    command: npx
    args: ["-y", "@bytebrew/mcp-crm"]
    env:
      CRM_API_KEY: "${CRM_API_KEY}"

models:
  glm-5:
    provider: openai
    api_key: "${OPENAI_API_KEY}"
  qwen-3-32b:
    provider: openai
    api_key: "${OPENAI_API_KEY}"
  claude-sonnet-4:
    provider: anthropic
    api_key: "${ANTHROPIC_API_KEY}"

triggers:
  morning-leads:
    cron: "0 9 * * 1-5"
    agent: sales-supervisor
    message: "Check for new leads from overnight and prioritize follow-ups."
```

## How to test

```bash
# Start a conversation with the supervisor
curl -N http://localhost:8080/api/v1/agents/sales-supervisor/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "I need a laptop for video editing under $1500"}'

# The supervisor will:
# 1. Analyze the request
# 2. Spawn the sales-agent to search products and check inventory
# 3. Return recommendations based on the results

# Follow up in the same session:
curl -N http://localhost:8080/api/v1/agents/sales-supervisor/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "I will take the second option", "session_id": "<id-from-previous>"}'
```

## Customization tips

- Replace `${CATALOG_API}` and `${ORDER_API}` with your actual API endpoints.
- Add a `knowledge` folder with product documentation for the support agent.
- Add a `apply_discount` tool with `confirmation_required: true` for price overrides.
- Adjust `max_steps` on the supervisor if complex multi-step orders time out.

---

## What's next

- [Multi-Agent Orchestration](/docs/concepts/multi-agent/)
- [Support Agent Example](/docs/examples/support-agent/)
- [Configuration Reference](/docs/getting-started/configuration/)
