# ByteBrew Engine — Regression Test Cases

Полный список тест-кейсов для регрессионного тестирования.
При добавлении нового функционала — обновлять этот файл.

**Последнее обновление:** 2026-03-22
**Модель для тестов:** OpenRouter `qwen/qwen3-coder-next`
**Принцип:** все команды берутся с публичного сайта bytebrew.ai

---

## TC-SITE: Сайт bytebrew.ai (12 TC)

### TC-SITE-01: Landing page
- Открыть https://bytebrew.ai
- **Ожидание:** Hero "Add an AI agent to your product", code sample, CTA кнопки

### TC-SITE-02: Navigation links
- Кликнуть каждую ссылку: Docs, Examples, Pricing, Download, Login
- **Ожидание:** все страницы загружаются, нет 404

### TC-SITE-03: Docs index
- Открыть https://bytebrew.ai/docs/
- **Ожидание:** Quick Start page (не splash, не 404)

### TC-SITE-04: All doc pages accessible
- Проверить все 29 страниц документации → HTTP 200
- **Список:** getting-started/(quick-start, configuration, api-reference); admin/(login, agents, models, mcp-servers, tasks, triggers, api-keys, settings, config-management, audit-log); concepts/(agents, multi-agent, tools, tasks, knowledge, triggers); deployment/(docker, model-selection, production); integration/(rest-api, multi-agent, byok); examples/(hr-assistant, support-agent, sales-agent)

### TC-SITE-05: Search
- /docs/ → Ctrl+K → ввести "agent"
- **Ожидание:** результаты поиска

### TC-SITE-06: Light theme readability
- Переключить на Light → проверить: h1/h2 тёмные, body text контрастный, sidebar headings читаемые, callouts оранжевые, code blocks видны, tables с borders

### TC-SITE-07: Dark theme readability
- Переключить на Dark → всё читаемо

### TC-SITE-08: llms.txt
- curl https://bytebrew.ai/llms.txt → 200, ссылки на документацию

### TC-SITE-09: docker-compose.yml download
- curl https://bytebrew.ai/releases/docker-compose.yml → 200, валидный YAML

### TC-SITE-10: Favicon
- /docs/ → favicon кружка SVG (не default)

### TC-SITE-11: Cache headers
- HTML: Cache-Control: no-cache
- CSS (_astro/): Cache-Control: immutable

### TC-SITE-12: /examples/ page
- Открыть https://bytebrew.ai/examples
- **Ожидание:** 3 карточки (HR Assistant, Support Agent, Sales Agent), feature tags, "Try Demo →"

### TC-SITE-13: Landing — порты и URL в примерах
- Hero code sample → `localhost:8443` (не 8080)
- Step 2 Deploy → `curl localhost:8443/api/v1/health` (не 8080/health)
- Step 3 Integrate → `curl localhost:8443/api/v1/agents/{name}/chat` (не 8080/v1/chat)
- **Ожидание:** все порты и пути соответствуют реальному docker-compose

### TC-SITE-14: Landing — SSE event types
- Step 3/response example → event types: `message_delta`, `message`, `done` (не `content`)
- **Ожидание:** совпадают с API Reference

### TC-SITE-15: Landing — скриншоты
- Секция "See it in action" → Web Client screenshot загружается
- Admin Dashboard screenshot загружается
- Step 1 → Admin Dashboard screenshot загружается
- **Ожидание:** все img видны (нет broken image alt text)

### TC-SITE-16: Landing — YAML примеры
- YAML в hero и Step 1 → tools: web_search (не knowledge_search)
- **Ожидание:** tools соответствуют документации Quick Start

---

## TC-INST: Docker Installation (7 TC)

### TC-INST-01: Download docker-compose
- curl команда из Quick Start → файл скачивается

### TC-INST-02: docker compose up
- Контейнеры стартуют, db healthcheck OK

