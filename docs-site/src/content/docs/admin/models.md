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
| `azure_openai` | Azure-hosted OpenAI models. Uses deployment-based URLs and `api-key` header auth. Requires `api_version`. |
| `google` | Google Gemini models via the `generateContent` API. Uses `x-goog-api-key` header. |
| `openrouter` | OpenRouter aggregator. Access 100+ models via a single API key. Base URL is preset automatically. |
| `deepseek` | DeepSeek models (e.g. DeepSeek V3.2). OpenAI-compatible with preset base URL. |
| `mistral` | Mistral AI models (e.g. Mistral Medium 3). OpenAI-compatible with preset base URL. |
| `xai` | xAI Grok models (e.g. Grok 4.1). OpenAI-compatible with preset base URL. |
| `zai` | Z.ai GLM models (e.g. GLM-5). OpenAI-compatible with preset base URL. |

## Adding a model

Click "Add Model" and fill in the fields:

- **Display name** -- a human-readable name used in the agent configuration dropdown.
- **Provider** -- select from the supported providers above.
- **Model name** -- the model identifier as expected by the provider API (e.g., `llama3.2`, `claude-sonnet-4-20250514`). For Azure OpenAI, this is the deployment name.
- **Base URL** -- custom endpoint URL. Required for Ollama and third-party providers. Leave empty for providers with preset URLs (OpenAI, Anthropic, OpenRouter, DeepSeek, Mistral, xAI, Z.ai). For Azure OpenAI, this is your Azure resource URL (e.g., `https://my-company.openai.azure.com`).
- **API Key** -- provider API key. Not needed for Ollama. Use the `${VAR}` syntax when configuring via YAML.
- **API Version** -- (Azure OpenAI only) the API version string, e.g. `2024-10-21`.

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

# Azure OpenAI (deployment-based)
models:
  gpt4-azure:
    provider: azure_openai
    base_url: "https://my-company.openai.azure.com"
    model_name: "gpt-4o-deploy"
    api_version: "2024-10-21"
    api_key: ${AZURE_OPENAI_KEY}

# Google Gemini
models:
  gemini-pro:
    provider: google
    model_name: "gemini-3.1-pro"
    api_key: ${GOOGLE_API_KEY}

# OpenRouter (base_url is preset automatically)
models:
  openrouter-claude:
    provider: openrouter
    model_name: "anthropic/claude-sonnet-4-20250514"
    api_key: ${OPENROUTER_API_KEY}

# DeepSeek (preset base_url)
models:
  deepseek-v3:
    provider: deepseek
    model_name: "deepseek-chat"
    api_key: ${DEEPSEEK_API_KEY}

# Mistral (preset base_url)
models:
  mistral-medium:
    provider: mistral
    model_name: "mistral-medium-3"
    api_key: ${MISTRAL_API_KEY}

# xAI (preset base_url)
models:
  grok:
    provider: xai
    model_name: "grok-4.1"
    api_key: ${XAI_API_KEY}

# Z.ai / GLM (preset base_url)
models:
  glm-5:
    provider: zai
    model_name: "glm-5"
    api_key: ${ZAI_API_KEY}
```

---

## What's next

- [MCP Servers](/docs/admin/mcp-servers/)
- [Configuration: Models](/docs/getting-started/configuration/#model-configuration)
