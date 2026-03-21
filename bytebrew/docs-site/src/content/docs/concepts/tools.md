---
title: Tools (MCP + Declarative)
description: Connect agents to the outside world with built-in tools, declarative HTTP tools, and MCP server integrations.
---

Tools are the bridge between an agent's reasoning and the outside world. Without tools, an agent can only generate text. With tools, it can search the web, query databases, create orders, send notifications, and interact with any API.

## Types of tools

| Type | Description |
|------|-------------|
| Built-in | Pre-built tools included with the engine: `web_search`, `knowledge_search`, `manage_tasks`, `ask_user`. |
| Declarative HTTP | Custom tools defined in YAML that make HTTP requests. No code required. |
| MCP | External tools provided by Model Context Protocol servers. Supports any MCP-compatible server. |
| Kit | Pre-packaged tool bundles. The `developer` kit adds read_file, edit_file, bash, and other dev tools. |

## Built-in tools

| Tool | Zone | Description |
|------|------|-------------|
| `web_search` | Safe | Search the internet for information. Returns relevant web page snippets. |
| `knowledge_search` | Safe | Search the agent's knowledge base (RAG). Automatically injected when `knowledge:` path is set. |
| `manage_tasks` | Safe | Create, list, update, and complete tasks. Enables persistent task tracking across sessions. |
| `ask_user` | Safe | Pause execution and ask the user a question. Useful for clarification or confirmation. |

## Declarative HTTP tools

Connect agents to any REST API without writing code. Define the endpoint, parameters, authentication, and the engine handles the HTTP request:

```yaml
tools:
  # GET request with query parameters
  get_weather:
    type: http
    method: GET
    url: "https://api.weather.com/v1/current"
    description: "Get current weather for a city"
    params:
      location: "{{city}}"
      units: "metric"
    auth:
      type: bearer
      token: ${WEATHER_API_KEY}

  # POST request with JSON body
  create_ticket:
    type: http
    method: POST
    url: "${HELPDESK_API}/tickets"
    description: "Create a support ticket"
    body:
      subject: "{{subject}}"
      description: "{{description}}"
      priority: "{{priority}}"
    confirmation_required: true     # Ask user before executing
```

## MCP tools

MCP (Model Context Protocol) is an open standard for connecting AI agents to external tools. Any MCP-compatible server works with ByteBrew:

```yaml
mcp_servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: ${GITHUB_TOKEN}

  # Then reference it in agent config:
agents:
  dev-agent:
    model: glm-5
    mcp_servers:
      - github            # All tools from the GitHub MCP server
```

## Per-agent tool isolation

Each agent sees only the tools listed in its configuration. This is a security and reliability feature:

- A customer support agent should not have access to `bash` or `delete_file`.
- A researcher should not be able to `create_order`.
- Different agents can use different MCP servers with different credentials.

:::note[Tool names must be unique]
Tool names are globally unique across your configuration. If you define a custom tool `search` and an MCP server also exposes a tool named `search`, the custom tool takes precedence and the MCP tool is shadowed.
:::

---

## What's next

- [Configuration: Tools](/docs/getting-started/configuration/#tool-configuration-declarative-yaml)
- [Admin: MCP Servers](/docs/admin/mcp-servers/)
- [Tasks & Jobs](/docs/concepts/tasks/)