### TC-INST-03: Health check
- curl из Quick Start → `{"status":"ok",...}`
- **Edge case:** порт 8443 в документации = реальный порт

### TC-INST-04: Admin Dashboard accessible
- URL из Quick Start Step 5 → Login форма
- **Ожидание:** default credentials admin/changeme работают

### TC-INST-05: Update Engine
- docker compose pull && up -d → рестарт без потери данных

### TC-INST-06: Clean shutdown
- docker compose down -v → всё удалено

### TC-INST-07: Idempotent restart
- down + up → health OK

---

## TC-ADMIN: Admin Dashboard (18 TC)

### TC-ADMIN-01: Login correct → /admin/health
### TC-ADMIN-02: Login wrong → error, stays /admin/login (не /login)
### TC-ADMIN-03: Logo visible (не broken image)
### TC-ADMIN-04: Health page → Status ok, Version, Uptime, Agents
### TC-ADMIN-05: Agents CRUD (create, edit, delete, empty name → error)
### TC-ADMIN-06: Models CRUD (create, duplicate name → error)
### TC-ADMIN-07: MCP Servers page → empty state + "Add Custom"
### TC-ADMIN-08: Triggers page → empty state + "Add Trigger"
### TC-ADMIN-09: API Keys → Generate → bb_ token
### TC-ADMIN-10: Settings → BYOK toggles, logging level, security section
### TC-ADMIN-11: Config → Export/Import/Reload
### TC-ADMIN-12: Audit Log → entries with filters
### TC-ADMIN-13: SPA refresh /admin/agents → не 404
### TC-ADMIN-14: Logout → /admin/login
### TC-ADMIN-15: Text readability → нет белого на белом
### TC-ADMIN-16: Forms → labels, inputs, buttons visible
### TC-ADMIN-17: Sidebar → active link highlighted
### TC-ADMIN-18: Empty state → "No agents configured"

---

## TC-API: REST API (12 TC)

### TC-API-01: POST /auth/login → JWT token
### TC-API-02: POST /auth/login wrong password → 401 "invalid credentials"
### TC-API-03: GET /health → 200 JSON
### TC-API-04: GET /agents без auth → 401 "unauthorized"
### TC-API-05: Models CRUD → create + list
### TC-API-06: Agents CRUD → create + list + get
### TC-API-07: PUT agent → name preserved (не пустой)
### TC-API-08: DELETE agent → 404 after delete
### TC-API-09: Duplicate agent name → error (не 500)
### TC-API-10: Invalid model type → no crash
### TC-API-11: Config export → YAML
### TC-API-12: Config reload → 200

---

## TC-CHAT: SSE Chat (7 TC)

### TC-CHAT-01: Simple chat → SSE stream (message_delta, message, done)
### TC-CHAT-02: Event types match API Reference docs
### TC-CHAT-03: Session persistence → agent remembers previous messages
### TC-CHAT-04: Nonexistent agent → "agent not found" (не panic)
### TC-CHAT-05: Agent without model → error (не 500)
### TC-CHAT-06: Chat without auth → 401
### TC-CHAT-07: Invalid session_id → starts fresh (не crash)

---

## TC-DOC: Documentation accuracy (8 TC)

### TC-DOC-01: Ports match → 8443 in Quick Start = docker-compose
### TC-DOC-02: curl commands from Quick Start work
### TC-DOC-03: SSE event types in API Reference = real (message_delta)
### TC-DOC-04: Login endpoint POST /auth/login documented
### TC-DOC-05: Default credentials admin/changeme + .env documented
### TC-DOC-06: Update instructions (docker compose pull + up -d) documented
### TC-DOC-07: host.docker.internal explained in Deployment/Docker
### TC-DOC-08: Example pages (hr-assistant, support-agent, sales-agent) have Quick Start + agents.yaml + example conversations

---

## TC-CLOUD: Cloud Web bytebrew.ai (10 TC)

### TC-CLOUD-01: /examples page loads
- 3 карточки, feature tags, "Try Demo →"

