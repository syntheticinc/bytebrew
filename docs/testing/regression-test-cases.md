# ByteBrew Engine — Regression Test Cases

Полный список тест-кейсов для регрессионного тестирования.
При добавлении нового функционала — обновлять этот файл.

**Последнее обновление:** 2026-03-23
**Модель для тестов:** OpenRouter `qwen/qwen3-coder-next`
**Принцип:** все команды берутся с публичного сайта bytebrew.ai

---

## TC-SITE: Сайт bytebrew.ai (16 TC)

### TC-SITE-01: Landing page
**Шаги:**
1. Открыть https://bytebrew.ai в браузере

**Ожидание:**
- Hero-секция содержит заголовок "Add an AI agent to your product"
- Присутствует code sample (YAML-блок с конфигурацией агента)
- CTA-кнопки "Get Started" и "View Examples" видны и кликабельны

**PASS:** Landing page загружена, hero/code sample/CTA отображаются корректно

### TC-SITE-02: Navigation links
**Шаги:**
1. Открыть https://bytebrew.ai
2. Кликнуть ссылку "Docs" в навигации
3. Кликнуть ссылку "Examples" в навигации
4. Кликнуть ссылку "Pricing" в навигации
5. Кликнуть ссылку "Download" в навигации
6. Кликнуть ссылку "Login" в навигации

**Ожидание:**
- Каждая ссылка ведёт на существующую страницу (HTTP 200, не 404)
- Docs → /docs/, Examples → /examples, Pricing → /pricing, Download → /download, Login → /login или cloud login

**PASS:** Все 5 навигационных ссылок открывают существующие страницы без ошибок

### TC-SITE-03: Docs index
**Шаги:**
1. Открыть https://bytebrew.ai/docs/

**Ожидание:**
- Загружается Quick Start page (заголовок "Quick Start" или "Getting Started")
- Sidebar с навигацией по документации виден
- Не отображается splash-страница или 404

**PASS:** /docs/ показывает Quick Start с sidebar навигацией

### TC-SITE-04: All doc pages accessible
**Шаги:**
1. Проверить HTTP 200 для каждой из 29 страниц документации:
   - Getting Started: `/docs/getting-started/quick-start`, `/docs/getting-started/configuration`, `/docs/getting-started/api-reference`
   - Admin: `/docs/admin/login`, `/docs/admin/agents`, `/docs/admin/models`, `/docs/admin/mcp-servers`, `/docs/admin/tasks`, `/docs/admin/triggers`, `/docs/admin/api-keys`, `/docs/admin/settings`, `/docs/admin/config-management`, `/docs/admin/audit-log`
   - Concepts: `/docs/concepts/agents`, `/docs/concepts/multi-agent`, `/docs/concepts/tools`, `/docs/concepts/tasks`, `/docs/concepts/knowledge`, `/docs/concepts/triggers`
   - Deployment: `/docs/deployment/docker`, `/docs/deployment/model-selection`, `/docs/deployment/production`
   - Integration: `/docs/integration/rest-api`, `/docs/integration/multi-agent`, `/docs/integration/byok`
   - Examples: `/docs/examples/hr-assistant`, `/docs/examples/support-agent`, `/docs/examples/sales-agent`

**Ожидание:**
- Все 29 URL возвращают HTTP 200
- Каждая страница содержит контент (не пустая, не заглушка)

**PASS:** Все 29 страниц документации доступны (HTTP 200) и содержат контент

### TC-SITE-05: Search
**Шаги:**
1. Открыть https://bytebrew.ai/docs/
2. Нажать Ctrl+K (или кликнуть иконку поиска)
3. Ввести "agent" в строку поиска

**Ожидание:**
- Открывается модальное окно поиска
- Результаты поиска содержат ссылки на страницы с упоминанием "agent"
- Клик по результату переходит на соответствующую страницу

**PASS:** Поиск находит релевантные результаты по запросу "agent"

### TC-SITE-06: Light theme readability
**Шаги:**
1. Открыть https://bytebrew.ai/docs/
2. Переключить тему на Light (если не по умолчанию)
3. Проверить визуальные элементы на нескольких страницах

**Ожидание:**
- h1/h2 заголовки — тёмный цвет текста на светлом фоне
- Body text — достаточный контраст для чтения
- Sidebar headings — читаемые, не сливаются с фоном
- Callout-блоки — оранжевые/цветные, отличаются от основного текста
- Code blocks — видны, с фоном отличным от основного
- Tables — с borders, данные различимы

**PASS:** Все текстовые элементы читаемы в Light theme

### TC-SITE-07: Dark theme readability
**Шаги:**
1. Открыть https://bytebrew.ai/docs/
2. Переключить тему на Dark
3. Проверить те же элементы что в TC-SITE-06

**Ожидание:**
- Все текстовые элементы читаемы (светлый текст на тёмном фоне)
- Code blocks видны, не сливаются с фоном
- Ссылки различимы от обычного текста

**PASS:** Все текстовые элементы читаемы в Dark theme

### TC-SITE-08: llms.txt
**Шаги:**
1. `curl -s -o /dev/null -w "%{http_code}" https://bytebrew.ai/llms.txt`
2. `curl -s https://bytebrew.ai/llms.txt`

**Ожидание:**
- HTTP 200
- Содержимое — текстовый файл со ссылками на страницы документации
- Ссылки ведут на реальные страницы (можно проверить выборочно)

**PASS:** llms.txt доступен, содержит ссылки на документацию

### TC-SITE-09: docker-compose.yml download
**Шаги:**
1. `curl -s -o /dev/null -w "%{http_code}" https://bytebrew.ai/releases/docker-compose.yml`
2. `curl -s https://bytebrew.ai/releases/docker-compose.yml | head -20`

**Ожидание:**
- HTTP 200
- Содержимое — валидный YAML (начинается с `version:` или `services:`)
- Содержит сервисы: `engine`, `db` (как минимум)

**PASS:** docker-compose.yml скачивается и содержит валидную конфигурацию

### TC-SITE-10: Favicon
**Шаги:**
1. Открыть https://bytebrew.ai/docs/ в браузере
2. Проверить иконку во вкладке браузера

**Ожидание:**
- Favicon — кружка (ByteBrew SVG логотип), не стандартная иконка браузера
- Не broken image

**PASS:** Кастомный favicon (кружка) отображается во вкладке

### TC-SITE-11: Cache headers
**Шаги:**
1. `curl -sI https://bytebrew.ai/docs/ | grep -i cache-control`
2. Найти URL любого CSS-файла из `_astro/` в source и проверить: `curl -sI https://bytebrew.ai/_astro/<file>.css | grep -i cache-control`

**Ожидание:**
- HTML страницы: `Cache-Control` содержит `no-cache` или `max-age=0`
- CSS из `_astro/`: `Cache-Control` содержит `immutable` или большой `max-age`

**PASS:** HTML не кешируется, статика (_astro/) кешируется с immutable

### TC-SITE-12: /examples/ page
**Шаги:**
1. Открыть https://bytebrew.ai/examples

**Ожидание:**
- 3 карточки: HR Assistant, Support Agent, Sales Agent
- Каждая карточка содержит feature tags (напр. "RAG", "MCP Tools", "Multi-Agent")
- Каждая карточка содержит кнопку "Try Demo →"
- Клик на карточку ведёт на `/examples/<name>`

**PASS:** Страница /examples/ показывает 3 карточки с tags и "Try Demo →"

### TC-SITE-13: Landing — порты и URL в примерах
**Шаги:**
1. Открыть https://bytebrew.ai
2. Проверить Hero code sample — URL содержит `localhost:8443`
3. Проверить Step 2 Deploy — curl команда использует `localhost:8443/api/v1/health`
4. Проверить Step 3 Integrate — curl команда использует `localhost:8443/api/v1/agents/{name}/chat`

**Ожидание:**
- Все порты = `8443` (не `8080`)
- Все API пути = `/api/v1/...` (не `/v1/...` или другие)
- Порты и пути совпадают с реальным docker-compose.yml

**PASS:** Все порты 8443 и пути /api/v1/ на landing соответствуют docker-compose

### TC-SITE-14: Landing — SSE event types
**Шаги:**
1. Открыть https://bytebrew.ai
2. Найти Step 3 / response example с SSE events
3. Проверить типы событий в примере

**Ожидание:**
- Event types в примере: `message_delta`, `message`, `done` (не `content`, не `chunk`)
- Совпадают с типами из API Reference (/docs/getting-started/api-reference)

**PASS:** SSE event types на landing совпадают с API Reference

### TC-SITE-15: Landing — скриншоты
**Шаги:**
1. Открыть https://bytebrew.ai
2. Проскроллить до секции "See it in action" → проверить Web Client screenshot
3. Проверить Admin Dashboard screenshot
4. Проскроллить до Step 1 → проверить Admin Dashboard screenshot

**Ожидание:**
- Все img элементы загружены (нет broken image / alt text вместо картинки)
- Скриншоты показывают актуальный UI (не устаревшие версии)

**PASS:** Все скриншоты на landing загружаются и отображаются

### TC-SITE-16: Landing — YAML примеры
**Шаги:**
1. Открыть https://bytebrew.ai
2. Проверить YAML в hero-секции
3. Проверить YAML в Step 1

**Ожидание:**
- `tools:` содержит `web_search` (не `knowledge_search` или другие несуществующие)
- YAML примеры соответствуют документации Quick Start

**PASS:** YAML примеры на landing содержат корректные tool names

---

## TC-INST: Docker Installation (7 TC)

### TC-INST-01: Download docker-compose
**Предусловие:** Docker и docker compose установлены

**Шаги:**
1. Выполнить curl-команду из Quick Start: `curl -o docker-compose.yml https://bytebrew.ai/releases/docker-compose.yml`
2. Проверить файл: `cat docker-compose.yml | head -5`

**Ожидание:**
- Файл скачивается без ошибок
- Содержит валидный YAML с секцией `services:`

**PASS:** docker-compose.yml скачан, содержит services

### TC-INST-02: docker compose up
**Предусловие:** docker-compose.yml скачан (TC-INST-01), `.env` настроен с OPENAI_API_KEY или OPENROUTER_API_KEY

**Шаги:**
1. `docker compose up -d`
2. `docker compose ps`

**Ожидание:**
- Все контейнеры в статусе "Up" или "Running"
- DB контейнер — healthcheck "healthy"
- Engine контейнер стартовал без crash loop

**PASS:** Все контейнеры запущены, db healthy

### TC-INST-03: Health check
**Предусловие:** Контейнеры запущены (TC-INST-02)

**Шаги:**
1. `curl -s http://localhost:8443/api/v1/health`

**Ожидание:**
- HTTP 200
- JSON содержит `"status":"ok"`
- Порт `8443` совпадает с документацией Quick Start

**PASS:** Health endpoint возвращает `{"status":"ok",...}` на порту 8443

### TC-INST-04: Admin Dashboard accessible
**Предусловие:** Контейнеры запущены (TC-INST-02)

**Шаги:**
1. Открыть http://localhost:8443/admin/ в браузере
2. Ввести username: `admin`, password: `changeme`
3. Нажать "Login"

**Ожидание:**
- Страница /admin/ показывает форму логина
- Default credentials `admin`/`changeme` принимаются
- После логина — redirect на /admin/health (или dashboard)

**PASS:** Admin Dashboard доступен, default credentials работают

### TC-INST-05: Update Engine
**Предусловие:** Контейнеры запущены, есть данные (агенты, модели)

**Шаги:**
1. `docker compose pull`
2. `docker compose up -d`
3. `curl -s http://localhost:8443/api/v1/health`

**Ожидание:**
- Pull и restart проходят без ошибок
- Health check возвращает `{"status":"ok",...}`
- Ранее созданные агенты и модели сохранились (проверить через Admin или API)

**PASS:** Update без потери данных, health OK после рестарта

### TC-INST-06: Clean shutdown
**Шаги:**
1. `docker compose down -v`
2. `docker compose ps`

**Ожидание:**
- Все контейнеры остановлены и удалены
- Volumes удалены (флаг `-v`)
- `docker compose ps` не показывает контейнеров

**PASS:** Все контейнеры и volumes удалены

### TC-INST-07: Idempotent restart
**Предусловие:** Контейнеры были запущены и остановлены (`docker compose down`, без `-v`)

**Шаги:**
1. `docker compose down` (без -v)
2. `docker compose up -d`
3. `curl -s http://localhost:8443/api/v1/health`

**Ожидание:**
- Контейнеры стартуют без ошибок
- Health check возвращает `{"status":"ok",...}`
- Данные сохранились (volumes не удалялись)

**PASS:** Restart без потери данных, health OK

---

## TC-ADMIN: Admin Dashboard (18 TC)

### TC-ADMIN-01: Login correct credentials
**Шаги:**
1. Открыть http://localhost:8443/admin/login
2. Ввести username: `admin`, password: `changeme`
3. Нажать "Login"

**Ожидание:**
- Redirect на /admin/health (dashboard)
- Sidebar навигация видна
- Health page отображает статус сервера

**PASS:** Логин с admin/changeme → redirect на /admin/health

### TC-ADMIN-02: Login wrong credentials
**Шаги:**
1. Открыть http://localhost:8443/admin/login
2. Ввести username: `admin`, password: `wrongpassword`
3. Нажать "Login"

**Ожидание:**
- Сообщение об ошибке (напр. "Invalid credentials")
- Остаёмся на /admin/login (не /login, не redirect)
- Форма не очищается (username сохраняется)

**PASS:** Ошибочный пароль → error message, остаёмся на /admin/login

### TC-ADMIN-03: Logo visible
**Шаги:**
1. Залогиниться в Admin Dashboard
2. Проверить логотип в sidebar или header

**Ожидание:**
- Логотип отображается (не broken image, не alt text)
- Логотип — ByteBrew branding (кружка или текст)

**PASS:** Логотип виден и корректно отображается

### TC-ADMIN-04: Health page
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Перейти на /admin/health (или кликнуть "Health" в sidebar)

**Ожидание:**
- Status: "ok" (зелёный индикатор)
- Version: отображается версия Engine
- Uptime: отображается время работы
- Agents: количество настроенных агентов

