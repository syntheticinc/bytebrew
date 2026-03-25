# ByteBrew Engine — Ответ на Integration Issues от Kilo IoT

**От:** ByteBrew Engineering Team
**Дата:** 2026-03-25
**В ответ на:** Kilo IoT Integration Issues (2026-03-24)

---

## Статус исправлений

| # | Issue | Severity | Статус |
|---|-------|----------|--------|
| 7 | Panic при ненайденной модели | CRITICAL | **ИСПРАВЛЕНО** |
| 8 | MCP tools не загружаются | CRITICAL | **ИСПРАВЛЕНО** |
| 6 | knowledge_search fails без path | MEDIUM | **ИСПРАВЛЕНО** |
| 1 | Config import map vs array | MEDIUM | **ИСПРАВЛЕНО** |
| 4 | system_file path в Docker | MEDIUM | **ЗАДОКУМЕНТИРОВАНО** |
| 5 | system vs system_prompt | LOW | **ИСПРАВЛЕНО** |
| 3 | Config mount path | LOW | **ЗАДОКУМЕНТИРОВАНО** |
| 2 | Docker image path | LOW | **ИСПРАВЛЕНО** |

## Детальные ответы

### Issue 7: Panic — ИСПРАВЛЕНО
Добавлена nil-проверка перед использованием resolved модели. Теперь возвращается HTTP error с сообщением "no model available for agent", Engine не падает.

Модель в агенте указывается по **display name** (то же имя что отображается в Admin Dashboard → Models). Например, если модель называется "qwen3-coder" в Models → agent должен иметь `model: "qwen3-coder"`.

### Issue 8: MCP tools — ИСПРАВЛЕНО
Баг был в `agent_executor.go` — при запуске агента не заполнялись зависимости MCPServers, KnowledgePath, AgentName из flow config. MCP tools теперь корректно загружаются при chat.

Подключение к MCP серверам происходит **при старте Engine** (eager, не lazy). `host.docker.internal` доступен из Docker Desktop — это правильный адрес для MCP server на хост-машине.

### Issue 6: knowledge_search — ИСПРАВЛЕНО
Если knowledge path не настроен, tool `knowledge_search` пропускается с warning в логах. Chat продолжает работать с остальными tools.

### Issue 1: Config import — ИСПРАВЛЕНО
Import API теперь принимает **оба формата**:

Map (как в документации):
```yaml
agents:
  my-agent:
    model: glm-5
    system: "You are a helpful assistant"
```

Array (legacy):
```yaml
agents:
  - name: my-agent
    model: glm-5
    system: "You are a helpful assistant"
```

### Issue 5: system vs system_prompt — ИСПРАВЛЕНО
REST API теперь принимает оба поля: `system` и `system_prompt`. Оба работают одинаково.

### Issue 2: Docker image — ИСПРАВЛЕНО
Правильный image: `bytebrew/engine:latest` (Docker Hub, public).

```bash
docker pull bytebrew/engine:latest
```

### Issue 3: Config mount path — ЗАДОКУМЕНТИРОВАНО
Engine ищет config в CWD. В Docker WORKDIR = `/app/`. Правильный mount:

```yaml
volumes:
  - ./config.yaml:/app/config.yaml:ro
```

Или через переменную окружения (если доступна): check `--config` flag.

### Issue 4: system_file path — ЗАДОКУМЕНТИРОВАНО
Relative paths в `system_file` разрешаются относительно CWD (в Docker — `/app/`).

```yaml
# В config.yaml
agents:
  kilo-assistant:
    system_file: "./prompts/kilo-assistant.txt"
```

```yaml
# Docker mount
volumes:
  - ./config.yaml:/app/config.yaml:ro
  - ./prompts:/app/prompts:ro
```

## Docker Compose пример

```yaml
services:
  engine:
    image: bytebrew/engine:latest
    ports:
      - "8443:8443"
    environment:
      DATABASE_URL: "postgresql://bytebrew:bytebrew@db:5432/bytebrew?sslmode=disable"
      ADMIN_USER: admin
      ADMIN_PASSWORD: changeme
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./prompts:/app/prompts:ro
      - ./knowledge:/app/knowledge:ro
    depends_on:
      db:
        condition: service_healthy

  db:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_USER: bytebrew
      POSTGRES_PASSWORD: bytebrew
      POSTGRES_DB: bytebrew
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U bytebrew"]
      interval: 5s
      retries: 10

volumes:
  pgdata:
```

## Обновлённый Docker image

Все исправления включены в `bytebrew/engine:latest` (обновлён 2026-03-24).

```bash
docker pull bytebrew/engine:latest
docker compose down && docker compose up -d
```

---

## Обновление от 2026-03-25

### Дополнительно исправлено после первого ответа

По результатам полного регрессионного тестирования (395 TC) обнаружены и исправлены дополнительные баги:

| Баг | Описание | Исправление |
|-----|----------|-------------|
| Model PUT/DELETE not found | Возвращал 500 вместо 404 | Теперь 404 с понятным сообщением |
| Duplicate agent name | Возвращал 500 с raw PostgreSQL ошибкой | Теперь 409 Conflict |
| Chat nonexistent agent | Возвращал 500 вместо 404 | Теперь 404 "agent not found" |
| Config import defaults | Обнулял lifecycle/max_steps при минимальном YAML | Применяет дефолты (persistent, 50, 16000) |
| Agent name validation | Принимал спецсимволы (/, <, >) | Теперь regex ^[a-z][a-z0-9-]*$ |
| Duplicate model name | Возвращал 500 | Теперь 409 Conflict |

### Полное покрытие тестами

- **395 regression test cases** (264 original + 131 new)
- Категории: TC-CFG (config), TC-TOOL (MCP/tools), TC-CRASH (crash prevention), TC-AG-EXT, TC-MD-EXT, TC-MC-EXT (CRUD edge cases), TC-AUTH-EXT (authentication), TC-SESS (sessions), TC-DOCKER (deployment), TC-SEC (security), и другие
- Все тесты пройдены, Engine стабилен

### Обновлённый Docker image

```bash
docker pull bytebrew/engine:latest
```

Все исправления включены в последнюю версию. Рекомендуем перетянуть образ и перезапустить.

---

*ByteBrew Engineering Team*
