# ByteBrew — Требования к платформе

**Дата:** 15 марта 2026
**Позиционирование:** Добавь AI-агента в свой продукт
**Модель:** Free community + paid Enterprise (QuestDB / PostHog / MySQL)

---

## Оглавление

0. [Как компания интегрирует ByteBrew](#0-как-компания-интегрирует-bytebrew)
1. [Архитектура](#1-архитектура)
2. [Agent Model](#2-agent-model)
3. [Tools (MCP)](#3-tools-mcp)
4. [Knowledge (RAG)](#4-knowledge-rag)
5. [Kits (Engine Extensions)](#5-kits-engine-extensions)
6. [Agent Configuration](#6-agent-configuration)
7. [Интерфейсы](#7-интерфейсы)
8. [Job System](#8-job-system)
9. [Security & Compliance](#9-security--compliance)
10. [Observability](#10-observability)
11. [Deployment & Distribution](#11-deployment--distribution)
12. [Архитектура продуктов: Engine vs ByteBrew Code](#12-архитектура-продуктов-engine-vs-bytebrew-code)
13. [Тарификация и лицензирование](#13-тарификация-и-лицензирование)
14. [Kits и примеры конфигураций](#14-kits-и-примеры-конфигураций)
15. [Milestones](#15-milestones)

---

## 0. Как компания интегрирует ByteBrew

### 4 точки интеграции

```
Продукт компании (shop, IoT, SaaS, dev tool)
    │
    ├── 1. Tools (MCP / declarative)     агент ДЕЛАЕТ что-то в продукте
    │      "search_products", "create_order", "get_telemetry"
    │
    ├── 2. Knowledge (папка с документами) агент ЗНАЕТ о бизнесе
    │      FAQ, каталог, спецификации, политики
    │
    ├── 3. Triggers (webhooks + cron)      продукт ЗАПУСКАЕТ агента
    │      "температура > 80°C", "каждое утро в 8:00"
    │
    └── 4. Embed (REST API + WS)           пользователь ОБЩАЕТСЯ с агентом
           POST /api/v1/agents/sales/chat → SSE stream
```

### Пошаговая интеграция

**Шаг 1: Установить ByteBrew**
```bash
docker-compose up -d
```

**Шаг 2: Описать агента в YAML**
```yaml
agents:
  - name: "sales-assistant"
    system_prompt: "Ты — консультант магазина..."
    tools:
      custom:
        - name: "search_products"
          endpoint: "GET https://api.shop.com/products"
          params: { query: "string", max_price: "number" }
        - name: "create_order"
          endpoint: "POST https://api.shop.com/orders"
          confirmation_required: true
      builtin: [ask_user, knowledge_search]
    knowledge: "./docs/sales/"
    rules:
      - "Не давай скидку больше 15%"
```

**Шаг 3: Подключить свой API** (одним из способов)
- **Declarative tools** (YAML) — для простых REST API, без кода
- **MCP сервер** — для сложных интеграций, компания пишет MCP-обёртку

**Шаг 4: Положить документы** в папку
```
docs/sales/
├── faq.md
├── return-policy.md
└── product-catalog.md
```

**Шаг 5: Настроить triggers** (если нужна автоматизация)
```yaml
triggers:
  - type: "webhook"
    path: "/api/alerts"
    job: { title: "Алерт", agent: "sales-assistant" }
  - type: "cron"
    schedule: "0 8 * * *"
    job: { title: "Утренний отчёт", agent: "sales-assistant" }
```

**Шаг 6: Дать пользователям доступ** (одним из способов)
- REST API + SSE: `POST /api/v1/agents/sales/chat`
- WebSocket: `ws://bytebrew:8443/ws?agent=sales`
- CLI: `bytebrew chat --agent sales`
- Mobile: QR pairing → выбор агента → chat

### Что компания НЕ делает

- Не пишет код agent engine
- Не разбирается в LLM, промптах, ReAct loop
- Не строит infrastructure (Docker Compose поднимает всё)
- Не платит per-token (свои модели, inference = $0)

**Время интеграции:** часы-дни (если у продукта есть API). Не месяцы.

---

## 1. Архитектура

### Принцип

ByteBrew — autonomous AI agent engine. Инфраструктура как MySQL для данных: ставится куда угодно, используется под что угодно. Стартап встраивает в свой продукт, enterprise автоматизирует процессы, агентство строит решения для клиентов, разработчик экспериментирует.

```
┌─────────────────────────────────────────────────────┐
│           Клиенты (CLI, Mobile, REST API, WS)        │
└──────────┬───────────────┬───────────────┬──────────┘
           │               │               │
           ▼               ▼               ▼
    ┌─────────────────────────────────────────────┐
    │              ByteBrew Engine                  │
    │                                             │
    │  ┌─────────┐  ┌─────────┐  ┌────────────┐  │
    │  │  Agent  │  │  Job    │  │  Event     │  │
    │  │  Engine │  │  Queue  │  │  Store     │  │
    │  └────┬────┘  └─────────┘  └────────────┘  │
    │       │                                     │
    │  ┌────┼──────────────────────────────────┐  │
    │  │    ▼                                  │  │
    │  │  ┌──────┐  ┌──────┐  ┌──────┐        │  │
    │  │  │Tools │  │Knowl.│  │ Kits │        │  │
    │  │  │(MCP) │  │(RAG) │  │      │        │  │
    │  │  └──┬───┘  └──┬───┘  └──┬───┘        │  │
    │  │     │         │         │             │  │
    │  └─────┼─────────┼─────────┼─────────────┘  │
    │        ▼         ▼         ▼                │
    │   Внешние API   Документы   Engine-level    │
    │   (MCP)         (папки)     extensions      │
    │                             (LSP, indexing)  │
    └─────────────────────────────────────────────┘
```

### Компоненты

| Компонент | Ответственность | Статус |
|-----------|----------------|:------:|
| **Agent Engine** | ReAct loop, tool execution, context management, sub-agent spawn | **Есть** (Eino) |
| **Agent Spawn** | Гибкие связи через `can_spawn`, lifecycle management | **Есть** (нужен рефакторинг: hardcoded → YAML-driven) |
| **Job Queue** | Приём jobs от всех источников, приоритизация, распределение | Нужно |
| **Event Store** | Персистентность событий, guaranteed delivery | **Есть** (SQLite) |
| **Model Gateway** | Абстракция над LLM (OpenAI API, Ollama, vLLM) | **Есть** |
| **Tool Registry** | Built-in + MCP + kit-provided tools | Частично (built-in есть, MCP нет) |
| **Knowledge Store** | RAG: документы из папок, vector search | Частично (code indexing есть) |
| **Kits** | Engine-level extensions (LSP, code indexing) — Go-код, compile-time | Частично (LSP + indexing есть, нужен refactor в kit) |
| **API Layer** | REST API + SSE, WebSocket, CLI, Mobile | Частично (WS + Mobile есть, REST API нужен) |
| **Config Engine** | YAML parsing, validation, hot-reload | Нужно |
| **Transport** | WS (direct) + Bridge (NAT traversal) + E2E encryption | **Есть** |

---

## 2. Agent Model

### Принцип: агенты как nodes, связи как конфигурация

Нет жёсткой иерархии "supervisor → sub-agent". Пользователь сам определяет:
- Сколько агентов
- Кто кого может спаунить
- Какой lifecycle у каждого
- Как они взаимодействуют

```
Agent:
├── prompt:          КТО ты
├── flow:            КАК ты работаешь
├── tools:           ЧТО ты можешь делать (LLM решает когда)
├── kit:             доменные расширения (LSP, code indexing — engine вызывает автоматически)
├── knowledge:       ЧТО ты знаешь (RAG)
├── rules:           ограничения
├── can_spawn:       КАКИХ агентов ты можешь создавать
├── lifecycle:       persistent | spawn
└── tool_execution:  sequential | parallel
```

### Паттерны использования (все конфигурируемые)

**Flat:** несколько независимых агентов, каждый принимает задачи напрямую
```yaml
agents:
  - name: "sales"
    # принимает задачи от пользователей через embed
  - name: "support"
    # принимает тикеты через webhook
  - name: "analytics"
    # работает по cron
```

**Координатор + исполнители:** один агент делегирует другим
```yaml
agents:
  - name: "coordinator"
    can_spawn: ["researcher", "writer"]
  - name: "researcher"
    lifecycle: "spawn"
  - name: "writer"
    lifecycle: "spawn"
```

**Цепочка:** агент спаунит следующего
```yaml
agents:
  - name: "analyst"
    can_spawn: ["report-writer"]
  - name: "report-writer"
    lifecycle: "spawn"
    can_spawn: ["reviewer"]
  - name: "reviewer"
    lifecycle: "spawn"
```

**Глубокая иерархия** (как coding pipeline):
```yaml
agents:
  - name: "supervisor"
    can_spawn: ["code-agent"]
  - name: "code-agent"
    lifecycle: "persistent"      # живёт весь цикл задачи
    can_spawn: ["review", "e2e-test"]
  - name: "review"
    lifecycle: "spawn"           # свежий контекст каждый раз
  - name: "e2e-test"
    lifecycle: "spawn"
```

### Lifecycle агентов

| Тип | Поведение | Когда использовать |
|-----|-----------|-------------------|
| **persistent** | Живёт пока жив его scope. Накапливает контекст | Основной исполнитель (code agent, sales agent) |
| **spawn** | Создан → выполнил → вернул summary → умер. Чистый контекст | Независимая проверка (review, test), одноразовые задачи |

**Scope persistent агента** зависит от того, кто его создал:

| Кто создал | Scope | Пример |
|-----------|-------|--------|
| Другой агент (spawn) | Пока parent не получит summary | Code-agent живёт пока Supervisor не примет результат |
| Task system (cron/webhook/API) | Пока задача не завершена | IoT-agent обработал alarm → задача закрыта → agent умер |
| Embed (user chat) | Пока сессия не закончилась | Sales-agent живёт пока покупатель в чате |

### Горизонтальное взаимодействие между агентами

Если агенту нужны данные другого домена — дай ему соответствующий tool. Sales нужен склад → tool `check_stock`. Support нужен billing → tool `get_invoices`. Агенты не общаются напрямую — каждый имеет свои tools.

### Как работает spawn: LLM решает, конфиг ограничивает

`can_spawn` — это **tools** которые engine автоматически создаёт для агента. Если у агента `can_spawn: ["researcher", "writer"]`, он получает два tool'а: `spawn_researcher(task)` и `spawn_writer(task)`. LLM сам решает когда и зачем вызвать — как решает когда вызвать `web_search` или `create_order`.

**Пример: coordinator получает задачу "Напиши статью про квантовые компьютеры"**

```
coordinator (ReAct loop):
  Думает: "Нужно исследовать тему, потом написать статью."

  → вызывает spawn_researcher("Найди ключевые факты про квантовые компьютеры")
      researcher работает (web_search, web_fetch)
      ← возвращает summary: "5 ключевых фактов: ..."

  Думает: "Факты есть. Теперь нужно написать."

  → вызывает spawn_writer("Напиши статью на основе фактов: ...")
      writer работает
      ← возвращает summary: "Статья готова: ..."

  → отвечает пользователю: "Вот статья."
```

**Конфиг определяет ЧТО МОЖНО** (какие агенты, какие tools, кто кого может спаунить).
**LLM решает КОГДА и ЗАЧЕМ** — через reasoning в ReAct loop.

### Цепочки спаунов

Spawn-агент тоже может спаунить (если у него есть `can_spawn`):

```
coordinator → spawn researcher("исследуй тему")
                  researcher → spawn fact-checker("проверь факт X")
                                   ← fact-checker: "подтверждён"
                  ← researcher: "проверенные факты: ..."
             → spawn writer("напиши статью")
                  ← writer: "готово"
             → spawn reviewer("проверь статью")
                  ← reviewer: "2 замечания"
             → spawn writer("исправь: ...")
                  ← writer: "исправлено"
← coordinator: "Статья готова."
```

### Взаимодействие между агентами

- Спаунящий агент передаёт **задачу** (prompt + контекст)
- Spawn-агент возвращает **summary** (не весь контекст — изоляция)
- `can_spawn` = whitelist (безопасность: агент не может создать кого угодно)
- Агент без `can_spawn` — leaf node, работает сам
- Если агент не может решить задачу — эскалация к спаунившему или к пользователю

### Требования

- [ ] Агенты определяются в YAML, связи через `can_spawn`
- [ ] Каждый агент: свой prompt, tools, kit, knowledge, rules, lifecycle
- [ ] `persistent` lifecycle: накапливает контекст
- [ ] `spawn` lifecycle: чистый контекст, возвращает summary
- [ ] `can_spawn`: whitelist агентов которых можно создать
- [ ] `tool_execution`: `sequential` (безопасно, без конфликтов) или `parallel` (быстрее)
- [ ] Произвольная глубина спауна (конфигурируемая, default limit для безопасности)
- [ ] Эскалация: spawn → parent → пользователь
- [ ] Агент без `can_spawn` — leaf node, работает сам

---

## 3. Tools (MCP)

### Принцип

Tools = что агент **может делать**. Подключаются через MCP (Model Context Protocol) — стандарт поддерживаемый Claude, Cursor, OpenAI.

### Два способа подключения

**Способ 1: MCP сервер** (для сложных интеграций)
```yaml
tools:
  mcp_servers:
    shop-api:
      type: "http"
      url: "http://shop-api:3000/mcp"
      # MCP сервер предоставляет tools автоматически
```

Компания пишет MCP-обёртку над своим API (или использует готовые MCP серверы).

**Способ 2: Декларативные tools** (quick start, без кода)
```yaml
tools:
  custom:
    - name: "search_products"
      description: "Поиск товаров в каталоге"
      endpoint: "GET https://api.shop.com/products"
      params:
        query: { type: "string", required: true }
        category: { type: "string" }
        max_price: { type: "number" }
      auth:
        type: "bearer"
        token_env: "SHOP_API_TOKEN"

    - name: "create_order"
      description: "Оформить заказ"
      endpoint: "POST https://api.shop.com/orders"
      body: { user_id: "string", items: "array" }
      confirmation_required: true    # агент спросит пользователя перед вызовом
```

ByteBrew автоматически превращает декларативное описание в MCP-compatible tool.

### Built-in tools

| Tool | Описание | Универсальный? |
|------|----------|:--------------:|
| `web_search` | Поиск в интернете | Да |
| `web_fetch` | Загрузка веб-страниц | Да |
| `ask_user` | Запрос к пользователю (блокирующий) | Да |
| `knowledge_search` | Поиск по knowledge base | Да |
| `manage_tasks` | Создание / отслеживание задач с опциональной вложенностью (persistent state) | Да |
| `file_read/write/edit` | Работа с файлами | Конфигурируется per-agent |
| `shell_exec` | Выполнение команд | Конфигурируется per-agent |
| `git_*` | Git операции | Конфигурируется per-agent |

### manage_tasks: persistent state + engine lifecycle

`manage_tasks` — не просто CRUD tool. Это **persistent state** (SQLite) с **engine-level lifecycle**.

**Зачем:** при длинных задачах контекст агента сжимается (rewrite/compress). Без external state агент "забудет" какие sub-tasks создал и какие уже выполнены. manage_tasks = внешняя память которая переживает потерю контекста.

**Как работает:**

```
1. Supervisor вызывает manage_tasks("create", tasks=[...])
   → tasks сохраняются в SQLite
   → engine начинает следить за lifecycle

2. Supervisor делегирует sub-task → spawn code-agent
   → code-agent завершился
   → engine автоматически:
     a) обновляет task status (completed/failed)
     b) inject'ит в контекст supervisor'а: "Task 1 completed. Pending: Task 2, Task 3"

3. Supervisor видит pending tasks → делегирует следующий

4. Все tasks completed → engine сигналит supervisor'у: "Все задачи выполнены"

5. Timeout без прогресса → эскалация
```

**Активация через конфиг:** если `manage_tasks` есть в списке tools агента — engine активирует lifecycle tracking при старте. Не ждёт первый вызов. Sales agent без manage_tasks в конфиге — engine не вмешивается.

**Один tool, subtasks опциональны:**
- Coding supervisor: tasks с subtasks (Task "REST API" → Subtask "/users", Subtask "/orders", Subtask "тесты")
- IoT coordinator: flat tasks без вложенности (Task "Зона А", Task "Зона Б")
- Research agent: tasks с subtasks или без — агент сам решает через промпт

**Usage определяется промптом агента**, не tool'ом. Tool один, промпт разный.

### Требования

- [ ] MCP: stdio, HTTP, SSE транспорты
- [ ] Декларативные tools (YAML → MCP tool)
- [ ] Per-agent tool access (агент видит только свои tools)
- [ ] `confirmation_required` — агент спрашивает перед опасными действиями
- [ ] Tool call timeout (конфигурируемый)
- [ ] Tool call logging (для audit)
- [ ] Auto-lifecycle: start при первом вызове, stop при неактивности

---

## 4. Knowledge (RAG)

### Принцип

Knowledge = что агент **знает**. Положил файлы в папку → агент их знает.

```yaml
agents:
  - name: "sales"
    knowledge: "./docs/sales/"       # всё что в папке

  - name: "support"
    knowledge: "./docs/support/"     # своя папка — своё знание
```

### Встроенный RAG

Engine сканирует папку → чанкает документы → строит embeddings → vector search. Агент автоматически получает tool `knowledge_search`.

Где работает хорошо: FAQ, KB-статьи, каталог товаров, спецификации устройств — структурированные короткие документы (~80% use cases).

Где работает плохо: длинные юридические документы (чанк теряет контекст), таблицы/графики (чанкинг ломает структуру).

### Внешний search через MCP

Для компаний с существующей инфраструктурой поиска (Elasticsearch, Algolia) — подключают свой search как MCP tool. Встроенный RAG не нужен.

### Требования

- [ ] Knowledge = путь к папке в YAML
- [ ] Документы: markdown, PDF, HTML, txt
- [ ] Chunking → embedding → vector search
- [ ] Per-agent knowledge (агент видит только свою папку)
- [ ] Model-agnostic embeddings (OpenAI, Ollama, local)
- [ ] Внешний search — через MCP (уже покрыто секцией Tools)

---

## 5. Kits (Engine Extensions)

### Принцип

Kits — Go-модули для доменов где нужно больше чем tools + prompt. Kit расширяет engine: добавляет session-level state, автоматические hooks, дополнительные tools.

```
Tool (MCP):   LLM решает вызвать → результат
Kit:          engine вызывает автоматически на события + предоставляет доп. tools + хранит state per session
```

Engine universal — не знает про LSP, coding или IoT. Kit подключается через конфиг, engine вызывает его через Go interface.

### Архитектура

```
bytebrew/
├── engine/              # universal
└── kits/
    └── developer/       # LSP, code indexing, watchdog, symbol tools
        ├── kit.go       # implements Kit interface
        ├── lsp.go       # LSP серверы, auto-install, diagnostics
        └── index.go     # code indexing, watchdog, embeddings
```

```go
type Kit interface {
    Name() string
    OnSessionStart(ctx, session) error    // создать LSP, запустить watchdog, проиндексировать
    OnSessionEnd(ctx, session) error      // cleanup
    Tools() []Tool                        // search_symbols, get_file_structure, lsp_*
    PostToolCall(ctx, toolName, result) *Enrichment  // LSP diagnostics после file_edit
}
```

### Конфигурация

```yaml
agents:
  - name: "code-agent"
    kit: "developer"     # engine загружает kit, добавляет его tools и hooks

  - name: "sales-agent"
    # нет kit — engine чистый, только tools + prompt
```

`kit: "developer"` → engine автоматически:
- При session start: запускает LSP серверы, индексирует код, запускает watchdog
- Добавляет tools: `search_symbols`, `get_file_structure`, `get_function`, `get_class`, `lsp_diagnostics`
- После `file_edit` / `file_write`: автоматически запускает LSP diagnostics → inject ошибки в контекст

### Чем kit отличается от MCP tool

| | MCP Tool | Kit |
|--|---------|-----|
| Кто вызывает | LLM (reasoning) | Engine (автоматически) + LLM (kit's tools) |
| Session state | Нет | Да (per-session: индексы, LSP серверы, watchdog) |
| Hooks | Нет | Да (post_tool_call → auto-enrichment) |
| Доп. tools | Нет | Да (search_symbols, lsp_diagnostics) |
| Пример | `search_products()` | Developer kit: LSP + indexing + symbol tools |

### Требования

- [ ] Kit Go interface (Name, OnSessionStart, OnSessionEnd, Tools, PostToolCall)
- [ ] Compile-time loading (kit = Go-код в репозитории, компилируется вместе с engine)
- [ ] Активация через YAML: `kit: "developer"`
- [ ] Kit предоставляет tools — engine автоматически добавляет их агенту
- [ ] Kit hooks (PostToolCall) — engine вызывает автоматически, результат inject в контекст
- [ ] Per-session state (kit создаёт/чистит при session start/end)
- [ ] Kits в отдельной директории (`kits/`), изолированы от engine

---

## 6. Agent Configuration

### Полный пример: E-commerce Sales Agent

```yaml
# bytebrew.yaml

engine:
  host: "0.0.0.0"
  port: 8443
  data_dir: "/var/lib/bytebrew"

models:
  providers:
    - name: "llama-4"
      type: "openai-compatible"
      base_url: "http://gpu:8000/v1"
      model: "meta-llama/Llama-4-70B"

agents:
  - name: "sales-assistant"
    model: "llama-4"
    tool_execution: "sequential"

    system_prompt: |
      Ты — AI-консультант интернет-магазина TechShop.
      Помогаешь покупателям найти и купить подходящий товар.
      Тон: дружелюбный, экспертный. Не навязывай.

    flow:
      steps:
        - "Выясни что ищет клиент (для чего, бюджет, предпочтения)"
        - "Найди подходящие товары через search_products"
        - "Проверь наличие через check_stock"
        - "Предложи 2-3 варианта с объяснением"
        - "Если клиент готов — оформи заказ через create_order"

      escalation:
        triggers: ["жалоба", "возврат", "менеджер"]
        action: "transfer_to_human"
        webhook: "https://api.shop.com/escalation"

    tools:
      mcp_servers:
        shop-api:
          type: "http"
          url: "http://shop-api:3000/mcp"
      builtin: [knowledge_search, ask_user]

    knowledge: "./docs/sales/"

    rules:
      - "Не давай скидку больше 15%"
      - "Всегда проверяй наличие перед предложением"
      - "Персональные данные не запрашивай кроме email"

    confirm_before:
      - "create_order"
      - "apply_promo"

    can_spawn: ["product-researcher"]

  - name: "product-researcher"
    lifecycle: "spawn"
    model: "llama-4"
    system_prompt: |
      Исследуй товар детально: характеристики, отзывы,
      сравнение с аналогами. Верни краткое резюме.
    tools:
      builtin: [knowledge_search, web_search]
```

### Полный пример: IoT Assistant

```yaml
agents:
  - name: "iot-assistant"
    model: "qwen-3"
    tool_execution: "sequential"

    system_prompt: |
      Ты — AI-ассистент IoT платформы SmartFactory.
      Помогаешь операторам управлять устройствами и автоматизацией.

    tools:
      mcp_servers:
        factory-api:
          type: "http"
          url: "http://factory-api:4000/mcp"
      builtin: [knowledge_search, ask_user, manage_tasks]

    knowledge: "./docs/devices/"

    triggers:
      - type: "webhook"
        path: "/api/alerts"
        job:
          title: "Алерт"
          agent: "iot-assistant"
      - type: "cron"
        schedule: "0 8 * * *"
        job:
          title: "Утренний отчёт по зонам"
          description: "Проверить все зоны, собрать телеметрию, сформировать отчёт"
          agent: "iot-assistant"

    can_spawn: ["report-generator"]

    escalation:
      triggers: ["критический алерт", "устройство не отвечает"]
      webhook: "https://factory-api:4000/escalation"

  - name: "report-generator"
    lifecycle: "spawn"
    model: "qwen-3"
    system_prompt: "Сформируй структурированный отчёт по данным."
    tools:
      mcp_servers:
        factory-api:
          type: "http"
          url: "http://factory-api:4000/mcp"
```

### YAML: что покрывает, а что нет

**YAML хватит для ~70% use cases:** agent prompt, tools, knowledge sources, triggers, rules, sub-agents, escalation. Это основные сценарии (IoT мониторинг, sales assistant, support, onboarding).

**Для оставшихся 30% нужны расширения:** custom MCP servers (код), kits (Go-код для engine-level интеграций), сложная бизнес-логика в tool handlers. Это нормально — MySQL тоже конфигурируется, но stored procedures пишутся на SQL.

**Путь:** YAML — точка входа (быстрый старт). MCP/kits — для продвинутых интеграций.

### Требования

- [ ] YAML-конфигурация: agents, tools, knowledge, kit, can_spawn, triggers, rules
- [ ] Валидация конфига при старте (fail fast, понятные ошибки)
- [ ] Hot-reload без рестарта (SIGHUP или API)
- [ ] Несколько агентов на одном инстансе
- [ ] Per-agent model selection
- [ ] REST API для CRUD агентов (программная конфигурация)

---

## 7. Интерфейсы

### Quick Start: CLI + Mobile

```bash
# CLI — работает с любым агентом
bytebrew chat --agent sales
bytebrew chat --agent iot-monitor

# Mobile — QR pairing, уже работает
```

CLI уже есть, нужно абстрагировать от coding (generic chat с любым агентом).
Mobile уже есть, нужно добавить выбор агента.

### Embed: REST API + WebSocket

Компания встраивает ByteBrew в свой продукт через API. UI строит сама.

**REST API + SSE streaming:**
```
POST /api/v1/agents/{agent_name}/chat
{
  "user_id": "customer-123",
  "message": "Ищу ноутбук для видеомонтажа",
  "session_id": "optional-session-for-continuity"
}

→ Stream of events (SSE):
  { "type": "thinking", "content": "..." }
  { "type": "tool_call", "tool": "search_products", "args": {...} }
  { "type": "message", "content": "Вот что я нашёл..." }
```

**WebSocket (real-time):**
```
ws://bytebrew.company.com/ws?agent=sales-assistant&user=customer-123
```

### Требования

- [ ] CLI: `bytebrew chat --agent NAME` (generic, любой агент)
- [ ] Mobile: выбор агента, generic chat
- [ ] REST API с SSE streaming
- [ ] WebSocket API (уже есть, нужно абстрагировать)
- [ ] User context passing (user_id, metadata → агент знает кто пишет)
- [ ] Session continuity (history within session)
- [ ] Multi-tenant: один ByteBrew instance → несколько API endpoints

---

## 8. Job System

### Терминология

- **Job** — внешняя задача от пользователя/trigger'а. Входная точка в engine. Создаётся из chat/cron/webhook/API.
- **Task** (manage_tasks) — внутренняя декомпозиция агентом. Агент разбивает job на tasks для отслеживания прогресса.

```
Job (внешнее)                    Tasks (внутреннее)
"Добавь REST API"    →  Agent →  Task 1: "endpoint /users"
                                 Task 2: "endpoint /orders"
                                 Task 3: "тесты"
```

### Принцип: Job — единая точка входа

Любой источник создаёт **job**. Агент всегда работает с job, не с raw prompt.

```
Chat:     пользователь написал → job
Cron:     trigger сработал → job (из конфига)
Webhook:  event пришёл → job (из webhook body + конфиг)
API:      программно → job
```

### Agent flow (всегда одинаковый)

```
1. Agent получил job
2. Нужно уточнить? → ask_user (chat) или работает с тем что есть (background)
3. Декомпозиция → manage_tasks (внутренние tasks для отслеживания)
4. Автономная работа (spawn sub-agents, выполнение)
5. Все tasks completed → результат job
```

### Triggers создают jobs

```yaml
triggers:
  - type: "cron"
    schedule: "0 8 * * *"
    job:
      title: "Утренний отчёт по зонам"
      description: "Проверить все зоны, собрать телеметрию, сформировать отчёт"
      agent: "iot-monitor"

  - type: "webhook"
    path: "/api/critical-alert"
    job:
      title: "Критический алерт"
      agent: "iot-monitor"
      # description берётся из webhook body
```

### Два режима: interactive vs background

| Источник | Режим | ask_user |
|----------|:-----:|----------|
| **Chat (CLI/mobile/API)** | Interactive | Агент может спрашивать, пользователь отвечает |
| **Cron** | Background | Агент работает с тем что есть в job description |
| **Webhook** | Background | Агент работает с тем что пришло в webhook body |
| **API** | Конфигурируется | `mode: "interactive"` → needs_input + notify |

Background job + агенту нужен ввод → job переходит в `needs_input` → уведомление (mobile push, webhook callback) → пользователь отвечает → агент продолжает.

### Требования

- [ ] Job — единая точка входа для всех источников
- [ ] Triggers создают jobs (не шлют raw prompts)
- [ ] Interactive mode: ask_user работает, пользователь в диалоге
- [ ] Background mode: агент работает автономно, needs_input при необходимости
- [ ] Job status: created / running / completed / failed / needs_input
- [ ] Notification при needs_input (mobile push, webhook callback)
- [ ] Job queue с приоритизацией
- [ ] Параллельные jobs (до N per agent)
- [ ] Job persistence (SQLite для dev, PostgreSQL для production — через Storage interface)
- [ ] Job creation via API

---

## 9. Security & Compliance

### Уже реализовано

| Свойство | Детали |
|----------|--------|
| E2E encryption | X25519 + XChaCha20-Poly1305 |
| Transport encryption | WS over TLS |
| Device pairing | QR + public key exchange |
| Persistent identity | Server keypair в SQLite |

### Нужно для платформы

| Требование | Приоритет | Описание |
|------------|:---------:|----------|
| **Auth (API keys)** | P0 | Bearer tokens для embed / API / webhooks |
| **Audit log** | P0 | Все действия агента: кто инициировал, что сделал, когда |
| **Rules / Guardrails** | P0 | Конфигурируемые ограничения (из YAML) |
| **Confirmation before action** | P0 | `confirm_before` — агент спрашивает перед опасными tools |
| **Rate limiting** | P1 | Per-user, per-agent лимиты |
| **OIDC/SSO** | P2 | Enterprise SSO для management dashboard |
| **RBAC** | P2 | Роли для management: admin, editor, viewer |
| **Data isolation** | P1 | Multi-tenant: данные разных клиентов изолированы |

---

## 10. Observability

| Компонент | Приоритет | Описание |
|-----------|:---------:|----------|
| **Structured logs** | P0 | JSON logs, slog. Уже есть |
| **Task history** | P0 | Все задачи + результаты |
| **Agent events** | P0 | Event Store. Уже есть |
| **Analytics** | P1 | Кол-во диалогов, средняя длина, escalation rate, tool usage |
| **Metrics** | P1 | Prometheus: tasks/sec, latency, token usage |
| **Traces** | P2 | OpenTelemetry: full task trace |
| **Admin API** | P0 | REST: agents status, tasks, config, health |

---

## 11. Deployment & Distribution

### Database: SQLite (dev) → PostgreSQL (production)

```yaml
# Quick start / Development — zero deps
engine:
  database: "sqlite"           # ./data/bytebrew.db, всё в одном файле

# Production — надёжный shared storage
engine:
  database:
    type: "postgresql"
    url: "postgresql://user:pass@db:5432/bytebrew"
```

PostgreSQL используется как **обычная реляционная БД** для state persistence:
- Jobs (queue, statuses, history)
- Events (event store, guaranteed delivery, replay)
- Tasks (manage_tasks persistent state)
- Sessions (metadata)
- Agent runs (history)
- Server identity + devices (bridge pairing)

**Не используется для:** vector search (RAG — in-memory или компания подключает свой через MCP), pub/sub (для scaling — специализированные решения).

**SQLite:** single node, до ~50 concurrent sessions. Для quick start и development.
**PostgreSQL:** тысячи concurrent sessions, shared state для scaling.

### Масштабирование — архитектурно заложено, не реализуется сейчас

Текущий scope: **single node**. Для первых клиентов хватит с запасом (PostgreSQL + single engine = сотни concurrent sessions).

Что закладываем в архитектуру чтобы можно было масштабировать позже:
- **GORM** (уже в проекте) — абстракция над DB, SQLite/PostgreSQL через смену драйвера
- **Event broadcasting через interface** — сейчас in-memory channels, потом можно подменить на NATS/аналог
- **Job Queue через GORM** — сейчас PostgreSQL, потом можно distributed
- **Session state через GORM** — не in-memory maps напрямую

Что **НЕ делаем** сейчас: NATS, Redis, distributed locks, sticky sessions, multi-node deployment, Helm chart.

### Community (бесплатно)

```bash
# Quick start (SQLite, zero deps)
docker-compose up -d

# Конфигурация
vim bytebrew.yaml
curl -X POST http://localhost:8443/api/config/reload
```

Компоненты:
- ByteBrew Engine (Go)
- PostgreSQL (опционально, для production)

**Community включает:** полный engine, agents, tools, knowledge, embed API, cron/webhooks/triggers. Всё что нужно для production (single node).

### Enterprise (платно)

Всё из Community +
- Dashboard + analytics (Web UI)
- SSO / RBAC / audit log
- Multi-tenant management
- Priority support + SLA
- Horizontal scaling (когда будет реализовано)

### Managed Cloud (будущее)

ByteBrew хостим мы. Для тех кому не нужен self-hosted.

### Требования

- [ ] GORM как ORM (уже в проекте) — SQLite + PostgreSQL через смену драйвера
- [ ] Конфиг: `database: "sqlite"` или `database: { type: "postgresql", url: "..." }`
- [ ] Auto-migration при upgrade (GORM AutoMigrate)
- [ ] Docker Compose для quick start (engine + опциональный PostgreSQL)
- [ ] Health check endpoints
- [ ] Event broadcasting, Job Queue, Session state — через interfaces (чтобы потом заменить на distributed)

---

## 12. Архитектура продукта

### Принцип: Engine и Code — разные продукты

**Engine** = скомпилированный closed-source бинарник. Взаимодействие только через API.
**Code** = отдельный продукт (как сторонняя компания). НЕ имеет доступа к коду Engine. Взаимодействует ТОЛЬКО через Engine API. Своё лицензирование, своя тарификация.
**CLI и Mobile** = open-source. Могут использоваться и Engine напрямую, и Code.

### Компоненты

```
ByteBrew Engine (closed-source бинарник, мы разрабатываем)
├── Agent Engine (ReAct, tools, MCP, kits)
├── Developer Kit (LSP, indexing, git)
├── Job System (cron, webhooks, queue)
├── REST API + WS API
├── Web Dashboard
├── Bridge (опциональный)
└── YAML config
    CE бесплатный / EE платный

ByteBrew Code (отдельный сервис, как сторонняя компания)
├── Code BFF server
│   ├── Лицензирование (JWT, per-seat, cloud-api)
│   ├── Code-специфичная логика
│   └── Прокси к Engine API
├── Использует open-source CLI и Mobile
└── Своя тарификация, свой сайт (code.bytebrew.ai)

Open-source (MIT):
├── CLI (bytebrew-cli) — reference client
├── Mobile (bytebrew-mobile-app) — reference client
└── Bridge relay (bytebrew-bridge)
```

### Closed-source vs Open-source

| Компонент | Лицензия |
|-----------|---------|
| **Engine (bytebrew-srv)** | Closed-source |
| **Enterprise code** | Closed-source |
| **Code BFF** | Closed-source (своя кодовая база) |
| **CLI (bytebrew-cli)** | Open-source (MIT) |
| **Mobile (bytebrew-mobile-app)** | Open-source (MIT) |
| **Bridge relay (bytebrew-bridge)** | Open-source (MIT) |

### ByteBrew Code — отдельный продукт поверх Engine

Code = BFF сервер с лицензированием и Code-специфичной обвязкой. Взаимодействует с Engine ТОЛЬКО через API.

```
Сервер компании:
  [Engine CE]  ← скачан с bytebrew.ai, бесплатный
       ↑ REST API + WS API
  [Code BFF]   ← ставится рядом, обвязка с лицензированием
       ↑ WS
  [CLI]        ← open-source, разработчики подключаются
  [Mobile]     ← open-source, через Bridge
```

Engine не знает про Code лицензирование. Code BFF проверяет лицензию и проксирует к Engine API.

**Что именно в Code BFF** — определяется при исследовании текущей кодовой базы (разделение engine-логики и Code-специфики из текущего bytebrew-srv).

### Developer Kit на сервере

```yaml
# bytebrew.yaml — coding daemon
agents:
  - name: "supervisor"
    kit: "developer"
    can_spawn: ["code-agent"]
    tools:
      builtin: [ask_user, manage_tasks, web_search]

  - name: "code-agent"
    kit: "developer"
    lifecycle: "persistent"
    can_spawn: ["review", "e2e-test"]
    tools:
      builtin: [file_read, file_write, file_edit, shell_exec, grep_search, glob, git_commit, git_push]

  - name: "review"
    lifecycle: "spawn"
    tools:
      builtin: [file_read, grep_search, glob]

  - name: "e2e-test"
    lifecycle: "spawn"
    tools:
      builtin: [shell_exec]
      mcp_servers:
        playwright:
          type: "stdio"
          command: "npx @anthropic/playwright-mcp"

kit_config:
  developer:
    default_repo: "https://github.com/company/project.git"
    branch: "main"
    worktree_dir: "/tmp/bytebrew-worktrees"
```

Kit при session start:
1. `git clone` или `git worktree add` (если repo уже клонирован)
2. Запуск LSP servers для этого worktree
3. Индексирование кодовой базы
4. File watcher

Kit при session end:
1. Cleanup worktree (если задача завершена)
2. Stop LSP
3. Stop watcher

### Bridge

Bridge **остаётся в engine** как опциональный модуль:

```yaml
bridge:
  enabled: true
  url: "wss://bridge.bytebrew.ai"
```

Для coding daemon — включён (CTO с телефона). Для sales/IoT — по желанию.

### Web Dashboard

Engine включает **встроенный web dashboard:**

- Список агентов (из YAML)
- Job dashboard (running, completed, failed, needs_input)
- Event logs (что делал агент)
- Health / metrics

### CLI и Mobile — роль

**Admin/dev tools**, не end-user interface:

| Кто | CLI | Mobile | Dashboard (web) |
|-----|:---:|:------:|:---------------:|
| Разработчик | Даёт задачи агенту, видит прогресс | — | Мониторинг |
| DevOps | Проверка, config | — | Мониторинг |
| CTO | — | Статус, задачи на ходу | Мониторинг |

Для coding daemon: CLI = основной интерфейс разработчика. Mobile = CTO monitoring.
Для sales/IoT: конечные пользователи через REST API. CLI/Mobile = admin tools.

### Адаптация CLI и Mobile

**CLI:**
- `--agent NAME` — выбор агента
- `bytebrew agents` — список доступных агентов
- `bytebrew init` — интерактивный онбординг:
  - Если запущен в папке с git → "Использовать текущую папку? [Y/n]"
  - Если нет → "Укажите путь или URL репозитория"
  - Записывает project path в конфиг engine
- `--server HOST:PORT` — подключение к удалённому engine (для daemon mode)
- Убрать legacy код: `grepSearch`, `globSearch` (не используется, file search на стороне engine)
- Остальной UI уже universal (chat, tool calls, streaming, plans)

**Mobile:**
- Agent selector screen
- Остальной UI уже universal

### projectRoot: engine знает из конфига

**Сейчас:** CLI передаёт `project_root` при `create_session`. Engine использует этот path.

**Daemon:** engine берёт projectRoot из **своего конфига** (`kit_config.developer.project_path`). CLI может не знать где код на сервере.

```yaml
kit_config:
  developer:
    project_path: "/srv/projects/myproject"     # локальный path на сервере
    # или:
    repo: "git@github.com:company/project.git"  # engine клонирует при init
    clone_path: "/srv/bytebrew/projects"         # куда клонировать
```

Логика: если `project_path` задан → использовать напрямую. Если `repo` задан → git clone в `clone_path` при первом запуске.

При `create_session`: если CLI передал `project_root` → использовать (desktop mode, backward compat). Если пустой → engine берёт из kit_config (daemon mode).

---

## 13. Тарификация и лицензирование

### Модель: единая для всех use cases

Один engine, одна тарификация. Coding agent (ByteBrew Code) и sales/IoT agent — один и тот же продукт, одна лицензия.

| | Community Edition | Enterprise Edition |
|--|------------------|-------------------|
| **Engine** | Полный, без лимитов | Полный, без лимитов |
| **Kits** | Все (включая developer) | Все |
| **Deployment** | Single node | + Horizontal scaling (multi-node) |
| **Observability** | Базовые логи (JSON, EventStore) | + AI Observability Dashboard (prompt analytics, quality metrics, cost tracking, session explorer) |
| **Support** | GitHub issues | Priority + SLA |
| **Бинарник** | `bytebrew-ce` | `bytebrew-ee` |
| **Цена** | $0 | Contact us |

**Community = полнофункциональный engine без runtime ограничений.** Как QuestDB OSS.
**Enterprise = scaling + AI observability.** Production capabilities которые нельзя собрать снаружи.

**Детальные Enterprise requirements:** `06_enterprise_requirements.md`

### Механизм: два бинарника

```
bytebrew-srv/
├── internal/          # core engine (общий код)
├── enterprise/        # enterprise фичи (scaling, observability)
├── cmd/
│   ├── ce/main.go     # Community build
│   └── ee/main.go     # Enterprise build
```

Enterprise код физически отсутствует в CE binary (как QuestDB).

### Сайты

| Сайт | Назначение | Контент |
|------|-----------|---------|
| **bytebrew.ai** | Engine platform | Landing, pricing (CE free / EE contact us), docs, API reference, quick start, YAML reference, examples |
| **code.bytebrew.ai** | ByteBrew Code (coding agent) | Product page, features, getting started, CLI/Mobile downloads |

### bytebrew.ai — структура

```
bytebrew.ai/
├── / (landing)              — УТП, "Добавь AI-агента в свой продукт", примеры
├── /pricing                 — CE (бесплатно, скачать) / EE (Contact Us + форма заявки → email)
├── /docs                    — Quick start, YAML reference, API reference
├── /docs/quick-start        — docker-compose up → YAML → готово
├── /docs/agents             — Agent model, can_spawn, lifecycle
├── /docs/tools              — MCP, declarative tools, builtin
├── /docs/kits               — Developer kit, custom kits
├── /docs/jobs               — Triggers, cron, webhooks
├── /docs/api                — REST API + WS reference
├── /examples                — YAML примеры (sales, IoT, coding, support)
├── /enterprise              — EE features (scaling, observability), Contact Us форма
└── /download                — CE binary download (Linux, macOS, Windows)
```

### code.bytebrew.ai — структура

```
code.bytebrew.ai/
├── / (landing)              — "AI Developer на вашем сервере", features, demo video
├── /features                — Developer Kit (LSP, indexing), multi-agent pipeline, git integration
├── /getting-started         — Установка engine + developer kit config, CLI подключение
├── /cli                     — CLI download, docs (open-source GitHub link)
├── /mobile                  — Mobile app download (open-source GitHub link)
└── /enterprise              — EE features для coding (observability: task quality, cost per PR), Contact Us
```

### Enterprise Contact Us

На обоих сайтах:
- Страница /enterprise с описанием EE фич
- Форма: имя, email, компания, use case, количество agents/users
- Submit → email на sales@bytebrew.ai
- Автоответ: "Спасибо, свяжемся в течение 24 часов"

### Что нужно реализовать

- [ ] Community dashboard (базовый web UI в engine: agents, jobs, logs)
- [ ] Сайт bytebrew.ai (landing, pricing, docs, download, enterprise contact form)
- [ ] Сайт code.bytebrew.ai (landing, features, getting started, CLI/Mobile links)
- [ ] Enterprise contact form → email notification

---

## 14. Kits и примеры конфигураций

### Kits — Go-код для доменов с engine-level интеграцией

Когда домену нужно больше чем tools + prompt (session state, watchdog, auto-enrichment):

| Kit | Что даёт | Когда нужен |
|-----|----------|-------------|
| `developer` | LSP (10 серверов, auto-install), code indexing (chunker, embeddings, watchdog), symbol tools | Coding agent |

Kit активируется в YAML: `kit: "developer"`. Engine загружает Go-код kit'а при старте.

Для остальных доменов (sales, support, IoT) kits не нужны — хватает tools + prompt + knowledge.

### Примеры конфигураций

Репозиторий содержит готовые YAML-примеры (`examples/`):

```
examples/
├── developer.yaml      # coding agent (с kit)
├── sales.yaml          # sales assistant
├── iot.yaml            # IoT monitor
├── support.yaml        # customer support
└── researcher.yaml     # web research agent
```

Пользователь копирует пример → меняет prompt, tools, knowledge под себя.

---

## 15. Milestones

### Фокус: seed round

**Engine (core):**

| # | Milestone | Что работает | Приоритет |
|---|-----------|-------------|:---------:|
| 1 | **Agent abstraction** | Configurable agents (YAML): prompt, tools, knowledge, sub-agents | P0 |
| 2 | **Tool Registry + Spawn** | Dynamic per-agent tools, generic spawn, MCP client | P0 |
| 3 | **Developer Kit** | LSP + indexing → kit. Coding pipeline через YAML | P0 |
| 4 | **Job System** | Cron, webhooks, async jobs, job queue | P0 |
| 5 | **REST API + SSE** | Embed API для интеграции | P0 |
| 6 | **Web Dashboard** | Базовый: agents, jobs, logs | P0 |
| 7 | **Knowledge (RAG)** | Документы из папки → vector search | P1 |
| 8 | **manage_tasks integration** | Engine lifecycle для persistent task tracking | P1 |

**Open-source clients:**

| # | Milestone | Что работает | Приоритет |
|---|-----------|-------------|:---------:|
| 9 | **CLI abstraction** | `--agent NAME`, generic chat, `bytebrew agents` | P0 |
| 10 | **Mobile abstraction** | Agent selector, generic chat | P1 |
| 11 | **Open-source release** | CLI + Mobile + Bridge relay → GitHub MIT | P1 |

**Бизнес:**

| # | Milestone | Что работает | Приоритет |
|---|-----------|-------------|:---------:|
| 12 | **Сайт bytebrew.ai** | Landing, pricing (CE free / EE contact us), docs, download | P0 |
| 13 | **Сайт code.bytebrew.ai** | ByteBrew Code landing, features, CLI/Mobile links | P1 |
| 14 | **Enterprise page** | EE features описание + Contact Us форма → email | P0 |
| 15 | **Seed demo** | Видео: 2 домена на одном engine + dashboard | P0 |

**Enterprise:** не продаём сейчас. Показываем модель на сайте. При обращении — онбордим как early adopters.

### Что нужно для seed pitch

1. **Working demo** — coding agent (developer kit) + sales/IoT agent на одном engine
2. **Dashboard** — web UI показывающий агентов, jobs, логи
3. **REST API** — curl POST → SSE stream → agent работает
4. **Сайт bytebrew.ai** — landing + pricing + docs + enterprise contact form
5. **Сайт code.bytebrew.ai** — ByteBrew Code showcase + CLI/Mobile downloads
6. **Pilot** — 1 компания тестирует
7. **Positioning** — "Добавь AI-агента в свой продукт"

### Что НЕ входит в seed milestone

- Horizontal scaling (архитектурно заложено, не реализуется)
- OIDC/SSO, RBAC (Enterprise P2)
- Cloud managed version
- Mobile SDK для чужих apps

---

## Сценарий демо (seed pitch)

```
Часть 1: "Dashboard" (1 мин)
  — bytebrew.ai: landing page, pricing, docs
  — Web Dashboard: список агентов (coding-supervisor, sales-assistant)
  — Job dashboard: running jobs, completed, failed

Часть 2: "Coding Agent — ByteBrew Code" (2 мин)
  — code.bytebrew.ai: product page
  — CLI (open-source): bytebrew chat --agent coding-supervisor
  — Задача: "Добавь endpoint GET /api/health"
  — Агент: git clone → LSP → пишет код → тесты → review → PR
  — Mobile (open-source): CTO видит прогресс с телефона

Часть 3: "Sales Agent — тот же engine" (2 мин)
  — curl POST /api/v1/agents/sales/chat → SSE stream
  — "Ищу ноутбук для видеомонтажа"
  — Агент: интервьюирует → ищет → предлагает

Часть 4: Enterprise (30 сек)
  — /enterprise page: scaling + AI observability
  — Contact Us форма
  — "Фичи в разработке, онбордим early adopters"

Часть 5: Ключевой момент (30 сек)
  "Один engine. Coding и sales — один YAML конфиг.
   Community бесплатный. Enterprise — scaling + observability.
   CLI и Mobile — open-source.
   Модель — ваша. Inference — бесплатный."
```