**PASS:** Health page показывает Status ok, Version, Uptime, Agents count

### TC-ADMIN-05: Agents CRUD
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Перейти на /admin/agents
2. Нажать "Add Agent" → заполнить: Name: `test-agent`, Model: выбрать существующую, System Prompt: `You are a test agent` → Save
3. Проверить что агент появился в списке
4. Кликнуть на агент → Edit → изменить System Prompt → Save
5. Проверить что изменения сохранились
6. Удалить агент → Confirm
7. **Edge case:** создать агент с пустым именем → Save

**Ожидание:**
- Create: агент создаётся и появляется в списке
- Edit: изменения сохраняются
- Delete: агент удаляется из списка
- Empty name: отображается ошибка валидации (не 500, не crash)

**PASS:** CRUD операции работают, пустое имя → validation error

### TC-ADMIN-06: Models CRUD
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Перейти на /admin/models
2. Нажать "Add Model" → заполнить: Name: `test-model`, Provider: OpenAI/OpenRouter, Model ID, API Key → Save
3. Проверить что модель появилась в списке
4. **Edge case:** создать модель с именем которое уже существует → Save

**Ожидание:**
- Create: модель создаётся и появляется в списке
- Duplicate name: отображается ошибка (не 500, не silent fail)

**PASS:** Model создаётся, duplicate name → error message

### TC-ADMIN-07: MCP Servers page
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Перейти на /admin/mcp-servers

**Ожидание:**
- Если нет MCP серверов: empty state ("No MCP servers configured" или аналог)
- Кнопка "Add Custom" (или "Add MCP Server") видна и кликабельна

**PASS:** MCP Servers page загружается с empty state и кнопкой добавления

### TC-ADMIN-08: Triggers page
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Перейти на /admin/triggers

**Ожидание:**
- Если нет триггеров: empty state ("No triggers configured" или аналог)
- Кнопка "Add Trigger" видна и кликабельна

**PASS:** Triggers page загружается с empty state и кнопкой добавления

### TC-ADMIN-09: API Keys
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Перейти на /admin/api-keys
2. Нажать "Generate" (или "Create API Key")
3. Скопировать сгенерированный ключ

**Ожидание:**
- Генерируется ключ с префиксом `bb_`
- Ключ отображается один раз для копирования
- Ключ появляется в списке API Keys (маскированный)

**PASS:** API Key генерируется с префиксом bb_, отображается в списке

### TC-ADMIN-10: Settings page
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Перейти на /admin/settings

**Ожидание:**
- BYOK (Bring Your Own Key) toggles присутствуют
- Секция Logging level (debug/info/warn/error) присутствует
- Секция Security присутствует
- Все toggle/select элементы кликабельны

**PASS:** Settings page показывает BYOK toggles, logging level, security section

### TC-ADMIN-11: Config Export/Import/Reload
**Предусловие:** Залогинен в Admin Dashboard, есть хотя бы 1 агент

**Шаги:**
1. Перейти на /admin/config
2. Нажать "Export" → скачивается YAML файл
3. Проверить содержимое YAML (должны быть агенты, модели)
4. Нажать "Reload" → конфигурация перезагружается

**Ожидание:**
- Export: скачивается валидный YAML с текущей конфигурацией
- Reload: конфигурация перезагружается без ошибок, toast/notification "Config reloaded"

**PASS:** Export скачивает YAML, Reload перезагружает конфигурацию без ошибок

### TC-ADMIN-12: Audit Log
**Предусловие:** Залогинен в Admin Dashboard, выполнены операции (создание агента, логин)

**Шаги:**
1. Перейти на /admin/audit-log
2. Проверить наличие записей
3. Попробовать фильтры (по типу действия, дате)

**Ожидание:**
- Записи аудита отображаются (логин, создание агента, etc.)
- Каждая запись содержит: timestamp, действие, пользователь
- Фильтры работают (сужают список)

**PASS:** Audit Log показывает записи с timestamps, фильтры работают

### TC-ADMIN-13: SPA refresh
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Перейти на http://localhost:8443/admin/agents
2. Нажать F5 (полная перезагрузка страницы)

**Ожидание:**
- Страница /admin/agents загружается (не 404, не white screen)
- Контент отображается корректно
- Сессия сохраняется (не просит логин повторно)

**PASS:** Refresh на /admin/agents не вызывает 404 или потерю сессии

### TC-ADMIN-14: Logout
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Нажать "Logout" в sidebar или header
2. Проверить URL

**Ожидание:**
- Redirect на /admin/login
- Попытка перейти на /admin/agents → redirect на /admin/login
- Сессия инвалидирована

**PASS:** Logout → redirect на /admin/login, защищённые страницы недоступны

### TC-ADMIN-15: Text readability
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Пройти по страницам: Health, Agents, Models, Settings
2. Проверить контрастность текста

**Ожидание:**
- Нет белого текста на белом фоне
- Нет чёрного текста на чёрном фоне
- Labels, values, headings — все читаемы
- Placeholder text в inputs — видим (серый, не невидимый)

**PASS:** Весь текст на всех страницах Admin Dashboard читаем

### TC-ADMIN-16: Forms
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Открыть форму создания агента (/admin/agents → Add)
2. Открыть форму создания модели (/admin/models → Add)
3. Проверить визуальные элементы

**Ожидание:**
- Labels видны и связаны с inputs
- Input поля имеют borders / видимые границы
- Buttons (Save, Cancel) видны и стилизованы
- Required fields отмечены (asterisk или аналог)

**PASS:** Forms отображаются с видимыми labels, inputs, buttons

### TC-ADMIN-17: Sidebar active link
**Предусловие:** Залогинен в Admin Dashboard

**Шаги:**
1. Перейти на /admin/agents → проверить sidebar
2. Перейти на /admin/models → проверить sidebar
3. Перейти на /admin/settings → проверить sidebar

**Ожидание:**
- Текущая страница подсвечена в sidebar (другой цвет фона или текста)
- Подсвечен именно текущий пункт, не другой

**PASS:** Sidebar highlighted link соответствует текущей странице

### TC-ADMIN-18: Empty state
**Предусловие:** Залогинен в Admin Dashboard, нет созданных агентов

**Шаги:**
1. Перейти на /admin/agents (при отсутствии агентов)

**Ожидание:**
- Отображается empty state: "No agents configured" (или аналогичное сообщение)
- Кнопка "Add Agent" видна
- Не отображается пустая таблица без пояснения

**PASS:** Пустой список агентов показывает empty state message

---

## TC-API: REST API (12 TC)

### TC-API-01: Login → JWT token
**Шаги:**
1. `curl -s -X POST http://localhost:8443/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"changeme"}'`

**Ожидание:**
- HTTP 200
- JSON response содержит поле `"token"` (JWT строка)
- Token начинается с `eyJ` (base64-encoded JWT header)

**PASS:** POST /api/v1/auth/login → 200 с JWT token

### TC-API-02: Login wrong password → 401
**Шаги:**
1. `curl -s -w "\n%{http_code}" -X POST http://localhost:8443/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"wrongpassword"}'`

**Ожидание:**
- HTTP 401
- Body содержит `"invalid credentials"` или аналогичное сообщение об ошибке
- Не 500, не stack trace

**PASS:** Неверный пароль → 401 "invalid credentials"

### TC-API-03: Health → 200 JSON
**Шаги:**
1. `curl -s http://localhost:8443/api/v1/health`

**Ожидание:**
- HTTP 200
- Content-Type: `application/json`
- JSON содержит `"status":"ok"`

**PASS:** GET /api/v1/health → 200 `{"status":"ok",...}`

### TC-API-04: Agents без auth → 401
**Шаги:**
1. `curl -s -w "\n%{http_code}" http://localhost:8443/api/v1/agents`

**Ожидание:**
- HTTP 401
- Body содержит `"unauthorized"` или аналогичное сообщение
- Список агентов НЕ возвращается

**PASS:** GET /api/v1/agents без Authorization header → 401

### TC-API-05: Models CRUD
**Предусловие:** JWT token получен (TC-API-01)

**Шаги:**
1. Создать модель: `curl -s -X POST http://localhost:8443/api/v1/models -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"name":"test-model","provider":"openai","model_id":"gpt-4o","api_key":"sk-test"}'`
2. Список моделей: `curl -s http://localhost:8443/api/v1/models -H "Authorization: Bearer <token>"`

**Ожидание:**
- Create: HTTP 200/201, response содержит созданную модель с `id`
- List: HTTP 200, JSON array содержит `test-model`

**PASS:** Model создаётся и появляется в списке

### TC-API-06: Agents CRUD
**Предусловие:** JWT token получен, модель создана (TC-API-05)

**Шаги:**
1. Создать агент: `curl -s -X POST http://localhost:8443/api/v1/agents -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"name":"test-agent","model":"test-model","system_prompt":"You are a test agent"}'`
2. Список: `curl -s http://localhost:8443/api/v1/agents -H "Authorization: Bearer <token>"`
3. Получить по имени: `curl -s http://localhost:8443/api/v1/agents/test-agent -H "Authorization: Bearer <token>"`

**Ожидание:**
- Create: HTTP 200/201, response содержит `test-agent`
- List: JSON array содержит `test-agent`
- Get: HTTP 200, JSON с полями name, model, system_prompt

**PASS:** Agent CRUD: create → list → get работают корректно

### TC-API-07: PUT agent → name preserved
**Предусловие:** Агент `test-agent` создан (TC-API-06)

**Шаги:**
1. `curl -s -X PUT http://localhost:8443/api/v1/agents/test-agent -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"system_prompt":"Updated prompt"}'`
2. `curl -s http://localhost:8443/api/v1/agents/test-agent -H "Authorization: Bearer <token>"`

**Ожидание:**
- PUT: HTTP 200
- GET после PUT: `name` = `test-agent` (не пустой, не null)
- `system_prompt` = `Updated prompt`

**PASS:** PUT agent обновляет prompt, name сохраняется

### TC-API-08: DELETE agent → 404 after
**Предусловие:** Агент `test-agent` создан

**Шаги:**
1. `curl -s -w "\n%{http_code}" -X DELETE http://localhost:8443/api/v1/agents/test-agent -H "Authorization: Bearer <token>"`
2. `curl -s -w "\n%{http_code}" http://localhost:8443/api/v1/agents/test-agent -H "Authorization: Bearer <token>"`

**Ожидание:**
- DELETE: HTTP 200/204
- GET после DELETE: HTTP 404

**PASS:** DELETE agent → последующий GET возвращает 404

### TC-API-09: Duplicate agent name → error
**Предусловие:** Агент `test-agent` создан

**Шаги:**
1. Создать второй агент с тем же именем: `curl -s -w "\n%{http_code}" -X POST http://localhost:8443/api/v1/agents -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"name":"test-agent","model":"test-model","system_prompt":"Duplicate"}'`

**Ожидание:**
- HTTP 400 или 409 (не 500)
- Body содержит понятное сообщение об ошибке (напр. "already exists")

**PASS:** Дублирующее имя агента → 400/409 с error message, не 500

### TC-API-10: Invalid model type → no crash
**Предусловие:** JWT token получен

**Шаги:**
1. `curl -s -w "\n%{http_code}" -X POST http://localhost:8443/api/v1/models -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"name":"bad-model","provider":"nonexistent_provider","model_id":"xyz"}'`

**Ожидание:**
- HTTP 400 или 422 (не 500, не crash)
- Сервер продолжает работать (health check OK)

**PASS:** Invalid provider → error response, сервер не падает

### TC-API-11: Config export → YAML
**Предусловие:** JWT token получен, есть агенты/модели

**Шаги:**
1. `curl -s http://localhost:8443/api/v1/config/export -H "Authorization: Bearer <token>"`

**Ожидание:**
- HTTP 200
- Content-Type содержит `yaml` или `text/plain`
- Body — валидный YAML с секциями agents, models

**PASS:** Config export возвращает YAML с текущей конфигурацией

### TC-API-12: Config reload → 200
**Предусловие:** JWT token получен

**Шаги:**
1. `curl -s -w "\n%{http_code}" -X POST http://localhost:8443/api/v1/config/reload -H "Authorization: Bearer <token>"`

**Ожидание:**
- HTTP 200
- Конфигурация перезагружена из БД
- Сервер продолжает работать (health check OK)

**PASS:** Config reload → 200, сервер работает

---

## TC-CHAT: SSE Chat (7 TC)

### TC-CHAT-01: Simple chat SSE stream
**Предусловие:** Engine запущен, агент настроен (напр. `hr-assistant`), модель подключена

**Шаги:**
1. Получить JWT: `curl -s -X POST http://localhost:8443/api/v1/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"changeme"}'`
2. Отправить: `curl -s -N -X POST http://localhost:8443/api/v1/agents/hr-assistant/chat -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"message":"Hello"}'`

**Ожидание:**
- Response Content-Type: `text/event-stream`
- SSE поток содержит events: `message_delta` (1+ раз), `message` (1 раз), `done` (1 раз)
- `done` event содержит `session_id` (UUID формат)
- Ответ агента — осмысленный текст (не пустой, не error)

**PASS:** SSE stream с message_delta + message + done events, ответ не пустой

### TC-CHAT-02: Event types match API Reference
**Предусловие:** Агент настроен с MCP tools

**Шаги:**
1. Отправить сообщение которое вызовет tool call (напр. "Look up employee EMP001")
2. Собрать все `event:` типы из SSE stream

**Ожидание:**
- Все event types из потока входят в множество: `message_delta`, `message`, `tool_call`, `tool_result`, `confirmation`, `thinking`, `done`, `error`
- Нет нестандартных event types (напр. `content`, `chunk`, `delta`)
- Типы совпадают с документацией API Reference

**PASS:** Все SSE event types из потока совпадают с документированными

### TC-CHAT-03: Session persistence
**Предусловие:** Агент настроен

**Шаги:**
1. Отправить: `{"message":"My name is TestUser42"}` → получить `session_id` из `done` event
2. Отправить: `{"message":"What is my name?", "session_id":"<session_id из шага 1>"}` с тем же session_id

**Ожидание:**
- Ответ на второе сообщение содержит "TestUser42"
- Агент помнит контекст предыдущего сообщения

