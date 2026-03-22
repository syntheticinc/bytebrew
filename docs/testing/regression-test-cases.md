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

## TC-EXAMPLE: bytebrew-examples repo (8 TC)

### TC-EXAMPLE-01: HR Assistant self-hosted
- git clone → cd hr-assistant → cp .env.example .env → docker compose up -d
- **Ожидание:** engine + db + mcp-server + service стартуют, health OK

### TC-EXAMPLE-02: HR chat works
- POST /api/v1/chat/hr-assistant {"message": "What's the PTO policy?"}
- **Ожидание:** knowledge_search tool call, policy answer

### TC-EXAMPLE-03: Support Agent self-hosted
- cd support-agent → docker compose up -d → health OK

### TC-EXAMPLE-04: Support multi-agent spawn
- POST /api/v1/chat/support-router {"message": "My API returns 500 errors"}
- **Ожидание:** agent_spawn event → technical agent → parallel diagnostics

### TC-EXAMPLE-05: Sales Agent self-hosted
- cd sales-agent → docker compose up -d → health OK

### TC-EXAMPLE-06: Sales confirmation flow
- POST /api/v1/chat/sales-agent {"message": "Create a quote for 5 ThinkPads"}
- **Ожидание:** confirmation_required event → approve → quote created

### TC-EXAMPLE-07: Hosted demos health
- curl https://bytebrew.ai/examples/hr-assistant/api/v1/health → 200
- curl https://bytebrew.ai/examples/support-agent/api/v1/health → 200
- curl https://bytebrew.ai/examples/sales-agent/api/v1/health → 200

### TC-EXAMPLE-08: Hosted demo chat
- POST https://bytebrew.ai/examples/hr-assistant/api/v1/chat/hr-assistant → real SSE response (не mock)

---

## Итого: 88 TC

| Категория | Кол-во |
|-----------|--------|
| TC-SITE | 16 |
| TC-INST | 7 |
| TC-ADMIN | 18 |
| TC-API | 12 |
| TC-CHAT | 7 |
| TC-DOC | 8 |
| TC-CLOUD | 11 |
| TC-EXAMPLE | 8 |
| **ВСЕГО** | **87** |
