# ByteBrew Engine — Regression Test Cases

Полный список тест-кейсов для регрессионного тестирования.
При добавлении нового функционала — обновлять этот файл.

**Последнее обновление:** 2026-03-22

---

## TC-SITE: Сайт bytebrew.ai

### TC-SITE-01: Landing page
- Открыть https://bytebrew.ai
- **Ожидание:** Hero "Add an AI agent to your product" видимый, code sample загружается

### TC-SITE-02: Navigation links
- Кликнуть Docs → /docs/
- Кликнуть Pricing → /pricing (или scroll)
- Кликнуть Download → /download
- **Ожидание:** все страницы загружаются, нет 404

### TC-SITE-03: Docs index
- Открыть https://bytebrew.ai/docs/
- **Ожидание:** Quick Start page (не splash, не 404)

### TC-SITE-04: All doc pages accessible
- Проверить все 29 страниц документации → HTTP 200
- **Список:** getting-started/quick-start, configuration, api-reference; admin/login, agents, models, mcp-servers, tasks, triggers, api-keys, settings, config-management, audit-log; concepts/agents, multi-agent, tools, tasks, knowledge, triggers; deployment/docker, model-selection, production; integration/rest-api, multi-agent, byok; examples/sales-agent, support-agent, devops-monitor, iot-analyzer

### TC-SITE-05: Search
- Открыть /docs/ → Ctrl+K → ввести "agent"
- **Ожидание:** результаты поиска появляются

### TC-SITE-06: Light theme readability
- Переключить на Light theme
- **Проверить:**
  - h1 "Quick Start" — тёмный, чёткий
  - h2 headings — тёмные
  - Body text — контрастный
  - Sidebar section headings — читаемые
  - Sidebar links — тёмные
  - Callout заголовки — оранжевые (brand accent)
  - Callout body text — тёмный, читаемый
  - Inline code — видимый
  - Code blocks — с контрастным фоном
  - Tables — borders видны
  - Right sidebar "On this page" — текст виден

### TC-SITE-07: Dark theme readability
- Переключить на Dark theme
- **Проверить:** заголовки, текст, code, callouts, sidebar — всё читаемо на тёмном фоне

### TC-SITE-08: llms.txt
- curl https://bytebrew.ai/llms.txt
- **Ожидание:** 200, plain text с ссылками на документацию

### TC-SITE-09: docker-compose.yml download
- curl -fsSL https://bytebrew.ai/releases/docker-compose.yml
- **Ожидание:** 200, валидный YAML с engine + db services

### TC-SITE-10: Favicon
- Открыть /docs/ → проверить favicon в tab
- **Ожидание:** кружка SVG (не Starlight default, не vite default)

### TC-SITE-11: Cache headers
- curl -sI https://bytebrew.ai/docs/getting-started/quick-start/ → Cache-Control: no-cache
- curl -sI https://bytebrew.ai/docs/_astro/{hash}.css → Cache-Control: immutable
- **Ожидание:** HTML не кэшируется, assets кэшируются

---

## TC-INST: Docker Installation

### TC-INST-01: Download docker-compose
- Скопировать curl команду из Quick Start Step 1
- **Ожидание:** docker-compose.yml скачивается

### TC-INST-02: docker compose up
- `docker compose up -d`
- **Ожидание:** engine + db контейнеры стартуют, db healthcheck OK

### TC-INST-03: Health check
- Скопировать curl из Quick Start → выполнить
- **Ожидание:** `{"status":"ok","version":"...","agents_count":0}`
- **Edge case:** порт в документации = реальный порт docker-compose

### TC-INST-04: Admin Dashboard accessible
- Открыть URL из Quick Start Step 5
- **Ожидание:** Login форма загружается

### TC-INST-05: Update Engine
- `docker compose pull && docker compose up -d`
- **Ожидание:** новый image скачивается, контейнер пересоздаётся, данные сохраняются
- **Edge case:** agents/models созданные до обновления — на месте

### TC-INST-06: Clean shutdown
- `docker compose down -v`
- **Ожидание:** контейнеры и volumes удалены

### TC-INST-07: Idempotent restart
- `docker compose down && docker compose up -d`
- **Ожидание:** всё работает заново, health OK

---

## TC-ADMIN: Admin Dashboard

### TC-ADMIN-01: Login correct password
- Ввести admin / changeme → Sign in
- **Ожидание:** redirect на /admin/health, Dashboard загружается