**PASS:** Агент помнит предыдущие сообщения в рамках session_id

### TC-CHAT-04: Nonexistent agent → error
**Предусловие:** JWT token получен

**Шаги:**
1. `curl -s -w "\n%{http_code}" -X POST http://localhost:8443/api/v1/agents/nonexistent-agent-xyz/chat -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"message":"Hello"}'`

**Ожидание:**
- HTTP 404 или SSE stream с `error` event содержащим "agent not found"
- Не panic, не 500 без body

**PASS:** Несуществующий агент → "agent not found", без crash

### TC-CHAT-05: Agent without model → error
**Предусловие:** Создан агент `no-model-agent` без привязки к модели

**Шаги:**
1. `curl -s -X POST http://localhost:8443/api/v1/agents/no-model-agent/chat -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"message":"Hello"}'`

**Ожидание:**
- HTTP 400/422 или SSE stream с `error` event
- Сообщение об ошибке объясняет что модель не настроена
- Не 500, не stack trace

**PASS:** Агент без модели → понятная ошибка, не 500

### TC-CHAT-06: Chat without auth → 401
**Шаги:**
1. `curl -s -w "\n%{http_code}" -X POST http://localhost:8443/api/v1/agents/hr-assistant/chat -H "Content-Type: application/json" -d '{"message":"Hello"}'`

**Ожидание:**
- HTTP 401
- Body содержит "unauthorized"
- SSE stream НЕ начинается

**PASS:** Chat без Authorization → 401

### TC-CHAT-07: Invalid session_id → fresh session
**Предусловие:** JWT token получен, агент настроен

**Шаги:**
1. `curl -s -N -X POST http://localhost:8443/api/v1/agents/hr-assistant/chat -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"message":"Hello","session_id":"invalid-uuid-format"}'`

**Ожидание:**
- Не crash, не panic
- Либо SSE stream с новой сессией (done event с новым session_id)
- Либо error event с понятным сообщением
- Сервер продолжает работать (health check OK после запроса)

**PASS:** Invalid session_id → новая сессия или понятная ошибка, без crash

---

## TC-DOC: Documentation accuracy (8 TC)

### TC-DOC-01: Ports match
**Шаги:**
1. Открыть https://bytebrew.ai/docs/getting-started/quick-start
2. Найти все упоминания портов
3. Проверить docker-compose.yml: `curl -s https://bytebrew.ai/releases/docker-compose.yml | grep -i port`

**Ожидание:**
- Quick Start использует порт `8443`
- docker-compose.yml маппит `8443:8443`
- Нет упоминаний порта `8080` или других устаревших портов

**PASS:** Порт 8443 в Quick Start = порт в docker-compose.yml

### TC-DOC-02: curl commands work
**Предусловие:** Контейнеры запущены (TC-INST-02)

**Шаги:**
1. Скопировать curl-команду health check из Quick Start и выполнить
2. Скопировать curl-команду логина из Quick Start и выполнить
3. Скопировать curl-команду chat из Quick Start (с полученным token) и выполнить

**Ожидание:**
- Все curl-команды из документации выполняются без модификации
- Результаты соответствуют описанным в документации

**PASS:** Все curl-команды из Quick Start работают as-is

### TC-DOC-03: SSE event types documented correctly
**Шаги:**
1. Открыть https://bytebrew.ai/docs/getting-started/api-reference
2. Найти описание SSE event types
3. Сравнить с реальным SSE потоком (TC-CHAT-01)

**Ожидание:**
- Документация описывает: `message_delta`, `message`, `done`, `tool_call`, `tool_result`, `thinking`, `error`
- Не упоминает устаревшие типы (`content`, `chunk`)
- Формат event data совпадает с реальным

**PASS:** SSE event types в API Reference совпадают с реальным поведением

### TC-DOC-04: Login endpoint documented
**Шаги:**
1. Открыть https://bytebrew.ai/docs/getting-started/api-reference
2. Найти описание authentication / login endpoint

**Ожидание:**
- Документирован: `POST /api/v1/auth/login`
- Описан request body: `{"username":"...","password":"..."}`
- Описан response: `{"token":"..."}`
- Указаны default credentials: `admin`/`changeme`

**PASS:** Login endpoint полностью документирован с примерами

### TC-DOC-05: Default credentials documented
**Шаги:**
1. Проверить Quick Start: https://bytebrew.ai/docs/getting-started/quick-start
2. Проверить Docker deployment: https://bytebrew.ai/docs/deployment/docker

**Ожидание:**
- Default credentials `admin`/`changeme` указаны
- Описано как изменить через `.env` или environment variables
- Предупреждение о смене credentials для production

**PASS:** Default credentials документированы с инструкцией по смене

### TC-DOC-06: Update instructions documented
**Шаги:**
1. Открыть https://bytebrew.ai/docs/deployment/docker

**Ожидание:**
- Описан процесс обновления: `docker compose pull && docker compose up -d`
- Указано что данные сохраняются при update (volumes)

**PASS:** Update инструкция документирована (pull + up -d)

### TC-DOC-07: host.docker.internal explained
**Шаги:**
1. Открыть https://bytebrew.ai/docs/deployment/docker
2. Найти описание host.docker.internal

**Ожидание:**
- Объяснено зачем нужен `host.docker.internal` (доступ к Ollama на хосте)
- Описаны платформенные различия (Docker Desktop vs Linux)

**PASS:** host.docker.internal объяснён в Deployment/Docker

### TC-DOC-08: Example pages complete
**Шаги:**
1. Открыть https://bytebrew.ai/docs/examples/hr-assistant
2. Открыть https://bytebrew.ai/docs/examples/support-agent
3. Открыть https://bytebrew.ai/docs/examples/sales-agent

**Ожидание:**
- Каждая страница содержит: Quick Start инструкцию (clone, docker compose up)
- Содержит agents.yaml конфигурацию
- Содержит примеры разговоров (example conversations)
- Все curl-команды используют порт 8443

**PASS:** Все 3 example pages содержат Quick Start + agents.yaml + примеры разговоров

---

## TC-CLOUD: Cloud Web bytebrew.ai (11 TC)

### TC-CLOUD-01: /examples page loads
**Шаги:**
1. Открыть https://bytebrew.ai/examples

**Ожидание:**
- 3 карточки: HR Assistant, Support Agent, Sales Agent
- Feature tags на каждой карточке (напр. "RAG", "MCP Tools")
- Кнопка "Try Demo →" на каждой карточке

**PASS:** /examples показывает 3 карточки с tags и "Try Demo →"

### TC-CLOUD-02: HR Assistant example page
**Шаги:**
1. Открыть https://bytebrew.ai/examples/hr-assistant

**Ожидание:**
- Секция "What this demonstrates" с описанием capabilities
- Chat UI (input поле + кнопка Send)
- Suggestion chips (предложенные вопросы)
- Секция "Run it yourself" с инструкциями
- Ссылка на GitHub репозиторий

**PASS:** HR Assistant page загружается со всеми секциями

### TC-CLOUD-03: Support Agent example page
**Шаги:**
1. Открыть https://bytebrew.ai/examples/support-agent

**Ожидание:**
- Те же секции что в TC-CLOUD-02: "What this demonstrates", Chat UI, suggestion chips, "Run it yourself", GitHub link
- Контент специфичен для Support Agent (не копия HR)

**PASS:** Support Agent page загружается со всеми секциями

### TC-CLOUD-04: Sales Agent example page
**Шаги:**
1. Открыть https://bytebrew.ai/examples/sales-agent

**Ожидание:**
- Те же секции что в TC-CLOUD-02
- Контент специфичен для Sales Agent

**PASS:** Sales Agent page загружается со всеми секциями

### TC-CLOUD-05: Auth popup trigger
**Шаги:**
1. Открыть https://bytebrew.ai/examples/hr-assistant (не залогиненным)
2. Ввести сообщение в chat input
3. Нажать Send

**Ожидание:**
- Появляется popup авторизации
- Popup содержит: поля email/password, кнопку "Sign in with Google", ссылку "Forgot password?"
- Chat сообщение НЕ отправляется до авторизации

**PASS:** Send без авторизации → popup с email/password + Google + forgot password

### TC-CLOUD-06: Auth popup login
**Предусловие:** Зарегистрирован аккаунт на bytebrew.ai

**Шаги:**
1. Открыть example page → нажать Send → popup появляется (TC-CLOUD-05)
2. Ввести email и password
3. Нажать "Sign in"

**Ожидание:**
- Popup закрывается
- Сообщение автоматически отправляется
- SSE streaming начинается, ответ от агента появляется

**PASS:** Login в popup → popup закрывается → сообщение отправляется

### TC-CLOUD-07: Dashboard links
**Предусловие:** Залогинен на bytebrew.ai

**Шаги:**
1. Перейти на Dashboard
2. Кликнуть ссылку "Documentation"
3. Вернуться на Dashboard, кликнуть "GitHub"
4. Вернуться на Dashboard, кликнуть "Installation Guide"

**Ожидание:**
- "Documentation" → https://bytebrew.ai/docs/ (не docs.bytebrew.ai, не 404)
- "GitHub" → https://github.com/syntheticinc/bytebrew-examples (не #, не 404)
- "Installation Guide" → /download page

**PASS:** Все Dashboard ссылки ведут на корректные URL

### TC-CLOUD-07b: Navigation after auth
**Предусловие:** Залогинен на bytebrew.ai

**Шаги:**
1. Проверить навигацию (header) после логина

**Ожидание:**
- Навигация содержит: Docs, Examples, Pricing, Download (те же ссылки что до логина)
- Дополнительно: Dashboard, Settings (или Account)
- Ссылки Docs/Examples/Pricing/Download НЕ пропали после авторизации

**PASS:** После логина навигация содержит и публичные, и приватные ссылки

### TC-CLOUD-08: Real agent response via hosted demo
**Предусловие:** Залогинен на bytebrew.ai

**Шаги:**
1. Открыть https://bytebrew.ai/examples/hr-assistant
2. Отправить сообщение "What's the PTO policy?"

**Ожидание:**
- SSE streaming от hosted demo Engine (не mock/hardcoded)
- Ответ содержит информацию о PTO policy (из knowledge base)
- Tool calls могут быть видны inline (knowledge_search)
- Ответ появляется с streaming-эффектом (не целиком)

**PASS:** Реальный streaming ответ от Engine про PTO policy

### TC-CLOUD-09: Rate limit display
**Предусловие:** Залогинен на bytebrew.ai

**Шаги:**
1. Отправить сообщение в demo chat
2. Проверить отображение лимита (напр. "14/15 messages remaining")
3. Отправить ещё несколько сообщений → счётчик уменьшается
4. (Опционально) Исчерпать лимит → отправить 15 сообщений

**Ожидание:**
- После каждого сообщения: "N/15 messages remaining" (N уменьшается)
- При исчерпании лимита: "0/15 messages remaining", input disabled
- Сообщение "Rate limit reached" или аналог

**PASS:** Rate limit отображается и уменьшается, при исчерпании → input disabled

### TC-CLOUD-10: Session persistence in demo
**Предусловие:** Залогинен на bytebrew.ai

**Шаги:**
1. Открыть https://bytebrew.ai/examples/hr-assistant
2. Отправить: "My name is RegressionTestUser"
3. Дождаться ответа
4. Отправить: "What's my name?"

**Ожидание:**
- Второй ответ содержит "RegressionTestUser"
- Агент помнит контекст в рамках одной сессии
- session_id сохраняется между сообщениями

**PASS:** Demo chat сохраняет контекст между сообщениями в сессии

---

## TC-EXAMPLE: bytebrew-examples repo (12 TC)

### TC-EXAMPLE-01: HR Assistant self-hosted + Web Client
**Предусловие:** Docker установлен, есть OpenAI/OpenRouter API key

**Шаги:**
1. `git clone https://github.com/syntheticinc/bytebrew-examples.git`
2. `cd bytebrew-examples/hr-assistant`
3. `cp .env.example .env` → заполнить API key
4. `docker compose up -d`
5. `docker compose ps` — проверить контейнеры
6. `curl -s http://localhost:8443/api/v1/health` — проверить Engine
7. Открыть http://localhost:3000 в браузере
8. Залогиниться admin/changeme
9. Выбрать агент hr-assistant → отправить сообщение

**Ожидание:**
- Контейнеры: engine + db + mcp-server + web-client — все Running
- Health check: `{"status":"ok",...}`
- Web Client на http://localhost:3000 загружается
- Sidebar показывает agent `hr-assistant`
- Chat работает: SSE streaming, tool calls видны inline

**PASS:** HR Assistant example полностью работает: docker up → web-client → chat с streaming

### TC-EXAMPLE-02: HR — Knowledge Search (RAG)
**Предусловие:** HR Assistant запущен (TC-EXAMPLE-01)

**Шаги:**
1. Отправить через Web Client или curl: `{"message": "What's the PTO policy for employees with 2+ years?"}`
   ```
   curl -s -N -X POST http://localhost:8443/api/v1/agents/hr-assistant/chat \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"message":"What is the PTO policy for employees with 2+ years?"}'
   ```

**Ожидание:**
- SSE stream содержит `tool_call` event с tool name `knowledge_search`
- Ответ содержит данные из pto-policy.md (knowledge base, 57 chunks, 5 документов)
- Ответ содержит конкретные цифры по tenure (15/20/25 days)
- Это RAG, не выдуманный ответ — данные из реального knowledge base

**PASS:** Agent вызывает knowledge_search, ответ содержит конкретные PTO цифры из knowledge base

### TC-EXAMPLE-03: HR — Employee Lookup + Leave Balance (MCP tools)
**Предусловие:** HR Assistant запущен (TC-EXAMPLE-01)

**Шаги:**
1. Отправить: `{"message": "Look up employee EMP001 and check their leave balance"}`

**Ожидание:**
- SSE stream содержит `tool_call` event с tool name `get_employee` → данные сотрудника
- SSE stream содержит `tool_call` event с tool name `get_leave_balance` → остатки отпусков
- Ответ содержит информацию о сотруднике EMP001 и его leave balance
- Это MCP tools, не hallucination — данные из MCP server

**PASS:** Agent вызывает get_employee + get_leave_balance, возвращает реальные данные

