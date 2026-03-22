---
title: Tasks & Job System
description: Persistent task tracking, background execution, and the task lifecycle in ByteBrew Engine.
---

The task system gives agents persistent memory for work items that survive context window limits and session boundaries. Tasks are also the mechanism for background execution -- triggers create tasks that agents process autonomously.

## Why tasks matter

- **Persistence** -- tasks survive context window compression. Even if the agent forgets the conversation, it always knows what tasks are pending.
- **Background work** -- cron and webhook triggers create tasks that agents work on without user interaction.
- **Cross-session tracking** -- a user can create a task in one session and check its status in another.
- **Audit trail** -- every task has a status history, making it easy to track what happened and when.

## Task lifecycle

```
# Task status flow:
#
#   pending ──> in_progress ──> completed
#                    |
#                    |──> needs_input ──> in_progress (after input)
#                    |
#                    |──> failed
#                    |
#                    |──> escalated
#
#   Any active status ──> cancelled (manual cancellation)
```

| Status | Description |
|--------|-------------|
| `pending` | Task created, waiting to be picked up. Transitions to in_progress when the agent starts. |
| `in_progress` | Agent is actively working. Can transition to completed, failed, needs_input, or escalated. |
| `needs_input` | Agent paused and waiting for user input or confirmation. |
| `completed` | Task finished successfully. Terminal state. |
| `failed` | Task failed due to an error. Terminal state. |
| `escalated` | Agent determined the task needs human attention. Terminal state. |
| `cancelled` | Cancelled by user or API. Terminal state. |

## The manage_tasks tool

Agents interact with tasks through the built-in `manage_tasks` tool. The LLM decides when and how to use it based on the conversation:

```yaml
# Example conversation flow:
#
# User: "Track the quarterly report preparation"
# Agent: [calls manage_tasks: action=create, title="Quarterly report"]
#         "I've created a task to track that. I'll work on it."
#
# User: "What's on my plate?"
# Agent: [calls manage_tasks: action=list, status=pending]
#         "You have 3 pending tasks:
#          1. Quarterly report preparation
#          2. Customer feedback analysis
#          3. Team standup summary"
#
# Agent: [calls manage_tasks: action=update, id=task_abc, status=completed]
#         "Done! The quarterly report task is now complete."

# Enable for an agent:
agents:
  project-manager:
    model: glm-5
    tools:
      - manage_tasks        # Adds task tracking capability
    system: |
      You are a project manager. Track all work items as tasks.
      When a user mentions something to do, create a task for it.
```

## Task sources

Tasks can be created from multiple sources:

| Source | Description |
|--------|-------------|
| `dashboard` | Created manually through the Admin Dashboard task form. |
| `api` | Created programmatically via POST /api/v1/tasks. |
| `agent` | Created by an agent using the manage_tasks tool. |
| `cron` | Created automatically by a cron trigger at the scheduled time. |
| `webhook` | Created when an external service sends a POST to a webhook endpoint. |

---

## What's next

- [Admin: Tasks](/docs/admin/tasks/)
- [Triggers](/docs/concepts/triggers/)
- [API Reference: Tasks](/docs/getting-started/api-reference/#tasks)
