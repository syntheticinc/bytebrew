# ByteBrew Engine — Ответ на Integration Issues Round 4

**От:** ByteBrew Engineering Team
**Дата:** 2026-03-26
**В ответ на:** Kilo IoT — forward_headers не форвардятся в MCP POST

---

## Статус исправлений

| Issue | Severity | Статус | Commit |
|-------|----------|--------|--------|
| **#17** forward_headers не персистятся в API | CRITICAL | **ИСПРАВЛЕНО** | `be5d718a` |
| **#18** forward_headers не форвардятся в MCP POST | CRITICAL | **ИСПРАВЛЕНО** | `dc4a6c1d` |
| **#16** SSE Content-Length блокирует streaming | HIGH | **ИСПРАВЛЕНО** | `b33cb035` |
| **#19** Non-streaming (stream:false) пустой message | MEDIUM | **ИСПРАВЛЕНО** | `f60a5ebb` |

---

## Issue #17: forward_headers не персистятся в API — ИСПРАВЛЕНО

### Корневая причина

`forward_headers` поле существовало в DB модели (`MCPServerModel.ForwardHeaders`), но отсутствовало в API request/response structs и adapter mapping. PUT/POST `/api/v1/mcp-servers` принимали `forward_headers` без ошибки, но не сохраняли и не возвращали в response.

### Что исправлено

- Добавлено `forward_headers` в `CreateMCPServerRequest` и `MCPServerResponse`
- Adapter маппинг: Create/Update/List корректно marshal/unmarshal JSON ↔ []string
- Config import/export: `forward_headers` включается в YAML

### Как проверить

```bash
# Create MCP server with forward_headers
curl -X POST http://localhost:8443/api/v1/mcp-servers \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "kilo-platform",
    "type": "sse",
    "url": "http://host.docker.internal:8087/sse",
    "forward_headers": ["Authorization", "X-Org-Id", "X-User-Id"]
  }'

# Response contains forward_headers:
# {"id":1,"name":"kilo-platform",...,"forward_headers":["Authorization","X-Org-Id","X-User-Id"]}
```

---

## Issue #18: forward_headers не форвардятся в MCP POST — ИСПРАВЛЕНО

### Корневая причина

Две проблемы:

1. **ChatHandler.forwardHeaders was static.** Список headers для извлечения из chat request устанавливался один раз при старте Engine. После добавления MCP server через API + `config/reload`, ChatHandler НЕ обновлял свой список → не извлекал headers из chat request → RequestContext был пустой → MCP transport не имел headers для forwarding.

2. **Цепочка:** Chat request headers → `buildRequestContext()` (использовал stale list) → пустой `RequestContext` → `applyForwardHeaders()` (нечего применять) → MCP POST без headers.

### Что исправлено

- `ChatHandler.forwardHeaders` заменён на `forwardHeadersFn func() []string` — вызывается на каждый request
- `atomic.Value` хранит актуальный список, разделяемый между ChatHandler и configReloader
- `reconnectMCPServers()` обновляет atomic store после переподключения
- Config reload теперь обновляет и MCP transport, и ChatHandler

### Правильный flow для Kilo

```
1. POST /api/v1/mcp-servers — создать MCP server с forward_headers ✅
2. POST /api/v1/config/reload — Engine обновляет:
   - MCP transport (reconnect с forward_headers) ✅
   - ChatHandler forward list (atomic update) ✅ NEW
3. POST /api/v1/agents/{name}/chat с headers:
   - Authorization: Bearer $TOKEN
   - X-Org-Id: org-123
   - X-User-Id: user-456
4. Engine извлекает X-Org-Id, X-User-Id из request → RequestContext
5. MCP tool call → POST /message включает:
   - X-Org-Id: org-123
   - X-User-Id: user-456
```

### Как проверить

```bash
docker pull bytebrew/engine:latest
docker compose down && docker compose up -d

# Setup
curl -X POST /api/v1/mcp-servers -d '{"name":"kilo","type":"sse","url":"...","forward_headers":["X-Org-Id","X-User-Id"]}'
curl -X POST /api/v1/config/reload

# Chat с headers
curl -X POST /api/v1/agents/kilo-assistant/chat \
  -H "X-Org-Id: org-123" \
  -H "X-User-Id: user-456" \
  -d '{"message":"List devices"}'

# В логах MCP server должно быть:
# headers=map[..., X-Org-Id:[org-123], X-User-Id:[user-456], ...]
```

---

## Issue #16: SSE Content-Length — ИСПРАВЛЕНО

### Корневая причина

Go's `net/http` буферизирует small responses и автоматически устанавливает `Content-Length`. Для SSE streaming это приводит к тому, что весь ответ отправляется сразу, а не стримится.

### Что исправлено

Добавлен `w.WriteHeader(http.StatusOK)` перед первым `Write` — коммитит headers немедленно, заставляя Go использовать chunked transfer encoding.

---

## Issue #19: Non-streaming пустой message — ИСПРАВЛЕНО

### Корневая причина

Engine отправляет trailing ANSWER event с пустым content как "completion signal" после окончания streaming. `handleNonStreaming` обрабатывал `message` event как `message = content` (replace), поэтому пустой trailing event перезаписывал реальный ответ.

### Что исправлено

Пустой content в `message` event игнорируется — сохраняется accumulated text из `message_delta` events.

---

## Регрессионное тестирование

Проведён полный регресс (105+ TC):

| Блок | Результат |
|------|-----------|
| API CRUD + Auth + Validation | 27/27 PASS |
| Admin Dashboard | 13/13 PASS |
| Config Import/Export | 10/10 PASS |
| SSE Streaming | 8/8 PASS |
| Helm Chart | 5/5 PASS |
| Security + Registry + Tasks + Tokens | 14/14 PASS |
| Docs Site + Cloud Web | 29/29 PASS |

---

## Обновлённый Docker image

```bash
docker pull bytebrew/engine:latest
docker compose down && docker compose up -d
```

Все исправления (#16, #17, #18, #19) включены.

**Важно:** После обновления Docker image:
1. MCP servers нужно пересоздать с `forward_headers` (или config import)
2. `POST /api/v1/config/reload` — применяет изменения

---

*ByteBrew Engineering Team*
