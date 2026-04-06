# PRD: ByteBrew Cloud + Engine v2

**Дата:** 2026-04-06
**Статус:** Ready for development
**Ambiguity Score:** 13.2% (threshold: 20%)
**Источник:** Deep Interview (10 rounds) + Gap Analysis Interview (10 rounds, 23 decisions)

---

## 1. Product Vision

**Positioning:** "Not another AI chatbot. ByteBrew is the open-source agent brewery."

**Суть:** ByteBrew — open-source agent runtime. Пользователь описывает задачу → ByteBrew создаёт AI-агентов с памятью, навыками и мышлением → агенты работают автономно. Простая задача — один агент. Сложная — команда.

**Три продукта, один engine:**

| Продукт | Что | Для кого |
|---------|-----|---------|
| **Cloud** | Managed service на bytebrew.ai | Все (быстрый старт, без хостинга) |
| **CE** | Self-hosted, бесплатно (BSL 1.1) | Devs, компании (полный контроль) |
| **EE** | Self-hosted + enterprise features (future, под feature flag) | Enterprise (SSO, audit, compliance) |

---

## 2. License

**BSL 1.1** (Business Source License)
- **Additional Use Grant:** любое использование КРОМЕ предоставления ByteBrew как managed service третьим лицам
- **Change Date:** 4 года
- **Change License:** Apache 2.0
- Self-host: бесплатно
- Embedding в свой продукт: бесплатно (через forward_headers, API)
- Multi-tenancy в СВОЁМ продукте: разрешено
- Перепродажа ByteBrew как hosted platform: запрещено

---

## 3. Cloud Architecture

### 3.1 Multi-Tenant на одной БД

```
bytebrew.ai
     │
     ▼
┌─────────────────────────────────┐
│  ByteBrew Engine (один процесс) │
│  ┌───────────────────────────┐  │
│  │  Auth: JWT + tenant_id    │  │
│  │  Rate limiter: per-tenant │  │
│  ├───────────────────────────┤  │
│  │  API: scoped by tenant_id │  │
│  │  автоматически            │  │
│  ├───────────────────────────┤  │
│  │  Agent Runtime            │  │
│  │  (shared, сессии          │  │
│  │   изолированы по tenant)  │  │
│  └───────────────────────────┘  │
│              │                   │
│  ┌───────────────────────────┐  │
│  │  PostgreSQL (одна БД)     │  │
│  │  tenant_id на всех таблицах│  │
│  └───────────────────────────┘  │
└─────────────────────────────────┘
```

### 3.2 Что нужно реализовать

**Существует (cloud-api):**
- User registration (email + Google OAuth)
- JWT auth + refresh tokens
- Stripe integration (checkout, portal, webhooks)
- Teams (roles, invites)
- License signing (Ed25519 JWT)

**Нужно добавить:**
- `tenant_id` на все таблицы engine (agents, triggers, models, mcp_servers, sessions, memory)
- Tenant middleware (auto-scope все запросы)
- Rate limiting per tenant
- Cloud-specific Stripe products (Free/Pro/Business тиры)
- Quota enforcement (API calls, storage, agents per schema)
- Default model integration (GLM 4.7 proxy)

### 3.3 Cloud Security Model

```
Cloud агенты МОГУТ:
  ✅ MCP tools (внешние API через verified/custom MCP servers)
  ✅ Knowledge/RAG (документы в storage tenant'а)
  ✅ Web search (через MCP — Tavily, Brave, etc.)
  ✅ Structured output
  ✅ Inter-agent communication (can_spawn, flows, transfer)

Cloud агенты НЕ МОГУТ:
  ❌ File system access (читать/писать файлы на сервере)
  ❌ Shell execution (выполнять команды)
  ❌ Local network access
  ❌ Доступ к данным других tenants
```

### 3.4 Acceptance Criteria

- AC-CLOUD-01: Пользователь регистрируется → получает tenant → видит пустой workspace
- AC-CLOUD-02: Tenant A не видит данных Tenant B (agents, sessions, memory)
- AC-CLOUD-03: API calls считаются per-tenant, при превышении лимита → 429 с сообщением
- AC-CLOUD-04: Storage считается per-tenant (memory + knowledge + sessions)
- AC-CLOUD-05: Агент в Cloud НЕ может вызвать file/shell tools (ошибка, не silent fail)
- AC-CLOUD-06: Rate limit per-tenant работает (burst protection)

---

## 4. Pricing

### 4.1 Тарифная сетка Cloud

| | Free | Pro $29/мо | Business $99/мо | Enterprise |
|--|:---:|:---:|:---:|:---:|
| Schemas | 1 | 5 | ∞ | ∞ |
| Agents per schema | 10 | ∞ | ∞ | ∞ |
| API calls/мес | 1,000 | 50,000 | 500,000 | Custom |
| Storage | 100 MB | 5 GB | 50 GB | Custom |
| MCP | Verified only | All | All | All |
| Widgets | 1 | 3 | ∞ | ∞ |
| Team members | 1 | 3 | 10 | ∞ |
| forward_headers | ❌ | ✅ | ✅ | ✅ |
| OPS Mode | ❌ | ❌ | ✅ | ✅ |
| Default model (GLM 4.7) | 100 req/мес | 100 req/мес | 100 req/мес | 100 req/мес |
| BYOK | ✅ | ✅ | ✅ | ✅ |
| 14-day Pro trial | ✅ | — | — | — |
| Support | Community | Email | Priority | Dedicated |

**Принцип:** тарифы ограничивают КОЛИЧЕСТВО (schemas, calls, storage), не ВОЗМОЖНОСТИ. Все engine capabilities (Memory, Flows, Gates, Inspect, Recovery, ReAct, can_spawn, Knowledge) — одинаковы на всех тарифах.

### 4.2 Inference Model

```
Default model (GLM 4.7):
  Все тарифы = 100 req/мес (бесплатно)
  Модель фиксирована, нельзя поменять
  Исчерпал → "Добавьте свой API ключ"

BYOK (свой ключ):
  Все тарифы = доступно
  Любая модель (OpenRouter, OpenAI, Anthropic)
  Пользователь платит провайдеру напрямую
  Без лимита inference с нашей стороны
```

**Наши расходы на GLM:** ~$0.001/req × 100 = $0.10/пользователь/мес

### 4.3 CE vs Cloud

```
CE (self-hosted, бесплатно):
  = Полный engine без ограничений
  + File/shell tools (свой сервер)
  + Любые модели (свои ключи)
  + Безлимитно всё
  - Сам хостишь, обновляешь, настраиваешь

Cloud:
  = CE по возможностям (кроме file/shell)
  + Managed infrastructure
  + Default model из коробки
  + Support
  + 14-day Pro trial
  - Лимиты по тарифу
  - Нет file/shell tools (security)
```

### 4.4 Stripe Implementation

Существующие тиры (`personal`, `teams`, `engine_ee`) — остаются под feature flag для будущего EE.

Новые Stripe Products для Cloud:
- `bytebrew_cloud_free` — Free tier
- `bytebrew_cloud_pro` — $29/мо (monthly + annual)
- `bytebrew_cloud_business` — $99/мо (monthly + annual)

### 4.5 Acceptance Criteria

- AC-PRICE-01: Free пользователь создаёт до 1 schema, 10 agents, 1000 API calls
- AC-PRICE-02: При превышении лимита → понятное сообщение + CTA upgrade
- AC-PRICE-03: Stripe checkout работает для Pro и Business
- AC-PRICE-04: 14-day Pro trial активируется без карты
- AC-PRICE-05: После trial → возврат на Free (не блокировка)
- AC-PRICE-06: Default model (GLM 4.7) работает без API ключа, лимит 100 req
- AC-PRICE-07: BYOK: пользователь вставляет ключ → модели доступны → лимит inference снят

### 4.6 Quota Enforcement UX

**Usage Dashboard (Admin → Settings → Usage):**
- Bar charts per metric: API calls (used/limit), Storage (used/limit), Schemas (used/limit), Agents per schema
- Current plan badge + billing cycle dates
- "Manage Plan" button → Stripe Customer Portal

**Warning levels:**

| % Used | UI Action |
|--------|-----------|
| 80% | Yellow banner top-of-page: "80% of API calls used this month" |
| 95% | Red banner: "Almost at limit — upgrade to avoid interruption" |
| 100% | Modal block: "Limit reached" + [Upgrade Plan] button |

**Upgrade flow:**
1. User clicks "Upgrade" → Stripe Checkout (hosted payment page)
2. Successful payment → Stripe webhook → plan updated in DB → limits increased immediately
3. User redirected back to Admin → success toast "Plan upgraded to Pro"
4. Downgrade: через Stripe Customer Portal, effective at end of billing cycle

**AC (Quota UX):**
- AC-PRICE-08: Usage dashboard показывает текущее потребление per-metric с bar charts
- AC-PRICE-09: Warning banner при 80% использования любого лимита
- AC-PRICE-10: Hard block modal при 100% с upgrade CTA
- AC-PRICE-11: Stripe Checkout upgrade → лимиты увеличиваются мгновенно после оплаты

---

## 5. Brewery UX — AI Assistant

### 5.1 Концепция

AI Assistant — **главная точка входа**. Не canvas, не форма, не wizard. Чат.

```
Пользователь → Chat с AI Assistant → Assistant routing → Action
```

### 5.2 Layout

```
┌──────────────────────────────────────────────────────────┐
│  Toolbar: [Schema ▾] [+ Schema] [Auto Layout] [+ Agent] │
├──────────────────────────────────────────────────────────┤
│                                                          │
│                      CANVAS                              │
│          (визуализация, синхронная)                       │
│                                                          │
│   Ноды появляются/исчезают по мере работы ассистента    │
│                                                          │
├──────────────────────────────────────────────────────────┤
│  ═══════════ drag handle ═══════════                     │
│  [AI Assistant] [Test Flow]              ← tabs          │
│                                                          │
│  Chat / test messages                    [message input] │
└──────────────────────────────────────────────────────────┘
```

Canvas сверху, Bottom Panel снизу (resizable). Два таба: AI Assistant (чат с builder agent) и Test Flow (тест schema). Всё что ассистент делает → видно на canvas в реальном времени.

### 5.3 Routing Logic (адаптация OMC)

AI Assistant классифицирует каждый запрос:

