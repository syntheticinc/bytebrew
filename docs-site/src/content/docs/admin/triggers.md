---
title: "Admin Dashboard: Triggers"
description: Configure cron schedules and webhook endpoints to run agents autonomously.
---

Triggers enable agents to run autonomously without user interaction. Use cron triggers for scheduled tasks (daily reports, periodic checks) and webhook triggers for event-driven workflows (order created, payment received, deployment completed).

## Cron triggers

Schedule agents to run at specific times using standard cron syntax. Each cron trigger creates a background task at the scheduled time.

### Common cron patterns

| Expression | Description |
|------------|-------------|
| `*/5 * * * *` | Every 5 minutes |
| `0 */2 * * *` | Every 2 hours |
| `0 9 * * 1-5` | Weekdays at 9:00 AM |
| `0 9,17 * * *` | Daily at 9:00 AM and 5:00 PM |
| `0 0 * * *` | Every day at midnight |
| `0 0 * * 0` | Every Sunday at midnight |
| `0 0 1 * *` | First day of each month |

```yaml
triggers:
  morning-report:
    cron: "0 9 * * 1-5"              # Weekdays at 9 AM
    agent: supervisor
    message: "Generate daily report"  # Message sent to the agent
```

## Webhook triggers

Expose HTTP endpoints that external services can call to trigger agents. The incoming request body is forwarded to the agent as the message.

- The webhook URL follows the pattern `/api/v1/webhooks/<path>`.
- Incoming POST body is forwarded to the agent as the task message.
- Configure a `secret` for HMAC signature verification (recommended for production).
- The webhook request is authenticated via the `X-Webhook-Secret` header.

```yaml
triggers:
  order-webhook:
    type: webhook
    path: /webhooks/orders             # POST /api/v1/webhooks/orders
    agent: sales-agent
    secret: ${WEBHOOK_SECRET}          # Signature verification

# Trigger the webhook externally:
# curl -X POST http://localhost:8443/api/v1/webhooks/orders \
#   -H "X-Webhook-Secret: your-secret" \
#   -H "Content-Type: application/json" \
#   -d '{"order_id": "12345", "event": "created"}'
```

## Managing triggers

- **Enable/disable** -- toggle triggers on and off without deleting them. Disabled triggers are retained in configuration but do not fire.
- **Edit** -- change the schedule, agent, message, or secret at any time.
- **Delete** -- permanently remove a trigger. Existing tasks created by the trigger are not affected.
- **History** -- view recent trigger executions in the Audit Log, including task IDs and outcomes.

---

## What's next

- [Core Concepts: Triggers](/docs/concepts/triggers/)
- [Configuration: Triggers](/docs/getting-started/configuration/#trigger-configuration)
- [Tasks](/docs/admin/tasks/)
