# ByteBrew Pivot — Acceptance Criteria

**Формат:** каждая фича описана как пошаговый сценарий который можно выполнить руками или автоматизировать. Используется для ручного тестирования, regression, и документации.

**Хранение тестов:** `bytebrew-srv/tests/acceptance/` — acceptance тесты (shell scripts / Go tests). Каждый TC-ID = файл.

---

## Phase 1: Config Engine + Agent Abstraction

### AC-1.1: YAML с несколькими агентами парсится корректно

**Предусловие:** файл `examples/developer.yaml` содержит agents: supervisor, code-agent, review, e2e-test

1. Запустить сервер: `go run ./cmd/server --config examples/developer.yaml`
2. Проверить: сервер стартовал без ошибок
3. Проверить в логах: "Loaded 4 agents: supervisor, code-agent, review, e2e-test"
4. Проверить: port file записан (`cat $APPDATA/bytebrew/server.port`)

### AC-1.2: Legacy config без `agents:` работает как раньше

1. Запустить сервер с текущим `config.yaml` (без `agents:` секции)
2. Проверить: сервер стартовал, работает как ByteBrew 1.x
3. CLI session: отправить "привет"
4. Проверить: агент ответил (supervisor mode)

### AC-1.3: Валидация конфига — дубликат имени

1. Создать YAML с двумя агентами `name: "sales"`
2. Запустить сервер
3. Проверить: ошибка при старте "duplicate agent name: sales"

### AC-1.4: Валидация конфига — невалидный can_spawn

1. Создать YAML с агентом `can_spawn: ["nonexistent"]`
2. Запустить сервер
3. Проверить: ошибка "agent 'nonexistent' referenced in can_spawn not found"

### AC-1.5: Валидация конфига — отсутствует system_prompt

1. Создать YAML с агентом без `system_prompt` и без `system_prompt_file`
2. Запустить сервер
3. Проверить: ошибка "agent 'X': system_prompt or system_prompt_file required"

### AC-1.6: system_prompt_file загружается из файла

1. Создать `prompts/sales.md` с текстом "Ты — sales консультант"
2. YAML: `system_prompt_file: "./prompts/sales.md"`
3. Запустить сервер
4. Проверить: агент использует промпт из файла

### AC-1.7: Rules и Flow.Steps inject в system prompt

1. YAML: rules: ["Не давай скидку > 15%"], flow.steps: ["Выясни потребности", "Предложи товар"]
2. Запустить, CLI session, спросить "Какие у тебя правила?"
3. Проверить: агент знает про правила и workflow

---

### AC-1.8: Per-agent model selection

1. YAML: agent A с `model: "llama-4"`, agent B с `model: "qwen-3"`
2. Запустить сервер
3. Проверить: agent A использует llama-4
4. Проверить: agent B использует qwen-3

---

## Phase 2: Dynamic Tool Registry + Agent Spawn

### AC-2.1: Agent видит только свои builtin tools

1. YAML: agent "restricted" с `builtin: [ask_user, web_search]`
2. Запустить сервер, CLI session с этим агентом
3. Попросить агента: "Прочитай файл /etc/passwd"
4. Проверить: агент НЕ может (нет read_file tool)
5. Попросить: "Выполни команду ls"
6. Проверить: агент НЕ может (нет shell_exec tool)

### AC-2.2: Agent с can_spawn видит spawn tools

1. YAML: supervisor с `can_spawn: ["researcher"]`, researcher с `lifecycle: "spawn"`
2. CLI session: "Исследуй тему AI agents"
3. Проверить: supervisor вызывает tool `spawn_researcher`
4. Проверить: researcher работает, возвращает summary
5. Проверить: supervisor получает summary и отвечает пользователю

### AC-2.3: Agent без can_spawn не может спаунить

1. YAML: agent "leaf" без `can_spawn`
2. CLI session: попросить делегировать задачу
3. Проверить: агент НЕ имеет spawn tools, работает сам

### AC-2.4: Spawn lifecycle "spawn" — agent умирает после return

1. YAML: reviewer с `lifecycle: "spawn"`
2. Supervisor спаунит reviewer
3. Проверить: reviewer выполнил задачу, вернул summary
4. Проверить: reviewer НЕ доступен после завершения (не накапливает контекст)