### TC-ADMIN-02: Login wrong password
- Ввести admin / wrongpassword → Sign in
- **Ожидание:** ошибка "Invalid credentials", остаёмся на /admin/login
- **Edge case:** НЕ redirect на /login (без /admin/ prefix)

### TC-ADMIN-03: Logo visible
- После логина → sidebar
- **Ожидание:** ByteBrew логотип видимый (не broken image)

### TC-ADMIN-04: Health page
- /admin/health
- **Ожидание:** Status: ok, Version, Uptime, Agents count — все поля заполнены

### TC-ADMIN-05: Agents CRUD
- Agents → Create Agent → заполнить name, system prompt, model → Save
- **Ожидание:** агент в списке
- Edit → изменить system prompt → Save
- Delete → подтвердить
- **Edge case:** создать агента с пустым именем → ошибка

### TC-ADMIN-06: Models CRUD
- Models → Add Model → заполнить name, type, model_name, base_url → Save
- **Ожидание:** модель в списке
- **Edge case:** дублирующееся имя → ошибка

### TC-ADMIN-07: MCP Servers page
- /admin/mcp
- **Ожидание:** страница загружается, empty state или список

### TC-ADMIN-08: Triggers page
- /admin/triggers
- **Ожидание:** страница загружается

### TC-ADMIN-09: API Keys
- API Keys → Create → select scopes → Create
- **Ожидание:** token bb_... показывается
- **Edge case:** token не показывается повторно

### TC-ADMIN-10: Settings page
- /admin/settings
- **Ожидание:** настройки загружаются

### TC-ADMIN-11: Config export
- Config → Export
- **Ожидание:** YAML скачивается/показывается

### TC-ADMIN-12: Audit Log
- /admin/audit
- **Ожидание:** записи о действиях (login, create agent, etc.)

### TC-ADMIN-13: SPA refresh
- Находясь на /admin/agents → F5
- **Ожидание:** страница перезагружается корректно (не 404)

### TC-ADMIN-14: Session expired
- Удалить JWT из localStorage → перейти на /admin/agents
- **Ожидание:** redirect на /admin/login

### TC-ADMIN-15: Text readability
- Пройти по всем страницам admin
- **Ожидание:** нет белого текста на белом фоне, все labels/inputs/buttons контрастны

### TC-ADMIN-16: Forms visibility
- Create Agent form → все поля видимы
- Create Model form → все поля видимы
- **Ожидание:** labels, placeholders, borders видны

### TC-ADMIN-17: Sidebar active state
- Кликнуть на разные ссылки в sidebar
- **Ожидание:** активная ссылка выделена цветом/фоном

### TC-ADMIN-18: Empty state
- При пустом списке agents/models
- **Ожидание:** "No agents configured" / "No models configured" — не пустая страница

---

## TC-API: REST API

### TC-API-01: Login
- POST /api/v1/auth/login {"username":"admin","password":"changeme"}
- **Ожидание:** 200, {"token":"eyJ...","expires_at":"..."}

### TC-API-02: Login wrong password
- POST /api/v1/auth/login {"username":"admin","password":"wrong"}
- **Ожидание:** 401

### TC-API-03: Health
- GET /api/v1/health
- **Ожидание:** 200, {"status":"ok",...}

### TC-API-04: Agents without auth
- GET /api/v1/agents (no Authorization header)
- **Ожидание:** 401

### TC-API-05: Models CRUD
- POST /api/v1/models → create
- GET /api/v1/models → list
- **Ожидание:** модель создаётся и видна в списке

### TC-API-06: Agents CRUD
- POST /api/v1/agents → create
- GET /api/v1/agents → list
- GET /api/v1/agents/{name} → details
- **Ожидание:** агент создаётся, видна в списке

### TC-API-07: PUT agent preserves name
- PUT /api/v1/agents/{name} {"tools": []}
- GET /api/v1/agents/{name}
- **Ожидание:** name не стал пустым

### TC-API-08: DELETE agent
- DELETE /api/v1/agents/{name}
- GET /api/v1/agents/{name}
- **Ожидание:** 404 after delete

### TC-API-09: Duplicate agent name
- POST /api/v1/agents с уже существующим name
- **Ожидание:** ошибка (не 500 Internal Server Error)

### TC-API-10: Invalid model type
- POST /api/v1/models {"type": "invalid_type", ...}
- **Ожидание:** ошибка валидации (не 500)

### TC-API-11: Config export
- GET /api/v1/config/export
- **Ожидание:** 200, YAML с agents и models

### TC-API-12: Config reload
- POST /api/v1/config/reload
- **Ожидание:** 200

---

## TC-CHAT: SSE Chat

