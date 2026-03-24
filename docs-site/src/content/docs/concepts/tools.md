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

### Safe Zone

No side effects. Safe to run without confirmation.

| Tool | Description |
|------|-------------|
| `ask_user` | Pause and ask user structured questions. Supports `input_type` (text, single_select, multi_select, confirm) and rich options with `value`/`description`. See [ask_user details](#ask_user-structured-questions) below. |
| `show_structured_output` | Display structured data blocks to the user (summary tables, action buttons). See [structured output](#structured-output) below. |
| `web_search` | Search the internet for information. Returns relevant web page snippets. |
| `knowledge_search` | Search the agent's knowledge base (RAG). Automatically injected when `knowledge:` path is set. |
| `manage_tasks` | Create and manage work tasks. Enables persistent task tracking across sessions. |
| `manage_subtasks` | Manage subtasks within a parent task. |

### Caution Zone

Operations that access external data or modify reversible state.

| Tool | Description |
|------|-------------|
| `web_fetch` | Fetch URL content. |
| `glob` | Find files matching a pattern. |
| `grep_search` | Search file contents with regex. |
| `search_code` | Find code symbols by name. |

### Dangerous Zone

Operations with significant side effects. Consider adding to `confirm_before`.

| Tool | Description |
|------|-------------|
| `read_file` | Read any file from the filesystem. |
| `write_file` | Create or overwrite files. |
| `edit_file` | Modify file contents. |
| `execute_command` | Run shell commands. |

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

## Tool Confirmation (`confirm_before`)

Human-in-the-loop safety gate for destructive actions. The `confirm_before` configuration requires explicit user approval before a tool executes. Use this for any tool that creates orders, modifies data, sends emails, runs commands, or has other irreversible side effects.

Tools listed in `confirm_before` require user approval before execution:

```yaml
agents:
  sales-agent:
    tools:
      - create_order
      - web_search
    confirm_before:
      - create_order    # Approval required before placing orders
```

When the agent calls a confirmed tool, execution pauses and a `confirmation` SSE event is sent to the client. The client approves or rejects via the respond endpoint.

## Parallel vs Sequential Execution

```yaml
agents:
  technical-agent:
    tool_execution: parallel    # Run multiple tool calls simultaneously
```

Default is `sequential`. Use `parallel` when tools are independent (e.g., checking service status + fetching logs at the same time).

## ask_user: structured questions

The `ask_user` tool supports rich, structured questions with typed inputs and selectable options. An agent can ask 1-5 questions in a single call. Each question supports the following fields:

| Field | Required | Description |
|-------|----------|-------------|
| `text` | Yes | The question text displayed to the user. |
| `input_type` | No | Input type: `text` (default), `single_select`, `multi_select`, or `confirm`. |
| `options` | No | Array of 2-5 options (for select types). Each option has `label` (required), `value` (optional), and `description` (optional). |
| `default` | No | Default answer pre-filled for the user. |
| `columns` | No | Grid columns for card-style layout (e.g., 2, 3). |

### Examples

**Single select with descriptions:**

```json
{
  "text": "What platform are you building for?",
  "input_type": "single_select",
  "options": [
    {"label": "iOS", "value": "ios", "description": "iPhone and iPad"},
    {"label": "Android", "value": "android", "description": "Google Play"},
    {"label": "Both", "value": "both", "description": "Cross-platform"}
  ],
  "columns": 3
}
```

**Multi-select:**

```json
{
  "text": "Which integrations do you need?",
  "input_type": "multi_select",
  "options": [
    {"label": "Slack"},
    {"label": "Email"},
    {"label": "Webhook"},
    {"label": "SMS"}
  ]
}
```

**Confirmation:**

```json
{
  "text": "Should I proceed with the deployment?",
  "input_type": "confirm"
}
```

The SSE event type for ask_user is `user_input_required`. See [REST API: Handling user_input_required events](/docs/integration/rest-api/#handling-user_input_required-events) for client implementation details.

## Structured output

The `show_structured_output` tool lets agents present organized data to the user -- tables, summaries, and action buttons. Unlike `ask_user`, structured output is display-only and does not pause execution.

| Field | Required | Description |
|-------|----------|-------------|
| `output_type` | Yes | Type of output (e.g., `summary_table`). |
| `title` | No | Title displayed above the output block. |
| `description` | No | Description text. |
| `rows` | No | Array of `{label, value}` pairs for table rows. |
| `actions` | No | Array of `{label, type, value}` action buttons. `type` is `primary` or `secondary`. |

### Example

An agent analyzing a project might produce:

```json
{
  "output_type": "summary_table",
  "title": "Project Overview",
  "rows": [
    {"label": "Name", "value": "MyApp"},
    {"label": "Framework", "value": "React + Go"},
    {"label": "Test Coverage", "value": "87%"}
  ],
  "actions": [
    {"label": "Run Tests", "type": "primary", "value": "run_tests"},
    {"label": "Skip", "type": "secondary", "value": "skip"}
  ]
}
```

The client receives this as a `structured_output` SSE event. Action button clicks can be sent back as regular chat messages.

---

## What's next

- [Configuration: Tools](/docs/getting-started/configuration/#tool-configuration-declarative-yaml)
- [Admin: MCP Servers](/docs/admin/mcp-servers/)
- [Tasks & Jobs](/docs/concepts/tasks/)
