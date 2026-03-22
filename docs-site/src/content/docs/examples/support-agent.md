---
title: "Support Agent"
description: "Multi-agent orchestration with parallel tool execution — build a support system that routes issues to specialized agents."
---

A multi-agent support system where a router agent analyzes incoming issues and spawns specialized agents (billing, technical) to handle them — with parallel tool execution for fast diagnostics.

## What this demonstrates

| Engine Feature | How it's used |
|---|---|
| **Agent spawn** | Router agent spawns billing or technical specialists on demand |
| **Parallel tool execution** | Technical agent runs multiple diagnostic tools simultaneously |
| **8 MCP tools** | Real integrations: ticketing, billing API, infrastructure checks |
| **Multi-agent orchestration** | 3 agents with distinct roles and tool sets |

## Architecture

```
User
 |
 v
+----------------------------------+
|        Router Agent              |
|  Classifies issue, spawns the   |
|  right specialist agent          |
|                                  |
|  Tools: spawn_agent, ask_user    |
+--------+---------------+--------+
         |               |
         v               v
+--------------+ +------------------+
| Billing Agent| | Technical Agent   |
|              | |                   |
| Tools:       | | Tools:            |
| - get_invoice| | - check_status    |
| - get_sub    | | - check_logs      |
| - apply_refund| | - run_diagnostic |
| - create_ticket| | - restart_service|
|              | | - create_ticket   |
+--------------+ +------------------+
         |               |
         v               v
   Billing API     Infrastructure
```

## Quick start

```bash
git clone https://github.com/syntheticinc/bytebrew-examples
cd bytebrew-examples/support-agent
docker compose up
```

The example includes mock billing and infrastructure APIs so you can run the full workflow locally.

## Agent configuration

```yaml
# agents.yaml
agents:
  - name: router
    model: gpt-4o
    system_prompt: |
      You are a support router. Analyze the user's issue and spawn
      the appropriate specialist agent:
      - "billing" for payment, invoice, subscription, or refund issues
      - "technical" for outages, errors, performance, or connectivity issues
      Ask clarifying questions if the issue category is ambiguous.
    tools:
      - spawn_agent
      - ask_user

  - name: billing
    model: gpt-4o-mini
    system_prompt: |
      You are a billing support specialist. Look up invoices and
      subscriptions, process refunds up to $50 automatically,
      and create tickets for larger amounts.
    tools:
      - create_ticket
    mcp_servers:
      - billing-api

  - name: technical
    model: gpt-4o-mini
    system_prompt: |
      You are a technical support specialist. Run diagnostics
      in parallel when possible to speed up resolution.
      Check service status, logs, and run tests simultaneously.
    tools:
      - create_ticket
    mcp_servers:
      - infra-api
    parallel_tool_calls: true

mcp_servers:
  - name: billing-api
    url: http://billing:3000/mcp
    tools: [get_invoice, get_subscription, apply_refund]

  - name: infra-api
    url: http://infra:3001/mcp
    tools: [check_status, check_logs, run_diagnostic, restart_service]
```

## Example conversations

### Technical issue (parallel diagnostics)

The router spawns a technical agent, which runs multiple checks in parallel:

```
User: My API is returning 502 errors since this morning.

SSE events:
  event: tool_call
  data: {"tool": "spawn_agent", "input": {"agent": "technical", "task": "User reports 502 errors on API since this morning"}}

  event: agent_spawned
  data: {"agent": "technical", "id": "agent-tech-1"}

  --- Technical agent runs 3 tools in parallel ---

  event: tool_call
  data: {"tool": "check_status", "input": {"service": "api-gateway"}, "parallel_group": "diag-1"}

  event: tool_call
  data: {"tool": "check_logs", "input": {"service": "api-gateway", "since": "6h", "level": "error"}, "parallel_group": "diag-1"}

  event: tool_call
  data: {"tool": "run_diagnostic", "input": {"test": "api-health"}, "parallel_group": "diag-1"}

  event: tool_result
  data: {"tool": "check_status", "result": {"status": "degraded", "uptime": "99.2%", "error_rate": "12%"}}

  event: tool_result
  data: {"tool": "check_logs", "result": {"errors": [{"msg": "upstream timeout", "count": 847, "first_seen": "06:12 UTC"}]}}

  event: tool_result
  data: {"tool": "run_diagnostic", "result": {"api_health": "fail", "db_connection": "ok", "cache": "ok"}}

  event: tool_call
  data: {"tool": "restart_service", "input": {"service": "api-gateway"}}

  event: tool_result
  data: {"tool": "restart_service", "result": {"status": "restarted", "healthy_after": "12s"}}

  event: message
  data: {"content": "I found the issue: your API gateway was experiencing upstream timeouts (847 errors since 06:12 UTC, 12% error rate). Database and cache are healthy — the problem was isolated to the gateway process. I've restarted the service and it's healthy again. Error rate is back to normal. Ticket TK-4521 created for post-mortem."}
```

### Billing issue (refund flow)

The router spawns a billing agent to handle a refund:

```
User: I was double-charged on my last invoice.

SSE events:
  event: tool_call
  data: {"tool": "spawn_agent", "input": {"agent": "billing", "task": "User reports double charge on last invoice"}}

  event: agent_spawned
  data: {"agent": "billing", "id": "agent-bill-1"}

  event: tool_call
  data: {"tool": "get_invoice", "input": {"customer_id": "cust-229", "latest": true}}

  event: tool_result
  data: {"invoice_id": "inv-8841", "amount": 49.00, "items": [{"desc": "Pro Plan", "amount": 49.00}, {"desc": "Pro Plan (duplicate)", "amount": 49.00}]}

  event: tool_call
  data: {"tool": "apply_refund", "input": {"invoice_id": "inv-8841", "amount": 49.00, "reason": "duplicate charge"}}

  event: tool_result
  data: {"refund_id": "ref-112", "status": "processed", "amount": 49.00}

  event: message
  data: {"content": "You're right — invoice INV-8841 shows a duplicate charge of $49.00 for the Pro Plan. I've processed a refund of $49.00 (ref: REF-112). It should appear on your statement within 3-5 business days."}
```

## Run it yourself

Full source code with Docker Compose, mock APIs, and example scenarios:

[github.com/syntheticinc/bytebrew-examples/support-agent](https://github.com/syntheticinc/bytebrew-examples/tree/main/support-agent)