| Сигнал | Действие |
|--------|----------|
| Нет schemas, первый визит | Interview → Assembly |
| Vague request ("сделай что-то для бизнеса") | Interview |
| Clear new schema ("сделай support flow с 3 агентами для магазина обуви") | Short interview → Assembly |
| Modification ("добавь агента в support flow") | Direct execution |
| Config change ("измени prompt у classifier") | Direct execution |
| Integration ("подключи Notion") | Direct execution |
| Simple question ("как работают flows?") | Answer |

**Ambiguity detection** (по паттернам OMC):
- Нет конкретных имён агентов/tools → vague
- Word count ≤ 15 → vague
- Есть конкретные файлы/agents/tools → clear
- Escape: "просто сделай" → skip interview

### 5.4 Full Flow

```
1. PROMPT
   User: "Нужен support для интернет-магазина обуви"

2. ROUTING
   Assistant: [classify → new schema + vague → interview mode]

3. INTERVIEW (снижение энтропии)
   Agent: "Какие каналы? (чат, email, telegram?)"
   User: "Чат на сайте"
   Agent: "Типичные вопросы?"
   User: "Доставка, возврат, размеры"
   Agent: "Есть CRM или база заказов?"
   User: "Google Sheets"
   → Энтропия достаточно низкая → assembly

4. ASSEMBLY (canvas синхронизируется)
   Agent: "Собираю схему..."
   → Canvas: trigger node появляется (fade in)
   → Canvas: classifier agent появляется
   → Canvas: support-agent появляется
   → Canvas: escalation agent появляется
   → Canvas: edges анимированно рисуются
   → Canvas: MCP иконка на support-agent (Google Sheets)

5. SELF-TEST
   Agent: "Тестирую с mock данными..."
   → Canvas: ноды подсвечиваются по мере работы
   → "Все агенты отвечают. Flow работает."

6. EXTERNAL INTEGRATIONS (если нужны)
   Agent: "Для Google Sheets подключите MCP сервер:
           Settings → MCP → Google Sheets → Connect

           Для production API вашего магазина —
           вот ТЗ для разработчика:
           [integration-spec.md]"

7. DELIVER
   Agent: "Готово! Протестируйте ниже."
   → Bottom panel: Test Flow активируется

8. USER TEST
   User: "Какой размер у Nike Air Max?"
   → Pipeline работает → ответ
```

### 5.5 Canvas ↔ Assistant синхронизация

| Действие ассистента | Canvas анимация |
|--------------------|-----------------|
| Создал schema | Canvas переключается на новую схему |
| Создал agent | Нода fade in + scale |
| Добавил flow/transfer/loop edge | Edge анимированно рисуется |
| Обновил agent config | Нода pulse (glow) |
| Подключил MCP | Иконка MCP на ноде |
| Удалил agent | Нода fade out |
| Self-test | Ноды подсвечиваются по ходу выполнения |

### 5.6 Assistant на ВСЕХ страницах Admin

Bottom panel с AI Assistant доступен на **каждой странице** admin (Canvas, Agents, MCP Servers, Models, Triggers, и т.д.), не только на Canvas.

**Schema selector в chat header:**
- Dropdown `[Support Schema ▾]` прямо в chat panel header
- Assistant работает в контексте выбранной schema
- Переключение schema меняет контекст assistant (доступные агенты, triggers, edges)

**Правило: 1 entry agent per schema:**
- Каждая schema имеет ровно 1 entry agent (агент на который ведут triggers)
- При тестировании через chat — сообщение отправляется entry agent'у выбранной schema
- Entry agent определяется наличием входящих trigger edges

**"Open Chat" ссылка удалена из sidebar:**
- Admin assistant = bottom panel (на всех страницах)
- `/chat/` web-client = для конечных пользователей (отдельный продукт)
- Не путать admin-ассистента (настраивает систему) с user-чатом (общается с агентами)

### 5.7 Live Animation — визуальная оркестрация

Когда Assistant настраивает систему, пользователь **видит в реальном времени** что происходит:

| Действие assistant | Визуальный эффект |
|-------------------|-------------------|
| Создаёт агента | Canvas: нода fade-in + scale. Drill-in: поля заполняются с анимацией печати |
| Меняет system prompt | Drill-in page открывается, текст стримится в textarea |
| Добавляет capability | Capability block появляется с slide-down анимацией |
| Подключает MCP | MCP page: сервер добавляется, статус connecting → connected |
| Создаёт trigger | Triggers page: строка появляется в таблице |
| Переключает страницу | Sidebar: active item переключается, page transition |

**Принцип:** Не "assistant сделал, посмотри результат", а "смотри как assistant делает" — полная визуальная прозрачность процесса.

**Техническая реализация (backend):**
- Assistant actions → SSE events (`admin.field_update`, `admin.page_navigate`, `admin.node_create`)
- Admin UI подписан на SSE → применяет изменения с анимацией
- Каждый event содержит: target (page/component/field), action, value, animation_type

### 5.8 Builder Assistant — системный агент

- Встроен в Engine, не создаётся пользователем
- Не виден в списке агентов пользователя
- Не редактируется, не удаляется
- Изолирован от пользовательских агентов (отдельный namespace)
- Использует модель из Engine settings
- Admin tools: create/update/delete agent, trigger, schema, edge, MCP connection

### 5.9 Acceptance Criteria

- AC-UX-01: Новый пользователь видит пустой canvas + AI Assistant с приветствием
- AC-UX-02: Vague запрос → ассистент задаёт уточняющие вопросы (interview)
- AC-UX-03: Clear запрос → ассистент выполняет напрямую
- AC-UX-04: При создании agent/edge → нода/edge появляется на canvas с анимацией
- AC-UX-05: При обновлении агента → нода pulse-анимация
- AC-UX-06: Self-test → ноды подсвечиваются по мере выполнения
- AC-UX-07: Builder assistant не виден в /admin/agents пользователя
- AC-UX-08: Builder assistant нельзя удалить или редактировать

**AC (Agent Configuration UX):**
- AC-UX-01 (config): Каждая capability имеет уникальную SVG иконку (Lucide icons в стиле admin menu, НЕ emoji/аббревиатуры)
- AC-UX-02 (config): Секции drill-in page collapsible
- AC-UX-03 (config): Model & Lifecycle параметры в 2-колоночном layout

**AC (Assistant on all pages):**
- AC-UX-09: Bottom panel с AI Assistant доступен на ВСЕХ страницах admin, не только Canvas
- AC-UX-10: Schema selector dropdown в chat header — переключает контекст assistant
- AC-UX-11: 1 entry agent per schema — chat отправляет сообщение entry agent'у
- AC-UX-12: "Open Chat" ссылка удалена из admin sidebar

**AC (Live Animation — BACKEND-DEFERRED):**
- AC-UX-13: При создании агента через assistant — нода fade-in на canvas
- AC-UX-14: При изменении system prompt — текст стримится в textarea drill-in page
- AC-UX-15: При добавлении capability — block slide-down анимация
- AC-UX-16: Assistant actions → SSE events → Admin UI применяет с анимацией

---

## 6. Canvas Model

### 6.1 Three Node Types

| Нода | Что | Визуал |
|------|-----|--------|
| **Agent** | ReAct агент с tools, reasoning, memory | AgentNode (persistent/spawn дифференциация) |
| **Trigger** | Entry point: user-message, cron, webhook | TriggerNode |
| **Gate** | Проверка условия между агентами | GateNode (новый) |

### 6.2 Five Edge Types

| Edge | Семантика | Визуал |
|------|-----------|--------|
| `can_spawn` | LLM решает (dynamic) | красный, solid |
| `triggers` | Cron/webhook запускает | фиолетовый, dashed |
| `flow` | Всегда после (deterministic) | зелёный, solid |
| `transfer` | Hand-off, завершается | синий, solid |
| `loop` | Цикл через gate | оранжевый, curved |

### 6.2.1 Trigger → Entry Agent Rule

**Правило:** Триггеры (`triggers` edge) могут ссылаться ТОЛЬКО на entry agents.

- **Entry agent** = агент без входящих flow/transfer edges от других агентов (первый в pipeline)
- Несколько триггеров (webhook + cron) → один entry agent = допустимо
- Триггер → sub-agent или агент в середине/конце цепочки = запрещено
- Если нужен отдельный entry point → отдельная schema

**Обоснование:** одна schema = один логический pipeline. Параллельные несвязанные цепочки внутри одной schema — архитектурная ошибка, для этого нужны разные schemas.

**AC:**
- AC-TRIGGER-01: Создание trigger с target = entry agent → успех
- AC-TRIGGER-02: Создание trigger с target = non-entry agent (имеет входящий flow/transfer) → ошибка валидации
- AC-TRIGGER-03: Несколько триггеров (webhook + cron) на один entry agent → допустимо

### 6.3 Edge Configuration

| Режим | Default? | Когда |
|-------|:---:|-------|
| Full output | ✅ | Следующий агент получает весь output |
| Field mapping | | `input_field: "backend_task"` — конкретное поле |
| Custom prompt | | Шаблон с переменными из output |

**Edge Configuration UX (Side Panel при клике на edge):**

| Поле | UI Element |
|------|-----------|
| Edge type | Read-only badge (flow / transfer / loop / can_spawn / triggers) |
| Mode | Radio: Full Output (default) / Field Mapping / Custom Prompt |

**Full Output:** Нет доп. настроек — следующий агент получает весь output.

**Field Mapping:**
```
Source field         →    Target field
[output.task]         →   [input.backend_task]    [✕]
[output.priority]     →   [input.priority]        [✕]
                                          [+ Add mapping]
```
Key-value pairs с autocomplete для source fields (на основе output schema предыдущего агента, если есть).

**Custom Prompt:** Template textarea с переменными `{{output}}`, `{{output.field}}`:
```
Summarize the following for the next agent:
Task: {{output.task}}
Context: {{output.context}}
```
Autocomplete для `{{}}` переменных из output schema.

**Кнопки:** [Save] [Delete Edge]

**AC (Edge Configuration):**
- AC-EDGE-01: Клик на edge → Side Panel с конфигурацией
- AC-EDGE-02: Field Mapping: key-value pairs с add/remove
- AC-EDGE-03: Custom Prompt: template textarea с `{{}}` переменными
- AC-EDGE-04: Delete Edge удаляет edge и закрывает panel

### 6.4 Parallel Execution

- Несколько `flow` edges из одной ноды = fork
- Gate с несколькими входами = join (ждёт всех)

