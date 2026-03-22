# ByteBrew Engine — Regression Test Cases

Полный список тест-кейсов для регрессионного тестирования.
При добавлении нового функционала — обновлять этот файл.

**Последнее обновление:** 2026-03-22
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