### TC-CLOUD-02: /examples/hr-assistant page loads
- "What this demonstrates", chat UI, suggestion chips, "Run it yourself", GitHub link

### TC-CLOUD-03: /examples/support-agent page loads
### TC-CLOUD-04: /examples/sales-agent page loads

### TC-CLOUD-05: Auth popup trigger
- Без авторизации → нажать Send → popup авторизации
- **Ожидание:** email/password + Google + forgot password

### TC-CLOUD-06: Auth popup login
- Ввести email/password → Sign in → popup закрывается, сообщение отправляется

### TC-CLOUD-07: Dashboard links
- Dashboard → "Documentation" → bytebrew.ai/docs/ (не docs.bytebrew.ai, не 404)
- Dashboard → "GitHub" → github.com/syntheticinc/bytebrew-examples (не #)
- Dashboard → "Installation Guide" → /download page

### TC-CLOUD-07b: Navigation after auth
- После логина → навигация содержит: Docs, Examples, Pricing, Download (не только Dashboard/Settings)
- **Ожидание:** те же ссылки что и до логина + Dashboard/Settings

### TC-CLOUD-08: /examples/ → HR → chat → реальный ответ от Engine
- Отправить "What's the PTO policy?" → SSE streaming от hosted demo (не mock)

### TC-CLOUD-09: Rate limit display
- Отправить сообщение → "14/15 messages remaining"
- Отправить 15 → "0/15 messages remaining" → input disabled

### TC-CLOUD-10: Session persistence in demo
- Message 1 → Message 2 (same session) → agent remembers context

---

## TC-EXAMPLE: bytebrew-examples repo (10 TC)

### TC-EXAMPLE-01: HR Assistant self-hosted + Web Client
- git clone → cd hr-assistant → cp .env.example .env → docker compose up -d
- **Ожидание:** engine + db + mcp-server + web-client стартуют, health OK
- **Ожидание:** http://localhost:3000 → Web Client загружается
- **Ожидание:** Web Client показывает agent hr-assistant в списке
- **Ожидание:** можно отправить сообщение → SSE streaming → tool calls видны

### TC-EXAMPLE-02: HR — Knowledge Search (RAG)
- POST /chat/hr-assistant {"message": "What's the PTO policy for employees with 2+ years?"}
- **Ожидание:** agent вызывает knowledge_search tool
- **Ожидание:** ответ содержит данные из pto-policy.md (accrual table)
- **Ожидание:** ответ содержит конкретные цифры (15/20/25 days by tenure)
- **Это агент, не LLM:** использует RAG для поиска по knowledge base (57 chunks, 5 документов)

### TC-EXAMPLE-03: HR — Employee Lookup + Leave Balance (MCP tools)
- POST /chat/hr-assistant {"message": "Look up employee EMP001 and check their leave balance"}
- **Ожидание:** agent вызывает get_employee tool → возвращает данные сотрудника
- **Ожидание:** agent вызывает get_leave_balance tool → возвращает остатки отпусков
- **Это агент, не LLM:** использует MCP tools для работы с HR данными

### TC-EXAMPLE-04: HR — Escalation trigger
- POST /chat/hr-assistant {"message": "I have a complex situation and I need to escalate this to a human"}
- **Ожидание:** agent распознаёт ключевое слово "escalate" или "need human"
- **Ожидание:** agent говорит что эскалирует к HR специалисту
- **Это агент, не LLM:** реагирует на configured escalation triggers

### TC-EXAMPLE-05: Support — Technical diagnostics (MCP tools)
- POST /chat/support-router {"message": "My API is returning 500 errors since this morning"}
- **Ожидание:** agent вызывает check_service_status tool → статус API gateway
- **Ожидание:** agent вызывает get_customer или get_error_logs для диагностики
- **Ожидание:** ответ содержит данные диагностики (uptime, error rate)
- **Это агент, не LLM:** использует MCP tools для реальной диагностики

### TC-EXAMPLE-06: Support — Billing + Customer lookup (MCP tools)
- POST /chat/support-router {"message": "I was double-charged, my customer ID is CUST-001"}
- **Ожидание:** agent вызывает get_customer tool → данные клиента
- **Ожидание:** agent анализирует подписку и предлагает действия (refund, ticket)
- **Это агент, не LLM:** использует MCP tools для работы с клиентскими данными

### TC-EXAMPLE-07: Sales — Product search (MCP tools)
- POST /chat/sales-agent {"message": "I need 5 laptops for my team, budget $1200 each"}
- **Ожидание:** agent вызывает search_products tool → список ноутбуков
- **Ожидание:** ответ содержит конкретные продукты с ценами и спецификациями
- **Это агент, не LLM:** использует MCP tools для реального каталога товаров

### TC-EXAMPLE-08: Sales — Discount with business rules (Settings)
- POST /chat/sales-agent {"message": "Can I get a 20% bulk discount?"}
- **Ожидание:** agent вызывает get_settings tool → max_discount_percent=15
- **Ожидание:** agent отказывает или предлагает максимум 15%
- **Это агент, не LLM:** следует бизнес-правилам из Settings API

### TC-EXAMPLE-09: Hosted demos health
- curl https://bytebrew.ai/examples/hr-assistant/api/v1/health → 200
- curl https://bytebrew.ai/examples/support-agent/api/v1/health → 200
- curl https://bytebrew.ai/examples/sales-agent/api/v1/health → 200

### TC-EXAMPLE-10: Hosted demos — real agent responses via UI
- Отправить сообщение через UI на bytebrew.ai/examples/hr-assistant
- Отправить сообщение через UI на bytebrew.ai/examples/sales-agent
- **Ожидание:** реальные ответы от Engine (не mock), streaming SSE
- **Ожидание:** MCP tool calls видны inline между текстом (tool name, arguments, result)
- **Ожидание:** tool call results expandable (клик → полный JSON)

### TC-EXAMPLE-11: Rate limit persistence
- Отправить несколько сообщений → счётчик уменьшается
- Обновить страницу (F5) → счётчик НЕ сбрасывается (синхронизация с сервером)
- **Ожидание:** счётчик совпадает с серверным значением из /health endpoint
- **Ожидание:** при исчерпании лимита → input disabled, сообщение "Rate limit reached"

### TC-EXAMPLE-12: Web Client в docker-compose
- git clone → cd hr-assistant → docker compose up -d
- **Ожидание:** web-client доступен на http://localhost:3000
- **Ожидание:** login admin/changeme → sidebar с агентами → chat работает
- **Ожидание:** tool calls видны с expandable деталями, markdown rendering

---

## Итого: 91 TC

| Категория | Кол-во | Покрытие |
|-----------|--------|----------|
| TC-SITE | 16 | Сайт, docs, темы, скриншоты, порты |
| TC-INST | 7 | Docker install, update, restart |
| TC-ADMIN | 18 | Dashboard CRUD, auth, UX |
| TC-API | 12 | REST API, errors |
| TC-CHAT | 7 | SSE streaming, sessions |
| TC-DOC | 8 | Документация = реальность |
| TC-CLOUD | 11 | /examples/, auth popup, dashboard links |
| TC-EXAMPLE | 12 | **Агентное поведение** (MCP tools, RAG, rate limit, web-client) |
| **ВСЕГО** | **91** |

### Примечания к TC-EXAMPLE
- Multi-agent spawn (can_spawn) работает через gRPC/WS path (CLI, mobile), но не через HTTP REST API path (web demos)
- Hosted demos демонстрируют: MCP tool calls, Knowledge Search (RAG), parallel tool execution, escalation triggers, business rules через Settings
- confirm_before работает через gRPC path, в HTTP REST API path confirmation events передаются но UI не обрабатывает их интерактивно