### TC-EXAMPLE-04: HR — Escalation trigger
**Предусловие:** HR Assistant запущен (TC-EXAMPLE-01)

**Шаги:**
1. Отправить: `{"message": "I have a complex situation and I need to escalate this to a human"}`

**Ожидание:**
- Agent распознаёт ключевое слово "escalate" или "need human"
- Ответ содержит информацию об эскалации к HR специалисту
- Это configured escalation trigger, не просто LLM response

**PASS:** Agent реагирует на escalation trigger и сообщает об эскалации

### TC-EXAMPLE-05: Support — Technical diagnostics (MCP tools)
**Предусловие:** Support Agent запущен (`cd support-agent && docker compose up -d`)

**Шаги:**
1. Отправить: `{"message": "My API is returning 500 errors since this morning"}`
   ```
   curl -s -N -X POST http://localhost:8443/api/v1/agents/support-router/chat \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"message":"My API is returning 500 errors since this morning"}'
   ```

**Ожидание:**
- SSE stream содержит `tool_call` event: `check_service_status` → статус API gateway
- Возможно: `get_customer` или `get_error_logs` для дополнительной диагностики
- Ответ содержит данные диагностики (uptime, error rate, status)
- Это MCP tools для реальной диагностики, не шаблонный ответ

**PASS:** Agent вызывает diagnostic tools и возвращает реальные данные о статусе

### TC-EXAMPLE-06: Support — Billing + Customer lookup (MCP tools)
**Предусловие:** Support Agent запущен

**Шаги:**
1. Отправить: `{"message": "I was double-charged, my customer ID is CUST-001"}`

**Ожидание:**
- SSE stream содержит `tool_call` event: `get_customer` → данные клиента CUST-001
- Agent анализирует подписку и предлагает конкретные действия (refund, ticket)
- Ответ содержит данные клиента из MCP server, не generic ответ

**PASS:** Agent вызывает get_customer для CUST-001 и предлагает действия

### TC-EXAMPLE-07: Sales — Product search (MCP tools)
**Предусловие:** Sales Agent запущен (`cd sales-agent && docker compose up -d`)

**Шаги:**
1. Отправить: `{"message": "I need 5 laptops for my team, budget $1200 each"}`
   ```
   curl -s -N -X POST http://localhost:8443/api/v1/agents/sales-agent/chat \
     -H "Authorization: Bearer <token>" \
     -H "Content-Type: application/json" \
     -d '{"message":"I need 5 laptops for my team, budget $1200 each"}'
   ```

**Ожидание:**
- SSE stream содержит `tool_call` event: `search_products` → список ноутбуков
- Ответ содержит конкретные продукты с ценами и спецификациями
- Продукты соответствуют бюджету ($1200)
- Это MCP tools для реального каталога товаров

**PASS:** Agent вызывает search_products и возвращает ноутбуки в рамках бюджета

### TC-EXAMPLE-08: Sales — Discount with business rules (Settings)
**Предусловие:** Sales Agent запущен

**Шаги:**
1. Отправить: `{"message": "Can I get a 20% bulk discount?"}`

**Ожидание:**
- SSE stream содержит `tool_call` event: `get_settings` → `max_discount_percent=15`
- Agent отказывает в 20% или предлагает максимум 15%
- Agent следует бизнес-правилам из Settings API, не выдумывает лимиты

**PASS:** Agent проверяет max_discount через get_settings и ограничивает скидку до 15%

### TC-EXAMPLE-09: Hosted demos health
**Шаги:**
1. `curl -s -w "\n%{http_code}" https://bytebrew.ai/examples/hr-assistant/api/v1/health`
2. `curl -s -w "\n%{http_code}" https://bytebrew.ai/examples/support-agent/api/v1/health`
3. `curl -s -w "\n%{http_code}" https://bytebrew.ai/examples/sales-agent/api/v1/health`

**Ожидание:**
- Все 3 endpoint возвращают HTTP 200
- Body содержит `{"status":"ok",...}`

**PASS:** Все 3 hosted demo engines доступны и healthy

### TC-EXAMPLE-10: Hosted demos — real agent responses via UI
**Предусловие:** Залогинен на bytebrew.ai

**Шаги:**
1. Открыть https://bytebrew.ai/examples/hr-assistant → отправить сообщение "What benefits do you offer?"
2. Открыть https://bytebrew.ai/examples/sales-agent → отправить сообщение "Show me your laptop options"

**Ожидание:**
- Реальные ответы от Engine (не mock/hardcoded), SSE streaming
- MCP tool calls видны inline между текстом (tool name, arguments, result)
- Tool call results expandable (клик → полный JSON)
- Ответы осмысленные и специфичные для каждого агента

**PASS:** Hosted demos возвращают реальные streaming ответы с видимыми tool calls

### TC-EXAMPLE-11: Rate limit persistence
**Предусловие:** Залогинен на bytebrew.ai

**Шаги:**
1. Отправить несколько сообщений в demo chat → записать счётчик (напр. "12/15 remaining")
2. Обновить страницу (F5)
3. Проверить счётчик после обновления

**Ожидание:**
- Счётчик уменьшается с каждым сообщением
- После F5: счётчик НЕ сбрасывается (синхронизация с сервером)
- Счётчик совпадает с серверным значением
- При исчерпании лимита: input disabled, сообщение "Rate limit reached"

**PASS:** Rate limit синхронизирован с сервером, не сбрасывается при F5

### TC-EXAMPLE-12: Web Client в docker-compose
**Предусловие:** Docker установлен

**Шаги:**
1. `git clone https://github.com/syntheticinc/bytebrew-examples.git`
2. `cd bytebrew-examples/hr-assistant`
3. `cp .env.example .env` → заполнить API key
4. `docker compose up -d`
5. Открыть http://localhost:3000
6. Залогиниться: admin/changeme
7. Выбрать агента в sidebar → отправить сообщение

**Ожидание:**
- Web Client доступен на http://localhost:3000
- Login admin/changeme → sidebar с агентами
- Chat работает: SSE streaming, markdown rendering
- Tool calls видны с expandable деталями (клик → JSON)

**PASS:** Web Client в docker-compose: login → agents sidebar → chat с tool calls

---

### TC-DOC-09: YAML examples valid format
- Открыть каждую страницу docs с YAML примером
- **Ожидание:** agents как map (name: key), не list (- name:)
- **Ожидание:** tools как list строк, не объектов
- **Ожидание:** mcp_servers type: stdio или sse (не http)

### TC-DOC-10: curl examples work
- Скопировать каждый curl из Quick Start и API Reference
- Выполнить на localhost:8443
- **Ожидание:** каждый curl возвращает описанный результат (или осмысленную ошибку если нет LLM)

### TC-DOC-11: Landing page claims match reality
- Проверить каждый feature claim на landing page
- **Ожидание:** Multi-agent ✓, MCP tools ✓, Cron ✓, Knowledge/RAG ✓, REST+SSE+WS ✓, BYOK ✓, Admin Dashboard ✓

### TC-DOC-12: Docker compose example ports
- Открыть deployment/docker docs page
- **Ожидание:** один порт 8443 (не два порта 8080+8443)
- **Ожидание:** ENGINE_PORT default 8443

### TC-DOC-13: API response format
- Открыть API Reference, проверить response examples
- **Ожидание:** GET /agents → JSON array (не {agents:[...]})
- **Ожидание:** POST /auth/login → {"token":"..."}

---

## TC-AUTH: Email Verification + Google Auth (10 TC)

### TC-AUTH-01: Register sends verification email
**Шаги:** POST /api/v1/auth/register с email + password
**Ожидание:** 201 с `{user_id, message}`, НЕ содержит access_token. Resend отправляет email.
**PASS:** `{"data":{"user_id":"...","message":"registration successful, please check your email"}}`

### TC-AUTH-02: Login before verification blocked
**Шаги:** POST /api/v1/auth/login с тем же email/password
**Ожидание:** Ошибка `EMAIL_NOT_VERIFIED`
**PASS:** `{"error":{"code":"EMAIL_NOT_VERIFIED","message":"please verify your email before logging in"}}`

### TC-AUTH-03: Verify email with token
**Шаги:** POST /api/v1/auth/verify-email с `{"token":"xxx"}` (из DB или email link)
**Ожидание:** 200 с `{access_token, refresh_token, user_id}`
**PASS:** JWT tokens returned, email_verified=true в DB

### TC-AUTH-04: Login after verification
**Шаги:** POST /api/v1/auth/login с verified email
**Ожидание:** 200 с JWT tokens
**PASS:** Login successful

### TC-AUTH-05: Resend verification
**Шаги:** POST /api/v1/auth/resend-verification с `{"email":"xxx"}`
**Ожидание:** 200 с message (не раскрывает существование аккаунта)
**PASS:** `{"data":{"message":"if an account exists with this email, a verification link has been sent"}}`

### TC-AUTH-06: Expired/used token
**Шаги:** POST /api/v1/auth/verify-email с использованным или несуществующим token
**Ожидание:** Ошибка `INVALID_INPUT`
**PASS:** `{"error":{"code":"INVALID_INPUT","message":"invalid or expired verification token"}}`

### TC-AUTH-07: Google login auto-verifies email
**Шаги:** POST /api/v1/auth/google с valid Google ID token
**Ожидание:** JWT tokens, email_verified=true автоматически

### TC-AUTH-08: Google Sign-In button on Login page
**Шаги:** Открыть /login в браузере
**Ожидание:** Кнопка "Sign in with Google" видна, разделитель "or"

### TC-AUTH-09: Google Sign-In button on Register page
**Шаги:** Открыть /register в браузере
**Ожидание:** Кнопка "Sign up with Google" видна, разделитель "or"

### TC-AUTH-10: Register duplicate email
**Шаги:** POST /api/v1/auth/register с уже зарегистрированным email
**Ожидание:** Ошибка `ALREADY_EXISTS`
**PASS:** `{"error":{"code":"ALREADY_EXISTS","message":"email already registered"}}`

---

## TC-MCP: MCP Documentation Server (5 TC)

### TC-MCP-01: MCP server health
**Шаги:** curl https://mcp.bytebrew.ai/health
**Ожидание:** 200 OK

### TC-MCP-02: Tools list via MCP protocol
**Шаги:** Подключиться к MCP SSE, отправить `tools/list`
**Ожидание:** Возвращает tools: search_docs, get_doc, list_docs

### TC-MCP-03: search_docs functional test
**Шаги:** MCP `tools/call` с `search_docs(query: "how to configure multi-agent")`
**Ожидание:** Возвращает релевантные passages из документации

### TC-MCP-04: Quality — ответ содержит конкретные примеры
**Шаги:** `search_docs(query: "YAML example for MCP server configuration")`
**Ожидание:** Ответ содержит YAML пример с `type: stdio` или `type: sse`

### TC-MCP-05: Quality — Claude Code integration
**Шаги:** В Claude Code с подключённым MCP: "How do I add a new agent with web_search tool?"
**Ожидание:** Ответ ссылается на правильную документацию, содержит рабочий YAML пример

### TC-MCP-06: Landing page — MCP section visible
**Шаги:** Открыть https://bytebrew.ai, скроллить до секции "AI-Native Documentation"
**Ожидание:** 3 таба (Claude Code, Codex, Other), код для подключения, переключалка работает

### TC-MCP-07: Docs — MCP instructions in Quick Start
**Шаги:** Открыть https://bytebrew.ai/docs/getting-started/quick-start/
**Ожидание:** Секция "Connect your AI assistant" с инструкциями для Claude Code, Codex, Other

### TC-MCP-08: Claude Code add command works
**Шаги:** `claude mcp add bytebrew-docs --transport sse https://mcp.bytebrew.ai/sse`
**Ожидание:** MCP сервер добавлен, `search_docs` tool доступен

### TC-MCP-09: Codex config works
**Предусловие:** Codex CLI установлен
**Шаги:** Добавить `[mcp_servers.bytebrew-docs] url = "https://mcp.bytebrew.ai/sse"` в `~/.codex/config.toml`
**Ожидание:** Codex может вызвать search_docs tool
**Примечание:** НЕ ТЕСТИРОВАНО — добавлено на основании документации Codex, требует ручной проверки

### TC-MCP-10: Search results contain documentation URLs
**Шаги:** Вызвать `search_docs(query: "triggers cron webhook")`
**Ожидание:**
- Каждый result содержит Source с кликабельной ссылкой на bytebrew.ai/docs/
- URL формат: `[triggers.md](https://bytebrew.ai/docs/concepts/triggers/)`
- URLs динамически загружаются из sitemap при старте сервера (не хардкод)
**PASS:** Source содержит markdown ссылку на правильную страницу документации

### TC-MCP-11: URL mapping updates automatically
**Шаги:**
1. Добавить новую страницу в docs-site
2. `npm run build` → deploy → sitemap обновится
3. Перезапустить MCP сервер
4. Вызвать search_docs с запросом по новой странице
**Ожидание:** Result содержит URL на новую страницу без изменений в коде MCP сервера

### TC-MCP-12: No hallucinations in search results
**Шаги:** Вызвать `search_docs(query: "GraphQL support")`
**Ожидание:** "No results found" — не выдумывает несуществующие фичи
**PASS:** Возвращает пустой результат, не галлюцинирует

### TC-MCP-13: Hybrid search — keyword fallback
**Шаги:** Вызвать `search_docs(query: "confirm_before")`
**Ожидание:** Результаты из tools.md, configuration.md, multi-agent.md — keyword `confirm_before` матчится даже если vector similarity низкий

---

## TC-EE: EE Activation, Stripe, License Lifecycle (27 TC)

### TC-EE-01: stripe-setup создаёт Engine EE product
**Предусловие:** Stripe аккаунт настроен, stripe-setup скомпилирован
**Шаги:**
1. Запустить `go run ./cmd/stripe-setup`
2. Проверить Stripe Dashboard → Products

**Ожидание:**
- Product "ByteBrew Engine EE" создан в Stripe
- 2 price: monthly ($499/mo) и annual ($4,990/yr)
- Product metadata содержит идентификатор `engine_ee`

**PASS:** stripe-setup создаёт Engine EE product с 2 prices