### AC-2.5: Spawn lifecycle "persistent" — agent живёт весь scope

1. YAML: code-agent с `lifecycle: "persistent"`, supervisor спаунит его
2. Code-agent выполняет несколько шагов (пишет файл → запускает тесты → фиксит)
3. Проверить: code-agent сохраняет контекст между шагами
4. Проверить: после return summary code-agent завершается

### AC-2.6: Blocking spawn + user interrupt

1. Supervisor спаунит code-agent (blocking)
2. Пока code-agent работает — пользователь отправляет сообщение
3. Проверить: supervisor получает interrupt, может обработать сообщение
4. Проверить: code-agent продолжает работать (не останавливается)

### AC-2.7: Session-scoped contexts выживают при cancel turn

1. Supervisor спаунит code-agent
2. Пользователь отменяет текущий turn (cancel_session)
3. Проверить: code-agent НЕ останавливается (session context жив)
4. Пользователь отправляет новое сообщение
5. Проверить: supervisor видит что code-agent ещё работает

### AC-2.8: tool_execution sequential vs parallel

1. YAML: supervisor с `tool_execution: "sequential"`, code-agent с `tool_execution: "parallel"`
2. Проверить: supervisor вызывает tools последовательно
3. Проверить: code-agent может вызывать tools параллельно

### AC-2.9: DeclarativeTool — HTTP endpoint

1. YAML: custom tool `search_products` с endpoint `GET https://httpbin.org/get`
2. CLI session: попросить agent вызвать search_products
3. Проверить: HTTP запрос отправлен, ответ получен

### AC-2.10: confirmation_required — agent спрашивает перед вызовом

1. YAML: tool `create_order` с `confirmation_required: true`
2. CLI session: попросить оформить заказ
3. Проверить: agent спрашивает "Оформить заказ? [Да/Нет]" перед вызовом tool

### AC-2.11: Per-agent max_steps override

1. YAML: agent A с `max_steps: 10`, agent B без max_steps (default 50)
2. Agent A: задача требующая > 10 steps
3. Проверить: agent A останавливается после 10 steps
4. Agent B: задача требующая 20 steps
5. Проверить: agent B работает до 50 steps

---

## Phase 3: MCP Client

### AC-3.1: MCP stdio server — tools обнаружены

1. Настроить MCP сервер (stdio) в YAML
2. Запустить сервер
3. Проверить в логах: "MCP server 'X' connected, N tools discovered"

### AC-3.2: MCP tool вызывается агентом

1. MCP сервер предоставляет tool `echo(message)`
2. CLI session: "вызови echo с текстом hello"
3. Проверить: agent вызвал MCP tool, получил ответ

### AC-3.3: MCP сервер недоступен — graceful degradation

1. YAML: MCP сервер с невалидным command
2. Запустить сервер
3. Проверить: warning в логах "MCP server 'X' unavailable, skipping"
4. Проверить: agent работает без MCP tools

### AC-3.4: MCP HTTP transport

1. Настроить MCP HTTP сервер
2. Проверить: tools обнаружены через HTTP
3. Проверить: tool call через HTTP работает

---

## Phase 4: Developer Kit — CHECKPOINT 1

### AC-4.1: Agent с kit "developer" — LSP работает

1. YAML: code-agent с `kit: "developer"`, project_dir указывает на Go проект
2. CLI session: "Открой файл main.go"
3. Проверить: agent видит LSP tools (lsp_diagnostics, get_function, etc.)
4. "Измени функцию main() — добавь fmt.Println"
5. Проверить: после edit → LSP diagnostics появляются в контексте

### AC-4.2: Agent с kit "developer" — code indexing работает

1. CLI session: "Найди функцию NewServer в проекте"
2. Проверить: agent использует search_symbols → находит функцию
3. "Покажи структуру файла server.go"
4. Проверить: agent использует get_file_structure → показывает символы

### AC-4.3: Agent без kit — engine чистый

1. YAML: sales agent без `kit:`
2. CLI session: "Привет"
3. Проверить: agent НЕ имеет LSP tools (lsp, get_function, search_code etc.)
4. Проверить: agent работает нормально с обычными tools

