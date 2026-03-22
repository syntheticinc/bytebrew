---
title: "Sales Agent"
description: "Confirmation workflows, configurable business rules, and BYOK model selection — build a sales assistant with approval gates."
---

A sales assistant that searches products, generates quotes, and applies discounts — with confirmation gates that require user approval before any financial action.

## What this demonstrates

| Engine Feature | How it's used |
|---|---|
| **`confirm_before`** | Quotes and discounts require explicit user approval before execution |
| **Settings CRUD** | Business rules (max discount, approval thresholds) managed via Admin Dashboard |
| **BYOK (Bring Your Own Key)** | End users can connect their own OpenAI/Anthropic key for the sales agent |
| **MCP tools** | Product catalog and CRM integration |

## Architecture

```
User
 |
 v
+----------------------------------------+
|           Sales Agent                  |
|                                        |
|  Tools:                                |
|  - search_products (MCP)               |
|  - get_customer (MCP)                  |
|  - create_quote (MCP, confirm_before)  |
|  - apply_discount (MCP, confirm)       |
|  - submit_order (MCP, confirm)         |
|                                        |
|  Settings (Admin Dashboard):           |
|  - max_discount_pct: 20               |
|  - auto_approve_below: 500            |
|  - require_manager_above: 5000        |
+----------------------------------------+
         |              |
         v              v
   Product Catalog    CRM API
```

## Quick start

```bash
git clone https://github.com/syntheticinc/bytebrew-examples
cd bytebrew-examples/sales-agent
docker compose up
```

Open Admin Dashboard at `http://localhost:8443/admin` to configure business rules under Settings, then chat at `http://localhost:8443`.

### BYOK setup

Users can provide their own API key via the `X-Model-Key` header or through the web client settings page. The agent will use their key instead of the server's default.

## Agent configuration

```yaml
# agents.yaml
agents:
  - name: sales-assistant
    model: gpt-4o
    byok: true
    system_prompt: |
      You are a sales assistant for {{company_name}}.
      Help customers find products, create quotes, and process orders.

      Business rules (from settings):
      - Maximum discount: {{max_discount_pct}}%
      - Orders under ${{auto_approve_below}} are auto-approved
      - Orders over ${{require_manager_above}} need manager approval

      IMPORTANT: Always confirm quotes and discounts with the customer
      before finalizing. Never apply a discount above the maximum.
    tools:
      - ask_user
    mcp_servers:
      - catalog
      - crm
    confirm_before:
      - create_quote
      - apply_discount
      - submit_order

settings:
  company_name: "Acme Sales"
  max_discount_pct: 20
  auto_approve_below: 500
  require_manager_above: 5000

mcp_servers:
  - name: catalog
    url: http://catalog:3000/mcp
    tools: [search_products, get_product, check_inventory]

  - name: crm
    url: http://crm:3001/mcp
    tools: [get_customer, create_quote, apply_discount, submit_order]
```

## Example conversations

### Product search and quote (confirmation gate)

The agent finds products, then asks for confirmation before creating a quote:

```
User: I need 50 units of the enterprise router, model ER-200.

SSE events:
  event: tool_call
  data: {"tool": "search_products", "input": {"query": "enterprise router ER-200"}}

  event: tool_result
  data: {"products": [{"sku": "ER-200", "name": "Enterprise Router 200G", "price": 1299.00, "stock": 124}]}

  event: tool_call
  data: {"tool": "get_customer", "input": {"customer_id": "cust-087"}}

  event: tool_result
  data: {"name": "Contoso Ltd", "tier": "gold", "account_manager": "Sarah Chen"}

  --- confirm_before gate ---

  event: confirmation_required
  data: {"tool": "create_quote", "input": {"customer_id": "cust-087", "items": [{"sku": "ER-200", "qty": 50, "unit_price": 1299.00}], "total": 64950.00}, "message": "Create a quote for Contoso Ltd: 50x Enterprise Router 200G at $1,299.00 each — total $64,950.00. Proceed?"}

  event: user_input
  data: {"confirmed": true}

  event: tool_call
  data: {"tool": "create_quote", "input": {"customer_id": "cust-087", "items": [{"sku": "ER-200", "qty": 50, "unit_price": 1299.00}]}}

  event: tool_result
  data: {"quote_id": "Q-4412", "total": 64950.00, "valid_until": "2026-04-21"}

  event: message
  data: {"content": "Quote Q-4412 created for Contoso Ltd: 50x Enterprise Router 200G at $1,299.00/unit, total $64,950.00. Valid until April 21. Note: since this exceeds $5,000, manager approval will be required before the order can be submitted. Would you like to apply a volume discount?"}
```

### Discount with approval gate

Discounts also require confirmation, and the agent enforces the max discount rule:

```
User: Can you apply a 15% volume discount?

SSE events:
  --- confirm_before gate ---

  event: confirmation_required
  data: {"tool": "apply_discount", "input": {"quote_id": "Q-4412", "discount_pct": 15, "new_total": 55207.50}, "message": "Apply 15% discount to Q-4412? New total: $55,207.50 (saving $9,742.50). Proceed?"}

  event: user_input
  data: {"confirmed": true}

  event: tool_call
  data: {"tool": "apply_discount", "input": {"quote_id": "Q-4412", "discount_pct": 15}}

  event: tool_result
  data: {"quote_id": "Q-4412", "discount": "15%", "new_total": 55207.50}

  event: message
  data: {"content": "Done — 15% volume discount applied to Q-4412. Updated total: $55,207.50. I'll send this to Sarah Chen (your account manager) for manager approval since it's above the $5,000 threshold."}
```

### Discount rejected by rules

The agent enforces the `max_discount_pct` setting:

```
User: Actually, make it 25%.

SSE events:
  event: message
  data: {"content": "I can't apply a 25% discount — our maximum allowed discount is 20%. Would you like me to apply the maximum 20% instead? That would bring the total to $51,960.00."}
```

## Configuring business rules

All business rules are managed as Engine Settings through the Admin Dashboard or REST API:

```bash
# Update max discount via API
curl -X PUT http://localhost:8443/api/v1/settings/max_discount_pct \
  -H "Authorization: Bearer $API_KEY" \
  -d '{"value": "25"}'
```

Changes take effect immediately — no restart needed. The agent's system prompt template re-evaluates settings on every conversation.

## Run it yourself

Full source code with Docker Compose, mock catalog/CRM APIs, and configurable business rules:

[github.com/syntheticinc/bytebrew-examples/sales-agent](https://github.com/syntheticinc/bytebrew-examples/tree/main/sales-agent)
