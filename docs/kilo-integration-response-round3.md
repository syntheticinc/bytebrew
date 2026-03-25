# ByteBrew Engine — Ответ на Integration Issues Round 3

**От:** ByteBrew Engineering Team
**Дата:** 2026-03-25
**В ответ на:** Kilo IoT Integration Issues Round 3 (2026-03-25)

---

## Статус исправлений

| Issue | Severity | Статус | Commit |
|-------|----------|--------|--------|
| **#8 REOPENED** | CRITICAL | **ИСПРАВЛЕНО** | `53c934fc` |
| **#14 NEW** | CRITICAL | **ИСПРАВЛЕНО** | `53c934fc` |
| **#13** | MEDIUM | **ОТВЕТ** | — |

---

## Issue #8 REOPENED + Issue #14: MCP SSE transport — ИСПРАВЛЕНО

### Корневая причина

Две связанные проблемы в SSE transport:

1. **SSE connection умирала после connect timeout.** `Start()` создавал SSE connection с caller's context (10-секундный connect timeout). После успешного connect, timeout context отменялся → SSE `readSSE()` горутина тоже отменялась → persistent SSE stream закрывался.

2. **HTTP body vs SSE stream.** `mark3labs/mcp-go` SSE server возвращает `202 Accepted` с пустым body. Ответ JSON-RPC приходит через SSE stream как `event: message`. Engine теперь поддерживает оба варианта: сначала проверяет HTTP body, если пусто — ждёт SSE stream.

### Что исправлено

**SSE connection lifecycle:** `Start()` теперь использует `context.Background()` для SSE connection — stream живёт до вызова `Close()`, независимо от connect timeout.

**Config reload:** `POST /api/v1/config/reload` теперь переподключает MCP серверы — не нужен рестарт Engine после добавления MCP server через API.

### Правильный flow для Kilo

```
1. Engine стартует с DATABASE_URL env var
2. POST /api/v1/models — создать модель
3. POST /api/v1/mcp-servers — создать MCP server
   {"name":"kilo-platform","type":"sse","url":"http://host.docker.internal:8087/sse"}
4. POST /api/v1/agents — создать агент с mcp_servers
   {"name":"kilo-assistant","model":"qwen3-coder","system_prompt":"...","mcp_servers":["kilo-platform"]}
5. POST /api/v1/config/reload — Engine переподключает MCP серверы
   Ожидаемый лог: "MCP server connected name=kilo-platform tools=50"
6. POST /api/v1/agents/kilo-assistant/chat — MCP tools доступны
```

### Как проверить

```bash
docker pull bytebrew/engine:latest
docker compose down && docker compose up -d
```

После reload в логах должно быть:
```
INFO MCP server connected name=kilo-platform tools=50
```

При chat:
```
INFO creating ReAct agent tools_count=51  (50 MCP + ask_user)
```

---

## Issue #13: Config architecture — ОТВЕТ

### Рекомендуемый подход: Option C (env vars + API)

```yaml
# docker-compose.yml
services:
  engine:
    image: bytebrew/engine:latest
    environment:
      DATABASE_URL: "postgresql://bytebrew:bytebrew@db:5432/bytebrew?sslmode=disable"
      ADMIN_USER: admin
      ADMIN_PASSWORD: changeme
    ports:
      - "8443:8443"
```

Агенты, модели, MCP серверы настраиваются через API или Admin Dashboard. Config reload применяет изменения без рестарта.

### Для GitOps: Config Import API

```bash
# Экспорт текущей конфигурации
curl -sf http://localhost:8443/api/v1/config/export \
  -H "Authorization: Bearer $TOKEN" > config-backup.yaml

# Импорт конфигурации (поддерживает map и array формат)
curl -sf -X POST http://localhost:8443/api/v1/config/import \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/x-yaml" \
  --data-binary @agents.yaml
```

Формат `agents.yaml`:
```yaml
agents:
  kilo-assistant:
    model: qwen3-coder
    system: "You are a Kilo IoT assistant"
    tools:
      - ask_user
    mcp_servers:
      - kilo-platform

models:
  qwen3-coder:
    type: openai_compatible
    model_name: qwen/qwen3-coder-next
    base_url: https://openrouter.ai/api/v1

mcp_servers:
  - name: kilo-platform
    type: sse
    url: http://host.docker.internal:8087/sse
```

---

## Обновлённый Docker image

```bash
docker pull bytebrew/engine:latest
```

Все исправления (Issues #8, #14 + config reload MCP) включены.

---

---

## Обновление: Issue #15 — Chat empty body

### Корневая причина

`chatEnabled` flag устанавливался один раз при старте Engine. Если Engine стартует с `DATABASE_URL` env var без default LLM provider в конфиге — `chatEnabled = false`. Добавление модели через Dashboard или config import ПОСЛЕ старта не обновляло этот флаг → все chat requests возвращали 200 с пустым body.

### Исправление

Убран static `chatEnabled` check. Chat теперь работает если агенты настроены (проверяется динамически через registry). Модель можно добавить в любой момент после старта.

### Docker image обновлён

```bash
docker pull bytebrew/engine:latest
# Или конкретная версия:
docker pull bytebrew/engine:1.0.0
```

**Все Issues 8, 13, 14, 15 исправлены в текущем образе.**

---

*ByteBrew Engineering Team*