### TC-CHAT-01: Simple chat
- POST /api/v1/agents/{name}/chat {"message":"Hello"}
- **Ожидание:** SSE stream с events message_delta, message, done

### TC-CHAT-02: Event types match docs
- Проверить что реальные event types совпадают с API Reference
- **Ожидание:** message_delta, message, done (не content)

### TC-CHAT-03: Session persistence
- Message 1: "My name is Alex"
- Message 2 (same session_id): "What is my name?"
- **Ожидание:** агент помнит имя (зависит от модели, проверять через engine logs)

### TC-CHAT-04: Nonexistent agent
- POST /api/v1/agents/nonexistent/chat
- **Ожидание:** error event или HTTP error (не 500 panic)

### TC-CHAT-05: Agent without model
- Создать агента без model_id → chat
- **Ожидание:** error (не 500)

### TC-CHAT-06: Chat without auth
- POST /api/v1/agents/{name}/chat без Authorization header
- **Ожидание:** 401

### TC-CHAT-07: Invalid session_id
- POST /api/v1/agents/{name}/chat {"session_id":"nonexistent-id","message":"Hi"}
- **Ожидание:** starts fresh session (no error)

---

## TC-WEB: Web-client E2E

### TC-WEB-01: Web-client starts
- cd bytebrew-web-client && npm install && npm run dev
- **Ожидание:** http://localhost:3012 доступен

### TC-WEB-02: Login page
- Открыть http://localhost:3012
- **Ожидание:** Login форма с username/password

### TC-WEB-03: Login → agent list
- Ввести admin/changeme → Submit
- **Ожидание:** список агентов

### TC-WEB-04: Select agent → chat
- Кликнуть на агента
- **Ожидание:** chat UI с input полем

### TC-WEB-05: Send message → streaming
- Ввести "Hello" → Send
- **Ожидание:** ответ появляется посимвольно (SSE streaming)

### TC-WEB-06: Response renders
- Отправить сообщение требующее markdown
- **Ожидание:** bold, code blocks, lists рендерятся

### TC-WEB-07: Session sidebar
- Отправить несколько сообщений
- **Ожидание:** history видна в sidebar

### TC-WEB-08: Switch agent
- Переключить на другого агента
- **Ожидание:** новая сессия, старый чат не виден

### TC-WEB-09: Wrong password
- Ввести неправильный пароль
- **Ожидание:** ошибка, остаёмся на login

---

## TC-DOC: Documentation accuracy

### TC-DOC-01: Ports match
- Порт в Quick Start = порт в docker-compose.yml
- **Ожидание:** 8443 везде

### TC-DOC-02: curl commands work
- Скопировать ВСЕ curl из Quick Start → выполнить
- **Ожидание:** каждый работает как описано

### TC-DOC-03: SSE event types match
- Event types в API Reference = реальные при chat
- **Ожидание:** message_delta, message, done

### TC-DOC-04: Login endpoint documented
- API Reference содержит POST /api/v1/auth/login
- **Ожидание:** curl пример, формат ответа

### TC-DOC-05: Default credentials documented
- Quick Start содержит admin/changeme и инструкцию как менять
- **Ожидание:** .env пример

### TC-DOC-06: Update instructions
- Deployment/Docker содержит секцию "Updating the Engine"
- **Ожидание:** docker compose pull + up -d

### TC-DOC-07: host.docker.internal
- Deployment/Docker объясняет host.docker.internal
- **Ожидание:** что это, зачем, как использовать для Ollama

### TC-DOC-08: Examples reproducible
- Examples в docs содержат пошаговые инструкции
- **Ожидание:** можно воспроизвести от начала до конца

---

## TC-EXAMPLE: bytebrew-examples repo

### TC-EXAMPLE-01: Clone and setup
- git clone bytebrew-examples → cd company-assistant → cp .env.example .env
- **Ожидание:** файлы на месте, README понятен

### TC-EXAMPLE-02: docker compose up
- docker compose up -d
- **Ожидание:** engine + db + mcp-server стартуют

### TC-EXAMPLE-03: Health check
- curl http://localhost:8443/api/v1/health
- **Ожидание:** 200 ok

### TC-EXAMPLE-04: Agents preconfigured
- GET /api/v1/agents
- **Ожидание:** supervisor, hr-agent, it-support в списке

### TC-EXAMPLE-05: Chat works
- POST /api/v1/agents/supervisor/chat {"message":"I need to check my leave balance"}
- **Ожидание:** supervisor routes to hr-agent, tool call get_leave_balance visible in SSE