### TC-EE-02: stripe-setup повторный запуск (idempotent)
**Предусловие:** TC-EE-01 выполнен
**Шаги:**
1. Запустить `go run ./cmd/stripe-setup` повторно
2. Проверить Stripe Dashboard → Products

**Ожидание:**
- Product не дублируется (всё ещё 1 Engine EE product)
- Prices не дублируются
- Команда завершается без ошибок

**PASS:** Повторный stripe-setup не создаёт дубликаты

### TC-EE-03: Pricing page — EE pricing visible
**Предусловие:** `VITE_SHOW_EE_PRICING=true`, cloud-web собран
**Шаги:**
1. Открыть страницу Pricing на bytebrew.ai

**Ожидание:**
- Отображается Engine EE plan: $499/mo, $4,990/yr
- Period toggle (monthly/annual) работает
- CTA кнопка "Subscribe" или "Get Started"

**PASS:** Pricing page показывает EE с двумя периодами и toggle

### TC-EE-04: Pricing page — available фичи зелёные
**Предусловие:** TC-EE-03
**Шаги:**
1. Открыть Pricing page
2. Проверить список EE фич

**Ожидание:**
- Audit Log — зелёный (реализовано)
- Configurable Rate Limiting — зелёный
- Prometheus Metrics — зелёный

**PASS:** Реализованные EE фичи отмечены зелёным

### TC-EE-05: Pricing page — coming soon фичи серые
**Предусловие:** TC-EE-03
**Шаги:**
1. Открыть Pricing page
2. Проверить будущие EE фичи

**Ожидание:**
- Session Explorer, Cost Analytics, Quality Metrics, Prompt A/B Testing, PII Redaction — серые "Coming Soon"
- Не кликабельны как активные фичи

**PASS:** Будущие фичи отмечены серым "Coming Soon"

### TC-EE-06: Checkout — Engine EE monthly
**Предусловие:** Залогинен на bytebrew.ai
**Шаги:**
1. Открыть Pricing page
2. Выбрать monthly → нажать "Subscribe"

**Ожидание:**
- Redirect на Stripe Checkout с monthly price ($499/mo)
- Checkout session создана корректно

**PASS:** Monthly checkout создаёт Stripe session и redirect

### TC-EE-07: Checkout — Engine EE annual
**Предусловие:** Залогинен на bytebrew.ai
**Шаги:**
1. Открыть Pricing page
2. Переключить на annual → нажать "Subscribe"

**Ожидание:**
- Redirect на Stripe Checkout с annual price ($4,990/yr)

**PASS:** Annual checkout создаёт Stripe session и redirect

### TC-EE-08: Webhook — subscription.created → license
**Предусловие:** Stripe webhook настроен
**Шаги:**
1. Завершить Stripe Checkout (тестовый режим)
2. Stripe отправляет `subscription.created` webhook

**Ожидание:**
- Cloud API обрабатывает webhook
- EE license (JWT) сгенерирована
- License доступна для скачивания

**PASS:** Webhook создаёт EE license после оплаты

### TC-EE-09: License download — /license/download
**Предусловие:** TC-EE-08, license сгенерирована
**Шаги:**
1. GET /api/v1/license/download с авторизацией

**Ожидание:**
- HTTP 200
- Скачивается JWT файл (license.jwt)
- JWT содержит tier=engine_ee, exp > now

**PASS:** License JWT скачивается

### TC-EE-10: Engine — EE endpoint + Active license
**Предусловие:** Engine запущен, `~/.bytebrew/license.jwt` с valid EE license
**Шаги:**
1. Вызвать EE endpoint (напр. GET /api/v1/audit/tool-calls)

**Ожидание:**
- HTTP 200
- Данные возвращаются корректно

**PASS:** EE endpoint работает с active лицензией

### TC-EE-11: Engine — EE endpoint без license.jwt
**Предусловие:** Engine запущен, файл `~/.bytebrew/license.jwt` отсутствует
**Шаги:**
1. Вызвать EE endpoint (напр. GET /api/v1/audit/tool-calls)

**Ожидание:**
- HTTP 403
- Body: `{"error": "Enterprise Edition required"}`

**PASS:** EE endpoint без лицензии → 403

### TC-EE-12: Engine — EE endpoint + Grace period
**Предусловие:** Engine запущен, license.jwt с exp < now < exp+3d (grace period)
**Шаги:**
1. Вызвать EE endpoint

**Ожидание:**
- HTTP 200
- Response header `X-License-Warning: "License expires in N days"`
- Данные возвращаются корректно

**PASS:** Grace period — 200 OK + предупреждение в header

### TC-EE-13: Engine — EE endpoint + Blocked (expired > 3 дней)
**Предусловие:** Engine запущен, license.jwt с exp+3d < now
**Шаги:**
1. Вызвать EE endpoint

**Ожидание:**
- HTTP 403
- Body: `{"error": "Enterprise license expired"}`

**PASS:** Expired license (после grace) → 403

### TC-EE-14: Engine — CE endpoint без лицензии
**Предусловие:** Engine запущен, нет license.jwt
**Шаги:**
1. Вызвать CE endpoint: POST /api/v1/agents/test/chat
2. GET /api/v1/health

**Ожидание:**
- HTTP 200
- Чат, агенты, MCP — всё работает без EE лицензии

**PASS:** CE endpoints работают без лицензии

### TC-EE-15: Engine — CE endpoint + Blocked лицензия
**Предусловие:** Engine запущен, license.jwt expired > 3 дней
**Шаги:**
1. Вызвать CE endpoint: POST /api/v1/agents/test/chat

**Ожидание:**
- HTTP 200
- CE функционал продолжает работать даже при blocked лицензии

**PASS:** CE работает при expired EE лицензии

### TC-EE-16: Downgrade — rate limits отключаются
**Предусловие:** Engine с EE лицензией, configurable rate limits настроены
**Шаги:**
1. Удалить license.jwt (или дождаться expiry)
2. Отправить chat request с rate limit headers

**Ожидание:**
- Configurable rate limits не применяются
- Per-IP (CE) rate limiter продолжает работать

**PASS:** При downgrade configurable limits отключаются, per-IP сохраняется

### TC-EE-17: Downgrade — данные не теряются
**Предусловие:** Engine работал с EE, есть audit data, session_events
**Шаги:**
1. Удалить license.jwt → CE mode
2. Проверить данные через CE endpoints (sessions, events)

**Ожидание:**
- session_events сохранены
- Конфиги агентов, моделей на месте
- Данные не удаляются при downgrade

**PASS:** Данные сохраняются при переходе EE → CE

### TC-EE-18: Engine — license.jwt auto-load при старте
**Предусловие:** Файл `~/.bytebrew/license.jwt` существует с valid JWT
**Шаги:**
1. Запустить Engine
2. Вызвать EE endpoint

**Ожидание:**
- Engine автоматически загружает license.jwt при старте
- EE endpoint возвращает 200

**PASS:** License загружается автоматически при старте

### TC-EE-19: Engine — background ticker обновляет license
**Предусловие:** Engine запущен без license.jwt (CE mode)
**Шаги:**
1. Вызвать EE endpoint → 403
2. Положить valid license.jwt в `~/.bytebrew/`
3. Подождать ~5 минут
4. Вызвать EE endpoint

**Ожидание:**
- EE endpoint возвращает 200 (license подхвачена без рестарта)

**PASS:** Background ticker подхватывает новую лицензию без рестарта

### TC-EE-20: Engine — license.jwt удалён → CE mode
**Предусловие:** Engine запущен с valid license.jwt
**Шаги:**
1. Вызвать EE endpoint → 200
2. Удалить license.jwt
3. Подождать ~5 минут
4. Вызвать EE endpoint

**Ожидание:**
- EE endpoint возвращает 403 (LicenseInfo = nil)
- CE endpoints продолжают работать

**PASS:** Удаление license.jwt переводит в CE mode

### TC-EE-21: Admin — Grace баннер
**Предусловие:** License в grace period (exp < now < exp+3d)
**Шаги:**
1. Открыть Admin Dashboard

**Ожидание:**
- Баннер "License expiring in N days, renew" отображается
- Ссылка на продление лицензии

**PASS:** Grace баннер виден в Admin

### TC-EE-22: Admin — Blocked баннер
**Предусловие:** License expired > 3 дней
**Шаги:**
1. Открыть Admin Dashboard

**Ожидание:**
- Баннер "License expired, upgrade at bytebrew.ai"
- EE вкладки скрыты или показывают upgrade prompt

**PASS:** Blocked баннер виден, EE вкладки недоступны

### TC-EE-23: Admin — EE tabs visible (Active)
**Предусловие:** License active
**Шаги:**
1. Открыть Admin Dashboard
2. Проверить sidebar

**Ожидание:**
- Вкладки Audit (Tool Calls), Rate Limits видны и доступны

**PASS:** EE вкладки видны при active лицензии

### TC-EE-24: Admin — EE tabs hidden (no license)
**Предусловие:** Нет license.jwt
**Шаги:**
1. Открыть Admin Dashboard
2. Проверить sidebar

**Ожидание:**
- EE вкладки скрыты или показывают upgrade prompt
- CE вкладки работают нормально

**PASS:** EE вкладки скрыты без лицензии

### TC-EE-25: Billing page — manage subscription
**Предусловие:** Залогинен на bytebrew.ai, есть active subscription
**Шаги:**
1. Открыть Billing page
2. Нажать "Manage Subscription"

**Ожидание:**
- Redirect в Stripe Customer Portal
- Возможность cancel/change plan

**PASS:** Manage Subscription → Stripe Portal

### TC-EE-26: Billing page — current plan display
**Предусловие:** Залогинен, есть active EE subscription
**Шаги:**
1. Открыть Billing page

**Ожидание:**
- Показывает "Engine EE"
- Период (monthly/annual)
- Next billing date

**PASS:** Billing page показывает текущий план

### TC-EE-27: Upgrade — Blocked → новый JWT → Active
**Предусловие:** Engine с expired license (blocked), новый JWT получен
**Шаги:**
1. Вызвать EE endpoint → 403
2. Положить новый valid license.jwt
3. Подождать ~5 минут (или перезапустить)
4. Вызвать EE endpoint

**Ожидание:**
- EE endpoint возвращает 200
- Все EE фичи снова доступны

**PASS:** Обновление license.jwt восстанавливает EE функционал

---

## TC-REG: Model Registry, Tier Warnings, OpenRouter (14 TC)

### TC-REG-01: GET /api/v1/models/registry — полный каталог
**Предусловие:** Engine запущен
**Шаги:**
1. `curl -s http://localhost:8443/api/v1/models/registry -H "Authorization: Bearer <token>"`

**Ожидание:**
- HTTP 200
- JSON содержит массив моделей с полями: provider, model_name, tier, context_window, pricing, supports_tools
- Минимум 10+ моделей от 4+ провайдеров

**PASS:** Registry API возвращает полный каталог моделей

### TC-REG-02: Фильтр ?provider=anthropic
**Предусловие:** Engine запущен
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/models/registry?provider=anthropic" -H "Authorization: Bearer <token>"`

**Ожидание:**
- HTTP 200
- Все модели в результате имеют provider=anthropic
- Модели других провайдеров отсутствуют

**PASS:** Фильтр по provider работает корректно

### TC-REG-03: Фильтр ?tier=1
**Предусловие:** Engine запущен
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/models/registry?tier=1" -H "Authorization: Bearer <token>"`

**Ожидание:**
- HTTP 200
- Все модели имеют tier=1 (Orchestrator)
- Присутствуют: Claude Opus/Sonnet 4.6, GPT-5.4, Gemini 3.1 Pro

**PASS:** Фильтр по tier возвращает только Tier 1 модели

### TC-REG-04: Фильтр ?supports_tools=true
**Предусловие:** Engine запущен
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/models/registry?supports_tools=true" -H "Authorization: Bearer <token>"`

**Ожидание:**
- HTTP 200
- Все модели имеют supports_tools=true
- Модели без tool support отсутствуют

**PASS:** Фильтр по supports_tools работает

### TC-REG-05: Ключевые модели присутствуют в каталоге
**Предусловие:** Engine запущен
**Шаги:**
1. GET /api/v1/models/registry
2. Проверить наличие ключевых моделей

**Ожидание:**
- Claude Opus 4.6, Claude Sonnet 4.6 (Anthropic, Tier 1)
- GPT-5.4, GPT-5.2 (OpenAI, Tier 1)
- Gemini 3.1 Pro (Google, Tier 1)
- DeepSeek V3.2 (DeepSeek, Tier 1)
- GLM-5 (Zhipu, Tier 1)

**PASS:** Все ключевые модели присутствуют с правильными tier

### TC-REG-06: Admin — список моделей с tier badge
**Предусловие:** Залогинен в Admin Dashboard, модели созданы
**Шаги:**
1. Перейти на /admin/models

**Ожидание:**
- Рядом с каждой моделью из registry — badge "Tier 1", "Tier 2" или "Tier 3"
- Badge цветные (зелёный/жёлтый/красный или аналогичная индикация)

**PASS:** Tier badges отображаются рядом с моделями

### TC-REG-07: Admin — выбор модели из каталога
**Предусловие:** Залогинен в Admin Dashboard
**Шаги:**
1. Перейти на /admin/models → Add Model
2. Выбрать provider → раскрыть dropdown моделей
3. Выбрать модель из каталога

**Ожидание:**
- Dropdown содержит модели из registry с tier badges
- При выборе модели: auto-fill model_name, base_url (если есть)

**PASS:** Каталог моделей в dropdown с auto-fill

### TC-REG-08: Admin — custom model warning
**Предусловие:** Залогинен в Admin Dashboard
**Шаги:**
1. Перейти на /admin/models → Add Model
2. Ввести model_name не из registry (напр. "my-custom-model-xyz")

**Ожидание:**
- Жёлтый warning: "Model not in registry — not tested for agent use"
- Модель можно создать (warning не блокирует)

**PASS:** Custom model показывает warning, не блокирует создание

### TC-REG-09: Admin — agent + Tier 3 orchestrator warning
**Предусловие:** Залогинен, модель Tier 3 создана (напр. GPT-4o-mini)
**Шаги:**
1. Перейти на /admin/agents → Add/Edit Agent
2. Выбрать Tier 3 модель, включить can_spawn=true

