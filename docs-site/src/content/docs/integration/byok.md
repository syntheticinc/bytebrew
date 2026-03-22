---
title: BYOK (Bring Your Own Key)
description: Let API consumers override the LLM model per-request by providing their own API key — setup, headers, security, and use cases.
---

BYOK (Bring Your Own Key) allows API consumers to override the model for a single request by passing their own provider credentials in HTTP headers. The engine uses the consumer's key for that request only and never stores it.

## What is BYOK?

Normally, each agent uses the model configured in its definition (e.g., `model: gpt-4o`). With BYOK enabled, an API consumer can override this by specifying a different provider, model, and API key in request headers. The override applies to that single request only -- subsequent requests without BYOK headers use the default model.

This is useful for:

- **Multi-tenant platforms** -- each customer uses their own LLM API key and billing.
- **Testing** -- try different models without changing engine configuration.
- **Premium tiers** -- offer customers the option to use a more powerful model by providing their own key.

## Headers

| Header | Required | Description |
|--------|----------|-------------|
| `X-Model-Provider` | Yes | Provider type: `openai`, `anthropic`, `ollama`. |
| `X-Model-API-Key` | Yes | The consumer's API key for the specified provider. |
| `X-Model-Name` | Yes | Model identifier as expected by the provider (e.g., `gpt-4o`, `claude-sonnet-4-20250514`). |

All three headers must be present for BYOK to activate. If any header is missing, the agent uses its default model.

## curl example

```bash
curl -N http://localhost:8443/api/v1/agents/my-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "Content-Type: application/json" \
  -H "X-Model-Provider: anthropic" \
  -H "X-Model-API-Key: sk-ant-customer-provided-key" \
  -H "X-Model-Name: claude-sonnet-4-20250514" \
  -d '{"message": "Hello, tell me about your capabilities"}'
```

The agent's system prompt, tools, and all other configuration remain unchanged. Only the LLM backend is overridden.

## JavaScript example

```javascript
const response = await fetch('http://localhost:8443/api/v1/agents/my-agent/chat', {
  method: 'POST',
  headers: {
    'Authorization': 'Bearer bb_your_token',
    'Content-Type': 'application/json',
    'X-Model-Provider': 'openai',
    'X-Model-API-Key': customerApiKey,
    'X-Model-Name': 'gpt-4o',
  },
  body: JSON.stringify({ message: userMessage }),
});

const reader = response.body.getReader();
// Process SSE stream...
```

## Enabling BYOK

BYOK is disabled for all providers by default. To enable it:

1. Navigate to **Admin Dashboard** -> **Settings**.
2. Under **BYOK (Bring Your Own Key)**, toggle the providers you want to allow.
3. Save. Changes take effect immediately (no restart needed).

You can enable BYOK for some providers and disable it for others. For example, enable it for OpenAI but keep Anthropic disabled.

## When to use BYOK

| Scenario | BYOK useful? | Why |
|----------|-------------|-----|
| Multi-tenant SaaS | Yes | Each customer provides their own LLM key and pays their own API costs. |
| Internal team tools | Usually no | Use a shared organizational API key configured in the engine. |
| A/B testing models | Yes | Compare gpt-4o vs claude on the same agent without changing config. |
| Premium features | Yes | Let paying customers use a better model by providing their own key. |
| Development/staging | Yes | Developers test with their personal keys without affecting shared config. |

## Security

- **Keys are never stored.** The engine uses the key for the duration of that single HTTP request and discards it immediately after.
- **Keys are never logged.** Even at `debug` log level, API keys from BYOK headers are redacted.
- **BYOK is off by default.** An operator must explicitly enable it per-provider in Settings.
- **Stateless.** The engine does not cache, persist, or transmit the key anywhere except to the specified provider's API.
- **Provider validation.** If the consumer specifies a provider that is not enabled for BYOK, the request is rejected with HTTP 403.

:::caution[Trust boundary]
BYOK headers pass through your infrastructure. Ensure your reverse proxy (Caddy, nginx) does not log request headers that may contain API keys. In Caddy, headers are not logged by default. In nginx, check your `log_format` directive.
:::

---

## What's next

- [REST API Chat](/docs/integration/rest-api/)
- [Multi-Agent Config](/docs/integration/multi-agent/)
- [Settings](/docs/admin/settings/)
