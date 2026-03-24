---
title: Model Selection Guide
description: Choose the right LLM for your ByteBrew Engine agents — requirements, providers, recommended models, and BYOK.
---

Choosing the right model is critical for agent reliability. ByteBrew Engine works with any LLM that supports tool calling, but model quality directly affects how well agents use tools and follow instructions.

## Requirements

| Requirement | Importance | Why |
|-------------|------------|-----|
| Tool calling (function calling) | Mandatory | Agents need structured tool calls to interact with APIs, MCP servers, and built-in tools. Models without tool calling cannot use any tools. |
| Multi-turn conversation | Mandatory | Agents maintain conversation context across multiple exchanges. The model must handle system + user + assistant message sequences. |
| 32K+ context window | Recommended | Long conversations, tool results, and knowledge base passages consume context. 32K+ prevents premature context compression. |
| Instruction following | Recommended | The system prompt defines agent behavior, constraints, and output format. Better instruction following = more reliable agents. |

## Provider table

ByteBrew supports these providers out of the box:

| Provider | Type | API Key Required | Notes |
|----------|------|-----------------|-------|
| OpenAI | Cloud | Yes | Best tool calling support. Models: GPT-5.4, GPT-5.4 Mini. |
| Anthropic | Cloud | Yes | Native support. Models: Claude Opus 4.6, Claude Sonnet 4.6, Claude Haiku 4.5. |
| Azure OpenAI | Cloud | Yes | Azure-hosted OpenAI models. Deployment-based URLs, requires `api_version`. |
| Google (Gemini) | Cloud | Yes | Native Gemini API support. Models: Gemini 3.1 Pro, Gemini 2.5 Flash. |
| DeepSeek | Cloud | Yes | Cost-effective models. Preset base URL. |
| Mistral | Cloud | Yes | Mistral AI models. Preset base URL. |
| xAI | Cloud | Yes | Grok models. Preset base URL. |
| Z.ai (GLM) | Cloud | Yes | GLM models. Preset base URL. |
| Ollama | Local | No | Free, private. Requires Ollama installed on host. |
| OpenRouter | Cloud | Yes | Aggregator. Access 100+ models via single API key. Preset base URL. |
| Custom (vLLM, LiteLLM) | Self-hosted | Varies | Any OpenAI-compatible API endpoint via `openai_compatible` provider. |

See [Model Registry](/docs/deployment/model-registry/) for the full catalog of known models with capabilities, pricing, and tier classifications.

## Recommended models

### Cloud models (best quality)

| Model | Provider | Strengths | Best for |
|-------|----------|-----------|----------|
| gpt-4o | OpenAI | Excellent tool calling, fast | Supervisors, complex reasoning |
| gpt-4o-mini | OpenAI | Good quality, low cost | Specialist agents, high volume |
| claude-sonnet-4-20250514 | Anthropic | Strong reasoning, long context | Research agents, analysis |
| claude-3-haiku | Anthropic | Fast, cheap | Simple tasks, data retrieval |

### Local models (Ollama)

| Model | Parameters | VRAM | Tool calling quality |
|-------|-----------|------|---------------------|
| qwen2.5-coder:32b | 32B | 24 GB | Excellent. Best quality/hardware ratio for local deployment. |
| qwen2.5:14b | 14B | 12 GB | Good. Minimum recommended size for stable tool calling. |
| llama3.2:3b | 3B | 4 GB | Basic. Works for simple single-tool agents. Not recommended for multi-step tasks. |
| mistral:7b | 7B | 8 GB | Fair. Better instruction following than llama 7B, but tool calling can be inconsistent. |

:::tip[14B+ for stable tool calling]
Models below 14B parameters often produce malformed tool calls or call the wrong tool. For production use, start with 14B+ and test your specific use case. qwen2.5-coder 32B offers the best quality-to-hardware ratio for self-hosted deployments.
:::

## Ollama specifics

Ollama exposes two APIs: native (`/api`) and OpenAI-compatible (`/v1`). ByteBrew requires the OpenAI-compatible endpoint.

```yaml
# CORRECT: Use /v1 endpoint
models:
  local-model:
    provider: ollama
    model: qwen2.5-coder:32b
    base_url: "http://localhost:11434/v1"    # /v1 is required
    api_key: "ollama"

# WRONG: Native API does not support tool calling format
# base_url: "http://localhost:11434/api"     # Will not work
```

:::caution[Always use /v1]
The native Ollama `/api` endpoint uses a different request format that does not support tool calling in the way ByteBrew expects. Always use `http://localhost:11434/v1` (or `http://host.docker.internal:11434/v1` from Docker).
:::

### Installing and pulling models

```bash
# Install Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model (downloads once, cached locally)
ollama pull qwen2.5-coder:32b

# Verify it works
curl http://localhost:11434/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "qwen2.5-coder:32b", "messages": [{"role": "user", "content": "Hello"}]}'
```

## Adding a model via Admin Dashboard

1. Navigate to **Admin Dashboard** -> **Models**.
2. Click **Add Model**.
3. Select the provider (Ollama, OpenAI Compatible, Anthropic).
4. Fill in the model name, base URL (if needed), and API key.
5. Click **Save**. The engine validates the connection automatically.

## Adding a model via REST API

```bash
# Import a model configuration via YAML
curl -X POST http://localhost:8443/api/v1/config/import \
  -H "Authorization: Bearer bb_admin_token" \
  -H "Content-Type: application/x-yaml" \
  -d '
models:
  my-new-model:
    provider: openai
    model: gpt-4o
    api_key: ${OPENAI_API_KEY}
'

# Reload to apply
curl -X POST http://localhost:8443/api/v1/config/reload \
  -H "Authorization: Bearer bb_admin_token"
```

## Per-agent model assignment

Different agents can use different models. Use your best model for the supervisor and cheaper models for specialists:

```yaml
agents:
  supervisor:
    model: gpt-4o              # Best reasoning for coordination
  researcher:
    model: gpt-4o-mini         # Cheaper for data retrieval
  local-analyzer:
    model: qwen-local          # Free, private, no API costs

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

## BYOK: per-request model override

Bring Your Own Key lets API consumers override the model for a single request by passing headers:

```bash
curl -N http://localhost:8443/api/v1/agents/my-agent/chat \
  -H "Authorization: Bearer bb_your_token" \
  -H "X-Model-Provider: anthropic" \
  -H "X-Model-API-Key: sk-ant-customer-key" \
  -H "X-Model-Name: claude-sonnet-4-20250514" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'
```

BYOK must be enabled per-provider in **Settings**. See [BYOK integration guide](/docs/integration/byok/) for details.

---

## What's next

- [Model Registry](/docs/deployment/model-registry/)
- [Docker Deployment](/docs/deployment/docker/)
- [Production Deployment](/docs/deployment/production/)
- [BYOK Integration](/docs/integration/byok/)