**Ожидание:**
- Warning: "Not recommended as orchestrator — model may have limited tool calling"

**PASS:** Предупреждение при выборе Tier 3 модели для orchestrator

### TC-REG-10: Admin — agent + Tier 1 model, no warning
**Предусловие:** Залогинен, модель Tier 1 создана
**Шаги:**
1. Перейти на /admin/agents → Add/Edit Agent
2. Выбрать Tier 1 модель

**Ожидание:**
- Нет warning'ов о tier
- Модель принимается без замечаний

**PASS:** Tier 1 модель — нет предупреждений

### TC-REG-11: type=openrouter → сохранение с base_url
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/models с type=openrouter, api_key=<key>
2. GET /api/v1/models — проверить созданную модель

**Ожидание:**
- Модель сохранена с base_url=`https://openrouter.ai/api/v1`
- Provider type нормализован (openrouter → openai_compatible или сохранён)

**PASS:** OpenRouter модель сохранена с правильным base_url

### TC-REG-12: Admin — openrouter selected, base_url auto-filled
**Предусловие:** Залогинен в Admin Dashboard
**Шаги:**
1. Перейти на /admin/models → Add Model
2. Выбрать provider "OpenRouter"

**Ожидание:**
- base_url auto-filled: `https://openrouter.ai/api/v1`
- Поле base_url read-only (нельзя изменить)
- Поле API key доступно для ввода

**PASS:** OpenRouter — base_url заполнен автоматически, read-only

### TC-REG-13: Verify openrouter model
**Предусловие:** OpenRouter модель создана с valid API key
**Шаги:**
1. POST /api/v1/models/<id>/verify

**Ожидание:**
- Connectivity check проходит (ping + tool probe)
- Status: verified

**PASS:** OpenRouter verify проходит успешно

### TC-REG-14: Unknown model → verify → ok, warning остаётся
**Предусловие:** Создана модель с custom model_name (не из registry)
**Шаги:**
1. POST /api/v1/models/<id>/verify → connectivity ok
2. GET /api/v1/models/<id>

**Ожидание:**
- Verify проходит (модель работает)
- Warning "Model not in registry" остаётся (не удаляется после verify)

**PASS:** Unknown model — verify ok, warning не снимается

---

## TC-PROV: LLM Providers — Azure OpenAI, Google Gemini (19 TC)

### TC-PROV-01: Создание Azure OpenAI модели через API
**Предусловие:** Engine запущен, admin JWT получен
**Шаги:**
1. `curl -s -X POST http://localhost:8443/api/v1/models -H "Authorization: Bearer <token>" -H "Content-Type: application/json" -d '{"name":"azure-gpt","provider":"azure_openai","model_id":"gpt-4","api_key":"<key>","deployment_name":"my-gpt4","api_version":"2024-02-01","base_url":"https://myresource.openai.azure.com"}'`
2. GET /api/v1/models — проверить модель в списке

**Ожидание:**
- HTTP 201
- type=azure_openai
- deployment_name и api_version сохранены

**Примечание:** Body format идентичен OpenAI, отличается URL pattern и auth header

**PASS:** Azure OpenAI модель создана

### TC-PROV-02: Azure — URL construction
**Предусловие:** Azure модель создана (TC-PROV-01)
**Шаги:**
1. Отправить chat request через Azure модель (или проверить unit test)

**Ожидание:**
- Запрос уходит на `https://{resource}.openai.azure.com/openai/deployments/{deployment}/chat/completions?api-version={version}`
- Формат URL соответствует Azure OpenAI API

**PASS:** URL конструируется по Azure-паттерну

### TC-PROV-03: Azure — Auth header
**Предусловие:** Azure модель создана
**Шаги:**
1. Отправить chat request, проверить исходящий HTTP-запрос (unit test или proxy)

**Ожидание:**
- Header `api-key: <key>` установлен
- НЕ `Authorization: Bearer <key>`

**PASS:** Azure использует `api-key` header

### TC-PROV-04: Verify Azure модели
**Предусловие:** Azure модель создана с valid credentials
**Шаги:**
1. POST /api/v1/models/<id>/verify

**Ожидание:**
- Connectivity check проходит
- Tool calling probe может быть skipped (зависит от deployment)

**PASS:** Azure verify — connectivity ok

### TC-PROV-05: Azure — Missing deployment_name → error
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/models с type=azure_openai, без deployment_name

**Ожидание:**
- HTTP 400
- Validation error: deployment_name required for azure_openai

**PASS:** Отсутствие deployment_name → 400 validation error

### TC-PROV-06: Chat с Azure моделью
**Предусловие:** Azure модель создана, агент привязан к ней
**Шаги:**
1. POST /api/v1/agents/<agent>/chat с сообщением

**Ожидание:**
- SSE stream с message_delta, message, done
- Ответ корректный, не пустой

**PASS:** Chat через Azure OpenAI работает

### TC-PROV-07: Streaming с Azure
**Предусловие:** Azure модель создана, агент привязан
**Шаги:**
1. POST /api/v1/agents/<agent>/chat с сообщением
2. Наблюдать SSE stream

**Ожидание:**
- SSE stream работает (не батчевый ответ)
- message_delta приходят по частям

**PASS:** Azure streaming работает

### TC-PROV-08: Admin UI — Azure опция
**Предусловие:** Залогинен в Admin Dashboard
**Шаги:**
1. Перейти на /admin/models → Add Model
2. Выбрать provider "Azure OpenAI"

**Ожидание:**
- Появляются поля: deployment_name, api_version
- Поле base_url доступно (для Azure resource endpoint)

**PASS:** Admin UI показывает Azure-специфичные поля

### TC-PROV-09: Azure — Backwards compat
**Предусловие:** Существующие модели (OpenAI, Anthropic) настроены
**Шаги:**
1. Добавить Azure модель
2. Проверить существующие модели: GET /api/v1/models

**Ожидание:**
- Существующие модели не затронуты
- Chat с существующими моделями работает

**PASS:** Добавление Azure не ломает существующие модели

### TC-PROV-10: Конвертация Gemini — user message
**Шаги:**
1. Unit test: конвертация Eino user message → Gemini format

**Ожидание:**
- role: "user"
- parts: [{text: "message text"}]
- Формат соответствует Gemini API

**PASS:** User message конвертируется в Gemini format

### TC-PROV-11: Конвертация Gemini — assistant + tool_call
**Шаги:**
1. Unit test: конвертация Eino assistant message с tool call → Gemini format

**Ожидание:**
- role: "model"
- parts: [{functionCall: {name, args}}]

**PASS:** Assistant + tool_call конвертируется корректно

### TC-PROV-12: Конвертация Gemini — tool result
**Шаги:**
1. Unit test: конвертация tool result → Gemini format

**Ожидание:**
- parts: [{functionResponse: {name, response}}]

**PASS:** Tool result конвертируется в functionResponse

### TC-PROV-13: Конвертация Gemini — tool declarations
**Шаги:**
1. Unit test: конвертация Eino ToolInfo → Gemini FunctionDeclaration

**Ожидание:**
- Eino ToolInfo → Gemini FunctionDeclaration с name, description, parameters

**PASS:** Tool declarations конвертируются корректно

### TC-PROV-14: Создание Gemini модели через API
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/models с type=google, model_id="gemini-2.5-pro", api_key=<key>

**Ожидание:**
- HTTP 201
- type=google

**PASS:** Gemini модель создана

### TC-PROV-15: Gemini — Auth: API key in header
**Предусловие:** Gemini модель создана
**Шаги:**
1. Отправить chat request, проверить исходящий HTTP-запрос

**Ожидание:**
- Header `x-goog-api-key: <key>` установлен

**PASS:** Gemini использует x-goog-api-key header

### TC-PROV-16: Gemini — Streaming
**Предусловие:** Gemini модель создана, агент привязан
**Шаги:**
1. POST /api/v1/agents/<agent>/chat

**Ожидание:**
- SSE stream от streamGenerateContent endpoint
- message_delta приходят по частям

**PASS:** Gemini streaming работает

### TC-PROV-17: Gemini — Chat с function calling
**Предусловие:** Gemini модель создана, агент с tools
**Шаги:**
1. Отправить сообщение требующее tool call

**Ожидание:**
- Tool call корректно парсится из Gemini response
- Tool result отправляется обратно как functionResponse
- Финальный ответ содержит результат tool call

**PASS:** Function calling через Gemini работает

### TC-PROV-18: Admin UI — Google опция
**Предусловие:** Залогинен в Admin Dashboard
**Шаги:**
1. Перейти на /admin/models → Add Model
2. Выбрать provider "Google"

**Ожидание:**
- Появляются соответствующие поля (API key)
- Model ID dropdown содержит Gemini модели из registry

**PASS:** Admin UI показывает Google-специфичные поля

### TC-PROV-19: Gemini — Invalid API key → error
**Предусловие:** Gemini модель создана с invalid API key
**Шаги:**
1. POST /api/v1/models/<id>/verify

**Ожидание:**
- Verify fails с понятной ошибкой (invalid API key)
- Не panic, не 500 без body

**PASS:** Invalid API key → корректная ошибка

---

## TC-CTX: Request Context Propagation, MCP Header Forwarding (12 TC)

### TC-CTX-01: Chat request с X-User-Id header
**Предусловие:** Engine запущен, агент настроен
**Шаги:**
1. POST /api/v1/agents/<agent>/chat с header `X-User-Id: user-456`
2. Проверить что header извлечён и доступен в context (через audit log или unit test)

**Ожидание:**
- Header X-User-Id извлечён в RequestContext
- Доступен для downstream компонентов (audit, MCP forwarding)

**PASS:** X-User-Id извлечён из request и помещён в context

### TC-CTX-02: Chat request с X-Org-Id header
**Предусловие:** Engine запущен, агент настроен
**Шаги:**
1. POST /api/v1/agents/<agent>/chat с header `X-Org-Id: org-123`

**Ожидание:**
- Header X-Org-Id извлечён в RequestContext
- Доступен для downstream (MCP forwarding, rate limiting)

**PASS:** X-Org-Id извлечён из request

### TC-CTX-03: Chat request без context headers
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents/<agent>/chat без X-User-Id, X-Org-Id headers

**Ожидание:**
- RequestContext пустой (нет headers)
- Ошибки нет, chat работает нормально

**PASS:** Отсутствие context headers не вызывает ошибку

### TC-CTX-04: MCP SSE — forward_headers configured
**Предусловие:** MCP server (SSE) настроен с `forward_headers: ["X-Org-Id", "X-User-Id"]`
**Шаги:**
1. POST /api/v1/agents/<agent>/chat с headers X-Org-Id: org-123, X-User-Id: user-456
2. Agent вызывает MCP tool → проверить HTTP-запрос к MCP server

**Ожидание:**
- HTTP-запрос к MCP серверу содержит headers X-Org-Id: org-123, X-User-Id: user-456

**PASS:** Forwarded headers передаются в MCP SSE requests

### TC-CTX-05: MCP SSE — forward_headers не configured
**Предусловие:** MCP server (SSE) без forward_headers в конфиге
**Шаги:**
1. POST /api/v1/agents/<agent>/chat с headers X-Org-Id
2. Agent вызывает MCP tool

**Ожидание:**
- HTTP-запрос к MCP серверу НЕ содержит X-Org-Id
- Только стандартные headers (Content-Type, etc.)

**PASS:** Без forward_headers — дополнительные headers не передаются

### TC-CTX-06: MCP SSE — header в конфиге, нет в request
**Предусловие:** MCP server с forward_headers: ["X-Org-Id"]
**Шаги:**
1. POST /api/v1/agents/<agent>/chat БЕЗ header X-Org-Id
2. Agent вызывает MCP tool

**Ожидание:**
- X-Org-Id не форвардится (нет значения)
- Ошибки нет, tool call проходит

**PASS:** Отсутствующий header не форвардится, без ошибки

### TC-CTX-07: MCP stdio — _context в JSON-RPC params
**Предусловие:** MCP server (stdio) с forward_headers: ["X-Org-Id", "X-User-Id"]
**Шаги:**
1. POST /api/v1/agents/<agent>/chat с headers X-Org-Id: org-123, X-User-Id: user-456
2. Agent вызывает MCP stdio tool

**Ожидание:**
- JSON-RPC params содержит `_context: {"X-Org-Id": "org-123", "X-User-Id": "user-456"}`

**PASS:** stdio transport передаёт context в _context поле

### TC-CTX-08: MCP stdio — нет forward_headers
**Предусловие:** MCP server (stdio) без forward_headers в конфиге
**Шаги:**
1. POST /api/v1/agents/<agent>/chat
2. Agent вызывает MCP stdio tool

**Ожидание:**
- Нет `_context` в JSON-RPC params
- Tool call работает как обычно

**PASS:** Без forward_headers — нет _context в params

### TC-CTX-09: Несколько MCP серверов — разные forward_headers
**Предусловие:** MCP server A с forward_headers: ["X-Org-Id"], MCP server B с forward_headers: ["X-User-Id"]
**Шаги:**
1. POST /api/v1/agents/<agent>/chat с обоими headers
2. Agent вызывает tool из server A, затем tool из server B

**Ожидание:**
- Server A получает только X-Org-Id
- Server B получает только X-User-Id
- Каждый сервер получает только свои configured headers

**PASS:** Разные MCP серверы получают только свои forward_headers

### TC-CTX-10: Security — LLM не видит forwarded headers
**Предусловие:** forward_headers настроены
**Шаги:**
1. Проверить system prompt и tool schema

**Ожидание:**
- Forwarded headers НЕ попадают в system prompt
- Forwarded headers НЕ попадают в tool schema/parameters
- LLM не может видеть или манипулировать forwarded headers

**PASS:** Headers скрыты от LLM

### TC-CTX-11: Security — forward только configured headers
**Предусловие:** forward_headers: ["X-Org-Id"]
**Шаги:**
1. POST /api/v1/agents/<agent>/chat с headers X-Org-Id: org-123, X-Secret: sensitive-data

**Ожидание:**
- Только X-Org-Id форвардится
- X-Secret НЕ форвардится (не в конфиге)

