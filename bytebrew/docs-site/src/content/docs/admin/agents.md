---
title: "Admin Dashboard: Agents"
description: Create, configure, and manage AI agents through the Admin Dashboard visual editor.
---

The Agents page is your central hub for creating, configuring, and managing AI agents. Each agent is a self-contained entity with its own model, personality (system prompt), tools, and memory scope.

## Agent list view

The main view shows a table of all configured agents with key information at a glance:

- **Name** -- unique identifier (lowercase, alphanumeric, hyphens only).
- **Kit** -- preset tool bundle (`none` or `developer`).
- **Tools count** -- total number of tools available to the agent.
- **Knowledge** -- whether a knowledge base is configured (RAG).

Click any agent row to open a side panel with the full configuration. From there you can edit settings, view tools by security zone, or delete the agent.

## Creating an agent

Click "Create Agent" to open the agent form. Here is a walkthrough of each field:

| Field | Default | Description |
|-------|---------|-------------|
| `name` * | -- | Unique identifier. Lowercase, alphanumeric + hyphens. Used in API endpoints and spawn references. |
| `model` * | -- | Dropdown populated from configured models. Determines the LLM backend. |
| `system` * | -- | System prompt that defines agent behavior. This is the most important field. |
| `kit` | `none` | `none` = no preset tools. `developer` = adds read_file, edit_file, bash, and other dev tools. |
| `lifecycle` | `persistent` | `persistent` = accumulates context across sessions. `spawn` = fresh context each time. |
| `tool_execution` | `sequential` | `sequential` = one tool at a time. `parallel` = concurrent tool execution. |
| `max_steps` | `50` | Maximum reasoning iterations (1-500). Higher = more complex tasks, more tokens. |
| `max_context_size` | `16000` | Context window in tokens (1,000-200,000). Older messages are compressed when exceeded. |
| `tools` | `[]` | Select from available tools, grouped by security zone (Safe, Caution, Dangerous). |
| `mcp_servers` | `[]` | Multi-select from configured MCP servers. |
| `can_spawn` | `[]` | Which other agents this one can create at runtime. |
| `confirm_before` | `[]` | Tools that require user confirmation before execution. |

:::tip[Start simple, then iterate]
Begin with a focused system prompt, 2-3 tools, and the default settings. Test the agent through the chat interface, then add more tools and tweak `max_steps` and `max_context_size` based on the agent's actual workload.
:::

## YAML equivalent

Everything configured through the form can also be expressed in YAML:

```yaml
agents:
  my-agent:
    model: glm-5                    # Model dropdown
    system: |                       # System prompt
      You are a sales consultant for Acme Corp.
      Always be professional and helpful.
    kit: developer                  # Kit: none | developer
    lifecycle: persistent           # persistent | spawn
    tool_execution: parallel        # sequential | parallel
    max_steps: 100                  # 1-500
    max_context_size: 32000         # 1000-200000
    tools:                          # Grouped by security zone
      - web_search                  # Safe
      - edit_file                   # Caution
      - bash                        # Dangerous
    mcp_servers:
      - github-server
    can_spawn:
      - researcher
    confirm_before:
      - bash
      - create_order
```

---

## What's next

- [Models](/admin/models/)
- [MCP Servers](/admin/mcp-servers/)
- [Core Concepts: Agents](/concepts/agents/)
