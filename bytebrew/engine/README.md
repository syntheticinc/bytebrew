# ByteBrew Server

Go-based AI coding agent server with gRPC API, LLM integration, and agent orchestration.

## Stack

- **Go 1.24** + gRPC (bidirectional streaming)
- **PostgreSQL 16** (pgvector for vector search)
- **Eino** (v0.7, agent framework by ByteDance)
- **slog** (structured logging)
- **Viper** (configuration)

## Quick Start

### 1. PostgreSQL (pgvector)

```bash
# Docker (recommended)
docker compose up -d

# Port: 5499 (non-standard to avoid conflicts with cloud-api)
# DB: bytebrew, user: postgres, password: postgres
```

### 2. Configuration

```bash
cp config.yaml.example config.yaml
```

Minimal changes in `config.yaml`:

| Parameter | Description | Required |
|-----------|-------------|:---:|
| `llm.default_provider` | LLM provider (`ollama`, `openrouter`, `anthropic`) | yes |
| `llm.<provider>.api_key` | Provider API key | yes (except ollama) |
| `license.public_key_hex` | Ed25519 public key (hex) — from cloud-api keygen | for licensing |
| `provider.mode` | Provider mode (`byok`, `proxy`, `auto`) | no (default: `byok`) |
| `provider.cloud_api_url` | Cloud API URL (for proxy mode) | for proxy |

### 3. Run

```bash
go run ./cmd/server
```

Server listens on gRPC at `localhost:60401`.

## LLM Providers

### BYOK (Bring Your Own Key)

Direct connection to LLM using your own API key:

| Provider | Config | Models |
|----------|--------|--------|
| **Ollama** | `llm.ollama.base_url` | Any local model |
| **OpenRouter** | `llm.openrouter.api_key` | 100+ models |
| **Anthropic** | `llm.anthropic.api_key` | Claude 3.5/4 |

### Proxy Mode

Via Cloud API gateway (no key needed — uses platform key):

```yaml
provider:
  mode: proxy
  cloud_api_url: http://localhost:60402
```

### Smart Routing

Automatic model selection by agent role:
- Supervisor + Coder → powerful model (GLM-5)
- Reviewer + Tester → fast model (GLM-4.7)

Switch provider/model in CLI: `/provider`, `/model`.

## Licensing

### Offline (file)

```yaml
license:
  public_key_hex: "7d286e3f..."  # Ed25519 public key
  license_path: "~/.bytebrew/license.jwt"
```

Server validates the JWT signature offline, without contacting Cloud API.

### Relay (on-premises)

```yaml
relay:
  address: "http://relay.internal:8080"
```

For corporate installations — validation through a relay server.

## Architecture

Clean Architecture with dependency injection:

```
bytebrew-srv/
├── cmd/
│   ├── server/         # Entry point
│   └── testserver/     # Test server (mock LLM)
├── internal/
│   ├── domain/         # Entities (License, Agent, Tool)
│   ├── usecase/        # Business logic
│   ├── service/        # Services (agent pool, license)
│   ├── delivery/grpc/  # gRPC handlers + interceptors
│   └── infrastructure/
│       ├── llm/        # LLM clients (auto, proxy, selector)
│       ├── license/    # License validator
│       ├── tools/      # Agent tools
│       └── agents/     # REACT agent, callbacks
├── pkg/config/         # Configuration
├── api/proto/          # Protocol Buffers
└── prompts.yaml        # System prompts
```

## Tests

```bash
# All unit tests
go test ./...

# Prompt regression tests (require LLM key)
go test -tags prompt -v -timeout 300s ./tests/prompt_regression/...
```

## Logs

When `logging.output: file`, logs are written to `logs/`:

| File | Description |
|------|-------------|
| `bytebrew-srv.log` | Main server log |
| `<session>/step_N_context.json` | Full LLM context at each step |
| `<session>/context_summary.txt` | Session statistics |

`logging.clear_on_startup: true` — clears logs on startup (useful for debugging).
