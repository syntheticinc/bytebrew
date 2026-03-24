---
title: Model Registry
description: Browse the built-in catalog of known AI models and providers with tier classifications, capabilities, and pricing.
---

The Model Registry is a built-in, read-only catalog of known AI models and their providers. It helps you choose the right model for each agent role by providing capability data, tier classifications, and pricing information.

## Tier system

Every model in the registry is classified into one of three tiers based on its capabilities:

| Tier | Name | Description | Use case |
|------|------|-------------|----------|
| **Tier 1** | Orchestrator | Top-tier reasoning models with full agent capabilities. Best tool calling, instruction following, and multi-step planning. | Supervisor agents, complex workflows, multi-agent orchestration. |
| **Tier 2** | Sub-agent | Capable models optimized for speed and cost. Reliable tool calling for focused tasks. | Specialist agents, data retrieval, single-purpose tasks. |
| **Tier 3** | Utility | Lightweight models for simple operations. May not support tool calling. | Classification, summarization, data extraction. |

:::tip[Match tier to role]
Use Tier 1 models for your supervisor/orchestrator agents that coordinate other agents. Use Tier 2 for specialist sub-agents that perform focused tasks. Use Tier 3 for utility operations like text classification where tool calling is not needed.
:::

## Supported models

### Tier 1: Orchestrator

| Model | Provider | Context | Max Output | Tools | Vision | Input $/1M | Output $/1M |
|-------|----------|---------|------------|-------|--------|-----------|------------|
| Claude Opus 4.6 | Anthropic | 1M | 32K | Yes | Yes | $5.00 | $25.00 |
| Claude Sonnet 4.6 | Anthropic | 1M | 16K | Yes | Yes | $3.00 | $15.00 |
| GPT-5.4 | OpenAI | 272K | 16K | Yes | Yes | $2.50 | $15.00 |
| GPT-5.2 | OpenAI | 200K | 16K | Yes | Yes | $1.75 | $14.00 |
| Gemini 3.1 Pro | Google | 1M | 8K | Yes | Yes | $2.00 | $12.00 |
| Grok 4.1 | xAI | 2M | 16K | Yes | Yes | $3.00 | $15.00 |
| DeepSeek V3.2 | DeepSeek | 128K | 8K | Yes | No | $0.28 | $0.42 |
| GLM-5 | Z.ai | 200K | 8K | Yes | Yes | $1.00 | $3.20 |

### Tier 2: Sub-agent

| Model | Provider | Context | Max Output | Tools | Vision | Input $/1M | Output $/1M |
|-------|----------|---------|------------|-------|--------|-----------|------------|
| GPT-5.4 Mini | OpenAI | 128K | 16K | Yes | Yes | $0.25 | $2.00 |
| Claude Haiku 4.5 | Anthropic | 200K | 8K | Yes | Yes | $0.80 | $4.00 |
| Gemini 2.5 Flash | Google | 1M | 8K | Yes | Yes | $0.30 | $2.50 |
| GLM-4.7 | Z.ai | 128K | 4K | Yes | No | $0.60 | $2.20 |
| Mistral Medium 3 | Mistral | 128K | 8K | Yes | No | $0.40 | $2.00 |

### Tier 3: Utility

| Model | Provider | Context | Max Output | Tools | Vision | Input $/1M | Output $/1M |
|-------|----------|---------|------------|-------|--------|-----------|------------|
| GPT-5.4 Nano | OpenAI | 128K | 4K | No | No | $0.05 | $0.40 |

## API

### List models

```bash
# All models
curl http://localhost:8443/api/v1/models/registry

# Filter by provider
curl "http://localhost:8443/api/v1/models/registry?provider=anthropic"

# Filter by tier (1 = Orchestrator, 2 = Sub-agent, 3 = Utility)
curl "http://localhost:8443/api/v1/models/registry?tier=1"

# Filter by tool calling support
curl "http://localhost:8443/api/v1/models/registry?supports_tools=true"

# Combine filters
curl "http://localhost:8443/api/v1/models/registry?provider=openai&tier=2"
```

#### Response

```json
[
  {
    "id": "claude-sonnet-4-6",
    "display_name": "Claude Sonnet 4.6",
    "provider": "anthropic",
    "tier": 1,
    "context_window": 1000000,
    "max_output": 16000,
    "supports_tools": true,
    "supports_vision": true,
    "pricing_input": 3.0,
    "pricing_output": 15.0,
    "description": "Balanced reasoning model with full agent capabilities",
    "recommended_for": ["orchestrator", "sub_agent"]
  }
]
```

### List providers

```bash
curl http://localhost:8443/api/v1/models/registry/providers
```

```json
[
  {
    "id": "openai",
    "display_name": "OpenAI",
    "auth_type": "api_key",
    "website": "console.openai.com"
  },
  {
    "id": "anthropic",
    "display_name": "Anthropic",
    "auth_type": "api_key",
    "website": "console.anthropic.com"
  },
  {
    "id": "google",
    "display_name": "Google",
    "auth_type": "api_key",
    "website": "aistudio.google.com"
  },
  {
    "id": "azure_openai",
    "display_name": "Azure OpenAI",
    "auth_type": "api_key",
    "website": "portal.azure.com"
  },
  {
    "id": "openrouter",
    "display_name": "OpenRouter",
    "auth_type": "api_key",
    "website": "openrouter.ai"
  }
]
```

## Admin Dashboard

The Admin Dashboard displays tier badges next to each model in the Models page:

- **Tier 1** models show a green badge -- suitable for orchestrators.
- **Tier 2** models show a blue badge -- suitable for sub-agents.
- **Tier 3** models show a gray badge -- utility only.

When assigning a model to an agent, the dashboard shows a warning if the model tier does not match the agent's role (e.g., assigning a Tier 3 model to a supervisor agent that needs tool calling).

---

## What's next

- [Model Selection Guide](/docs/deployment/model-selection/)
- [Admin: Models](/docs/admin/models/)
- [Production Deployment](/docs/deployment/production/)
