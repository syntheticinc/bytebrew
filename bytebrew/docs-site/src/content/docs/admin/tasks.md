---
title: "Admin Dashboard: Tasks"
description: Create, monitor, and manage agent tasks through the ByteBrew Engine Admin Dashboard.
---

Tasks are units of work that agents process asynchronously. They can be created manually through the dashboard, programmatically via the API, or automatically by triggers (cron/webhook). The Tasks page gives you visibility into everything your agents are working on.

## Task list and filtering

The main view shows a paginated table of all tasks with powerful filters:

- **Status filter** -- `pending`, `in_progress`, `completed`, `failed`, `cancelled`, `needs_input`, `escalated`.
- **Source filter** -- `agent` (spawned by another agent), `cron`, `webhook`, `api`, `dashboard`.
- **Agent filter** -- filter by which agent is assigned to the task.

## Creating a task

Click "Create Task" and fill in the form:

- **Agent** -- select which agent should handle the task.
- **Title** -- short description of the task (shown in the list).
- **Description** -- detailed instructions for the agent (this becomes the message).

The agent starts working on the task immediately. You can track progress in the task detail view, which shows the agent's messages, tool calls, and results.

## Task actions

- **Cancel** -- available for tasks in `pending` or `in_progress` status. The agent stops working immediately.
- **Provide input** -- for tasks in `needs_input` status. The agent paused to ask a question or request confirmation. Type your response and the agent continues.
- **View details** -- click any task to see the full conversation, tool calls, and results.

## API equivalent

```bash
# Create a task
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer bb_token" \
  -H "Content-Type: application/json" \
  -d '{
    "agent": "researcher",
    "title": "Market analysis Q1",
    "description": "Analyze Q1 market trends for SaaS sector"
  }'

# Provide input to a waiting task
curl -X POST http://localhost:8080/api/v1/tasks/{id}/input \
  -H "Authorization: Bearer bb_token" \
  -H "Content-Type: application/json" \
  -d '{"input": "Focus on enterprise segment"}'

# Cancel a task
curl -X DELETE http://localhost:8080/api/v1/tasks/{id} \
  -H "Authorization: Bearer bb_token"
```

---

## What's next

- [Core Concepts: Tasks](/concepts/tasks/)
- [Triggers](/admin/triggers/)
- [API Reference](/getting-started/api-reference/)
