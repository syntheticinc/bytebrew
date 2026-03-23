---
title: "Admin Dashboard: MCP Servers"
description: Add, configure, and monitor Model Context Protocol server connections in ByteBrew Engine.
---

Model Context Protocol (MCP) is an open standard for connecting AI agents to external tools and data sources. The MCP Servers page lets you add, configure, and monitor MCP server connections.

## Transport types

| Type | Description |
|------|-------------|
| `stdio` | The engine spawns a local process and communicates over stdin/stdout. Best for npm packages and local scripts. |
| `sse` | The engine connects to a remote HTTP server via Server-Sent Events. Best for remote services and microservices. |

## Adding from catalog

The catalog contains pre-configured, well-known MCP servers. Adding one is a one-click operation:

- Click "Add from Catalog" on the MCP Servers page.
- Browse or search for the server you need (GitHub, filesystem, PostgreSQL, etc.).
- Click "Add" -- the name, command, and args are pre-filled.
- Fill in required environment variables (e.g., `GITHUB_TOKEN`) and save.
- The engine spawns the process and discovers available tools automatically.

## Adding a custom server

For servers not in the catalog, click "Add Custom" and fill in the form:

- **Name** -- unique identifier for referencing from agent configs.
- **Type** -- stdio or sse.
- **Command / URL** -- for stdio: the command to run. For http/sse: the server URL.
- **Args** -- command-line arguments (stdio only).
- **Environment variables** -- key-value pairs passed to the process (stdio only).

```yaml
# Stdio: Engine spawns the process
mcp_servers:
  github:
    command: npx
    args: ["-y", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_TOKEN: ${GITHUB_TOKEN}

  # Python MCP server
  custom-tools:
    command: python
    args: ["-m", "my_mcp_server"]
    env:
      DATABASE_URL: ${DATABASE_URL}

# SSE: Engine connects to a running server via Server-Sent Events
  analytics:
    type: sse
    url: "http://analytics-service:3000/mcp"

# SSE: Another remote MCP server
  realtime:
    type: sse
    url: "http://localhost:4000/sse"
```

## Monitoring and troubleshooting

Each MCP server shows a status indicator and the count of discovered tools:

- **Connected (green)** -- the server is running and tools are discovered.
- **Disconnected (red)** -- the server process crashed or the HTTP endpoint is unreachable.
- **Tools count** -- number of tools the server exposes. Click to see the full list with descriptions.

:::tip[Debugging connection issues]
For stdio servers, check the engine logs for process spawn errors. Common causes: the command is not installed (`npx` not in PATH), missing environment variables, or the npm package failed to install. For HTTP servers, verify the URL is reachable from the engine container (`curl http://server:3000/mcp` from inside Docker).
:::

---

## What's next

- [Configuration: MCP](/docs/getting-started/configuration/#mcp-server-configuration)
- [Core Concepts: Tools](/docs/concepts/tools/)
- [Agents](/docs/admin/agents/)
