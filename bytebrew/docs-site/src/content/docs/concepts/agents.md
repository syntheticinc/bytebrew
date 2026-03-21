---
title: Agents & Lifecycle
description: Understanding ByteBrew Engine agents — identity, capabilities, memory scope, and the persistent vs spawn lifecycle.
---

An agent in ByteBrew is an LLM-powered entity with a defined identity (system prompt), capabilities (tools), and memory scope (lifecycle). Agents are the fundamental building blocks of your AI-powered workflows.

## What is an agent?

At its core, an agent is a loop: receive input, reason about it using an LLM, optionally call tools to gather information or take actions, and return a response. The system prompt defines who the agent is and how it behaves.

- **Identity** -- the system prompt gives the agent a role, personality, and knowledge boundaries.
- **Capabilities** -- tools, MCP servers, and knowledge bases determine what the agent can do.
- **Memory** -- the lifecycle setting controls whether the agent remembers previous conversations.
- **Autonomy** -- the agent decides which tools to call and in what order based on the user's request.

## Lifecycle: persistent vs spawn

The `lifecycle` setting is one of the most important decisions you make when configuring an agent. It controls the agent's memory scope:

| Lifecycle | Description |
|-----------|-------------|
| `persistent` | Accumulates context across sessions. Remembers previous conversations. Best for: customer-facing agents, personal assistants, support bots. |
| `spawn` | Fresh context per invocation. No memory between calls. Terminates after completing its task and returns a summary. Best for: sub-agents, one-off research tasks, data processing. |

```yaml
agents:
  # Persistent: remembers customer history
  support-bot:
    model: glm-5
    lifecycle: persistent
    system: |
      You are a customer support agent. Remember
      previous interactions with each customer.

  # Spawn: fresh context, used for delegation
  researcher:
    model: qwen-3-32b
    lifecycle: spawn
    system: |
      Research the given topic thoroughly.
      Return a structured summary with sources.
    tools:
      - web_search
      - knowledge_search
```

:::tip[When to use spawn]
Use `spawn` for sub-agents in a multi-agent setup. When a supervisor spawns a researcher, the researcher gets a clean context focused solely on the research task. This keeps the sub-agent focused and prevents context pollution from unrelated conversations.
:::

## System prompts

The system prompt is the most important configuration for an agent. It defines the agent's personality, capabilities, constraints, and output format. You can set it inline or load it from a file:

```yaml
agents:
  # Inline (good for short prompts)
  greeter:
    model: glm-5
    system: "You are a friendly greeter. Welcome users and ask how you can help."

  # Multi-line inline (good for medium prompts)
  analyst:
    model: glm-5
    system: |
      You are a data analyst. When given data, you:
      1. Identify key trends and patterns
      2. Calculate relevant statistics
      3. Provide actionable recommendations

  # External file (good for long, version-controlled prompts)
  enterprise-agent:
    model: glm-5
    system_file: "./prompts/enterprise-agent.txt"
```

## Agent capabilities

Each agent can be configured with a unique combination of capabilities:

- **Built-in tools** -- `web_search`, `knowledge_search`, `manage_tasks`, `ask_user`.
- **Custom HTTP tools** -- declarative API calls defined in YAML (see [Tools docs](/docs/concepts/tools/)).
- **MCP servers** -- external tools via Model Context Protocol.
- **Knowledge base (RAG)** -- auto-indexed document folder for grounded responses.
- **Sub-agent spawning** -- ability to create and delegate to other agents.

---

## What's next

- [Multi-Agent Orchestration](/docs/concepts/multi-agent/)
- [Tools](/docs/concepts/tools/)
- [Admin: Agents](/docs/admin/agents/)
