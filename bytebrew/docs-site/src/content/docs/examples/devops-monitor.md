---
title: "Example: DevOps Monitor"
description: Alert handling agent with PagerDuty webhooks, cron health checks, and task management for operational workflows.
---

An alert handling agent that monitors infrastructure health, triages incoming PagerDuty alerts, and performs automated remediation. This example demonstrates webhook triggers, cron scheduling, and task management for operational workflows.

## What this demonstrates

- **Webhook trigger** -- PagerDuty alerts are forwarded to the agent in real-time.
- **Cron trigger** -- periodic health checks every 5 minutes.
- **Task management** -- the agent tracks open incidents as tasks.
- **Escalation** -- critical issues are flagged for human attention.

## Prerequisites

- PagerDuty (or similar alerting platform) configured to send webhooks.
- Service health check endpoints accessible from the engine.

## Full configuration

```yaml
# DevOps Monitor — Alert handling with webhooks and cron
agents:
  alert-handler:
    model: glm-5
    system: |
      You are a DevOps alert handler. Analyze incoming alerts,
      filter noise, identify real issues, suggest remediation.
      Escalate P1 incidents immediately.
    tools:
      - web_search
      - manage_tasks

triggers:
  pagerduty-webhook:
    type: webhook
    path: /webhooks/pagerduty
    agent: alert-handler
    secret: ${PAGERDUTY_SECRET}

  health-check:
    cron: "*/5 * * * *"
    agent: alert-handler
    message: "Check service health status for all monitored endpoints."

models:
  glm-5:
    provider: openai
    api_key: "${OPENAI_API_KEY}"
```

## How to test

```bash
# Simulate a PagerDuty alert
curl -X POST http://localhost:8080/api/v1/webhooks/pagerduty \
  -H "X-Webhook-Secret: your-pagerduty-secret" \
  -H "Content-Type: application/json" \
  -d '{
    "event": {
      "type": "incident.triggered",
      "data": {
        "title": "High CPU usage on web-server-03",
        "severity": "critical",
        "service": "web-cluster"
      }
    }
  }'

# The agent will analyze the alert, determine severity,
# and either handle it automatically or escalate to a human.

# Check what the health-check cron has found:
curl http://localhost:8080/api/v1/tasks?agent=alert-handler&status=completed \
  -H "Authorization: Bearer bb_your_token"
```

## Customization tips

- Add custom HTTP tools for automated remediation (restart services, scale infrastructure, clear caches).
- Add a `knowledge` folder with runbooks so the agent can follow documented procedures.
- Create a `can_spawn` relationship with a log-analyzer agent for deep-dive investigations.
- Set `confirmation_required: true` on any tool that modifies production infrastructure.

---

## What's next

- [Triggers](/docs/concepts/triggers/)
- [Tasks & Jobs](/docs/concepts/tasks/)
- [IoT Analyzer Example](/docs/examples/iot-analyzer/)