### AC-4.4: PostToolCall enrichment — LSP diagnostics после file_edit

1. Code-agent с kit "developer"
2. Agent пишет код с ошибкой (syntax error)
3. Проверить: после file_edit → LSP diagnostics автоматически append к результату
4. Проверить: agent видит ошибку и исправляет в следующем шаге

### AC-4.5: Full coding pipeline через YAML

1. `examples/developer.yaml`: supervisor → code-agent (persistent) → review (spawn) → e2e-test (spawn)
2. CLI session: "Добавь endpoint GET /api/health в Go проект"
3. Проверить: supervisor квалифицирует задачу
4. Проверить: supervisor создаёт tasks (manage_tasks)
5. Проверить: supervisor спаунит code-agent
6. Проверить: code-agent пишет код, запускает тесты
7. Проверить: code-agent спаунит review (свежий контекст)
8. Проверить: review возвращает OK или замечания
9. Проверить: если замечания → code-agent фиксит → повторяет
10. Проверить: code-agent возвращает summary supervisor'у
11. Проверить: supervisor финализирует

### AC-4.6: Два агента на одном engine одновременно

1. YAML: supervisor (coding) + sales-assistant
2. Запустить сервер
3. CLI session 1 (--agent supervisor): "Создай файл hello.go"
4. CLI session 2 (--agent sales): "Привет, что ты умеешь?"
5. Проверить: оба работают параллельно, не конфликтуют

### AC-4.7: Kit OnSessionEnd cleanup

1. Code-agent с kit "developer" работает (LSP started, watchdog running)
2. Сессия завершается (agent вернул результат)
3. Проверить: LSP серверы остановлены
4. Проверить: file watcher остановлен
5. Проверить: нет leaked goroutines/processes

---

## Phase 5: Job System

### ~~AC-5.1: Chat message → Job~~ — УДАЛЁН

> Chat message != Task (см. master.md AD-5). Агент сам создаёт Tasks через `manage_tasks` tool по результатам интервью с пользователем. Автоматическое создание Task из каждого chat message — неверная архитектура.

### AC-5.2: Cron trigger → Job

1. YAML: trigger cron `schedule: "*/1 * * * *"`, job: {title: "Test cron", agent: "sales"}
2. Запустить сервер
3. Ждать 1 минуту
4. Проверить: Job создан автоматически
5. Проверить: agent выполнил задачу
6. Проверить: Job status → "completed"

### AC-5.3: Webhook → Job

1. YAML: trigger webhook `path: "/api/webhooks/alert"`, job: {title: "Alert", agent: "iot-monitor"}
2. `curl -X POST http://localhost:PORT/api/webhooks/alert -d '{"message": "temperature high"}'`
3. Проверить: Job создан
4. Проверить: agent обработал с description из webhook body

### AC-5.4: Background task → ask_user недоступен

1. Cron/webhook создаёт task (background mode)
2. Agent пытается вызвать ask_user
3. Проверить: tool возвращает ошибку "ask_user not available in background mode"
4. Проверить: agent обрабатывает ошибку и продолжает без input
5. Если agent не может продолжить → task status = "failed" с причиной

### AC-5.5: Job cancel

1. Создать job через API
2. `DELETE /api/v1/jobs/{id}`
3. Проверить: job status → "cancelled", agent остановлен

---

## Phase 6: REST API + SSE — CHECKPOINT 2

### AC-6.1: Health check

1. `curl http://localhost:PORT/api/v1/health`
2. Проверить: 200 OK, body: `{"status": "ok"}`

### AC-6.2: List agents

1. `curl http://localhost:PORT/api/v1/agents`
2. Проверить: JSON array с агентами из YAML (name, description)

### AC-6.3: Chat via REST API + SSE

1. `curl -N -H "Accept: text/event-stream" -X POST http://localhost:PORT/api/v1/agents/sales/chat -d '{"message": "Привет", "user_id": "test-user"}'`
2. Проверить: SSE stream открылся
3. Проверить: получены events: `event: thinking`, `event: message`
4. Проверить: финальный event: `event: done` с status "completed"

