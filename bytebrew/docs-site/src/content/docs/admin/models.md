---
title: "Admin Dashboard: Models"
description: Configure LLM providers and endpoints in the ByteBrew Engine Admin Dashboard.
---

The Models page lets you configure LLM providers and endpoints. Each model entry defines how the engine connects to an LLM backend -- you can have multiple models from different providers and assign each agent its own model.

## Supported providers

| Provider | Description |
|----------|-------------|
| `ollama` | Local model inference via Ollama. Free, private, no API key needed. Requires Ollama installed on the host. |
| `openai_compatible` | Any API that follows OpenAI chat completions format. Works with OpenAI, DeepInfra, Together, Groq, vLLM, LiteLLM. |
| `anthropic` | Native Anthropic API. Supports Claude models with automatic message format conversion. |

## Adding a model

Click "Add Model" and fill in the fields:

- **Display name** -- a human-readable name used in the agent configuration dropdown.
- **Provider** -- select from the supported providers above.
- **Model name** -- the model identifier as expected by the provider API (e.g., `llama3.2`, `claude-sonnet-4-20250514`).
- **Base URL** -- custom endpoint URL. Required for Ollama and third-party providers. Leave empty for default OpenAI/Anthropic endpoints.
- **API Key** -- provider API key. Not needed for Ollama. Use the `${VAR}` syntax when configuring via YAML.

:::note[Model validation]
After adding a model, the engine attempts a test connection to verify the endpoint is reachable and the API key is valid. If the connection fails, the model is saved but marked with a warning indicator in the list.
:::

## Configuration examples

```yaml
# Ollama (local, no API key needed)
models:
  llama-local:
    provider: ollama
    model: llama3.2
    base_url: "http://localhost:11434/v1"

# OpenAI-compatible (DeepInfra)
models:
  qwen-3-32b:
    provider: openai_compatible
    model: Qwen/Qwen3-32B
    base_url: "https://api.deepinfra.com/v1/openai"
    api_key: ${DEEPINFRA_API_KEY}

# Anthropic
models:
  claude-sonnet-4:
    provider: anthropic
    model: claude-sonnet-4-20250514
    api_key: ${ANTHROPIC_API_KEY}
```

---

## What's next

- [MCP Servers](/docs/admin/mcp-servers/)
- [Configuration: Models](/docs/getting-started/configuration/#model-configuration)
