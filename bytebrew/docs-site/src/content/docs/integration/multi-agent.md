---
title: Multi-Agent Configuration
description: Complete guide to configuring multi-agent orchestration — orchestrator pattern, can_spawn, lifecycle, per-agent models, tool isolation, and nested spawning.
---

This guide covers the full configuration for multi-agent systems in ByteBrew Engine. Multi-agent orchestration lets a supervisor agent delegate tasks to specialist agents, each with their own model, tools, and constraints.

## Orchestrator pattern (hub-and-spoke)

The most common pattern is a single **orchestrator** (supervisor) agent that delegates to multiple **specialist** agents:

```
         ┌─────────────┐
         │  Orchestrator │  (persistent, powerful model)
         │  can_spawn:   │
         │  - researcher │
         │  - writer     │
         │  - reviewer   │
         └──────┬────────┘
                │
     ┌──────────┼──────────┐
     │          │          │
┌────▼───┐ ┌───▼────┐ ┌───▼────┐
│research│ │ writer │ │reviewer│  (spawn, cheaper models)
│  er    │ │        │ │        │
└────────┘ └────────┘ └────────┘
```

The orchestrator receives user messages and decides which specialist(s) to invoke based on reasoning. Each specialist runs with `lifecycle: spawn` -- fresh context, focused on the subtask, terminates after completion.

## can_spawn configuration

The `can_spawn` field lists which agents a given agent is allowed to create at runtime:

```yaml
agents:
  orchestrator:
    model: gpt-4o
    can_spawn:
      - researcher     # Engine creates spawn_researcher tool
      - writer         # Engine creates spawn_writer tool
      - reviewer       # Engine creates spawn_reviewer tool
```

For each entry in `can_spawn`, the engine auto-generates a tool named `spawn_<agent-name>`. The orchestrator's LLM sees these as regular tools and decides when to use them based on the system prompt and conversation context.

The generated spawn tool accepts a single `message` parameter -- the task description for the sub-agent:

```
# What the LLM sees:
Tool: spawn_researcher
Description: Spawn the 'researcher' agent with a task message
Parameters:
  message (required): The task to assign to the researcher agent
```

## Lifecycle: persistent vs spawn

| Setting | When to use | Behavior |
|---------|------------|----------|
| `persistent` | Orchestrators, customer-facing agents | Maintains conversation history across sessions. Never terminates. |
| `spawn` | Specialist sub-agents | Fresh context per invocation. Returns a summary when done, then terminates. |

:::tip[Rule of thumb]
The orchestrator should be `persistent` (remembers the conversation). All agents it spawns should be `spawn` (isolated, focused). This prevents context pollution between tasks.
:::

```yaml
agents:
  orchestrator:
    lifecycle: persistent     # Keeps conversation history
    can_spawn: [specialist]

  specialist:
    lifecycle: spawn          # Fresh for each delegation
```

## Nested spawn (orchestrator -> agent -> sub-agent)

Specialists can themselves spawn further sub-agents, creating a tree:

```yaml
agents:
  ceo:
    model: gpt-4o
    lifecycle: persistent
    can_spawn: [sales-lead, engineering-lead]

  sales-lead:
    model: gpt-4o-mini
    lifecycle: spawn
    can_spawn: [market-researcher, proposal-writer]

  engineering-lead:
    model: gpt-4o-mini
    lifecycle: spawn
    can_spawn: [code-reviewer, test-runner]

  market-researcher:
    model: gpt-4o-mini
    lifecycle: spawn
    tools: [web_search]

  proposal-writer:
    model: gpt-4o-mini
    lifecycle: spawn
    tools: [knowledge_search]

  code-reviewer:
    model: qwen-local
    lifecycle: spawn
    kit: developer

  test-runner:
    model: qwen-local
    lifecycle: spawn
    kit: developer
```

Results flow back up the tree: `market-researcher` returns to `sales-lead`, which returns to `ceo`.

