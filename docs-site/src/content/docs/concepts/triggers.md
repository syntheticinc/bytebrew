---
title: Triggers (Cron, Webhooks)
description: Enable agents to operate autonomously with scheduled cron triggers and event-driven webhook triggers.
---

Triggers enable agents to operate autonomously without waiting for user messages. They are the foundation of proactive AI workflows -- agents that monitor, report, and react to events on their own.

## Cron triggers

Schedule agents to run at specific times using standard 5-field cron expressions. At the scheduled time, the engine creates a background task with the configured message and assigns it to the specified agent.

```yaml
triggers:
  # Run every weekday morning
  daily-digest:
    cron: "0 9 * * 1-5"
    agent: reporter
    message: "Compile the daily digest from all data sources."

  # Run every 10 minutes
  health-check:
    cron: "*/10 * * * *"
    agent: monitor
    message: "Check all monitored services and report any issues."
```

### Common patterns

| Expression | Use case |
|------------|----------|
| `*/5 * * * *` | Every 5 minutes -- health checks, monitoring |
| `0 */2 * * *` | Every 2 hours -- periodic data sync |
| `0 9 * * 1-5` | Weekdays at 9 AM -- daily reports |
| `0 0 * * 0` | Sundays at midnight -- weekly summaries |
| `0 0 1 * *` | Monthly -- billing reports, audits |

## Webhook triggers

Expose HTTP endpoints that external services can call to activate agents. The webhook request body is forwarded to the agent as the task message, giving it full context about the event.

```yaml
triggers:
  # Stripe payment events
  stripe-payment:
    type: webhook
    path: /webhooks/stripe
    agent: billing-agent
    secret: ${STRIPE_WEBHOOK_SECRET}

  # GitHub PR events
  github-pr:
    type: webhook
    path: /webhooks/github
    agent: code-reviewer
    secret: ${GITHUB_WEBHOOK_SECRET}
```

### Calling a webhook

```bash
# External service sends a POST request:
curl -X POST http://localhost:8443/api/v1/webhooks/stripe \
  -H "X-Webhook-Secret: whsec_your_secret" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "payment_intent.succeeded",
    "data": {
      "customer_id": "cus_123",
      "amount": 9900,
      "currency": "usd"
    }
  }'

# The agent receives the full JSON body as its task message
# and can act on it (update records, send notifications, etc.)
```

## Use cases

- **Daily reports** -- cron trigger at 9 AM generates and distributes a summary.
- **Alert handling** -- PagerDuty/Datadog webhook triggers an agent to analyze and triage alerts.
- **Order processing** -- e-commerce webhook triggers an agent when a new order is placed.
- **CI/CD notifications** -- GitHub webhook triggers a code review agent on new pull requests.
- **Periodic health checks** -- cron trigger every 5 minutes monitors service endpoints.

---

## What's next

- [Admin: Triggers](/docs/admin/triggers/)
- [Tasks & Jobs](/docs/concepts/tasks/)
- [Example: DevOps Monitor](/docs/examples/devops-monitor/)