**PASS:** Только configured headers форвардятся

### TC-CTX-12: Backwards compat — нет forward_headers в конфиге
**Предусловие:** Старый конфиг без секции forward_headers
**Шаги:**
1. Запустить Engine со старым конфигом
2. Вызвать MCP tool

**Ожидание:**
- MCP работает как раньше (без forwarding)
- Нет ошибок при отсутствии forward_headers

**PASS:** Обратная совместимость с конфигом без forward_headers

---

## TC-AUDIT: Tool Call Audit Log — EE (13 TC)

### TC-AUDIT-01: GET /api/v1/audit/tool-calls с EE лицензией
**Предусловие:** Engine с EE лицензией, были выполнены tool calls
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/audit/tool-calls" -H "Authorization: Bearer <token>"`

**Ожидание:**
- HTTP 200
- JSON с пагинированным списком tool calls
- Каждый элемент содержит: tool_name, session_id, timestamp, status, duration_ms

**PASS:** Audit API возвращает пагинированный список tool calls

### TC-AUDIT-02: GET /api/v1/audit/tool-calls без EE лицензии
**Предусловие:** Engine без EE лицензии (CE mode)
**Шаги:**
1. `curl -s -w "\n%{http_code}" "http://localhost:8443/api/v1/audit/tool-calls" -H "Authorization: Bearer <token>"`

**Ожидание:**
- HTTP 403
- Body: `{"error": "Enterprise Edition required"}`

**PASS:** Audit endpoint без EE лицензии → 403

### TC-AUDIT-03: Фильтр по session_id
**Предусловие:** EE лицензия, несколько сессий с tool calls
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/audit/tool-calls?session_id=<uuid>" -H "Authorization: Bearer <token>"`

**Ожидание:**
- Только tool calls из указанной сессии
- Другие сессии не включены

**PASS:** Фильтр по session_id работает

### TC-AUDIT-04: Фильтр по tool_name
**Предусловие:** EE лицензия, разные tools использовались
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/audit/tool-calls?tool_name=knowledge_search" -H "Authorization: Bearer <token>"`

**Ожидание:**
- Только вызовы knowledge_search
- Другие tools не включены

**PASS:** Фильтр по tool_name работает

### TC-AUDIT-05: Фильтр по дате (from/to)
**Предусловие:** EE лицензия, tool calls за разные даты
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/audit/tool-calls?from=2026-03-01T00:00:00Z&to=2026-03-15T23:59:59Z" -H "Authorization: Bearer <token>"`

**Ожидание:**
- Только tool calls в указанном диапазоне дат
- Tool calls за пределами диапазона не включены

**PASS:** Фильтр по дате работает

### TC-AUDIT-06: Фильтр по user_id
**Предусловие:** EE лицензия, chat requests с X-User-Id header (Phase 4)
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/audit/tool-calls?user_id=user-456" -H "Authorization: Bearer <token>"`

**Ожидание:**
- Только tool calls с matching user_id (из RequestContext)

**PASS:** Фильтр по user_id работает

### TC-AUDIT-07: Pagination — page/per_page
**Предусловие:** EE лицензия, 20+ tool calls
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/audit/tool-calls?page=1&per_page=5" -H "Authorization: Bearer <token>"`
2. `curl -s "http://localhost:8443/api/v1/audit/tool-calls?page=2&per_page=5" -H "Authorization: Bearer <token>"`

**Ожидание:**
- Страница 1: 5 записей
- Страница 2: следующие 5 записей (не дубликаты)
- Total count корректен

**PASS:** Пагинация работает, total корректен

### TC-AUDIT-08: Duration вычислен
**Предусловие:** EE лицензия, tool calls выполнены
**Шаги:**
1. GET /api/v1/audit/tool-calls

**Ожидание:**
- Каждый tool call содержит duration_ms (число > 0)
- Duration вычислена из пары tool_call → tool_result events

**PASS:** duration_ms вычисляется корректно

### TC-AUDIT-09: Failed tool call в результатах
**Предусловие:** EE лицензия, был tool call который завершился с ошибкой
**Шаги:**
1. GET /api/v1/audit/tool-calls

**Ожидание:**
- Запись с status=failed видна в результатах
- error field содержит описание ошибки

**PASS:** Failed tool calls отображаются со status=failed

### TC-AUDIT-10: Нет X-User-ID в оригинальном request
**Предусловие:** EE лицензия, chat request без X-User-Id
**Шаги:**
1. Отправить chat → tool call
2. GET /api/v1/audit/tool-calls

**Ожидание:**
- user_id пуст (null или пустая строка)
- Ошибки нет, запись создана

**PASS:** Отсутствие X-User-Id → user_id пуст, без ошибки

### TC-AUDIT-11: Admin Dashboard — EE лицензия, вкладка Tool Calls
**Предусловие:** EE лицензия, залогинен в Admin
**Шаги:**
1. Перейти на /admin/audit-log
2. Найти вкладку "Tool Calls"

**Ожидание:**
- Вкладка "Tool Calls" видна
- Содержит таблицу с tool calls, фильтры

**PASS:** Вкладка Tool Calls видна при EE лицензии

### TC-AUDIT-12: Admin Dashboard — нет EE лицензии, вкладка скрыта
**Предусловие:** Нет EE лицензии, залогинен в Admin
**Шаги:**
1. Перейти на /admin/audit-log

**Ожидание:**
- Вкладка "Tool Calls" скрыта или показывает upgrade prompt
- Остальные audit записи (admin CRUD) видны

**PASS:** Tool Calls вкладка скрыта без EE лицензии

### TC-AUDIT-13: Backwards compat — session events не затронуты
**Предусловие:** Engine обновлён с Phase 5
**Шаги:**
1. GET /api/v1/sessions/<id>/events (существующий CE endpoint)

**Ожидание:**
- Endpoint работает как раньше
- Session events содержат tool_call/tool_result записи

**PASS:** Существующие session events API не затронуты

---

## TC-TRIG: Task Completion Webhook (11 TC)

### TC-TRIG-01: Task завершён — webhook fires
**Предусловие:** Trigger создан с on_complete_url (httpbin или requestbin)
**Шаги:**
1. Создать trigger с on_complete_url: "https://httpbin.org/post"
2. Trigger fires → task запускается и завершается

**Ожидание:**
- HTTP POST отправлен на configured URL
- Webhook вызывается после завершения задачи

**PASS:** Webhook fires при завершении задачи

### TC-TRIG-02: Webhook payload корректен
**Предусловие:** TC-TRIG-01
**Шаги:**
1. Проверить payload HTTP POST

**Ожидание:**
- JSON содержит: task_id, status, result, duration_ms, trigger_id, agent_name, timestamp
- Все поля заполнены корректно

**PASS:** Webhook payload содержит все необходимые поля

### TC-TRIG-03: Custom headers отправлены
**Предусловие:** Trigger с on_complete_headers: {"X-API-Key": "secret123"}
**Шаги:**
1. Task завершается → webhook fires

**Ожидание:**
- HTTP POST содержит header X-API-Key: secret123
- Custom headers из on_complete_headers передаются

**PASS:** Custom headers передаются в webhook

### TC-TRIG-04: Webhook URL недоступен
**Предусловие:** Trigger с on_complete_url указывающим на несуществующий сервер
**Шаги:**
1. Task завершается → webhook пытается отправить POST

**Ожидание:**
- Ошибка залогирована (connection refused/timeout)
- Task всё равно помечен как completed (webhook failure не влияет на task status)

**PASS:** Недоступный webhook — ошибка в логах, task complete

### TC-TRIG-05: Retry при 5xx
**Предусловие:** Webhook endpoint возвращает 500
**Шаги:**
1. Task завершается → webhook fires → 500 ответ

**Ожидание:**
- До 3 retries с exponential backoff
- Retries залогированы

**PASS:** Webhook retry до 3 раз при 5xx

### TC-TRIG-06: Нет webhook configured
**Предусловие:** Trigger без on_complete_url
**Шаги:**
1. Task завершается

**Ожидание:**
- HTTP POST не делается
- Нет ошибок, task complete как обычно

**PASS:** Без webhook URL — никаких HTTP calls

### TC-TRIG-07: Task failed — webhook fires со status=failed
**Предусловие:** Trigger с webhook, task завершается с ошибкой
**Шаги:**
1. Task fails (LLM error или timeout)

**Ожидание:**
- Webhook fires
- Payload содержит status=failed
- result содержит error description

**PASS:** Failed task отправляет webhook со status=failed

### TC-TRIG-08: API — create trigger с on_complete
**Предусловие:** JWT token получен
**Шаги:**
1. POST /api/v1/triggers с on_complete_url, on_complete_headers

**Ожидание:**
- HTTP 201
- Trigger создан с on_complete полями
- GET /api/v1/triggers/<id> содержит on_complete_url

**PASS:** Trigger с on_complete создаётся через API

### TC-TRIG-09: API — update trigger on_complete
**Предусловие:** Trigger создан (TC-TRIG-08)
**Шаги:**
1. PUT /api/v1/triggers/<id> с новым on_complete_url

**Ожидание:**
- HTTP 200
- on_complete_url обновлён

**PASS:** Trigger on_complete обновляется

### TC-TRIG-10: Backwards compat — триггеры без on_complete
**Предусловие:** Существующие триггеры без on_complete полей
**Шаги:**
1. GET /api/v1/triggers
2. Проверить что существующие триггеры работают

**Ожидание:**
- Существующие триггеры продолжают работать
- on_complete_url = null/пустой (не ошибка)

**PASS:** Старые триггеры работают без on_complete

### TC-TRIG-11: Admin UI — on_complete форма
**Предусловие:** Залогинен в Admin Dashboard
**Шаги:**
1. Перейти на /admin/triggers → Add/Edit Trigger
2. Проверить наличие секции On Complete

**Ожидание:**
- Поле URL для webhook
- Поле Headers (JSON или key-value)
- Секция видна в форме создания/редактирования триггера

**PASS:** Admin UI показывает on_complete поля в форме триггера

---

## TC-SSE: Typed ask_user, Structured Output (13 TC)

### TC-SSE-01: ask_user с input_type=single_select
**Предусловие:** Agent вызывает ask_user с input_type=single_select, options
**Шаги:**
1. Отправить сообщение вызывающее ask_user с single_select
2. Наблюдать SSE stream

**Ожидание:**
- SSE event user_question содержит input_type: "single_select"
- Options с labels и values

**PASS:** ask_user с input_type=single_select в SSE event

### TC-SSE-02: ask_user с input_type=multi_select
**Предусловие:** Agent вызывает ask_user с input_type=multi_select
**Шаги:**
1. Отправить сообщение вызывающее ask_user с multi_select

**Ожидание:**
- SSE event содержит input_type: "multi_select"
- Options с values для множественного выбора

**PASS:** ask_user с input_type=multi_select

### TC-SSE-03: ask_user с input_type=confirm
**Предусловие:** Agent вызывает ask_user с input_type=confirm
**Шаги:**
1. Отправить сообщение вызывающее ask_user с confirm

**Ожидание:**
- SSE event содержит input_type: "confirm"

**PASS:** ask_user с input_type=confirm

### TC-SSE-04: ask_user без input_type — backwards compat
**Предусловие:** Agent вызывает ask_user без input_type (старый формат)
**Шаги:**
1. Отправить сообщение вызывающее ask_user (старый формат)

**Ожидание:**
- SSE event содержит input_type: "text" (default)
- Обратная совместимость с существующими клиентами

**PASS:** Без input_type → default "text"

### TC-SSE-05: QuestionOption с value
**Предусловие:** ask_user с options, каждая с value
**Шаги:**
1. Отправить сообщение → ask_user с options

**Ожидание:**
- SSE event options содержат value (machine-readable)
- Value используется для matching ответа пользователя

**PASS:** QuestionOption с value в SSE

### TC-SSE-06: QuestionOption без value — value = label
**Предусловие:** ask_user с options без value field
**Шаги:**
1. Отправить сообщение → ask_user с options (без value)

**Ожидание:**
- Value по умолчанию = Label
- SSE event options: value совпадает с label

**PASS:** Без value → value = label

### TC-SSE-07: ask_user с columns field
**Предусловие:** ask_user с полем columns (для табличного отображения)
**Шаги:**
1. Отправить сообщение → ask_user с columns

**Ожидание:**
- SSE event содержит columns field
- Клиент может использовать columns для табличного UI

**PASS:** Columns field в ask_user SSE event

### TC-SSE-08: structured_output — summary_table
**Предусловие:** Agent вызывает structured_output tool с type=summary_table
**Шаги:**
1. Отправить сообщение вызывающее structured_output

**Ожидание:**
- SSE event type: "structured_output"
- Data содержит type: "summary_table", title, rows
- Rows — массив объектов с данными

**PASS:** structured_output event с summary_table

### TC-SSE-09: structured_output — actions
**Предусловие:** Agent вызывает structured_output с actions
**Шаги:**
1. Отправить сообщение → structured_output с action buttons

**Ожидание:**
- SSE event содержит actions array
- Каждый action: label, value

**PASS:** structured_output с action buttons

### TC-SSE-10: Backwards compat — старый клиент
**Предусловие:** Клиент не поддерживает input_type/structured_output
**Шаги:**
1. Отправить сообщение, получить SSE stream с новыми полями

**Ожидание:**
- Старые поля (questions, options) по-прежнему присутствуют
- Клиент может игнорировать новые поля без ошибок
- Не ломается если клиент не обрабатывает structured_output

**PASS:** Старые клиенты продолжают работать

### TC-SSE-11: Invalid/custom input_type — passthrough
**Предусловие:** Agent вызывает ask_user с кастомным input_type (напр. "kilo_team_form")
**Шаги:**
1. Отправить сообщение → ask_user с input_type="kilo_team_form"

**Ожидание:**
- input_type: "kilo_team_form" пробрасывается as-is
- Engine не валидирует, не блокирует кастомные типы
- Кастомный фронтенд может рендерить свой UI

**PASS:** Кастомный input_type пробрасывается без валидации

### TC-SSE-12: Multi-select response parsing
**Предусловие:** ask_user с input_type=multi_select
**Шаги:**
1. Пользователь отвечает с несколькими выбранными values
2. Проверить что agent получает массив значений