### 6.5 Gate Conditions

| Тип | Как |
|-----|-----|
| Auto | JSON Schema valid, regex, contains X |
| Human | Approval перед следующим шагом |
| LLM-based | LLM оценивает качество |
| All completed | Join — ждёт всех входящих |

**Gate Configuration UX (Side Panel при клике на gate ноду):**

| Condition Type | Конфигурация |
|---------------|-------------|
| Auto (JSON Schema) | JSON Schema editor или regex input или "contains" text |
| Human | Approval prompt text (что увидит пользователь), timeout (optional) |
| LLM-based | Judge prompt textarea, model selector dropdown, pass threshold (0.0-1.0) |
| All completed | Нет доп. конфигурации — auto-join (ждёт все входящие edges) |

**Общие поля:**
- **Max iterations** (для loop edges): default 3, max 10 — предотвращает бесконечные циклы
- **Timeout:** секунды (0 = no timeout, default 60)
- **On timeout/failure:** block (stop pipeline) / skip (пропустить gate, продолжить) / escalate

**AC (Gate):**
- AC-GATE-01: Gate ноду можно сконфигурировать через Side Panel
- AC-GATE-02: Каждый condition type имеет свою форму конфигурации
- AC-GATE-03: Max iterations предотвращает бесконечные loop cycles
- AC-GATE-04: On timeout/failure action выполняется корректно

### 6.6 Node Creation — Instant, No Modals

**Агенты и триггеры создаются прямо на canvas, без модальных окон:**

- **"+ Add Agent"** → нода мгновенно появляется на canvas с дефолтным именем (`new-agent-1`), дефолтной моделью. Пользователь кликает → drill-in page → настраивает все параметры.
- **"+ Add Trigger"** → триггер нода на canvas (`new-trigger-1`). Клик → настройка type, schedule, webhook path.

**Обоснование:** У агента десятки настроек (capabilities, lifecycle, tools, parameters, escalation, policies). Модалка с 3 полями (name, model, prompt) не имеет смысла — пользователь всё равно идёт в drill-in. Убираем промежуточный шаг.

**Agent defaults при instant creation:**

| Поле | Default |
|------|---------|
| name | `new-agent-{N}` (auto-increment per schema) |
| model | Первая модель в списке (или default model) |
| system_prompt | Пустой (placeholder: "Describe this agent's role...") |
| lifecycle | spawn |
| tools | Tier 1 Core only (ask_user, manage_tasks, wait) |
| capabilities | Нет (добавляются в drill-in) |
| max_turn_steps | 25 |
| max_context_size | 16000 |
| max_turn_duration | 120s |

**Trigger defaults при instant creation:**

| Поле | Default |
|------|---------|
| name | `new-trigger-{N}` (auto-increment per schema) |
| type | webhook |
| path | `/webhook/{auto-generated-uuid-short}` |
| target | Нет — пользователь соединяет trigger → agent edge вручную |
| schedule | Нет (появляется при смене type на cron) |
| headers | Пустой key-value list |

### 6.7 Trigger Schedule — Human-Readable

**Cron syntax (`0 9 * * *`) заменяется на human-readable scheduler:**

```
Repeat: [Every day ▾]  at [09:00 ▾]
```

**Presets:**
- Every N minutes (5, 15, 30, 60)
- Every hour at :00
- Every day at HH:MM
- Every weekday (Mon-Fri) at HH:MM
- Every Monday / Tuesday / ... at HH:MM
- Custom (advanced cron toggle для экспертов)

Внутренне конвертируется в cron. Пользователь видит только человеческий формат. "Advanced" toggle для raw cron.

### 6.8 Multiple Schemas

- Schema = именованная группа агентов + связи + триггеры
- Переключатель в toolbar
- CRUD: создание, удаление, переименование
- Export/Import per-schema
- Один агент может быть в нескольких schemas

### 6.7 Acceptance Criteria

- AC-CANVAS-01: Gate нода отображается на canvas, настраивается condition
- AC-CANVAS-02: Flow/transfer/loop edges создаются drag-and-drop
- AC-CANVAS-03: Edge configuration (full output / field mapping / custom prompt) в Side Panel при клике на edge
- AC-CANVAS-04: Parallel: несколько flow edges из одной ноды запускают агентов параллельно
- AC-CANVAS-05: Gate join: ждёт завершения всех входящих агентов
- AC-CANVAS-06: Schemas: создание, переключение, удаление, переименование
- AC-CANVAS-07: Export/Import per-schema (YAML)
- AC-CANVAS-08: "+ Add Agent" создаёт ноду на canvas мгновенно (без модалки), клик → drill-in
- AC-CANVAS-09: "+ Add Trigger" создаёт trigger ноду на canvas мгновенно (без модалки), клик → настройка
- AC-CANVAS-10: Trigger schedule — human-readable UI (presets: every day/hour/weekday + time picker)
- AC-CANVAS-11: Trigger schedule — "Advanced" toggle показывает raw cron input для экспертов

---

## 7. Admin UI

### 7.1 Drill-In UX (не Side Panel)

Клик на agent ноду → **проваливаемся в полноэкранную конфигурацию:**

```
← Support Flow / support-agent                    [Save] [Delete]

Model: [claude-sonnet ▾]    Lifecycle: [persistent ▾]

System Prompt:
┌────────────────────────────────────────────────────┐
│ You are a technical support agent...               │
└────────────────────────────────────────────────────┘

── Parameters ─────────────────────────
Max Turn Steps: [50]    Context Size: [16000]    Execution: [sequential ▾]
Max Turn Duration: [120s]

── Model Parameters ───────────────────
Temperature: [0.7]    Top P: [1.0]    Max Tokens: [4096]
Stop Sequences: [comma-separated]

── Capabilities ──────────────────────────── [+ Add ▾]

┌ 🧠 Memory ────────────────────────── [⚙] [✕] ┐
│ Cross-session · Per-user: yes                  │
└────────────────────────────────────────────────┘
┌ 📚 Knowledge ─────────────────────── [⚙] [✕] ┐
│ support-docs.pdf (2,341 chunks)                │
└────────────────────────────────────────────────┘

── Tools ──────────────────────────────
Core:      ask_user · show_structured_output · manage_tasks · wait (always on)
Auto:      memory_recall · memory_store (from Memory capability)
Self-host: ☑ read_file  ☑ write_file  ☐ execute_command
MCP:       tavily_search · github_create_issue (from connected MCP servers)

── Connections ────────────────────────
Receives from: classifier (flow)
Can spawn: ☐ researcher
```

### 7.2 Capability Blocks ([+ Add])

Каждый capability — модуль расширения агента. Добавляется через [+ Add] в drill-in, конфигурируется inline.

#### Memory
Агент запоминает информацию между сессиями. Scope: **per-schema, cross-session**.
- **Cross-session persistence** (bool): хранить memory между сессиями
- **Per-user isolation** (bool): изолировать memory по user_id
- **Retention** (string): **Unlimited** по умолчанию (без авто-удаления из БД)
- **Max entries** (number): максимальное количество записей (ограничение через лимит записей, не TTL)
- **Eviction**: FIFO (oldest entries removed first) при достижении max_entries
- **Hint:** "Агенты в разных схемах имеют раздельные memory spaces"

#### Knowledge (RAG)
Агент ищет в загруженных документах перед ответом.
- **Supported formats:** PDF, DOCX, DOC, TXT, MD, CSV
- **Sources** (file[]): загруженные документы
- **Top-K** (number, default 5): количество чанков для retrieval — **настраивается в agent capability config**, не в аргументах tool
- **Similarity threshold** (float 0-1, default 0.75): порог релевантности — **настраивается в agent capability config**, не в аргументах tool
- `knowledge_search` tool использует значения top_k и similarity_threshold из agent config

**File Listing UI:**
- Таблица: Имя файла | Тип | Размер | Дата загрузки | Статус индексации
- Статусы: `uploading → indexing → ready → error`
- Действия: Удалить, Переиндексировать
- Preview недоступен (RAG index, не viewer)

#### Output Guardrail
**Пост-валидация** output агента перед отправкой пользователю. Работает ПОСЛЕ генерации ответа (в отличие от Output Schema, который работает ДО).

**Три режима:**

**1. JSON Schema (post-validation):**
- LLM генерирует ответ → Engine валидирует ответ против JSON Schema
- JSON Schema editor позволяет редактировать произвольную схему
- При невалидном ответе → on_failure action

**2. LLM Judge (separate LLM call):**
- Main agent генерирует ответ → Engine отправляет ответ judge LLM с настраиваемым промптом
- Judge возвращает оценку (yes/no или score)
- UI чётко показывает что промпт — для judge LLM, не для основного агента ("Промпт для проверочного LLM")
- При no → on_failure action

**3. Webhook (POST with contract):**
- Engine отправляет POST на webhook URL с payload:
  ```json
  {
    "event": "guardrail_check",
    "agent": "support-agent",
    "session_id": "sess_123",
    "response": "Agent's generated response text",
    "metadata": { "model": "claude-sonnet", "turn": 3 }
  }
  ```
- Webhook возвращает: `{ "pass": true|false, "reason": "Optional explanation" }`
- Timeout: 10s. Retry: 1x. При timeout → on_failure action

**On failure behaviors:** retry (max 3) | error | fallback

**Webhook auth types:** none | api_key (`Authorization: Bearer <token>`) | forward_headers (прокидывает headers из входящего запроса) | oauth2 (Client ID + Secret → engine обновляет token)

#### Output Schema
**Отдельная capability от Output Guardrail.** Формирует структуру ответа ДО генерации (pre-generation) через `response_format` в LLM API call. LLM сам старается соответствовать схеме.

| | Output Schema | Output Guardrail |
|---|---|---|
| **Стадия** | До генерации | После генерации |
| **Механизм** | `response_format` в LLM API call | Валидация ответа (JSON/LLM/Webhook) |
| **Цель** | Формировать структуру ответа | Проверить качество/соответствие |
| **При ошибке** | LLM сам старается соответствовать | Retry / Error / Fallback |

**Оба могут быть включены одновременно** (Schema формирует, Guardrail проверяет).

- **Format** (enum): json_schema | plain_text
- **Enforce** (bool): блокировать ответ если не соответствует схеме
- **Schema** (text): JSON Schema определение

#### Escalation
Эскалация на пользователя или внешнюю систему.

