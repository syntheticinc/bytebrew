---
title: "Example: Support Agent"
description: Knowledge-based customer support agent with RAG, ticket creation with confirmation, and order tracking.
---

A knowledge-based customer support agent that answers questions from a documentation knowledge base, looks up order status, and creates support tickets for unresolved issues. This example demonstrates RAG, confirmation-required tools, and escalation patterns.

## What this demonstrates

- **Knowledge base (RAG)** -- the agent searches your support docs before answering.
- **Ticket creation with confirmation** -- the agent asks before creating a support ticket.
- **Order status lookup** -- real-time order tracking via HTTP tool.
- **Escalation behavior** -- the system prompt instructs the agent when to hand off to humans.

## Prerequisites

- A `./docs/support/` folder with your knowledge base documents (Markdown, TXT, PDF).
- A helpdesk API for ticket creation (or a mock endpoint).
- An order tracking API (or a mock endpoint).

## Full configuration

```yaml
# Support Agent — Knowledge-based with ticket creation
agents:
  support-bot:
    model: glm-5
    system: |
      You are a customer support agent. Use the knowledge base
      to answer questions. Create tickets for unresolved issues.
      Escalate urgent matters to human support.
    tools:
      - knowledge_search
      - create_ticket
      - order_status
    knowledge: "./docs/support/"

tools:
  create_ticket:
    type: http
    method: POST
    url: "${HELPDESK_API}/tickets"
    body:
      subject: "{{subject}}"
      description: "{{description}}"
      priority: "{{priority}}"
    confirmation_required: true

  order_status:
    type: http
    method: GET
    url: "${ORDER_API}/orders/{{order_id}}/status"

models:
  glm-5:
    provider: openai
    api_key: "${OPENAI_API_KEY}"
```

## How to test

```bash
# Ask a question that should be answered from the knowledge base
curl -N http://localhost:8080/api/v1/agents/support-bot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "What is your return policy for electronics?"}'

# Ask about an order
curl -N http://localhost:8080/api/v1/agents/support-bot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Where is my order #12345?"}'

# Trigger ticket creation (agent will ask for confirmation)
curl -N http://localhost:8080/api/v1/agents/support-bot/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "My laptop screen is cracked, I need a replacement"}'
```

## Customization tips

- Populate `./docs/support/` with your actual FAQ, return policy, and product guides.
- Add more tools for common support actions: refund processing, account lookup, shipping updates.
- Adjust the system prompt to match your brand voice and escalation thresholds.
- For high-volume support, set `max_context_size` to 8000 to reduce costs per conversation.

---

## What's next

- [Knowledge / RAG](/concepts/knowledge/)
- [Sales Agent Example](/examples/sales-agent/)
- [DevOps Monitor Example](/examples/devops-monitor/)