**Ожидание:**
- Несколько значений корректно парсятся
- Agent получает массив выбранных values

**PASS:** Multi-select response — массив значений

### TC-SSE-13: Confirm response parsing
**Предусловие:** ask_user с input_type=confirm
**Шаги:**
1. Пользователь отвечает "yes" или "no"

**Ожидание:**
- yes/no корректно обрабатывается
- Agent получает boolean-like ответ

**PASS:** Confirm yes/no обрабатывается корректно

---

## TC-RLIM: Configurable Rate Limiting — EE (15 TC)

### TC-RLIM-01: EE — лимит превышен → 429
**Предусловие:** EE лицензия, rate_limits configured: tier "free" = 50/24h
**Шаги:**
1. Отправить 51 POST /chat с X-Org-Id: org-test, X-Subscription-Tier: free

**Ожидание:**
- Первые 50 запросов: 200 OK
- 51-й запрос: 429 Too Many Requests
- Headers: Retry-After, X-RateLimit-Limit: 50, X-RateLimit-Remaining: 0, X-RateLimit-Reset

**PASS:** Лимит превышен → 429 с rate limit headers

### TC-RLIM-02: EE — разные key values независимы
**Предусловие:** EE лицензия, rate_limits по X-Org-Id
**Шаги:**
1. Исчерпать лимит для org-123 (50 requests → 429)
2. Отправить request с X-Org-Id: org-456

**Ожидание:**
- org-123: 429 (лимит исчерпан)
- org-456: 200 OK (свой отдельный счётчик)

**PASS:** Разные key values имеют независимые счётчики

### TC-RLIM-03: EE — tier "pro" → 500/day
**Предусловие:** EE лицензия, tier "pro" = 500/24h
**Шаги:**
1. Отправить requests с X-Subscription-Tier: pro

**Ожидание:**
- 500 requests проходят
- 501-й → 429
- X-RateLimit-Limit: 500

**PASS:** Tier "pro" ограничен 500/day

### TC-RLIM-04: EE — tier "enterprise" unlimited
**Предусловие:** EE лицензия, tier "enterprise" = unlimited: true
**Шаги:**
1. Отправить множество requests с X-Subscription-Tier: enterprise

**Ожидание:**
- Все requests проходят (нет лимита)
- Нет 429 ответов

**PASS:** Enterprise tier — без лимита

### TC-RLIM-05: EE — tier_header отсутствует → default_tier
**Предусловие:** EE лицензия, default_tier: "free"
**Шаги:**
1. Отправить request с X-Org-Id: org-test, БЕЗ X-Subscription-Tier

**Ожидание:**
- Используется default_tier "free"
- Лимит free tier применяется

**PASS:** Без tier_header → используется default_tier

### TC-RLIM-06: EE — key_header отсутствует
**Предусловие:** EE лицензия, rate_limits по X-Org-Id
**Шаги:**
1. Отправить request БЕЗ X-Org-Id

**Ожидание:**
- Rate limit rule не применяется для этого request (нет ключа)
- Request проходит

**PASS:** Без key_header → rate limit не применяется

### TC-RLIM-07: EE — несколько rate_limit rules
**Предусловие:** EE лицензия, 2 rules: per-org и per-user
**Шаги:**
1. Отправить request с X-Org-Id и X-User-Id
2. Исчерпать лимит по одному из rules

**Ожидание:**
- Все rules проверяются
- Первый deny → 429
- Если org limit ok, но user limit exceeded → 429

**PASS:** Несколько rules проверяются, первый deny → 429

### TC-RLIM-08: CE — нет EE лицензии, config игнорируется
**Предусловие:** Нет EE лицензии, rate_limits настроены в конфиге
**Шаги:**
1. Отправить requests

**Ожидание:**
- Configurable rate limits НЕ применяются (игнорируются)
- Per-IP (CE) rate limiter работает

**PASS:** Без EE лицензии — configurable limits игнорируются

### TC-RLIM-09: CE — per-IP сохранён
**Предусловие:** CE mode (нет EE лицензии)
**Шаги:**
1. Отправить множество requests без авторизации с одного IP

**Ожидание:**
- Per-IP rate limiting работает (unauthenticated DDoS защита)
- Requests лимитируются по IP

**PASS:** Per-IP rate limiter работает в CE mode

### TC-RLIM-10: Admin — rate limits UI (EE)
**Предусловие:** EE лицензия, залогинен в Admin
**Шаги:**
1. Перейти на /admin/settings → секция Rate Limits

**Ожидание:**
- Настройка rate limit rules видна
- Можно добавить/редактировать rules (key_header, tiers)

**PASS:** Rate limits UI доступна при EE лицензии

### TC-RLIM-11: Admin — rate limits UI (CE)
**Предусловие:** Нет EE лицензии, залогинен в Admin
**Шаги:**
1. Перейти на /admin/settings

**Ожидание:**
- Секция Rate Limits скрыта или показывает upgrade prompt
- CE per-IP настройки могут быть видны

**PASS:** Rate limits секция скрыта без EE лицензии

### TC-RLIM-12: Usage API
**Предусловие:** EE лицензия, requests были отправлены
**Шаги:**
1. `curl -s "http://localhost:8443/api/v1/rate-limits/usage?key_header=X-Org-Id&key_value=org-123" -H "Authorization: Bearer <token>"`

**Ожидание:**
- HTTP 200
- JSON с текущим count, limit, reset time

**PASS:** Usage API возвращает текущее потребление

### TC-RLIM-13: Sliding window accuracy
**Предусловие:** EE лицензия, rate limit с window: "1h"
**Шаги:**
1. Отправить requests до лимита → 429
2. Подождать window expiry (или mock time)

**Ожидание:**
- После истечения window — счётчик сбрасывается
- Новые requests проходят

**PASS:** Sliding window корректно сбрасывает счётчик

### TC-RLIM-14: Concurrent requests — thread safety
**Предусловие:** EE лицензия
**Шаги:**
1. Отправить 100 concurrent requests (напр. ab/hey/wrk)

**Ожидание:**
- Нет race conditions
- Counter корректен (не пропускает лишние requests сверх лимита)
- Нет panic/deadlock

**PASS:** Thread-safe счётчик при concurrent requests

### TC-RLIM-15: Backwards compat — нет rate_limits в config
**Предусловие:** Старый конфиг без секции rate_limits
**Шаги:**
1. Запустить Engine со старым конфигом

**Ожидание:**
- Только per-IP rate limiter работает
- Нет ошибок при отсутствии rate_limits секции
- Поведение как раньше

**PASS:** Без rate_limits в конфиге — работает как раньше

---

## TC-HELM: Helm Chart for Kubernetes (7 TC)

### TC-HELM-01: helm template renders
**Предусловие:** Helm установлен, chart в engine/deploy/helm/bytebrew/
**Шаги:**
1. `helm template bytebrew ./engine/deploy/helm/bytebrew/`

**Ожидание:**
- Нет template errors
- Сгенерированные YAML манифесты валидны
- Содержит: Deployment, Service, ConfigMap, Secret

**PASS:** helm template рендерится без ошибок

### TC-HELM-02: helm lint passes
**Предусловие:** Helm chart в engine/deploy/helm/bytebrew/
**Шаги:**
1. `helm lint ./engine/deploy/helm/bytebrew/`

**Ожидание:**
- Нет errors
- Нет critical warnings
- "1 chart(s) linted, 0 chart(s) failed"

**PASS:** helm lint проходит без ошибок

### TC-HELM-03: Custom values override
**Предусловие:** Helm chart
**Шаги:**
1. `helm template bytebrew ./engine/deploy/helm/bytebrew/ --set replicaCount=3 --set resources.limits.memory=1Gi`

**Ожидание:**
- Deployment replicas: 3
- resources.limits.memory: 1Gi
- Custom values корректно применяются

**PASS:** Custom values переопределяют defaults

### TC-HELM-04: HPA configured
**Предусловие:** Helm chart с autoscaling.enabled=true
**Шаги:**
1. `helm template bytebrew ./engine/deploy/helm/bytebrew/ --set autoscaling.enabled=true`

**Ожидание:**
- HPA manifest сгенерирован
- minReplicas, maxReplicas, targetCPUUtilization настроены

**PASS:** HPA настроен при autoscaling.enabled=true

### TC-HELM-05: Sticky sessions — Ingress annotation
**Предусловие:** Helm chart с ingress.enabled=true
**Шаги:**
1. `helm template bytebrew ./engine/deploy/helm/bytebrew/ --set ingress.enabled=true`

**Ожидание:**
- Ingress содержит annotation для session affinity (cookie-based)
- Необходимо для SSE connections

**PASS:** Ingress с sticky sessions annotation

### TC-HELM-06: ServiceMonitor для Prometheus
**Предусловие:** Helm chart с metrics.serviceMonitor.enabled=true
**Шаги:**
1. `helm template bytebrew ./engine/deploy/helm/bytebrew/ --set metrics.serviceMonitor.enabled=true`

**Ожидание:**
- ServiceMonitor manifest сгенерирован
- Endpoint: /metrics, port корректный
- Interval настроен

**PASS:** ServiceMonitor для Prometheus scraping

### TC-HELM-07: Secrets — API keys, DB URL
**Предусловие:** Helm chart
**Шаги:**
1. `helm template bytebrew ./engine/deploy/helm/bytebrew/ --set database.url="postgresql://..." --set apiKeys.openai="sk-..."`

**Ожидание:**
- Secret manifest содержит DB URL и API keys
- Deployment ссылается на Secret через envFrom или env.valueFrom

**PASS:** Sensitive данные в Kubernetes Secrets

---

## TC-METR: Prometheus Metrics — EE (6 TC)

### TC-METR-01: GET /metrics — Prometheus format
**Предусловие:** Engine запущен с EE лицензией
**Шаги:**
1. `curl -s http://localhost:8443/metrics`

**Ожидание:**
- HTTP 200
- Content-Type: text/plain (Prometheus exposition format)
- Содержит метрики: http_requests_total, http_request_duration_seconds, и др.

**PASS:** /metrics возвращает Prometheus text format

### TC-METR-02: HTTP counter increments
**Предусловие:** TC-METR-01
**Шаги:**
1. Записать текущее значение http_requests_total
2. Отправить несколько HTTP requests
3. Повторно GET /metrics

**Ожидание:**
- http_requests_total увеличилось на количество отправленных requests
- Labels: method, path, status_code

**PASS:** HTTP counter растёт с каждым request

### TC-METR-03: Duration histogram
**Предусловие:** Engine с EE лицензией
**Шаги:**
1. Отправить requests
2. GET /metrics

**Ожидание:**
- http_request_duration_seconds_bucket, _sum, _count присутствуют
- Наблюдения записываются в buckets

**PASS:** Duration histogram записывает наблюдения

### TC-METR-04: /metrics без auth
**Предусловие:** Engine запущен
**Шаги:**
1. `curl -s -w "\n%{http_code}" http://localhost:8443/metrics` (без Authorization header)

**Ожидание:**
- HTTP 200
- Metrics доступны без аутентификации (для Prometheus scraping)

**Примечание:** /metrics стандартно доступен без auth для совместимости с Prometheus

**PASS:** /metrics доступен без токена

### TC-METR-05: Tool call metrics
**Предусловие:** Engine с EE, tool calls выполнены
**Шаги:**
1. Выполнить chat с tool calls
2. GET /metrics

**Ожидание:**
- tool_calls_total metric присутствует
- Labels: tool_name
- Counter увеличился

**PASS:** tool_calls_total по tool_name

### TC-METR-06: LLM metrics
**Предусловие:** Engine с EE, LLM requests выполнены
**Шаги:**
1. Выполнить chat
2. GET /metrics

**Ожидание:**
- llm_requests_total metric присутствует
- Labels: provider (openai, anthropic, google, etc.)
- llm_request_duration_seconds histogram

**PASS:** LLM metrics по provider

---

## Итого: 256 TC

| Категория | Кол-во | Покрытие |
|-----------|--------|----------|
| TC-SITE | 16 | Сайт, docs, темы, скриншоты, порты |
| TC-INST | 7 | Docker install, update, restart |
| TC-ADMIN | 18 | Dashboard CRUD, auth, UX |
| TC-API | 12 | REST API, errors |
| TC-CHAT | 7 | SSE streaming, sessions |
| TC-DOC | 13 | Документация = реальность + контент валидация |
| TC-CLOUD | 11 | /examples/, auth popup, dashboard links |
| TC-EXAMPLE | 12 | Агентное поведение (MCP tools, RAG, rate limit, web-client) |
| TC-AUTH | 10 | Email verification, Google auth |
| TC-MCP | 13 | MCP docs server (functional + quality + URLs + hybrid search) |
| TC-EE | 27 | EE activation, Stripe, license lifecycle, downgrade |
| TC-REG | 14 | Model registry, tier warnings, OpenRouter preset |
| TC-PROV | 19 | LLM провайдеры (Azure OpenAI, Google Gemini) |
| TC-CTX | 12 | Request context propagation, MCP header forwarding |
| TC-AUDIT | 13 | Runtime audit log — EE (query, license check, filters) |
| TC-TRIG | 11 | Trigger webhooks (on_complete, retry, payload) |
| TC-SSE | 13 | Typed ask_user + structured_output (SSE extensions) |
| TC-RLIM | 15 | Configurable rate limiting — EE (per-header, tiers, usage) |
| TC-HELM | 7 | Helm chart (template, lint, values, affinity) |
| TC-METR | 6 | Prometheus metrics (/metrics, counters, histograms) |
| **ВСЕГО** | **256** |

### Примечания
- Multi-agent spawn работает через HTTP REST API и gRPC/WS
- TC-MCP-04/05 — quality tests: проверяют не только работоспособность но и качество RAG ответов
- TC-EE-* — требуют Stripe test mode и EE license для полного прогона
- TC-PROV-01..09 (Azure) — требуют Azure OpenAI credentials
- TC-PROV-10..19 (Gemini) — требуют Google API key
- TC-HELM-* — требуют Helm CLI, можно проверить без K8s кластера (helm template/lint)
- TC-METR-* — EE feature, требуют EE лицензию
