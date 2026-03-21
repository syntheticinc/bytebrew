---
title: Multi-Agent Orchestration
description: Build teams of specialized agents that collaborate on complex tasks using the orchestrator pattern.
---

Multi-agent orchestration lets you build teams of specialized agents that collaborate on complex tasks. A supervisor agent coordinates the team, delegating subtasks to specialist agents that each have their own tools and expertise.

## How it works

The orchestration model is simple but powerful:

- `can_spawn: [agent-name]` defines which agents a supervisor can create at runtime.
- The engine auto-generates a `spawn_<name>` tool for each allowed target.
- The LLM decides **when** to spawn based on reasoning. The config limits **what** is possible.
- Spawned agents run with `lifecycle: spawn` (fresh context, focused on the subtask).
- When the sub-agent completes, its summary is returned to the supervisor.
- The supervisor integrates the result and continues its own reasoning.

## Spawn tree architecture

In a multi-agent system, agents form a tree structure. The supervisor sits at the root and delegates to specialists. Specialists can even spawn their own sub-agents:

```
# Spawn tree visualization:
#
#   supervisor (persistent)
#   |-- sales-agent (spawn)
#   |   |-- inventory-checker (spawn)
#   |-- support-agent (spawn)
#   |-- researcher (spawn)
#
# Each spawn agent gets a fresh context focused solely on its task.
# Results flow back up the tree to the supervisor.
```

## When to use multi-agent

- **Complex workflows** -- a single agent cannot handle all aspects of a task (e.g., sales requires product lookup, inventory check, and order creation).
- **Specialized models** -- use a powerful model for the supervisor (reasoning) and cheaper models for specialists (data retrieval).
- **Tool isolation** -- a researcher should not have access to order creation tools, and vice versa.
- **Parallel processing** -- spawn multiple agents simultaneously to work on independent subtasks.

## Full example

A sales team with a supervisor that delegates to a sales consultant and a support agent:

```yaml
agents:
  supervisor:
    model: glm-5                  # Powerful model for coordination
    lifecycle: persistent         # Remembers customer interactions
    can_spawn:
      - sales-agent               # Engine creates spawn_sales_agent tool
      - researcher                # Engine creates spawn_researcher tool
    system: |
      You lead a sales team. When a customer asks about products,
      delegate to the sales-agent. When they need research on
      a topic, delegate to the researcher.

      After receiving results from sub-agents, synthesize
      a final response for the customer.

  sales-agent:
    model: qwen-3-32b             # Cheaper model for data lookup
    lifecycle: spawn              # Fresh context per delegation
    tools:
      - search_products
      - check_inventory
      - create_order
    system: |
      You are a sales consultant. Find products matching
      the customer's needs, check availability, and
      create orders when the customer is ready.

  researcher:
    model: claude-sonnet-4
    lifecycle: spawn
    tools:
      - web_search
      - knowledge_search
    system: |
      Research the given topic thoroughly.
      Return a structured report with:
      - Key findings
      - Supporting data
      - Sources
```

:::tip[Model selection strategy]
Use your most capable (and expensive) model for the supervisor, since it handles the complex reasoning of when to delegate, how to synthesize results, and what to tell the user. Use faster, cheaper models for specialist agents that mostly do data retrieval and simple processing.
:::

---

## What's next

- [Agents & Lifecycle](/docs/concepts/agents/)
- [Tools](/docs/concepts/tools/)
- [Example: Sales Agent](/docs/examples/sales-agent/)
