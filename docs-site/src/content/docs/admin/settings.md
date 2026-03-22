---
title: "Admin Dashboard: Settings"
description: Configure engine-wide settings including BYOK and logging levels.
---

The Settings page controls engine-wide preferences that affect all agents and API requests. Currently, it covers BYOK (Bring Your Own Key) configuration and logging levels.

## BYOK (Bring Your Own Key)

BYOK allows API consumers to override the model for a single request by passing their own API key in request headers. This is useful for multi-tenant deployments where each customer uses their own LLM account.

- BYOK is configured per-provider: you can enable it for OpenAI but disable for Anthropic.
- When enabled, the consumer passes `X-Model-Provider`, `X-Model-API-Key`, and `X-Model-Name` headers.
- The user-provided key is used for that single request only and is never stored or logged.
- If the headers are not present, the agent uses its configured model as normal.

```bash
# BYOK headers in a request (when enabled for the provider)
curl -N http://localhost:8443/api/v1/agents/my-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "X-Model-Provider: anthropic" \
  -H "X-Model-API-Key: sk-ant-customer-key" \
  -H "X-Model-Name: claude-sonnet-4-20250514" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'
```

## Logging level

Change the engine's logging verbosity at runtime without restarting:

| Level | Description |
|-------|-------------|
| `debug` | Most verbose. Logs every LLM call, tool execution, and internal state change. |
| `info` | Default. Logs agent activity, task lifecycle, and MCP connections. |
| `warn` | Only warnings and errors. Good for production with stable agents. |
| `error` | Only errors. Minimal output, useful for high-traffic deployments. |

:::tip[Debugging agents]
Set the logging level to `debug` temporarily when troubleshooting agent behavior. This shows the full LLM prompt, tool calls, and responses. Remember to set it back to `info` or `warn` for production -- debug logging generates significant output.
:::

---

## What's next

- [Config Management](/docs/admin/config-management/)
- [API Keys](/docs/admin/api-keys/)