**Actions:**
- `transfer_to_user` — передать пользователю управление (НЕ "transfer_to_human")
- `notify_webhook(url)` — отправить уведомление на webhook
- `send_message(text)` — отправить сообщение пользователю

**Typed Conditions (dropdown, НЕ CEL):**
- `confidence_below(threshold)` — агент не уверен в ответе (confidence 0.0-1.0 генерируется LLM агента)
- `topic_matches(pattern)` — тема соответствует паттерну
- `user_sentiment(negative)` — негативный сентимент пользователя
- `max_turns_exceeded(n)` — превышено N шагов
- `tool_failed(tool_name)` — инструмент не сработал
- `custom(prompt)` — LLM оценивает по промпту

- **Webhook URL** (string, optional): URL для уведомления
- **Webhook auth**: none | api_key | forward_headers | oauth2 (единый контракт для всех webhook'ов)

#### Recovery Policy
Автоматическое восстановление при сбоях. **Degrade scope: per-session** — при новой сессии агент начинает с полноценным набором компонентов. **1 автоматический recovery attempt перед escalation** (паттерн из codding-agent RecoveryRecipe).
- **Failure type** (enum): mcp_connection_failed | model_unavailable | tool_timeout | tool_auth_failure | context_overflow
- **Recovery action** (enum): retry | fallback | degrade | block
- **Retry count** (number, default 3): количество повторных попыток
- **Backoff** (enum): fixed | exponential
- **Fallback model** (string, optional): запасная модель при model_unavailable
- **Degrade scope**: per-session (degrade mode действует до конца сессии)

#### Agent Policies
Визуальные правила "When [condition] → Do [action]".

**Typed Conditions (dropdown, НЕ free text):**
- `before_tool_call` — перед вызовом tool
- `after_tool_call` — после вызова tool
- `tool_matches(pattern)` — tool соответствует паттерну (e.g. "delete_*")
- `time_range(start, end)` — временной диапазон
- `error_occurred` — произошла ошибка

**Actions:**
- `block` — блокирует tool execution с сообщением агенту
- `log_to_webhook` — логирует событие на webhook
- `notify` — отправляет уведомление на webhook
- `inject_header` — прокидывает custom header в MCP tool requests (ключевой action для auth в MCP серверы)
- `write_audit` — запись в audit log

**Webhook auth:** те же 4 типа (none | api_key | forward_headers | oauth2) для log_to_webhook и notify actions

**AC (Capabilities):**
- AC-CAP-01: Memory config сохраняется и применяется (cross-session, per-user, retention, max entries)
- AC-CAP-02: Knowledge config: upload файла, top-k и threshold влияют на retrieval
- AC-CAP-03: Guardrail: output проверяется по выбранному mode, on-failure действие выполняется
- AC-CAP-04: Output Schema: enforce блокирует несоответствующий ответ
- AC-CAP-05: Escalation: trigger condition → action выполняется
- AC-CAP-06: Recovery: при failure type → recovery action применяется
- AC-CAP-07: Policy rule "tool_matches(delete_*) → block" блокирует вызов

**AC (Output Guardrail — JSON Schema):**
- AC-GRD-JSON-01: Ответ LLM валидируется против JSON Schema
- AC-GRD-JSON-02: При невалидном ответе — retry (до 3 раз)
- AC-GRD-JSON-03: После 3 неудачных retry — fallback или error (по конфигу)
- AC-GRD-JSON-04: JSON Schema editor позволяет редактировать произвольную схему

**AC (Output Guardrail — LLM Judge):**
- AC-GRD-LLM-01: После генерации основного ответа — вызывается judge LLM
- AC-GRD-LLM-02: Judge промпт настраиваемый через UI
- AC-GRD-LLM-03: Judge возвращает yes/no → при no срабатывает on_failure
- AC-GRD-LLM-04: UI чётко показывает что промпт — для judge, не для основного агента

**AC (Output Guardrail — Webhook):**
- AC-GRD-WH-01: Engine отправляет POST с response payload на webhook URL
- AC-GRD-WH-02: Webhook возвращает `{"pass": true/false, "reason": "..."}`
- AC-GRD-WH-03: При pass=false → on_failure action (retry/error/fallback)
- AC-GRD-WH-04: При timeout (10s) → on_failure action
- AC-GRD-WH-05: Auth: Bearer token (настраиваемый в UI)

**AC (Output Schema):**
- AC-SCH-01: Output Schema передаётся как response_format в LLM API
- AC-SCH-02: Output Schema и Output Guardrail могут быть включены одновременно
- AC-SCH-03: Guardrail проверяет ПОСЛЕ генерации, Schema формирует ДО

**AC (Escalation):**
- AC-ESC-01: Терминология "transfer_to_user" (не "human")
- AC-ESC-02: Условия выбираются из dropdown (typed, не CEL)
- AC-ESC-03: confidence_below(0.7) корректно триггерит escalation
- AC-ESC-04: transfer_to_user прерывает агента и передаёт управление пользователю

**AC (Notify — Webhook):**
- AC-NOTIFY-01: Notify webhook отправляет JSON payload
- AC-NOTIFY-02: Auth types: none, api_key, forward_headers, oauth2
- AC-NOTIFY-03: При timeout — логирование, не блокирование агента

**AC (Knowledge — Formats):**
- AC-KB-FMT-01: Загрузка .pdf → успешная индексация
- AC-KB-FMT-02: Загрузка .docx → успешная индексация
- AC-KB-FMT-03: Загрузка .doc → успешная индексация
- AC-KB-FMT-04: Загрузка .txt, .md, .csv → успешная индексация
- AC-KB-FMT-05: Неподдерживаемый формат → внятная ошибка

**AC (Knowledge — File Listing):**
- AC-KB-LIST-01: Загруженный файл появляется в списке
- AC-KB-LIST-02: Отображаются: имя, тип, размер, дата, статус
- AC-KB-LIST-03: Статус корректно переключается: uploading → indexing → ready
- AC-KB-LIST-04: Можно удалить файл из knowledge base
- AC-KB-LIST-05: Можно переиндексировать файл

**AC (Knowledge — Parameters):**
- AC-KB-PARAM-01: top_k настраивается в Knowledge capability config (default: 5)
- AC-KB-PARAM-02: similarity_threshold настраивается в Knowledge capability config (default: 0.75)
- AC-KB-PARAM-03: knowledge_search tool использует значения из agent config

### 7.3 Bottom Panel (resizable)

Два таба, drag handle для resize:

| Таб | Что |
|-----|-----|
| 🤖 AI Assistant | Главный чат — interview, assembly, config |
| ▶ Test Flow | Тест schema: headers, agent selector, chat |

**Поведение Bottom Panel:**
- Resizable: drag handle, min height 150px, max 70% viewport
- Collapse/expand: toggle кнопка (▼/▲), collapsed state = thin bar (40px) с названиями табов
- Persistence: panel state (open/closed, height, active tab) сохраняется в localStorage между навигациями
- На Canvas page: canvas сверху + panel снизу
- На других pages (Agents, MCP, Models, etc.): page content сверху + panel снизу (full width)
- Tab memory: активный таб сохраняется при переходе между страницами

**Test Flow tab UX:**
- Schema selector наследуется из panel header (общий для Assistant и Test Flow)
- HTTP Headers editor: key-value pairs (Add/Remove rows), JSON import button
- Message input + Send button
- SSE response streaming с inline tool calls и reasoning steps
- Каждая сессия → link "View in Inspect" → переход на InspectPage с этой сессией

**AC (Bottom Panel):**
- AC-PANEL-01: Drag handle resize (min 150px, max 70% viewport)
- AC-PANEL-02: Collapse/expand toggle, collapsed = thin bar с tab labels
- AC-PANEL-03: Panel state (height, tab, open/closed) persists across page navigation (localStorage)
- AC-PANEL-04: Bottom panel доступен на ВСЕХ admin pages, не только Canvas

**AC (Test Flow):**
- AC-TESTFLOW-01: Test Flow tab имеет HTTP Headers key-value editor (add/remove/JSON import)
- AC-TESTFLOW-02: Headers прокидываются в agent → MCP tool calls через forward_headers
- AC-TESTFLOW-03: SSE response streaming показывает tool calls и reasoning inline
- AC-TESTFLOW-04: "View in Inspect" link переходит на InspectPage с текущей сессией

### 7.4 Agent Inspection (Inspect Page)

**Session List (paginated table, НЕ вкладки):**

| Колонка | Описание |
|---------|----------|
| Session ID | Short hash (8 chars), clickable → detail |
| Entry Agent | Имя entry agent'а schema |
| Status | completed / running / failed / blocked / timeout |
| Duration | Total wall-clock time |
| Tokens | Total tokens (prompt + completion) |
| Created | Timestamp (relative: "2 min ago", hover = absolute) |

- **Pagination:** 20 sessions per page, numbered pages + prev/next
- **Search:** по Session ID (prefix match) и agent name (contains)
- **Filters:** status dropdown (multi-select), date range picker
- **Sort:** by date (default desc), duration, tokens (clickable column headers)
- **Auto-refresh:** running sessions обновляются через SSE (status, duration)

**Session Detail (step timeline):**

```
← Sessions / Session #a3f2c8e1

Status: ✅ Completed · 4.3s · 2,270 tokens

Step 1 · 💭 Reasoning                              0.3s
  "Пользователь спрашивает о..."

Step 2 · 🔧 search_knowledge                       1.2s
  Input: {"query": "billing FAQ"}        [expand]
  Output: "Для изменения тарифа..."      [expand]

Step 3 · 🧠 memory_recall                          0.1s
  "Клиент обращался 3 дня назад..."      [expand]

Step 4 · 📚 knowledge_search                       0.8s
  Query: "return policy"                 [expand]

Step 5 · 🛡️ guardrail_check                        0.2s
  Mode: JSON Schema · Result: ✅ Pass

Step 6 · ✅ Final Answer                            1.7s
  "Ваш текущий тариф — Pro..."
```

**Unified Step Icons:**

| Kind | Icon | Описание |
|------|------|----------|
| reasoning | 💭 | LLM thinking step |
| tool_call | 🔧 | External tool execution |
| memory_recall | 🧠 | Memory retrieval (auto at session start) |
| memory_store | 🧠 | Memory save (agent decision) |
| knowledge_search | 📚 | RAG document retrieval |
| guardrail_check | 🛡️ | Output validation (JSON/LLM/Webhook) |
| final_answer | ✅ | Agent response to user |
| error | ⚠️ | Error occurred |
| escalation | 🚨 | Escalation triggered |
| task_dispatch | 📤 | Task sent to sub-agent |
| task_result | 📥 | Result received from sub-agent |
| task_timeout | ⏰ | Dead letter — task timed out |

**Dead Letter visibility (from §8.12):**
- Timed-out tasks отображаются с ⏰ icon и причиной: elapsed time, target agent
- Parent action (retry/escalate/abort) отображается как следующий step

**AC (Inspect):**
- AC-INSPECT-01: Session list — paginated table (20 per page), не вкладки
- AC-INSPECT-02: Search по session ID и agent name работает
- AC-INSPECT-03: Filter по status (multi-select dropdown) работает
- AC-INSPECT-04: Session detail — timeline с unified step icons и timing
- AC-INSPECT-05: Dead letter tasks видны с ⏰ icon и причиной timeout
- AC-INSPECT-06: Running sessions auto-refresh через SSE

### 7.5 Production Ready Cleanup

- Баг-фиксы текущего UI
- Дедупликация логики между страницами
- UX polish
- `/admin/agents` = глобальный список агентов (не inline canvas edit)
- AI ассистент в Bottom Panel (на всех страницах), не floating button

### 7.6 Acceptance Criteria

- AC-UI-01: Drill-in: клик на ноду → полноэкранная конфигурация с breadcrumb навигацией
- AC-UI-02: Capability blocks: [+ Add] → dropdown → новый block с inline config
- AC-UI-03: Bottom panel: resizable drag handle, два таба (Assistant / Test Flow)
- AC-UI-04: Inspect: session history с reasoning steps, tool calls, timing
- AC-UI-05: Единственная страница управления агентами (нет отдельного /admin/agents)
- AC-UI-06: Model Parameters: Temperature, Top P, Max Tokens, Stop Sequences настраиваются per-agent (универсальные параметры, поддерживаемые всеми LLM провайдерами)
- AC-UI-07: Tools section отображает tools по тирам (Core/Auto/Self-host/MCP) с реальными именами

---

## 8. Engine Features

### 8.1 Memory (cross-session, per-schema)

**БЛОКЕР.** Лендинг обещает "memory".

- Cross-session persistence (memory per-schema, cross-session by definition — в рамках сессии всё в контексте)
- **Per-schema scope:** агенты в разных схемах имеют раздельные memory spaces (НЕ "per-flow" — термин "Flow" не используется в контексте memory)
- User-managed: пользователь контролирует retention, очистку
- Storage: в tenant storage (PostgreSQL)
- **Default retention: Unlimited** — без авто-удаления из БД
- **Ограничение:** через `max_entries` (лимит записей), не через TTL
- При достижении лимита — FIFO (oldest entries removed first)

**Механизм (Hybrid):**
- `memory_recall` — автоматически в начале каждой сессии, inject релевантный контекст
- `memory_store` — tool, агент самостоятельно решает что важно сохранить
- Пользователь может явно попросить "запомни X" → агент вызывает memory_store
- UI: пользователь просматривает и удаляет записи (AC-MEM-03)

**AC:**
- AC-MEM-01: Агент помнит информацию из предыдущей сессии с тем же пользователем
- AC-MEM-02: Память schema A изолирована от schema B
- AC-MEM-03: Пользователь может просмотреть и очистить memory через UI
- AC-MEM-04: Memory хранится в tenant storage, учитывается в storage quota
- AC-MEM-TERM-01: В UI Memory capability нет упоминания "Flow" — используется "Schema"
- AC-MEM-TERM-02: Memory hint корректно описывает scope: "per-schema, cross-session"
- AC-MEM-RET-01: Default retention = Unlimited (не 30 дней)
- AC-MEM-RET-02: max_entries ограничивает количество, старые записи не удаляются автоматически
- AC-MEM-RET-03: При достижении max_entries — FIFO eviction (oldest entries removed first)

### 8.2 Flows (internal agent pipelines)

Flows = внутренняя агентская логика. Внутри одного `POST /agents/{name}/chat` может работать pipeline.

- Новые edge types: flow, transfer, loop
- Gate nodes с conditions
- Parallel execution (fork/join)
- Edge configuration (full output / field mapping / custom prompt)

**Backend:** расширить domain model (Flow, SpawnPolicy) для flow/transfer/loop + gates.

**AC:**
- AC-FLOW-01: Flow edge: после завершения Agent A автоматически запускается Agent B
- AC-FLOW-02: Transfer edge: Agent A передаёт контекст Agent B и завершается
- AC-FLOW-03: Loop edge: при fail gate → возврат к предыдущему агенту (max_iterations)
- AC-FLOW-04: Gate: auto-condition проверяет output → pass/fail
- AC-FLOW-05: Parallel: несколько flow edges = fork, gate с join = wait all
- AC-FLOW-06: Edge config: field mapping передаёт конкретное поле следующему агенту

### 8.3 Agent Lifecycle States

Явные состояния агента: `initializing → ready → running → needs_input → blocked → degraded → finished`

- SSE event `agent.state_changed`
- Видимо в Cloud UI (badge на ноде)
- `blocked` всегда с structured reason

**AC:**
- AC-STATE-01: Каждый агент имеет явное состояние, доступное через API
- AC-STATE-02: SSE event `agent.state_changed` при каждом переходе
- AC-STATE-03: UI показывает текущее состояние на ноде (badge/icon)
- AC-STATE-04: `blocked` содержит reason, видимый пользователю

### 8.4 Recovery Recipes

Domain-specific recovery per failure type. **1 автоматический recovery attempt перед escalation** (паттерн из codding-agent RecoveryRecipe с EscalationPolicy).

**Degrade scope: per-session.** При новой сессии — полноценная работа с полным набором компонентов.

| Failure Type | Recovery |
|-------------|----------|
| `mcp_connection_failed` | Reconnect один раз → degraded mode (агент работает без этого MCP) |
| `model_unavailable` | Retry с backoff → fallback model (если настроен) → block |
| `tool_timeout` | Retry если idempotent → skip tool → inform user |
| `tool_auth_failure` | No retry → inform user "ключ/доступ невалидный" |
| `context_overflow` | Auto-compact → retry turn |

**Escalation after recovery failure:** AlertHuman, LogAndContinue, Abort (настраивается).

**AC:**
- AC-REC-01: Degrade mode действует до конца сессии
- AC-REC-02: Новая сессия начинается с полноценным набором компонентов
- AC-REC-03: Один автоматический recovery attempt перед escalation
- AC-REC-04: Recovery events видны в Agent Inspection

### 8.5 Event Schema Versioning

Каждый SSE event имеет `schema_version`. Forward compatibility.

**AC:**
- AC-EVT-01: Все SSE events содержат `schema_version` field
- AC-EVT-02: Неизвестные event types безопасно игнорируются клиентом
- AC-EVT-03: Документирован event contract (types, fields, versions)

### 8.6 MCP Auth

Configurable auth per-MCP-server:

| Паттерн | Config |
|---------|--------|
| forward_headers | `auth.type: forward_headers` |
| API key | `auth.type: api_key`, `auth.key_env: ENV_VAR` |
| OAuth2 | `auth.type: oauth2`, `auth.client_id`, `auth.token_store: encrypted` |
| Service account | `auth.type: service_account`, `auth.token_env: ENV_VAR` |

**AC:**
- AC-AUTH-01: MCP server config принимает auth section
- AC-AUTH-02: forward_headers прокидываются от вызывающей системы в MCP calls
- AC-AUTH-03: API key читается из env variable, не хранится в plain text

### 8.7 Agent Policies

Визуальные правила "When [condition] → Do [action]" как capability-блок. Используют **typed conditions** (dropdown, НЕ free text) и **единый webhook auth контракт**.

**Typed Conditions:**
- `before_tool_call` — перед вызовом tool
- `after_tool_call` — после вызова tool
- `tool_matches(pattern)` — tool соответствует паттерну (e.g. "delete_*")
- `time_range(start, end)` — временной диапазон
- `error_occurred` — произошла ошибка

**Actions:**
- `block` — блокирует tool execution с сообщением агенту
- `log_to_webhook` — логирует событие на webhook
- `notify` — отправляет уведомление на webhook
- `inject_header` — прокидывает custom header в MCP tool requests (ключевой action для auth в MCP серверы)
- `write_audit` — запись в audit log

**Webhook auth для log_to_webhook / notify:** none | api_key | forward_headers | oauth2

**Три уровня permission:**
| Уровень | Default? | Что |
|---------|:---:|------|
| Standard | ✅ | Агент работает с назначенными tools. Dangerous = confirm_before |
| Restricted | | Один checkbox. Нет shell/file/external API без confirm |
| Custom | | Policy блок с конкретными правилами |

**AC:**
- AC-POL-01: Policy conditions — typed dropdown (не free text)
- AC-POL-02: inject_header прокидывает custom header в MCP tool requests
- AC-POL-03: Webhook auth для log_to_webhook / notify использует те же 4 auth types
- AC-POL-04: block action корректно блокирует tool execution с сообщением агенту

### 8.8 Tool Architecture (4-Tier Model)

Все tools в ByteBrew делятся на 4 уровня:

#### Tier 1 — Core Native (всегда доступны)
Минимальные tools для работы агента, встроены в engine:
- `ask_user` — взаимодействие с пользователем (safe)
- `show_structured_output` — структурированный вывод (safe)
- `spawn_{agent}` — запуск другого агента (auto-generated per agent)
- `manage_tasks` / `manage_subtasks` — трекинг задач в сессии (safe)
- `wait` — пауза на указанное время без прерывания ReAct цикла (safe)

#### Tier 2 — Capability-Injected (авто-добавляются с capability)
Автоматически доступны при включении capability:
- Memory: `memory_recall`, `memory_store`
- Knowledge: `knowledge_search`
- Escalation: `escalate`

Пользователь НЕ назначает эти tools вручную — они включаются при активации capability.

#### Tier 3 — Self-Hosted Only (заблокированы в Cloud)
Файловые и shell tools доступны ТОЛЬКО в self-hosted deployment:
- File: `read_file`, `write_file`, `edit_file`, `glob`, `grep_search`, `search_code`, `smart_search`
- Code analysis: `get_project_tree`, `get_function`, `get_class`, `get_file_structure`, `lsp`
- Shell: `execute_command`

В Cloud (AC-CLOUD-05) вызов Tier 3 tool → ошибка (не silent fail).

#### Tier 4 — MCP Tools (user-provided)
Все внешние интеграции через MCP серверы:
- Web search (Tavily, Brave, Exa)
- Web fetch
- Domain-specific (Google Sheets, Slack, GitHub, Stripe, etc.)
- Custom бизнес-логика

`web_search` и `web_fetch` — НЕ нативные tools. Доступны только через MCP.

**AC:**
- AC-TOOL-01: Tier 1 tools доступны каждому агенту без дополнительной настройки
- AC-TOOL-02: Tier 2 tools автоматически добавляются при включении capability
- AC-TOOL-03: Tier 3 tools заблокированы в Cloud deployment (ошибка, не silent fail)
- AC-TOOL-04: Tier 4 tools доступны через MCP server configuration
- AC-TOOL-05: web_search доступен ТОЛЬКО через MCP (не нативный tool)

### 8.9 Entity Relationships

**Агенты и MCP серверы — глобальные сущности.** Schemas ссылаются на них, не владеют ими.

**Entity Model:**
```
Agent (global) ←─── referenced by ───→ Schema (contains agent refs)
MCP Server (global) ←── assigned to ──→ Agent
Trigger (per-schema) ──── targets ────→ Agent (entry only)
Gate (per-schema) ──── connects ──────→ Agent (via edges)
```

**Drill-in UX:**
- Клик на агента в canvas → **навигация** на страницу глобального редактирования (не inline edit)
- Кнопка "← Вернуться на Canvas" для быстрого возврата
- Индикатор: "Используется в: Support Schema, Sales Schema"
- Изменение агента затрагивает ВСЕ схемы где он используется
- Аналогичный паттерн для MCP серверов (глобальные, с кросс-ссылками)

**AC:**
- AC-ENT-01: Агент — глобальная сущность (одна конфигурация для всех схем)
- AC-ENT-02: Клик на агента в canvas → навигация на страницу агента
- AC-ENT-03: На странице агента — "Используется в: [список схем]"
- AC-ENT-04: Кнопка "← Вернуться на Canvas" работает
- AC-ENT-05: MCP серверы — аналогичный паттерн (глобальные, с кросс-ссылками)

### 8.10 Persistent Agent Lifecycle

**Spawn vs Persistent sub-agents:**

| Тип | Поведение | Контекст |
|-----|-----------|----------|
| **spawn** | Task → result → контекст уничтожен → прекращает существование | Обнуляется |
| **persistent** | Task → result → контекст сохраняется → ждёт новый task | Накапливается |

**Lifecycle states** (из codding-agent WorkerBoot pattern):
```
Spawning → Ready → Running → [Blocked | Finished]
                       ↑                    |
                       └──── (new task) ────┘  (только для persistent)
```

**Task dispatch communication:**
- Parent создаёт task → persistent/spawn child исполняет → результат возвращается через event
- Паттерн: codding-agent TaskRegistry (`task_registry.rs`)
- Mailbox НЕ нужен для V2 — task dispatch достаточно

**Context preservation (persistent agent):**
- Контекст накапливается между задачами
- Auto-compaction при переполнении контекстного окна
- Явный reset через API

**Edge cases:**
- Parent обнулил контекст → persistent child продолжает работать независимо, принимает задачи от любого агента с правом spawn
- Удаление/пересоздание агента → контекст обнуляется

**AC (BACKEND-DEFERRED):**
- AC-LIFE-01: Spawn sub-agent выполняет задачу и уничтожается (контекст = 0 при повторном spawn)
- AC-LIFE-02: Persistent sub-agent выполняет задачу, сохраняет контекст, принимает вторую задачу с учётом предыдущего контекста
- AC-LIFE-03: Parent reset не влияет на persistent child — child продолжает принимать задачи
- AC-LIFE-04: Persistent agent context auto-compacts при переполнении окна

### 8.11 Inter-Agent Communication

**Механизм:** Task dispatch (НЕ mailbox).

- Parent создаёт task → persistent/spawn child исполняет → результат возвращается через event
- Паттерн: codding-agent TaskRegistry (`task_registry.rs`)

**AC (BACKEND-DEFERRED):**
- AC-COMM-01: Parent agent отправляет task persistent child через task dispatch
- AC-COMM-02: Persistent child возвращает результат через event
- AC-COMM-03: Parent получает результат и продолжает работу

### 8.12 Agent Resilience & Fault Tolerance

Production-ready агентная платформа должна обрабатывать зависания и сбои на всех уровнях: sub-agent, MCP tool, LLM API. В отличие от простых обёрток над LLM, ByteBrew обеспечивает structured fault handling.

**1. Heartbeat / Watchdog для sub-agents:**
- Persistent/spawn sub-agent шлёт heartbeat event каждые `heartbeat_interval` секунд (default: 15s)
- Parent agent имеет watchdog timer: если heartbeat не получен за `2 × heartbeat_interval` → agent считается stuck
- Action при stuck:
  - **spawn agent:** kill + re-spawn с тем же task (retry)
  - **persistent agent:** attempt graceful interrupt → если не ответил за 10s → force kill + escalate to parent
- Heartbeat передаётся через SSE event `agent.heartbeat { agent_id, timestamp, current_step }`

**2. MCP Tool Call Timeout:**
- Отдельный от `max_turn_duration` таймаут per-tool-call: `tool_call_timeout` (default: 30s)
- Если MCP сервер не вернул результат за `tool_call_timeout`:
  - Отменить вызов
  - Вернуть агенту structured error: `{ "error": "tool_timeout", "tool": "search_knowledge", "timeout_ms": 30000 }`
  - Агент решает: retry, skip, или escalate (по Recovery policy)
- Настраивается per-agent и per-MCP-server (server-level override > agent-level default)

**3. Dead Letter Queue для Task Dispatch:**
- Task получает `task_timeout` (default: 5 минут, настраиваемый)
- Если task не перешёл в `completed`/`failed` за timeout:
  - Task status → `timeout`
  - Parent получает event `task.timeout { task_id, agent_id, elapsed_ms }`
  - Parent может: re-dispatch к другому агенту, retry, или escalate
- Dead letter tasks видны в Inspect view с причиной timeout

**4. Circuit Breaker:**
- Per-MCP-server и per-model circuit breaker
- **Open** после 3 consecutive failures (timeout/error) в окне 60s
- **Half-open** через `circuit_reset_interval` (default: 120s) — пробует 1 запрос
- **Closed** если half-open запрос успешен
- В состоянии Open:
  - MCP: агент получает `tool_unavailable` вместо вызова → degraded mode
  - Model: engine пробует fallback model (если настроен) → degraded mode
- Circuit state виден в Admin UI (MCP page status, Agent health)

**Паттерны из codding-agent:**
- `recovery_recipes.rs` → heartbeat + watchdog pattern
- `mcp_lifecycle_hardened.rs` → MCP circuit breaker + degraded reporting
- `plugin_lifecycle.rs` → partial startup, graceful degradation

**AC (BACKEND-DEFERRED):**
- AC-RESIL-01: Sub-agent heartbeat отправляется каждые heartbeat_interval секунд
- AC-RESIL-02: Parent получает event при stuck sub-agent (2× heartbeat miss)
- AC-RESIL-03: Spawn agent при stuck — auto kill + re-spawn с retry
- AC-RESIL-04: Persistent agent при stuck — graceful interrupt → force kill → escalate
- AC-RESIL-05: MCP tool call timeout (30s default) возвращает structured error агенту
- AC-RESIL-06: tool_call_timeout настраивается per-agent и per-MCP-server
- AC-RESIL-07: Task dispatch timeout переводит task в status `timeout`, parent получает event
- AC-RESIL-08: Dead letter tasks видны в Inspect view
- AC-RESIL-09: Circuit breaker opens после 3 consecutive failures
- AC-RESIL-10: В состоянии circuit open — MCP tools возвращают tool_unavailable
- AC-RESIL-11: Circuit breaker half-open через reset interval, closes при успехе
- AC-RESIL-12: Circuit state отображается в Admin UI (MCP page, Agent health)

### 8.13 Testing Infrastructure

Минимальное решение для тестирования авторизованных цепочек (Kilo pattern).

**Компоненты:**

**1. Test Flow tab — HTTP Headers editor:**
- Key-value editor (как GraphQL Playground)
- Пользователь вставляет Zitadel/OAuth header → Engine прокидывает в MCP → MCP проверяет auth

**2. Trigger config — Custom Headers field:**
- Key-value pairs для webhook/cron триггеров

**3. Chat API — optional `headers` field:**
- В запросе `POST /api/v1/agents/{name}/chat` можно передать optional `headers` field
- Engine прокидывает эти headers в MCP tool calls (через forward_headers/inject_header)

**Цепочка:** Пользователь вставляет Zitadel header → Engine прокидывает в MCP → MCP проверяет auth.

**AC:**
- AC-TEST-01: Test Flow tab имеет HTTP Headers editor
- AC-TEST-02: Headers из Test Flow прокидываются в MCP tool calls
- AC-TEST-03: Trigger config имеет Custom Headers (key-value)
- AC-TEST-04 (BACKEND-DEFERRED): Chat API принимает optional headers field

---

## 9. Widget Embed

```html
<script src="https://bytebrew.ai/widget/{widget_id}.js"></script>
```

- Виджет = chat UI для конечных пользователей сайта клиента
- Подключается к конкретной schema / entry agent
- Стилизуется (цвета, позиция, приветственное сообщение)
- SSE streaming

### 9.1 Widget Configuration UX (Admin → Widgets page)

| Поле | Описание | Default |
|------|----------|---------|
| Name | Название виджета | "My Widget" |
| Schema | Привязка к schema (dropdown) | First schema |
| Primary color | Color picker | #6366f1 (indigo) |
| Position | bottom-right / bottom-left | bottom-right |
| Size | compact / standard / full | standard |
| Welcome message | Текст приветствия | "Hi! How can I help?" |
| Placeholder | Текст в message input | "Type a message..." |
| Avatar | Upload image или URL | ByteBrew default icon |
| Domain whitelist | Comma-separated domains | * (all domains) |

### 9.2 Embed Code

```html
<!-- ByteBrew Widget (Cloud) -->
<script src="https://bytebrew.ai/widget/{widget_id}.js"></script>

<!-- ByteBrew Widget (Self-hosted) -->
<script src="https://your-domain.com/widget/{widget_id}.js"></script>
```

**Preview:** Live preview рядом с configuration form — показывает виджет с текущими настройками.

### 9.3 Technical
- Widget загружает lightweight iframe с chat UI
- Auth: widget_id + domain whitelist (CORS origin check)
- Self-hosted: widget JS/CSS раздаётся Engine'ом (`/widget/{id}.js`)

**AC:**
- AC-WID-01: Widget script загружается и показывает chat bubble
- AC-WID-02: Конечный пользователь пишет → агент отвечает через SSE
- AC-WID-03: Widget стилизуется (primary color, position, welcome message)
- AC-WID-04: Widget ID привязан к tenant + schema
- AC-WID-05: Domain whitelist ограничивает embed origins (CORS)
- AC-WID-06: Self-hosted widget script раздаётся Engine'ом
- AC-WID-07: Live preview в Admin показывает текущую конфигурацию виджета

---

## 10. MCP Catalog

### 10.1 Trusted MCP Catalog

Курированный каталог проверенных MCP серверов. Поставляется как **built-in YAML файл** рядом с engine binary (`mcp-catalog.yaml`), обновляется с каждым релизом. **~10-15 серверов в V2.** Внешние реестры (mcp.run, Smithery) — **V3, не V2.**

**Auth types для MCP серверов:** none | api_key | forward_headers | oauth2

**Формат каталога:**
```yaml
catalog_version: "1.0"
servers:
  - name: "tavily-web-search"
    display: "Tavily Web Search"
    description: "AI-optimized web search, content extraction, site crawling"
    category: "search"
    verified: true
    packages:
      - type: "stdio"
        command: "npx"
        args: ["-y", "@mcptools/mcp-tavily"]
        env_vars:
          - name: "TAVILY_API_KEY"
            description: "Get key at tavily.com/app/api-keys"
            required: true
            secret: true
      - type: "remote"
        transport: "streamable-http"
        url_template: "https://mcp.tavily.com/mcp/?tavilyApiKey=${TAVILY_API_KEY}"
        env_vars:
          - name: "TAVILY_API_KEY"
            required: true
            secret: true
      - type: "docker"
        image: "mcp/tavily:latest"
        env_vars:
          - name: "TAVILY_API_KEY"
            required: true
            secret: true
    provided_tools:
      - name: "tavily_search"
        description: "Search the web"
      - name: "tavily_extract"
        description: "Extract content from URLs"
```

**Категории:** search, data, communication, dev-tools, productivity, payments, generic

**Стартовый каталог (~10-15 серверов):**

| Категория | Сервер | Tools |
|-----------|--------|-------|
| Search | Tavily Web Search | tavily_search, tavily_extract, tavily_crawl |
| Search | Brave Search | web_search, local_search |
| Search | Exa | web_search_exa, web_fetch_exa |
| Data | Google Sheets | read_sheet, write_sheet |
| Data | PostgreSQL | query, execute |
| Communication | Slack | send_message, read_channel |
| Communication | Email (Resend) | send_email |
| Dev Tools | GitHub | create_issue, create_pr, search_code |
| Productivity | Notion | search, create_page |
| Productivity | Linear | create_issue, list_issues |
| Payments | Stripe | create_payment, list_customers |
| Generic | HTTP/Webhook | http_request |

### 10.2 Транспорты MCP

**Основные (V2):**

| Тип | Описание | Use case |
|-----|----------|----------|
| stdio | Subprocess (command + args) | Dev, npm packages |
| streamable-http | HTTP POST + SSE response (MCP 2025-03-26) | **Production remote** |
| sse | Server-Sent Events | Legacy MCP servers |

**Дополнительные:**

| Тип | Описание | Use case |
|-----|----------|----------|
| websocket | WebSocket bidirectional | Real-time |
| docker | Docker container с stdio | Production self-hosted |

Auth на remote транспортах: none, API key, forward_headers, OAuth2.

### 10.3 Custom MCP

Пользователь может подключить любой MCP сервер (не из каталога):
- Указать transport type, URL/command, env vars, auth
- Custom MCP доступен всем тарифам (CE, Cloud Free+)

**UI:** MCP server page — **глобальная** (как agents). Два способа добавления:
- "Add from Catalog" — список с поиском, category filter
- "Add Custom Server" — форма (transport, URL/command, env vars, auth)

### 10.4 Acceptance Criteria

- AC-MCP-01: Built-in catalog содержит 10-15 проверенных серверов
- AC-MCP-02: "Add from Catalog" показывает список с поиском и category filter
- AC-MCP-03: "Add Custom Server" позволяет ввести transport, URL/command, env vars, auth
- AC-MCP-04: MCP серверы глобальные (не per-agent, не per-schema)
- AC-MCP-05: Агент может подключать MCP серверы из глобального списка
- AC-MCP-06: MCP server detail page показывает "Used by agents: [список агентов]"
- AC-MCP-07: Клик на agent в списке "Used by" → навигация на agent detail page

---

## 11. Landing Page

### Structure (по образцу Paperclip)

```
Section 1: Hero
  "Not another AI chatbot."
  "ByteBrew — the open-source agent brewery."
  "Describe an operation — agents assemble themselves to run it."
  [Try free]  [GitHub ★]  [Self-host →]

Section 2: What / What Not
  ✅ AI agents that reason, act, and coordinate
  ✅ 2000+ integrations via MCP
  ✅ Self-hosted, open source
  ❌ Not a chatbot wrapper
  ❌ Not a workflow builder
  ❌ Not "AI employee" hype

Section 3: How It Works (3 steps)
  1. Describe — опиши задачу
  2. Brew — ByteBrew собирает агентов
  3. Run — система работает автономно

Section 4: Demo video / Live demo

Section 5: Use Cases
  "Анализ закупок" / "AI-first продукт за неделю" / "Support agent"

Section 6: Pricing
  Free / Pro / Business / Enterprise

Section 7: Engine Capabilities
  Memory, Flows, Gates, Inspect, Recovery, MCP, ReAct...

Section 8: Quick Start
  docker run bytebrew/engine

Section 9: Open Source + Community
  GitHub, Discord, Docs
```

**AC:**
- AC-LAND-01: Landing page загружается, hero section виден
- AC-LAND-02: "Try free" ведёт на registration
- AC-LAND-03: Pricing section отражает актуальные тарифы
- AC-LAND-04: "Self-host" ведёт на docs с docker run инструкцией

---

## 12. Patterns from OMC & coding-agent

### 12.1 codding-agent Pattern Reuse

Переиспользуем **паттерны** (не код — Rust vs Go):

| Паттерн | Источник | Применение в ByteBrew V2 | Приоритет |
|---------|----------|--------------------------|-----------|
| Worker lifecycle state machine | `worker_boot.rs` | Persistent sub-agent states (§8.3, §8.9) | **P0** |
| Task registry + packets | `task_registry.rs` + `task_packet.rs` | Task dispatch (§8.9, §8.10) | **P0** |
| Policy engine (typed conditions) | `policy_engine.rs` | Agent Policies capability (§8.7) | **P1** |
| Recovery recipes | `recovery_recipes.rs` | Recovery capability (§8.4) | **P1** |
| Hook system (pre/post) | `hooks.rs` | Output Guardrail pipeline (§7.2) | **P1** |
| MCP lifecycle (degraded mode) | `mcp_lifecycle_hardened.rs` | MCP partial startup (§10) | **P1** |
| Permission enforcer | `permission_enforcer.rs` | Tool tier enforcement (§8.8) | **P2** |
| Plugin lifecycle | `plugin_lifecycle.rs` | Capability failure handling (§7.2) | **P2** |
| Summary compression | `summary_compression.rs` | Inspect view optimization (§7.4) | **P3** |

### 12.2 Включены в scope (must have)

| Паттерн | Источник | Где в PRD |
|---------|----------|-----------|
| Request routing (keyword + ambiguity detection) | OMC | §5.3 Routing Logic |
| Interview → Assembly pipeline | OMC deep-interview | §5.4 Full Flow |
| Self-test after assembly | OMC autopilot validation | §5.4 Step 5 |
| Recovery recipes per failure type | coding-agent | §8.4 |
| Agent lifecycle state machine | coding-agent + Codex PRD | §8.3 |
| Event schema versioning | Codex PRD | §8.5 |
| Canvas live sync with agent actions | Agent Flow (VS Code ext) | §5.5 |

### 12.3 Post-launch

| Паттерн | Источник | Зачем |
|---------|----------|-------|
| Semantic context compaction | coding-agent | Лучше quality в long sessions |
| Full hook system | coding-agent | Extensibility |
| Circuit breaker | coding-agent | Cascading failure protection |
| Policy engine (declarative rules) | coding-agent | CI/CD-grade automation |
| Green contract (verification levels) | coding-agent | Quality gates |

---

## 13. Implementation Dependencies

```
Phase 1: Foundation (parallel)
  ├── Cloud: tenant_id + middleware + auth integration [backend]
  ├── Cloud: Stripe products + quota enforcement [backend]
  ├── Engine: Memory system [backend]
  ├── Engine: Lifecycle states + SSE events [backend]
  └── UI: Production ready cleanup + drill-in UX [frontend]

Phase 2: Core Features (parallel, depends on Phase 1)
  ├── Engine: Flows (flow/transfer/loop edges + gates) [backend]
  ├── Engine: Recovery recipes [backend]
  ├── Engine: Event schema versioning [backend]
  ├── Engine: MCP auth [backend]
  ├── UI: Canvas sync with assistant actions + animations [frontend]
  ├── UI: Multiple schemas [frontend]
  ├── UI: Gate node + new edge types [frontend]
  └── Cloud: Default model (GLM 4.7) proxy [backend]

Phase 3: Delivery (depends on Phase 1-2)
  ├── Brewery UX: AI assistant routing + interview + assembly [backend + frontend]
  ├── Widget embed [frontend + backend]
  ├── Verified MCP servers (curated list + OAuth) [backend]
  ├── Landing page [frontend]
  └── License: BSL 1.1 in repo [repo]

Phase 4: Polish & Launch
  ├── Integration testing (all flows)
  ├── Security audit (Cloud sandbox, tenant isolation)
  ├── Documentation
  └── "Show HN" preparation
```

---

## 14. Open Source Preparation

### 14.1 Repo Restructuring

Текущая структура `bytebrew/` содержит всё в одном repo:

```
bytebrew/           ← монорепо
├── engine/         ← core engine (Go)
├── engine/admin/   ← admin UI (React)
├── cloud-api/      ← cloud API (Go)
├── cloud-web/      ← landing + website (React)
├── cli/            ← CLI client (TypeScript/Bun)
├── docs/           ← документация
└── ...
```

**Нужно определить:**
- Что выносится в отдельные repos?
- Что остаётся в монорепо?
- Что открываем, что остаётся закрытым?

**Предлагаемая структура:**

| Repo | Содержимое | Open Source? | Лицензия |
|------|-----------|:---:|----------|
| `bytebrew/engine` | Core engine + admin UI | ✅ | BSL 1.1 |
| `bytebrew/cli` | CLI client | ✅ | BSL 1.1 |
| `bytebrew/web-client` | Web chat client | ✅ | BSL 1.1 |
| `bytebrew/examples` | Примеры, templates | ✅ | Apache 2.0 |
| `bytebrew/docs` | Документация | ✅ | CC-BY-4.0 |
| `bytebrew/cloud-api` | Cloud API (billing, auth, tenants) | ❌ закрытый | Proprietary |
| `bytebrew/cloud-web` | Landing page bytebrew.ai | ❌ закрытый | Proprietary |
| `bytebrew/bridge` | WS relay (уже отдельный repo) | ✅ | BSL 1.1 |
| `bytebrew/relay` | Enterprise relay | ❌ закрытый | Proprietary |
| `bytebrew/mobile` | Flutter app (уже отдельный repo) | Решить | BSL 1.1? |

**Принцип:** Engine + клиенты = открыты (BSL 1.1). Cloud infra + website = закрыты. Examples/docs = максимально открыты (Apache/CC).

### 14.2 Repo Cleanup Checklist

Перед open source launch каждый открываемый repo должен:

- [ ] Убрать secrets, API keys, .env файлы из git history (`git filter-repo`)
- [ ] Убрать internal references (cloud-api URLs, внутренние комменты)
- [ ] Добавить LICENSE файл (BSL 1.1 с правильными параметрами)
- [ ] Добавить README.md (описание, quick start, contributing)
- [ ] Добавить CONTRIBUTING.md
- [ ] Добавить CODE_OF_CONDUCT.md
- [ ] Добавить .github/ (issue templates, PR templates, CI workflows)
- [ ] Проверить что build/test работает из чистого clone
- [ ] Убрать vendor lock-in (hardcoded URLs типа bytebrew.ai где не нужно)

### 14.3 BSL 1.1 License Text

Нужен полный текст лицензии для согласования с партнёром.

**Параметры BSL 1.1 для ByteBrew:**

```
Business Source License 1.1

Licensor:          Synthetic Inc.
Licensed Work:     ByteBrew Engine [version]
                   The Licensed Work is (c) [year] Synthetic Inc.

Additional Use Grant:
  You may make use of the Licensed Work, provided that you may not
  use the Licensed Work for a Managed Service.

  A "Managed Service" is a commercial offering that allows third
  parties to access and/or use the Licensed Work or a substantial
  set of the features or functionality of the Licensed Work as a
  service, where the service is provided to third parties on a
  hosted or managed basis.

  For clarity, the following uses are expressly permitted:
  - Self-hosting the Licensed Work for internal business purposes
  - Embedding the Licensed Work within your own product or service
    (e.g., using the API to power AI features in your application)
  - Using the Licensed Work to provide services to your own
    customers, provided that your customers do not directly access
    or operate the Licensed Work itself

Change Date:       Four years from the date of each release
Change License:    Apache License, Version 2.0
```

**Ключевые моменты для партнёра:**
1. "Managed Service" = нельзя хостить ByteBrew и продавать доступ к нему как сервис
2. Embedding разрешён явно — Kilo-кейс (forward_headers) легален
3. Self-hosting для внутренних целей — легален
4. Через 4 года → Apache 2.0 (полностью открыто)

### 14.3.1 CI/CD: Auto-update Change Date при релизе

BSL 1.1 применяется **per-version** — каждый релиз имеет свой Change Date (4 года от даты релиза). Автоматизировать в release workflow:

```yaml
# .github/workflows/release.yml (шаг перед tag/publish)
- name: Update BSL Change Date
  run: |
    CHANGE_DATE=$(date -d '+4 years' +%Y-%m-%d)
    sed -i "s/Change Date:.*/Change Date:          $CHANGE_DATE (four years from release date)/" LICENSE
    git add LICENSE
    git commit -m "chore: update BSL Change Date to $CHANGE_DATE"
```

Каждый git tag содержит LICENSE с актуальным Change Date. Старые теги не трогаются — их Change Date фиксирован на момент релиза.

**AC:**
- AC-OSS-01: LICENSE файл в каждом открытом repo с правильными параметрами
- AC-OSS-02: README с описанием, quick start, license badge
- AC-OSS-03: Git history чистая (нет secrets, нет .env)
- AC-OSS-04: `git clone` + `docker build` + `docker run` → работает из коробки
- AC-OSS-05: Полный текст лицензии отправлен партнёру для верификации
- AC-OSS-06: CI/CD release workflow автоматически обновляет Change Date в LICENSE

### 14.4 Docs & Landing (детализация)

Требования к документации и лендингу описаны в секциях 11 (Landing) и будут детализированы в отдельных implementation plans per-section. В PRD зафиксированы:
- Структура лендинга (9 секций)
- AC для лендинга (AC-LAND-01..04)
- Принцип: docs и landing делаются параллельно с engine/cloud другими агентами

Отдельные планы будут прорабатываться по секциям:
- Landing page content + design
- Documentation structure + content
- Quick start guide
- API reference
- Self-hosting guide (Docker, Helm, bare metal)

### 14.5 Documentation Page (bytebrew.ai/docs)

**Структура:**
- **Getting Started:** Docker quick start → Admin login → Create first schema → First chat
- **Concepts:** Schemas, Agents, Triggers, Gates, Edges, Capabilities, Memory, Knowledge
- **Configuration:** Models (BYOK), MCP Servers, Policies, Recovery, Escalation
- **API Reference:** REST endpoints, SSE event contracts, Webhook payloads
- **Self-Hosting:** Docker Compose, Kubernetes (Helm chart), Bare Metal (systemd + Caddy)
- **Widget:** Embed guide, styling, domain whitelist
- **Examples:** Use cases с full schema configs и curl commands

**AC:**
- AC-DOCS-01: Docs page загружается с sidebar навигацией по секциям
- AC-DOCS-02: Getting Started содержит `docker run` → working admin → first agent → first chat
- AC-DOCS-03: API Reference содержит все REST endpoints с request/response examples

---

## 15. Release Strategy

**V2 first → open source.** Никакого legacy кода.

- Все V2 фичи реализуются как единый релиз
- `flows.yaml` убирается — Schema в DB = source of truth
- Текущий main не выкатывается отдельно
- Первый public release = V2
- Никакого backward compatibility с flows.yaml
- Объём тестирования митигируется через product-level AC + Playwright E2E + автономная верификация

---

## 16. Interview Transcript Summary

### 16.1 Initial Deep Interview (10 rounds)

10 rounds of deep interview, key decisions:

| Round | Decision |
|-------|----------|
| 1 | Весь scope в одном релизе |
| 2 | Параллельная разработка через multi-agent, ~1 неделя |
| 3 | Cloud = full CE + managed + widget + MCP |
| 4 | EE под feature flag, Cloud = новые Stripe products |
| 5 | Default model GLM 4.7, 100 req всем, BYOK на всех тарифах |
| 6 | BSL 1.1 standard (no managed service), без отдельного запрета multi-tenancy |
| 7 | 13 must-have пунктов, agent policies вместо shell hooks, standard/restricted/custom permission |
| 8 | Schemas вместо agents как единица тарификации, memory не лимитируется |
| 9 | Pricing: Free/$29/$99, storage-based limits, BYOK everywhere |
| 10 | Brewery UX = AI assistant as main entry, OMC routing patterns, canvas sync |

### 16.2 Gap Analysis Interview (10 rounds, 23 decisions)

Дополнительные 10 раундов deep interview для закрытия 23 архитектурных и UX пробелов. Ambiguity: 18%.

| # | Decision | Секция PRD |
|---|----------|------------|
| 1 | Persistent Agent Lifecycle: spawn (destroy) vs persistent (keep context) | §8.10 |
| 2 | Mailbox НЕ нужен — Task dispatch достаточно | §8.11 |
| 3 | UX: SVG/Lucide иконки, collapsible sections, 2-колоночный layout | §5.7 |
| 4 | Memory: "Flow" → "Schema" в терминологии | §8.1 |
| 5 | Memory: Retention = Unlimited, лимит через max_entries | §8.1 |
| 6 | Output Guardrail — JSON Schema: пост-валидация, retry max 3 | §7.2 |
| 7 | Output Guardrail — LLM Judge: отдельный LLM call | §7.2 |
| 8 | Output Guardrail — Webhook: POST с контрактом, 10s timeout | §7.2 |
| 9 | Knowledge: PDF, DOCX, DOC, TXT, MD, CSV | §7.2 |
| 10 | Knowledge: File listing UI (table, statuses, actions) | §7.2 |
| 11 | Knowledge: top_k/similarity_threshold в agent config, не в tool args | §7.2 |
| 12 | Output Schema vs Guardrail: раздельные capabilities (pre vs post) | §7.2 |
| 13 | Escalation: transfer_to_user (не human), typed conditions (не CEL) | §7.2 |
| 14 | Notify: единый webhook контракт, 4 auth types | §7.2 |
| 15 | Recovery: degrade scope = per-session, 1 auto attempt | §8.4 |
| 16 | Agent Policies: typed conditions + inject_header action | §8.7 |
| 17 | MCP Catalog: built-in YAML 10-15 + custom, no external registries | §10 |
| 18 | Entity Relationships: Agents и MCP глобальные, canvas → navigate | §8.9 |
| 19 | UI Verification checklist | §7.2 (ACs) |
| 20 | AC Format: product-level для автономного тестирования | All ACs |
| 21 | Testing: Test Flow HTTP Headers, forward headers for Kilo pattern | §8.12 |
| 22 | codding-agent: 9 паттернов (P0-P3) для переиспользования | §12.1 |
| 23 | V2 first, no legacy, no flows.yaml | §15 |