## Per-agent model assignment

Different agents can use different models. Use expensive models where reasoning quality matters, and cheap/local models for simple tasks:

```yaml
agents:
  orchestrator:
    model: gpt-4o              # Best reasoning for coordination ($$$)
  researcher:
    model: gpt-4o-mini         # Good enough for web search ($)
  local-analyzer:
    model: qwen-local          # Free, runs on your GPU

models:
  gpt-4o:
    provider: openai
    api_key: ${OPENAI_API_KEY}
  gpt-4o-mini:
    provider: openai
    api_key: ${OPENAI_API_KEY}
  qwen-local:
    provider: ollama
    model: qwen2.5-coder:32b
    base_url: "http://localhost:11434/v1"
    api_key: "ollama"
```

## Tool whitelisting and scope isolation

Each agent only sees the tools in its own configuration. This is a critical security boundary:

```yaml
agents:
  customer-service:
    tools:
      - knowledge_search         # Can search docs
      - create_ticket            # Can create tickets
      # Cannot: bash, delete_file, create_order

  devops-bot:
    tools:
      - web_search               # Can search the web
      - manage_tasks             # Can track incidents
    kit: developer               # Has bash, read_file, etc.
      # Cannot: create_ticket, create_order

  order-processor:
    tools:
      - create_order             # Can create orders
      - check_inventory          # Can check stock
      # Cannot: bash, knowledge_search
```

MCP servers are also isolated per-agent:

```yaml
agents:
  github-bot:
    mcp_servers: [github]        # Only GitHub tools
  db-bot:
    mcp_servers: [database]      # Only database tools
```

## confirm_before for destructive operations

Require human approval before an agent executes sensitive tools:

```yaml
agents:
  order-agent:
    tools:
      - search_products          # Safe, no confirmation
      - create_order             # Dangerous, needs approval
      - refund_order             # Dangerous, needs approval
    confirm_before:
      - create_order
      - refund_order
```

When the agent calls `create_order`, the stream pauses with a `confirmation` event. The client must approve or reject before execution continues. See [REST API Chat: Handling confirmation events](/docs/integration/rest-api/#handling-confirmation-events).

## Complete multi-agent example (3 agents)

A content creation team with a project manager, researcher, and writer:

```yaml
agents:
  project-manager:
    model: gpt-4o
    lifecycle: persistent
    system: |
      You are a content project manager. When a user requests
      content, break it into research and writing tasks.
      Delegate research to the researcher and writing to the writer.
      Review the final output before returning it to the user.
    can_spawn:
      - researcher
      - writer
    tools:
      - manage_tasks

  researcher:
    model: gpt-4o-mini
    lifecycle: spawn
    system: |
      You are a research analyst. Given a topic, find relevant
      information, statistics, and examples. Return a structured
      brief with sources that a writer can use.
    tools:
      - web_search
      - knowledge_search

  writer:
    model: claude-sonnet-4
    lifecycle: spawn
    system: |
      You are a content writer. Given a research brief and
      content requirements, produce polished content.
      Follow the specified format (blog post, email, report).
      Cite sources from the research brief.
    tools:
      - knowledge_search

models:
  gpt-4o:
    provider: openai
    api_key: ${OPENAI_API_KEY}
  gpt-4o-mini:
    provider: openai
    api_key: ${OPENAI_API_KEY}
  claude-sonnet-4:
    provider: anthropic
    api_key: ${ANTHROPIC_API_KEY}
```

Test it:

```bash
curl -N http://localhost:8080/api/v1/agents/project-manager/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -d '{"message": "Write a blog post about the future of AI agents in enterprise software"}'
```

---

## What's next

- [Core Concepts: Multi-Agent](/docs/concepts/multi-agent/)
- [REST API Chat](/docs/integration/rest-api/)
- [Example: Sales Agent](/docs/examples/sales-agent/)