### AC-6.4: Create job via API

1. `curl -X POST http://localhost:PORT/api/v1/jobs -d '{"title": "Test", "description": "Тестовая задача", "agent": "sales"}'`
2. Проверить: 201 Created, body содержит job_id
3. `curl http://localhost:PORT/api/v1/jobs/{id}`
4. Проверить: job details с status

### AC-6.5: Auth — bearer token

1. `curl http://localhost:PORT/api/v1/agents` без токена
2. Проверить: 401 Unauthorized
3. `curl -H "Authorization: Bearer valid-token" http://localhost:PORT/api/v1/agents`
4. Проверить: 200 OK

### AC-6.6: Config hot-reload

1. Сервер запущен с YAML содержащим agent "sales"
2. Добавить agent "support" в YAML
3. `curl -X POST http://localhost:PORT/api/v1/config/reload`
4. Проверить: 200 OK
5. `curl http://localhost:PORT/api/v1/agents`
6. Проверить: "support" появился в списке

### AC-6.7: Два use case одновременно через разные интерфейсы

1. CLI (--agent supervisor): "Создай файл test.go" → coding pipeline
2. curl REST API (agent sales): "Привет" → sales отвечает
3. Проверить: оба работают параллельно
4. Проверить: events не пересекаются между сессиями

---

## Phase 7: Knowledge (RAG)

### AC-7.1: Agent с knowledge — search работает

1. Создать папку `docs/sales/` с файлом `faq.md` (10 вопросов-ответов)
2. YAML: agent с `knowledge: "./docs/sales/"`
3. CLI session: "Какова политика возврата?"
4. Проверить: agent находит ответ из faq.md через knowledge_search

### AC-7.2: Agent без knowledge — tool недоступен

1. YAML: agent без `knowledge:`
2. Проверить: `knowledge_search` tool НЕ доступен агенту

### AC-7.3: Per-agent knowledge isolation

1. Agent A: `knowledge: "./docs/a/"`, Agent B: `knowledge: "./docs/b/"`
2. Agent A: спросить про контент из docs/b/
3. Проверить: Agent A НЕ находит (не видит чужую knowledge)

---

## Phase 8: manage_tasks Engine Integration

### AC-8.1: manage_tasks — persistent state

1. Supervisor создаёт 3 tasks через manage_tasks
2. Проверить: tasks сохранены в SQLite
3. Context rewrite/compress (длинная сессия)
4. Проверить: supervisor всё ещё видит pending tasks (из SQLite, не из контекста)

### AC-8.2: Auto-update task при spawn completion

1. Supervisor создаёт task "Endpoint /users", спаунит code-agent с task_id
2. Code-agent завершается
3. Проверить: task status автоматически → "completed"
4. Проверить: supervisor получает inject "Task 1 completed. Pending: Task 2, Task 3"

### AC-8.3: Все tasks completed → engine signals

1. Supervisor создаёт 2 tasks, спаунит agents для обоих
2. Оба agent'а завершаются
3. Проверить: engine сигналит supervisor'у "Все задачи выполнены"

### AC-8.4: manage_plan удалён

1. Agent пытается вызвать `manage_plan`
2. Проверить: tool NOT FOUND

---

## Phase 9: CLI + Mobile Abstraction

### AC-9.1: `bytebrew chat --agent sales`

1. `bytebrew chat --agent sales`
2. Проверить: подключается к серверу, chat с sales agent
3. Нет coding-specific UI elements

### AC-9.2: `bytebrew chat --agent supervisor`

1. `bytebrew chat --agent supervisor`
2. Проверить: coding pipeline работает (spawn, plans, tools)

### AC-9.3: `bytebrew agents`

1. `bytebrew agents`
2. Проверить: список всех агентов из сервера (name, description)

### AC-9.4: `bytebrew task "описание" --agent sales`

1. `bytebrew task "Подготовь outreach" --agent sales`
2. Проверить: job создан, agent работает, результат показан

### AC-9.5: Mobile — выбор агента

1. Mobile app → подключиться к серверу
2. Проверить: список агентов отображается
3. Выбрать "sales" → chat → agent отвечает
4. Выбрать "supervisor" → chat → coding pipeline
