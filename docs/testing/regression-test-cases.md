# ByteBrew Engine — Regression Test Cases

Полный список тест-кейсов для регрессионного тестирования.
При добавлении нового функционала — обновлять этот файл.

**Последнее обновление:** 2026-03-25
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

## TC-MCP-SH: Streamable HTTP MCP Transport (6 TC)

### TC-MCP-SH-01: Создание streamable-http MCP сервера через Admin API
**Шаги:** `POST /api/v1/mcp-servers` с `type: "streamable-http"`, `url: "https://gitbook.com/docs/~gitbook/mcp"`
**Ожидание:** Сервер создан, `type` = `streamable-http` в ответе и в БД

### TC-MCP-SH-02: Подключение к Streamable HTTP серверу (SSE response)
**Предусловие:** MCP сервер gitbook-test создан с type=streamable-http
**Шаги:** Рестарт Engine
**Ожидание:** Лог `MCP server connected name=gitbook-test tools=1`, transport отправляет POST с `Accept: application/json, text/event-stream`, парсит SSE ответ

### TC-MCP-SH-03: Подключение к Streamable HTTP серверу (JSON response)
**Шаги:** Unit test `TestStreamableHTTP_JSONResponse` — mock server отвечает `application/json`
**Ожидание:** Транспорт парсит JSON ответ корректно

### TC-MCP-SH-04: Session ID management
**Шаги:** Unit test `TestStreamableHTTP_SessionIDManagement`
**Ожидание:** Транспорт сохраняет `Mcp-Session-Id` из ответа сервера и отправляет его в последующих запросах

### TC-MCP-SH-05: Forward headers
**Шаги:** Создать streamable-http сервер с `forward_headers: ["Authorization"]`, отправить запрос с RequestContext
**Ожидание:** Заголовок Authorization проксируется в MCP запрос (unit test pattern из http_transport)

### TC-MCP-SH-06: Tool call через Streamable HTTP
**Шаги:** Прямой вызов `tools/call` с `name: "searchDocumentation"` на GitBook MCP endpoint
**Ожидание:** SSE ответ с результатами поиска (titles, links, content)
**PASS:** Проверен через curl — GitBook возвращает результаты по запросу "getting started"

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

## TC-KILO: Integration Issues (8 TC)

### TC-KILO-01: Config import принимает map формат
**Предусловие:** Engine запущен, admin JWT получен
**Шаги:**
1. Подготовить YAML в map формате: `agents: my-agent: model: glm-5`
2. POST /api/v1/config/import с Content-Type: application/x-yaml

**Ожидание:**
- 200 OK, конфигурация импортирована
- Agent "my-agent" создан с моделью glm-5

**PASS:** Config import принимает map формат

### TC-KILO-02: Config import принимает array формат
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/config/import с array форматом agents

**Ожидание:**
- 200 OK, backwards compatible

**PASS:** Config import принимает array формат

### TC-KILO-03: Chat с ненайденной моделью — graceful error (не panic)
**Предусловие:** Agent создан, привязанная модель удалена
**Шаги:**
1. POST /api/v1/agents/{name}/chat

**Ожидание:**
- HTTP 500 с error message "no model available"
- Engine НЕ падает (no panic, no restart)

**PASS:** Graceful error при ненайденной модели

### TC-KILO-04: MCP tools загружаются из assigned MCP server
**Предусловие:** MCP server создан (type: sse), agent привязан к MCP server
**Шаги:**
1. POST /api/v1/agents/{name}/chat с запросом использующим MCP tool

**Ожидание:**
- Agent видит MCP tools (tools_count > 1)
- Tool call к MCP серверу выполняется

**PASS:** MCP tools загружаются и вызываются

### TC-KILO-05: knowledge_search без knowledge path — skip
**Предусловие:** Agent с knowledge_search в tools, без knowledge path
**Шаги:**
1. POST /api/v1/agents/{name}/chat

**Ожидание:**
- Chat работает (не ошибка)
- knowledge_search пропущен с warning в логах

**PASS:** knowledge_search пропускается без ошибки

### TC-KILO-06: Docker config mount path
**Предусловие:** Docker container запущен
**Шаги:**
1. Mount config: `-v ./config.yaml:/config.yaml:ro`
2. Проверить Engine загружает конфиг

**Ожидание:**
- Config загружен, agents создаются из конфига

**PASS:** Docker config mount работает

### TC-KILO-07: system_file relative path в Docker
**Предусловие:** Docker container с mounted prompts directory
**Шаги:**
1. Agent config: `system_file: "./prompts/my-prompt.txt"`
2. Mount: `-v ./prompts:/app/prompts:ro`

**Ожидание:**
- System prompt загружен из файла

**PASS:** system_file relative path работает в Docker

### TC-KILO-08: API принимает оба поля system и system_prompt
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с `system: "prompt text"`
2. POST /api/v1/agents с `system_prompt: "prompt text"`

**Ожидание:**
- Оба варианта приняты, agent создан с correct system prompt

**PASS:** Оба поля system и system_prompt принимаются

---

## TC-CFG: Config Import/Export Edge Cases (11 TC)

### TC-CFG-01: Config export → re-import produces identical config
**Предусловие:** Engine запущен, есть настроенные agents и models
**Шаги:**
1. GET /api/v1/config/export → сохранить YAML
2. POST /api/v1/config/import с сохранённым YAML
3. GET /api/v1/config/export → сравнить с оригиналом

**Ожидание:**
- Re-import завершается 200 OK
- Повторный export идентичен оригиналу (все поля сохранены)

**Примечание:** Критично для GitOps workflow — config должна быть reproducible

### TC-CFG-02: Config import с пустой секцией agents → 200 OK
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/config/import с YAML где `agents: []`

**Ожидание:**
- HTTP 200 OK
- Существующие agents не удалены
- Новые agents не созданы

**Примечание:** Пустой список не должен вызывать ошибку или side effects

### TC-CFG-03: Config import с agents = null → без crash
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/config/import с YAML где `agents:` (без значения, null)

**Ожидание:**
- HTTP 200 OK (или 400 с понятной ошибкой)
- Engine НЕ падает (no panic)
- Логи без stacktrace

**Примечание:** null vs пустой список — оба варианта должны быть safe

### TC-CFG-04: Config import с пустой секцией models → OK
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/config/import с YAML где `models: []`

**Ожидание:**
- HTTP 200 OK
- Существующие models не затронуты

**Примечание:** Аналогично TC-CFG-02, но для models

### TC-CFG-05: Config import с UTF-8 символами в system prompt
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/config/import с YAML где agent имеет `system: "Привет! 你好 مرحبا 🌍"`
2. GET /api/v1/agents → проверить system prompt

**Ожидание:**
- Agent создан успешно
- System prompt содержит все UTF-8 символы без искажений
- Emoji сохранены корректно

**Примечание:** Проверка что pipeline не ломает Unicode

### TC-CFG-06: Config import с дублирующимся именем агента
**Предусловие:** Engine запущен, agent "support" уже существует
**Шаги:**
1. POST /api/v1/config/import с YAML содержащим agent "support" с другим system prompt

**Ожидание:**
- Существующий agent обновлён (upsert), НЕ создан дубликат
- Или: 409 Conflict с понятным сообщением
- В БД ровно один agent с именем "support"

**Примечание:** Дубликаты имён — источник багов в multi-agent routing

### TC-CFG-07: Config export включает ВСЕ поля models
**Предусловие:** Engine запущен, model создана с type, model_name, base_url, api_key
**Шаги:**
1. GET /api/v1/config/export → проверить YAML

**Ожидание:**
- YAML содержит для каждой model: type, model_name, base_url
- has_api_key = true (api_key НЕ экспортируется в открытом виде)
- Нет потерянных полей

**Примечание:** api_key — секрет, экспортировать только флаг наличия

### TC-CFG-08: Config export включает все поля agents
**Предусловие:** Engine запущен, agent с confirm_before, tools, mcp_servers
**Шаги:**
1. GET /api/v1/config/export → проверить YAML

**Ожидание:**
- YAML содержит: name, system (или system_file), model, tools, mcp_servers, confirm_before
- Все массивы (tools, mcp_servers, confirm_before) корректно сериализованы
- Пустые массивы НЕ опущены (для явности)

**Примечание:** Неполный export → broken re-import

### TC-CFG-09: Config import с malformed YAML → 400
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/config/import с невалидным YAML (broken indentation, missing colon)

**Ожидание:**
- HTTP 400 Bad Request
- Тело ответа содержит YAML parse error с указанием строки
- НЕ HTTP 500 Internal Server Error
- Engine продолжает работать

**Примечание:** Разница между 400 и 500 — критична для UX

### TC-CFG-10: system_file path разрешается относительно директории конфига
**Предусловие:** Engine запущен, config.yaml в /app/config/
**Шаги:**
1. Config содержит agent с `system_file: "./prompts/agent.txt"`
2. Файл существует по пути /app/config/prompts/agent.txt
3. POST /api/v1/config/import

**Ожидание:**
- System prompt загружен из файла
- Путь разрешён относительно директории config.yaml, НЕ CWD

**Примечание:** Relative path resolution — частый source of bugs при Docker mount

### TC-CFG-11: Config import через API → agents видны сразу
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/config/import с новым agent "fresh-agent"
2. GET /api/v1/agents (сразу, без рестарта)
3. POST /api/v1/agents/fresh-agent/chat → отправить сообщение

**Ожидание:**
- fresh-agent виден в списке агентов сразу после import
- Chat с fresh-agent работает без рестарта Engine
- Не требуется reload/restart

**Примечание:** Hot reload конфигурации — ключевое преимущество API import vs file

---

## TC-TOOL: Tool Calling & MCP Edge Cases (8 TC)

### TC-TOOL-01: MCP tool call timeout → graceful error
**Предусловие:** MCP server настроен, но намеренно задерживает ответ > 30s
**Шаги:**
1. POST /api/v1/agents/{name}/chat с запросом, вызывающим MCP tool
2. MCP server не отвечает 30+ секунд

**Ожидание:**
- Timeout error event отправлен клиенту через SSE
- Agent НЕ зависает — продолжает обработку или завершает с ошибкой
- Engine не блокирует другие запросы

**Примечание:** Timeout должен быть конфигурируемым (default 30s)

### TC-TOOL-02: Tool call с отсутствующим required параметром
**Предусловие:** Agent с tool, имеющим required параметры
**Шаги:**
1. LLM генерирует tool call без required параметра (или с null)

**Ожидание:**
- Error event с деталями: какой параметр отсутствует
- Agent получает ошибку и может retry или сообщить пользователю
- НЕ panic, НЕ silent fail

**Примечание:** LLM часто забывает required параметры — нужна валидация

### TC-TOOL-03: Tool result > 1MB → handled gracefully
**Предусловие:** MCP tool возвращает результат размером > 1MB
**Шаги:**
1. POST /api/v1/agents/{name}/chat → tool call
2. MCP server возвращает JSON > 1MB

**Ожидание:**
- Результат обработан (chunked или truncated с предупреждением)
- Engine НЕ OOM (out of memory)
- SSE event доставлен клиенту

**Примечание:** Защита от случайного dump базы данных через MCP tool

### TC-TOOL-04: MCP server возвращает non-JSON → graceful error
**Предусловие:** MCP server настроен, но возвращает plain text или HTML
**Шаги:**
1. POST /api/v1/agents/{name}/chat → tool call к MCP server
2. MCP server отвечает HTML (например, 502 page от proxy)

**Ожидание:**
- Error event: "MCP server returned invalid response"
- Engine НЕ crash (no panic on json.Unmarshal)
- Agent получает ошибку и может сообщить пользователю

**Примечание:** Часто случается при proxy/nginx перед MCP server

### TC-TOOL-05: MCP tools/list возвращает невалидный формат
**Предусловие:** MCP server настроен, но tools/list возвращает битый JSON
**Шаги:**
1. Engine загружает tools из MCP server при старте или по запросу
2. MCP server tools/list возвращает невалидный формат

**Ожидание:**
- MCP server пропущен (не добавлен в tool list)
- Warning в логах с именем MCP server и причиной
- Остальные MCP servers загружены корректно
- Engine продолжает работать

**Примечание:** Один битый MCP server не должен ломать весь agent

### TC-TOOL-06: Circular agent spawn → предотвращение
**Предусловие:** Agent A может spawn Agent B, Agent B может spawn Agent A
**Шаги:**
1. POST /api/v1/agents/agent-a/chat с запросом, вызывающим spawn agent-b
2. Agent B в свою очередь вызывает spawn agent-a

**Ожидание:**
- Circular spawn предотвращён (max depth или cycle detection)
- Error event с сообщением о circular dependency
- Engine НЕ уходит в бесконечный цикл
- Ресурсы (goroutines, memory) не утекают

**Примечание:** Max spawn depth рекомендуется 3-5 уровней

### TC-TOOL-07: EE audit log записывает tool calls
**Предусловие:** EE лицензия активна, audit log включён
**Шаги:**
1. POST /api/v1/agents/{name}/chat → agent вызывает tool
2. GET /api/v1/audit-log → найти запись о tool call

**Ожидание:**
- Audit log содержит запись с type "tool_call"
- Запись содержит: tool name, arguments, result (или result summary)
- Timestamp и request_id присутствуют

**Примечание:** EE feature — требует активную EE лицензию

### TC-TOOL-08: Tool execution error → error SSE event
**Предусловие:** Agent с tool, tool выбрасывает ошибку при выполнении
**Шаги:**
1. POST /api/v1/agents/{name}/chat → LLM вызывает tool
2. Tool возвращает ошибку (runtime error, connection refused, etc.)

**Ожидание:**
- SSE event type "error" или "tool_error" отправлен клиенту
- Event содержит: tool name, error message
- Agent продолжает работу (может retry или ответить пользователю)
- НЕ silent fail (клиент ДОЛЖЕН знать об ошибке)

**Примечание:** Клиент должен иметь возможность показать ошибку пользователю

---

## TC-CRASH: Crash Prevention & Validation (9 TC)

### TC-CRASH-01: Создание agent с пустым именем → 400
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с `name: ""`

**Ожидание:**
- HTTP 400 Bad Request
- Тело ответа: validation error "name is required"
- Agent НЕ создан в БД

**Примечание:** Пустое имя — невалидный идентификатор для routing

### TC-CRASH-02: Создание agent со спецсимволами в имени
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с `name: "agent/with<special\"chars"`

**Ожидание:**
- HTTP 400 Bad Request
- Validation error: недопустимые символы в имени
- Agent НЕ создан

**Примечание:** Спецсимволы (/, ", <, >) ломают URL routing и HTML rendering

### TC-CRASH-03: Создание agent с именем > 255 символов
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с name длиной 300+ символов

**Ожидание:**
- HTTP 400 Bad Request
- Validation error: имя слишком длинное
- Agent НЕ создан

**Примечание:** Защита от overflow в БД и URL

### TC-CRASH-04: Создание agent с неизвестным tool → 400
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с `tools: ["nonexistent_tool_xyz"]`

**Ожидание:**
- HTTP 400 Bad Request
- Error message: unknown tool "nonexistent_tool_xyz"
- Agent НЕ создан
- Engine НЕ crash при попытке resolve tool

**Примечание:** Typo в tool name не должен вызывать panic при chat

### TC-CRASH-05: PUT /api/v1/agents/{nonexistent} → 404
**Предусловие:** Engine запущен, agent "ghost" НЕ существует
**Шаги:**
1. PUT /api/v1/agents/ghost с валидным body

**Ожидание:**
- HTTP 404 Not Found
- Error message: agent "ghost" not found
- НЕ создаёт новый agent (PUT ≠ upsert)

**Примечание:** Различие PUT (update) vs POST (create)

### TC-CRASH-06: DELETE agent дважды → 404 на второй раз
**Предусловие:** Engine запущен, agent "temp" существует
**Шаги:**
1. DELETE /api/v1/agents/temp → 200 OK
2. DELETE /api/v1/agents/temp → повторный запрос

**Ожидание:**
- Первый DELETE: 200 OK (или 204 No Content)
- Второй DELETE: 404 Not Found
- НЕ 500, НЕ panic

**Примечание:** Идемпотентность vs повторные запросы

### TC-CRASH-07: Chat с model с невалидным API key → error response
**Предусловие:** Engine запущен, model создана с невалидным api_key
**Шаги:**
1. POST /api/v1/agents/{name}/chat (agent привязан к model с bad key)

**Ожидание:**
- Error SSE event: "authentication failed" или "invalid API key"
- Engine НЕ panic (no nil pointer, no unhandled error)
- HTTP connection закрывается gracefully

**Примечание:** Невалидный API key — самая частая ошибка при настройке

### TC-CRASH-08: Создание model с дублирующимся именем → 409
**Предусловие:** Engine запущен, model "gpt-4" уже существует
**Шаги:**
1. POST /api/v1/models с `name: "gpt-4"` (уже существует)

**Ожидание:**
- HTTP 409 Conflict
- Error message: model "gpt-4" already exists
- Существующая model НЕ изменена

**Примечание:** Unique constraint на имя model

### TC-CRASH-09: DELETE model, привязанной к agent → error
**Предусловие:** Engine запущен, model "main-model" используется agent "support"
**Шаги:**
1. DELETE /api/v1/models/main-model

**Ожидание:**
- HTTP 409 Conflict (или 400 Bad Request)
- Error message: model is referenced by agent(s): "support"
- Model НЕ удалена
- Agent "support" продолжает работать

**Примечание:** Foreign key protection — удаление зависимости ломает агента

---

## TC-AG-EXT: Agent CRUD Edge Cases (4 TC)

### TC-AG-EXT-01: Создание agent с пустым system_prompt
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с `system_prompt: ""`

**Ожидание:**
- Agent создан успешно (200/201) ИЛИ validation error с подсказкой
- Если создан — при chat используется default system prompt или пустой prompt

**Примечание:** Пустой system_prompt — валидный кейс (agent без инструкций)

### TC-AG-EXT-02: Создание agent с пустым tools array
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с `tools: []`

**Ожидание:**
- HTTP 200/201 — agent создан
- Agent работает в chat без tool calling
- GET /api/v1/agents/{name} возвращает `tools: []`

**Примечание:** Agent без tools — валидный кейс (чистый LLM без инструментов)

### TC-AG-EXT-03: Смена модели agent через PUT
**Предусловие:** Engine запущен, agent "test-agent" создан с model "model-a", model "model-b" существует
**Шаги:**
1. PUT /api/v1/agents/test-agent с `model: "model-b"`
2. POST /api/v1/agents/test-agent/chat → отправить сообщение

**Ожидание:**
- PUT возвращает 200 OK
- Chat использует model-b (видно в логах или audit)
- Старая model-a больше не используется этим agent

**Примечание:** Горячая смена модели без перезапуска

### TC-AG-EXT-04: Имя agent максимальной длины (255 символов)
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с name длиной ровно 255 символов (валидных)

**Ожидание:**
- HTTP 200/201 — agent создан
- GET /api/v1/agents/{name} возвращает полное имя
- Agent доступен по URL с полным именем

**Примечание:** Граничное значение — ровно на лимите

---

## TC-MD-EXT: Model CRUD Edge Cases (4 TC)

### TC-MD-EXT-01: Создание model с пустым именем
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/models с `name: ""`

**Ожидание:**
- HTTP 400 Bad Request
- Validation error: name is required
- Model НЕ создана

**Примечание:** Пустое имя невалидно для routing

### TC-MD-EXT-02: Создание model с невалидным base_url
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/models с `base_url: "not-a-url"`

**Ожидание:**
- HTTP 400 Bad Request
- Validation error: invalid base_url format
- Model НЕ создана

**Примечание:** base_url должен быть валидным URL

### TC-MD-EXT-03: DELETE model, используемой в активной сессии
**Предусловие:** Engine запущен, model используется в активном chat
**Шаги:**
1. Начать chat с agent, привязанным к model
2. Во время streaming DELETE /api/v1/models/{name}

**Ожидание:**
- DELETE заблокирован (409 Conflict) ИЛИ предупреждение
- Активная сессия не прерывается crash-ем

**Примечание:** Защита от удаления ресурса во время использования

### TC-MD-EXT-04: Обновление api_key модели
**Предусловие:** Engine запущен, model "test-model" существует
**Шаги:**
1. PUT /api/v1/models/test-model с новым `api_key`
2. POST /api/v1/agents/{name}/chat → отправить сообщение

**Ожидание:**
- PUT возвращает 200 OK
- Новый api_key используется при следующем chat запросе
- Старый ключ больше не используется

**Примечание:** Ротация ключей без перезапуска Engine

---

## TC-MC-EXT: MCP Server Edge Cases (5 TC)

### TC-MC-EXT-01: MCP server с невалидным URL
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/mcp-servers с `url: "not-a-valid-url"`

**Ожидание:**
- HTTP 400 Bad Request
- Validation error: invalid URL format
- MCP server НЕ создан

**Примечание:** URL должен быть валидным для подключения

### TC-MC-EXT-02: MCP server с недоступным хостом
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/mcp-servers с `url: "http://192.0.2.1:9999"` (unreachable)
2. Назначить MCP server агенту
3. Запросить tools/list

**Ожидание:**
- Создание MCP server → 200/201 (конфиг сохранён)
- При tools/list → graceful error (timeout/connection refused)
- Engine НЕ crash, НЕ hang

**Примечание:** Создание != подключение, ошибка при runtime а не при создании

### TC-MC-EXT-03: MCP server с невалидным type
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/mcp-servers с `type: "invalid_type"`

**Ожидание:**
- HTTP 400 Bad Request
- Validation error: invalid type (expected "sse", "stdio", etc.)
- MCP server НЕ создан

**Примечание:** Перечисление допустимых типов

### TC-MC-EXT-04: Назначение несуществующего MCP server агенту
**Предусловие:** Engine запущен, MCP server "ghost-mcp" НЕ существует
**Шаги:**
1. PUT /api/v1/agents/{name} с `mcp_servers: ["ghost-mcp"]`

**Ожидание:**
- HTTP 404 Not Found ИЛИ validation error
- Agent НЕ обновлён с невалидной ссылкой

**Примечание:** Целостность ссылок между entities

### TC-MC-EXT-05: MCP connection timeout
**Предусловие:** Engine запущен, MCP server сконфигурирован с медленным endpoint
**Шаги:**
1. Назначить MCP server агенту
2. Отправить chat message, вызывающий MCP tool

**Ожидание:**
- При timeout → graceful error в SSE stream
- Engine НЕ crash, НЕ hang
- Сообщение об ошибке информативное (timeout, host unreachable)

**Примечание:** Таймауты не должны убивать процесс

---

## TC-AUTH-EXT: Authentication Edge Cases (9 TC)

### TC-AUTH-EXT-01: Tampered JWT signature
**Предусловие:** Engine запущен, валидный JWT получен
**Шаги:**
1. Изменить последний символ JWT signature
2. Отправить запрос с изменённым JWT в Authorization header

**Ожидание:**
- HTTP 401 Unauthorized
- Error message: invalid token signature

**Примечание:** Защита от подделки токенов

### TC-AUTH-EXT-02: Expired JWT token
**Предусловие:** Engine запущен
**Шаги:**
1. Использовать JWT с истёкшим exp claim

**Ожидание:**
- HTTP 401 Unauthorized
- Error message: token expired

**Примечание:** TTL enforcement

### TC-AUTH-EXT-03: Missing "Bearer" prefix
**Предусловие:** Engine запущен
**Шаги:**
1. Отправить запрос с `Authorization: <token>` (без "Bearer")

**Ожидание:**
- HTTP 401 Unauthorized
- Токен не принят без prefix

**Примечание:** Строгое соответствие RFC 6750

### TC-AUTH-EXT-04: "Bearer" без токена
**Предусловие:** Engine запущен
**Шаги:**
1. Отправить запрос с `Authorization: Bearer ` (пробел, без токена)

**Ожидание:**
- HTTP 401 Unauthorized
- Error message: token required

**Примечание:** Edge case пустого токена

### TC-AUTH-EXT-05: Concurrent logins одного пользователя
**Предусловие:** Engine запущен, пользователь существует
**Шаги:**
1. Выполнить POST /api/v1/auth/login параллельно 10 раз с одними credentials

**Ожидание:**
- Все 10 запросов возвращают 200 OK
- Каждый возвращает валидный (возможно разный) JWT
- Нет race condition, нет 500 ошибок

**Примечание:** Параллельный логин — частый сценарий

### TC-AUTH-EXT-06: API key с неправильным scope
**Предусловие:** Engine запущен, API key создан со scope "read"
**Шаги:**
1. Использовать API key для POST (write) операции

**Ожидание:**
- HTTP 403 Forbidden
- Error message: insufficient permissions / wrong scope

**Примечание:** Granular access control

### TC-AUTH-EXT-07: Invalid API key
**Предусловие:** Engine запущен
**Шаги:**
1. Отправить запрос с `X-API-Key: invalid-key-12345`

**Ожидание:**
- HTTP 401 Unauthorized
- Error message: invalid API key

**Примечание:** Несуществующий ключ

### TC-AUTH-EXT-08: Revoked/deleted API key
**Предусловие:** Engine запущен, API key создан и затем удалён
**Шаги:**
1. Создать API key → запомнить значение
2. Удалить API key через admin
3. Использовать удалённый API key в запросе

**Ожидание:**
- HTTP 401 Unauthorized
- Error message: API key not found / revoked

**Примечание:** Мгновенная инвалидация после удаления

### TC-AUTH-EXT-09: Разные пользователи с одинаковым паролем
**Предусловие:** Engine запущен, два пользователя с одинаковым паролем
**Шаги:**
1. POST /api/v1/auth/login для user-a
2. POST /api/v1/auth/login для user-b (тот же пароль)

**Ожидание:**
- Оба получают уникальные JWT
- Токены содержат разные user claims
- Доступ к ресурсам изолирован

**Примечание:** Пароль не влияет на уникальность токена

---

## TC-SESS: Session Edge Cases (7 TC)

### TC-SESS-01: Использование expired session_id
**Предусловие:** Engine запущен, session_id был активен но истёк (TTL)
**Шаги:**
1. POST /api/v1/agents/{name}/chat с session_id из истёкшей сессии

**Ожидание:**
- Создаётся новая сессия ИЛИ возвращается ошибка "session expired"
- НЕ crash, НЕ возвращает данные чужой сессии

**Примечание:** Expired session — не ошибка, а стандартный lifecycle

### TC-SESS-02: Invalid UUID format в session_id
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents/{name}/chat с `session_id: "not-a-uuid"`

**Ожидание:**
- HTTP 400 Bad Request ИЛИ создаётся новая сессия (зависит от дизайна)
- НЕ panic, НЕ SQL error в логах

**Примечание:** Невалидный формат не должен ломать парсинг

### TC-SESS-03: SQL injection в session_id
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents/{name}/chat с `session_id: "'; DROP TABLE sessions;--"`

**Ожидание:**
- HTTP 400 Bad Request
- SQL injection НЕ выполнен
- Таблица sessions цела

**Примечание:** Parameterized queries обязательны

### TC-SESS-04: Concurrent запросы с одним session_id
**Предусловие:** Engine запущен, активная сессия
**Шаги:**
1. Отправить 5 параллельных POST /api/v1/agents/{name}/chat с одним session_id

**Ожидание:**
- Нет race condition
- Нет дублирования сообщений в истории
- Все 5 запросов обработаны или часть отклонена (429)

**Примечание:** Concurrent writes в одну сессию

### TC-SESS-05: Chat с deleted session_id
**Предусловие:** Engine запущен, сессия была создана и затем удалена
**Шаги:**
1. Создать сессию → получить session_id
2. Удалить сессию (если есть API)
3. POST /api/v1/agents/{name}/chat с удалённым session_id

**Ожидание:**
- Создаётся новая сессия ИЛИ ошибка "session not found"
- НЕ crash

**Примечание:** Orphaned session reference

### TC-SESS-06: SSE client disconnect → cleanup
**Предусловие:** Engine запущен, SSE stream активен
**Шаги:**
1. Начать chat → SSE stream открыт
2. Закрыть клиент (обрыв соединения)
3. Подождать 30 секунд
4. Проверить goroutine count / memory

**Ожидание:**
- Goroutines, связанные с сессией, завершены
- Нет memory leak
- Нет zombie SSE connections

**Примечание:** Клиент может отвалиться в любой момент

### TC-SESS-07: Сообщение > 100KB
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents/{name}/chat с message длиной > 100KB

**Ожидание:**
- Сообщение обработано ИЛИ 413 Payload Too Large
- Engine НЕ crash, НЕ OOM
- Если обработано — ответ корректный

**Примечание:** Защита от oversize requests

---

## TC-DOCKER: Docker Deployment (7 TC)

### TC-DOCKER-01: Missing DATABASE_URL
**Предусловие:** Docker image собран
**Шаги:**
1. `docker run bytebrew/engine` без DATABASE_URL environment variable

**Ожидание:**
- Container выходит с ненулевым кодом
- В логах — понятное сообщение: "DATABASE_URL is required"
- НЕ panic, НЕ cryptic error

**Примечание:** Helpful error messages при missing config

### TC-DOCKER-02: Невалидный config path
**Предусловие:** Docker image собран
**Шаги:**
1. `docker run -v /nonexistent:/config bytebrew/engine` с указанием несуществующего конфига

**Ожидание:**
- Container стартует с defaults ИЛИ показывает понятную ошибку
- НЕ crash

**Примечание:** Fallback при отсутствии конфига

### TC-DOCKER-03: Volume permissions error
**Предусловие:** Docker image собран
**Шаги:**
1. `docker run -v /root-only-dir:/data bytebrew/engine` с read-only volume

**Ожидание:**
- Понятное сообщение об ошибке доступа
- Container не зависает

**Примечание:** Permission errors — частая проблема в Docker

### TC-DOCKER-04: Health check < 1s under load
**Предусловие:** Container запущен, под нагрузкой (100 concurrent requests)
**Шаги:**
1. Запустить нагрузку на /api/v1/agents/{name}/chat
2. Параллельно проверить GET /health (или /readyz)

**Ожидание:**
- Health check отвечает за < 1 секунду
- HTTP 200 OK

**Примечание:** Health check не должен блокироваться business logic

### TC-DOCKER-05: SIGTERM → graceful shutdown
**Предусловие:** Container запущен, активные SSE streams
**Шаги:**
1. Начать несколько chat sessions
2. `docker stop <container>` (SIGTERM)

**Ожидание:**
- Активные streams завершаются gracefully
- Container выходит за < 30 секунд
- Нет data corruption

**Примечание:** Kubernetes/Docker полагаются на SIGTERM для graceful shutdown

### TC-DOCKER-06: Container restart → state preserved
**Предусловие:** Container запущен с PostgreSQL volume
**Шаги:**
1. Создать agents, models через API
2. `docker restart <container>`
3. GET /api/v1/agents → проверить данные

**Ожидание:**
- Все agents, models сохранены
- Сессии (в БД) доступны
- Конфигурация не потеряна

**Примечание:** State в PostgreSQL, не в памяти

### TC-DOCKER-07: Read-only config mount
**Предусловие:** Docker image собран
**Шаги:**
1. `docker run -v ./config.yaml:/app/config.yaml:ro bytebrew/engine`

**Ожидание:**
- Container стартует нормально
- Config читается, но Engine не пытается писать в config file
- Все изменения конфигурации через API/DB

**Примечание:** Immutable infrastructure pattern

---

## TC-CONC: Concurrency (2 TC)

### TC-CONC-01: Создание agent с одинаковым именем параллельно
**Предусловие:** Engine запущен, agent "race-test" НЕ существует
**Шаги:**
1. Отправить 10 параллельных POST /api/v1/agents с `name: "race-test"`

**Ожидание:**
- Ровно один запрос возвращает 201 Created
- Остальные получают 409 Conflict
- Нет дублей в БД
- Нет 500 ошибок

**Примечание:** Unique constraint + concurrent writes

### TC-CONC-02: DELETE model во время активного chat
**Предусловие:** Engine запущен, model "active-model" используется в chat
**Шаги:**
1. Начать chat с agent, привязанным к "active-model"
2. Во время streaming — DELETE /api/v1/models/active-model

**Ожидание:**
- DELETE заблокирован (409) ИЛИ текущий stream завершается, следующий запрос ошибка
- Engine НЕ panic
- Нет data corruption

**Примечание:** Graceful handling конкурентных операций

---

## TC-PROV-EXT: Provider Edge Cases (11 TC)

### TC-PROV-EXT-01: openai_compatible с невалидным URL
**Предусловие:** Engine запущен
**Шаги:**
1. Создать model с provider "openai_compatible" и `base_url: "not-a-url"`
2. Попытаться отправить chat message

**Ожидание:**
- При chat → error в SSE stream (connection failed)
- Engine НЕ crash

**Примечание:** Валидация URL формата на уровне model creation или runtime

### TC-PROV-EXT-02: openai_compatible без api_key
**Предусловие:** Engine запущен
**Шаги:**
1. Создать model с provider "openai_compatible" без api_key
2. Отправить chat message

**Ожидание:**
- Error: authentication failed / API key required
- Engine НЕ crash

**Примечание:** Некоторые local LLM серверы не требуют key — зависит от провайдера

### TC-PROV-EXT-03: Провайдер возвращает unexpected response fields
**Предусловие:** Engine запущен, model сконфигурирована
**Шаги:**
1. Провайдер возвращает response с дополнительными unknown полями

**Ожидание:**
- Unknown поля игнорируются
- Парсинг не ломается
- Ответ доставлен клиенту

**Примечание:** Forward compatibility с новыми версиями API провайдера

### TC-PROV-EXT-04: OpenRouter — несуществующая модель
**Предусловие:** Engine запущен, model с provider "openrouter"
**Шаги:**
1. Указать model_id: "nonexistent/model-name"
2. Отправить chat message

**Ожидание:**
- Error: model not found / invalid model
- Сообщение содержит имя модели для диагностики

**Примечание:** Опечатка в model_id — частая ошибка

### TC-PROV-EXT-05: Azure — missing deployment
**Предусловие:** Engine запущен, Azure OpenAI model сконфигурирована
**Шаги:**
1. Указать несуществующий deployment_name
2. Отправить chat message

**Ожидание:**
- Error: deployment not found
- Детальное сообщение от Azure API

**Примечание:** Azure требует deployment, а не model name

### TC-PROV-EXT-06: Azure — wrong key format
**Предусловие:** Engine запущен
**Шаги:**
1. Создать Azure model с обычным OpenAI key (неправильный формат)
2. Отправить chat message

**Ожидание:**
- HTTP 401 от Azure API
- Error в SSE stream: authentication failed

**Примечание:** Azure keys отличаются от OpenAI keys по формату

### TC-PROV-EXT-07: Azure — key rotation
**Предусловие:** Engine запущен, Azure model работает
**Шаги:**
1. Обновить api_key через PUT /api/v1/models/{name}
2. Отправить chat message

**Ожидание:**
- Новый ключ используется немедленно
- Chat работает с новым ключом

**Примечание:** Ротация ключей без downtime

### TC-PROV-EXT-08: Gemini — tool support detection
**Предусловие:** Engine запущен, Gemini model сконфигурирована
**Шаги:**
1. Создать agent с tools, привязанный к Gemini model
2. Отправить chat message, вызывающий tool

**Ожидание:**
- Tool call корректно форматируется для Gemini API
- Ответ tool парсится правильно
- Или ошибка если модель не поддерживает tools

**Примечание:** Gemini имеет свой формат tool calling

### TC-PROV-EXT-09: Gemini — JSON response parsing edge cases
**Предусловие:** Engine запущен, Gemini model сконфигурирована
**Шаги:**
1. Запросить structured output от Gemini

**Ожидание:**
- JSON response корректно распарсен
- Нет ошибок при наличии nested objects, arrays, null values

**Примечание:** Gemini JSON format может отличаться от OpenAI

### TC-PROV-EXT-10: Gemini — несуществующая модель
**Предусловие:** Engine запущен
**Шаги:**
1. Создать model с provider "gemini" и model_id: "nonexistent-model"
2. Отправить chat message

**Ожидание:**
- Error: model not found
- Информативное сообщение об ошибке

**Примечание:** Список моделей Gemini ограничен

### TC-PROV-EXT-11: Gemini — rate limit 429
**Предусловие:** Engine запущен, Gemini model сконфигурирована
**Шаги:**
1. Отправить много запросов до получения 429 от Gemini

**Ожидание:**
- Error в SSE stream: rate limited
- Engine НЕ crash
- Retry logic (если есть) с backoff

**Примечание:** Rate limiting на стороне провайдера

---

## TC-KNOW: Knowledge/RAG Edge Cases (7 TC)

### TC-KNOW-01: Agent без knowledge path
**Предусловие:** Engine запущен, agent без knowledge конфигурации
**Шаги:**
1. POST /api/v1/agents/{name}/chat с обычным сообщением

**Ожидание:**
- Chat работает без RAG
- Нет ошибок, нет warning в логах
- Ответ генерируется на основе system_prompt и LLM

**Примечание:** Уже исправлено — skip knowledge если не настроен

### TC-KNOW-02: Agent с несуществующим knowledge path
**Предусловие:** Engine запущен
**Шаги:**
1. Создать/обновить agent с knowledge_path: "/nonexistent/path"

**Ожидание:**
- Ошибка при конфигурации ИЛИ при первом chat
- Понятное сообщение: path not found

**Примечание:** Валидация path при создании или lazy при использовании

### TC-KNOW-03: Knowledge path — файл вместо директории
**Предусловие:** Engine запущен
**Шаги:**
1. Указать knowledge_path на конкретный файл (не директорию)

**Ожидание:**
- Файл обработан как единственный документ ИЛИ ошибка "expected directory"
- НЕ crash

**Примечание:** Чёткое поведение для файла vs директории

### TC-KNOW-04: Неподдерживаемый формат файла в knowledge
**Предусловие:** Engine запущен, knowledge directory содержит .exe, .bin файлы
**Шаги:**
1. Добавить binary файл в knowledge directory
2. Запустить индексацию

**Ожидание:**
- Неподдерживаемые файлы пропущены (skipped)
- Warning в логах: "skipping unsupported file: X"
- Поддерживаемые файлы проиндексированы нормально

**Примечание:** Graceful skip, не hard error

### TC-KNOW-05: Пустой search query
**Предусловие:** Engine запущен, knowledge проиндексирован
**Шаги:**
1. Внутренний RAG search с пустым query

**Ожидание:**
- Пустой результат ИЛИ top-N документов
- НЕ crash, НЕ SQL error

**Примечание:** Edge case пустого запроса

### TC-KNOW-06: Очень длинный query (10k символов)
**Предусловие:** Engine запущен, knowledge проиндексирован
**Шаги:**
1. Отправить chat message длиной 10000+ символов (вызывает RAG search)

**Ожидание:**
- Запрос обработан (возможно, truncated для embedding)
- НЕ crash, НЕ OOM
- Ответ возвращён

**Примечание:** Embedding models имеют ограничение на длину input

### TC-KNOW-07: Large knowledge file > 100MB
**Предусловие:** Engine запущен, knowledge directory содержит файл > 100MB
**Шаги:**
1. Запустить индексацию с большим файлом

**Ожидание:**
- Файл обработан (возможно, chunked) ИЛИ пропущен с предупреждением
- Индексация не зависает
- Memory usage остаётся в разумных пределах

**Примечание:** Performance при больших файлах

---

## TC-TRIG-EXT: Trigger/Webhook Edge Cases (7 TC)

### TC-TRIG-EXT-01: Webhook retry с backoff timing
**Предусловие:** Engine запущен, webhook endpoint недоступен
**Шаги:**
1. Создать trigger с webhook URL
2. Trigger fires → endpoint возвращает 500

**Ожидание:**
- Retry с exponential backoff (1s, 2s, 4s...)
- Количество retries ограничено (max 3-5)
- После max retries — событие в логах

**Примечание:** Backoff предотвращает перегрузку endpoint

### TC-TRIG-EXT-02: Custom headers со спецсимволами
**Предусловие:** Engine запущен
**Шаги:**
1. Создать trigger с custom headers содержащими спецсимволы (unicode, кавычки)

**Ожидание:**
- Headers корректно закодированы в HTTP запросе
- Webhook endpoint получает правильные значения

**Примечание:** HTTP header encoding

### TC-TRIG-EXT-03: Webhook body > 1MB
**Предусловие:** Engine запущен, trigger настроен
**Шаги:**
1. Trigger fire с payload > 1MB (очень длинный chat result)

**Ожидание:**
- Payload truncated до лимита ИЛИ отправлен полностью
- НЕ OOM
- Webhook endpoint получает данные

**Примечание:** Защита от oversize payloads

### TC-TRIG-EXT-04: Webhook URL с credentials
**Предусловие:** Engine запущен
**Шаги:**
1. Создать trigger с URL вида `http://user:pass@example.com/webhook`

**Ожидание:**
- Заблокировано (credentials в URL — security risk) ИЛИ credentials stripped из логов
- Пароль НЕ отображается в audit logs / admin UI

**Примечание:** Credentials в URL — антипаттерн безопасности

### TC-TRIG-EXT-05: Task status update во время webhook delivery
**Предусловие:** Engine запущен, webhook в процессе отправки
**Шаги:**
1. Trigger fires → webhook sending
2. Параллельно обновить task status

**Ожидание:**
- Нет race condition
- Task status обновлён корректно
- Webhook получает актуальные данные

**Примечание:** Concurrent операции с одним task

### TC-TRIG-EXT-06: Несколько webhooks на одно событие
**Предусловие:** Engine запущен, два trigger с разными webhook URLs на одно событие
**Шаги:**
1. Fire событие, на которое подписаны два trigger

**Ожидание:**
- Оба webhook вызваны
- Порядок не гарантирован, но оба доставлены
- Ошибка в одном не блокирует другой

**Примечание:** Fan-out delivery

### TC-TRIG-EXT-07: Webhook delivery timeout
**Предусловие:** Engine запущен, webhook endpoint очень медленный
**Шаги:**
1. Webhook endpoint отвечает через 60+ секунд
2. Trigger fires

**Ожидание:**
- Timeout после configurable period (default 30s)
- Webhook request aborted
- Событие залогировано
- Engine НЕ hang

**Примечание:** Timeout предотвращает resource exhaustion

---

## TC-RLIM-EXT: Rate Limiting Edge Cases (6 TC)

### TC-RLIM-EXT-01: Missing tier header → default_tier
**Предусловие:** Engine запущен, rate limiting включён
**Шаги:**
1. Отправить запрос без tier header (X-Tier или аналогичного)

**Ожидание:**
- Применяется default_tier rate limit
- Запрос обработан (если в пределах лимита)

**Примечание:** Missing header — не ошибка, а fallback

### TC-RLIM-EXT-02: Per-IP rate limit → 429
**Предусловие:** Engine запущен, rate limiting по IP включён
**Шаги:**
1. Отправить requests с одного IP до превышения лимита

**Ожидание:**
- HTTP 429 Too Many Requests
- Retry-After header присутствует
- Другие IP не затронуты

**Примечание:** Изоляция rate limits по IP

### TC-RLIM-EXT-03: Sliding window accuracy
**Предусловие:** Engine запущен, rate limit = 10 req/min (sliding window)
**Шаги:**
1. Отправить 10 запросов в первые 30 секунд
2. Подождать 31 секунду
3. Отправить ещё 5 запросов

**Ожидание:**
- Первые 10 — accepted
- Запрос #11 (через 31 сек) — accepted (часть окна сдвинулась)
- Не fixed window, а sliding

**Примечание:** Sliding vs fixed window — разное поведение на границах

### TC-RLIM-EXT-04: Concurrent request counting
**Предусловие:** Engine запущен, rate limit включён
**Шаги:**
1. Отправить 100 запросов параллельно

**Ожидание:**
- Счётчик точный (±1)
- Нет race condition в counting
- Не больше limit запросов обработано

**Примечание:** Atomic counting под нагрузкой

### TC-RLIM-EXT-05: Custom header bucketing
**Предусловие:** Engine запущен, rate limit по custom header (e.g., X-Tenant-ID)
**Шаги:**
1. Отправить запросы с X-Tenant-ID: "tenant-a" → до лимита
2. Отправить запросы с X-Tenant-ID: "tenant-b"

**Ожидание:**
- tenant-a получает 429
- tenant-b обрабатывается нормально (свой bucket)

**Примечание:** Изоляция лимитов по tenant

### TC-RLIM-EXT-06: Prometheus metric accuracy
**Предусловие:** Engine запущен (EE), rate limiting включён
**Шаги:**
1. Отправить N запросов (часть accepted, часть rejected)
2. GET /metrics

**Ожидание:**
- bytebrew_rate_limit_total{result="accepted"} = правильное число
- bytebrew_rate_limit_total{result="rejected"} = правильное число
- Сумма = N

**Примечание:** Метрики для мониторинга rate limiting

---

## TC-COMPAT: Backwards Compatibility (5 TC)

### TC-COMPAT-01: Старый array формат конфигурации
**Предусловие:** Engine с конфигом в старом формате (agents как массив, не map)
**Шаги:**
1. Запустить Engine со старым YAML конфигом

**Ожидание:**
- Engine стартует без ошибок
- Agents создаются из старого формата
- Warning в логах: "deprecated config format, consider migration"

**Примечание:** Обратная совместимость с v1 конфигами

### TC-COMPAT-02: Старый формат JWT токена
**Предусловие:** Engine запущен, JWT сгенерирован старой версией
**Шаги:**
1. Использовать JWT из предыдущей версии Engine

**Ожидание:**
- JWT принят и валидирован
- Все claims доступны
- Новые claims отсутствуют — не ошибка

**Примечание:** JWT backward compatibility при обновлении

### TC-COMPAT-03: Старые SSE events
**Предусловие:** Engine запущен, клиент ожидает старые SSE event names
**Шаги:**
1. Отправить chat message
2. Наблюдать SSE stream

**Ожидание:**
- Старые event types (answer, tool_call) присутствуют
- Новые типы (если добавлены) не ломают парсинг старого клиента
- Engine НЕ crash

**Примечание:** SSE event backward compatibility

### TC-COMPAT-04: Оба поля system/system_prompt
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с `system: "text"` (старое поле)
2. POST /api/v1/agents с `system_prompt: "text"` (новое поле)

**Ожидание:**
- Оба варианта приняты
- Если оба указаны — system_prompt приоритетнее

**Примечание:** Alias полей для backward compatibility

### TC-COMPAT-05: DB migration → data integrity
**Предусловие:** БД от предыдущей версии Engine
**Шаги:**
1. Запустить новую версию Engine с БД от старой версии

**Ожидание:**
- Миграции применяются автоматически
- Существующие данные не потеряны
- Новые поля заполнены default значениями

**Примечание:** GORM auto-migration + custom migrations

---

## TC-SESS-EXT: Session Extra (3 TC)

### TC-SESS-EXT-01: Сессия с 1000+ сообщениями
**Предусловие:** Engine запущен, сессия накопила 1000+ messages
**Шаги:**
1. Отправить ещё одно сообщение в сессию с 1000+ messages

**Ожидание:**
- Ответ за разумное время (< 30 секунд)
- Context window management (truncation/summarization)
- НЕ OOM

**Примечание:** Performance при большой истории

### TC-SESS-EXT-02: Null bytes в SSE данных
**Предусловие:** Engine запущен
**Шаги:**
1. LLM ответ содержит null bytes (\x00)

**Ожидание:**
- Null bytes escaped или stripped в SSE stream
- SSE парсинг на клиенте не ломается
- Данные доставлены

**Примечание:** SSE spec не допускает null bytes

### TC-SESS-EXT-03: Event flood — множество событий за короткое время
**Предусловие:** Engine запущен, SSE stream активен
**Шаги:**
1. Agent генерирует много tool_call events подряд (10+ за секунду)

**Ожидание:**
- Все events доставлены клиенту
- Порядок сохранён
- SSE buffer не переполняется

**Примечание:** Burst traffic в SSE stream

---

## TC-SET: Settings (5 TC)

### TC-SET-01: Invalid JSON в settings
**Предусловие:** Engine запущен
**Шаги:**
1. PUT /api/v1/settings с невалидным JSON body

**Ожидание:**
- HTTP 400 Bad Request
- Error: invalid JSON
- Settings не изменены

**Примечание:** JSON validation на входе

### TC-SET-02: Settings с circular references
**Предусловие:** Engine запущен
**Шаги:**
1. PUT /api/v1/settings с JSON содержащим circular references (если возможно)

**Ожидание:**
- HTTP 400 Bad Request ИЛИ JSON парсер отклоняет
- Engine НЕ hang (infinite loop)

**Примечание:** JSON стандартно не поддерживает circular refs, но edge case

### TC-SET-03: Settings export/import round-trip
**Предусловие:** Engine запущен, settings заполнены
**Шаги:**
1. GET /api/v1/settings → сохранить response
2. Изменить одно значение
3. PUT /api/v1/settings с изменённым JSON
4. GET /api/v1/settings → сравнить

**Ожидание:**
- Round-trip без потери данных
- Изменённое значение обновлено
- Остальные значения не затронуты

**Примечание:** Идемпотентность настроек

### TC-SET-04: Secrets не видны в GET response
**Предусловие:** Engine запущен, settings содержат секреты (api keys, passwords)
**Шаги:**
1. GET /api/v1/settings

**Ожидание:**
- Секретные поля замаскированы ("****" или отсутствуют)
- API keys, passwords НЕ возвращаются в открытом виде

**Примечание:** Security: secrets exposure prevention

### TC-SET-05: Per-agent vs global settings precedence
**Предусловие:** Engine запущен, global setting и per-agent override существуют
**Шаги:**
1. Установить global setting X = "global"
2. Установить per-agent setting X = "agent-specific"
3. Chat с agent

**Ожидание:**
- Per-agent setting имеет приоритет
- Другие agents используют global value

**Примечание:** Settings hierarchy: agent > global > default

---

## TC-PERF: Performance (5 TC)

### TC-PERF-01: List 1000+ agents
**Предусловие:** Engine запущен, 1000+ agents в БД
**Шаги:**
1. GET /api/v1/agents?limit=100

**Ожидание:**
- Ответ < 2 секунды
- Pagination работает
- Нет N+1 query проблемы

**Примечание:** Performance при большом количестве agents

### TC-PERF-02: Chat с 1000+ сообщениями в сессии
**Предусловие:** Engine запущен, сессия с 1000+ messages
**Шаги:**
1. Отправить новое сообщение

**Ожидание:**
- First token < 5 секунд
- Context window management (не отправляет все 1000 в LLM)
- Memory usage стабилен

**Примечание:** Context truncation/summarization обязателен

### TC-PERF-03: Knowledge base с 10k+ файлами
**Предусловие:** Engine запущен, 10000+ файлов в knowledge directory
**Шаги:**
1. Выполнить RAG search

**Ожидание:**
- Результат < 1 секунда
- Relevance не деградирует с ростом базы
- Memory usage в пределах нормы

**Примечание:** Vector search performance

### TC-PERF-04: 100+ models в системе
**Предусловие:** Engine запущен, 100+ models настроено
**Шаги:**
1. GET /api/v1/models
2. Изменить model через PUT

**Ожидание:**
- List < 1 секунда
- Config reload < 1 секунда
- Нет memory spike при загрузке конфигурации

**Примечание:** Scalability по количеству models

### TC-PERF-05: 100+ concurrent SSE streams
**Предусловие:** Engine запущен
**Шаги:**
1. Открыть 100 параллельных SSE connections (разные chat sessions)

**Ожидание:**
- Все connections стабильны
- Нет OOM
- Goroutine count пропорционален количеству connections
- Health check отвечает

**Примечание:** SSE scalability — каждый stream = goroutine

---

## TC-OBS: Observability (4 TC)

### TC-OBS-01: Prometheus /metrics под нагрузкой
**Предусловие:** Engine запущен (EE), /metrics endpoint включён
**Шаги:**
1. Создать нагрузку (50+ requests)
2. GET /metrics

**Ожидание:**
- Ответ в valid Prometheus text format
- Метрики обновлены (counters > 0)
- Endpoint отвечает < 1 секунда

**Примечание:** Metrics endpoint не должен быть тяжёлым

### TC-OBS-02: Audit logs не содержат PII/passwords
**Предусловие:** Engine запущен, audit logging включён
**Шаги:**
1. Создать model с api_key
2. Логин пользователя с паролем
3. Проверить audit logs

**Ожидание:**
- api_key замаскирован в логах
- Пароль НЕ логируется
- Email может присутствовать (не считается secret)

**Примечание:** GDPR/security compliance

### TC-OBS-03: Structured JSON logging
**Предусловие:** Engine запущен с JSON log format
**Шаги:**
1. Выполнить несколько операций
2. Проверить stdout/log file

**Ожидание:**
- Каждая строка — valid JSON
- Обязательные поля: timestamp, level, msg
- slog format с context

**Примечание:** Parseable logs для log aggregation (ELK, Loki)

### TC-OBS-04: Request tracing IDs propagated
**Предусловие:** Engine запущен
**Шаги:**
1. Отправить запрос с X-Request-ID header
2. Проверить logs и response headers

**Ожидание:**
- X-Request-ID из request присутствует в логах
- X-Request-ID возвращается в response headers
- Если не передан — генерируется автоматически

**Примечание:** Distributed tracing support

---

## TC-SEC: Security (5 TC)

### TC-SEC-01: XSS в имени agent
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с `name: "<script>alert('xss')</script>"`

**Ожидание:**
- Имя отклонено (400) ИЛИ escaped в response
- В admin UI — нет XSS execution
- HTML entities escaped

**Примечание:** XSS prevention

### TC-SEC-02: SQL injection в model name
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/models с `name: "'; DROP TABLE models;--"`

**Ожидание:**
- SQL injection НЕ выполнен
- Ошибка валидации ИЛИ имя сохранено как строка
- Таблица models цела

**Примечание:** GORM parameterized queries защищают

### TC-SEC-03: API key не в error messages
**Предусловие:** Engine запущен, model с невалидным api_key
**Шаги:**
1. Отправить chat message → ошибка auth

**Ожидание:**
- Error message НЕ содержит api_key
- api_key замаскирован или опущен
- Только "authentication failed" без ключа

**Примечание:** Secrets не должны утекать в responses

### TC-SEC-04: CORS headers correct
**Предусловие:** Engine запущен
**Шаги:**
1. OPTIONS /api/v1/agents (preflight)
2. Проверить CORS headers

**Ожидание:**
- Access-Control-Allow-Origin: настроенные origins (не *)
- Access-Control-Allow-Methods: GET, POST, PUT, DELETE
- Access-Control-Allow-Headers включает Authorization

**Примечание:** CORS misconfiguration — частая проблема

### TC-SEC-05: Rate limiting не обходится сменой IP
**Предусловие:** Engine запущен, rate limiting по API key (не только IP)
**Шаги:**
1. Отправить запросы с одним API key, разных IP → до лимита
2. Отправить ещё запрос с другого IP, тем же API key

**Ожидание:**
- Rate limit срабатывает по API key, не по IP
- Смена IP не сбрасывает счётчик

**Примечание:** Rate limit bucketing по правильному идентификатору

---

## TC-CONC-EXT: Concurrency Extra (4 TC)

### TC-CONC-EXT-01: Config reload во время chat
**Предусловие:** Engine запущен, chat session активна
**Шаги:**
1. Начать chat → SSE stream
2. Параллельно — обновить конфигурацию (PUT settings / reload)

**Ожидание:**
- Текущий stream продолжает работать
- Новые requests используют обновлённый конфиг
- НЕ crash, НЕ прерывание stream

**Примечание:** Hot reload без downtime

### TC-CONC-EXT-02: Multiple SSE streams на одну сессию
**Предусловие:** Engine запущен
**Шаги:**
1. Открыть два SSE connection с одним session_id

**Ожидание:**
- Оба получают events ИЛИ второй отклонён
- Нет дублирования messages
- Нет data corruption

**Примечание:** Определённое поведение для multi-subscriber

### TC-CONC-EXT-03: DB transaction consistency after crash
**Предусловие:** Engine запущен
**Шаги:**
1. Начать операцию, записывающую в несколько таблиц
2. Kill Engine в середине операции
3. Перезапустить Engine

**Ожидание:**
- Данные в consistent state (транзакция rollback)
- Нет partial writes
- Engine стартует нормально

**Примечание:** ACID compliance

### TC-CONC-EXT-04: Rate limit concurrent counting accuracy
**Предусловие:** Engine запущен, rate limit = 100 req/min
**Шаги:**
1. Отправить 200 параллельных запросов

**Ожидание:**
- Ровно ~100 accepted (±5%)
- ~100 rejected (429)
- Atomic counter, нет over-admission

**Примечание:** Точность counting под high concurrency

---

## TC-EXTRA: Miscellaneous (3 TC)

### TC-EXTRA-01: Empty request body на POST endpoints
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с пустым body (Content-Length: 0)

**Ожидание:**
- HTTP 400 Bad Request
- Error: request body required
- НЕ panic

**Примечание:** Защита от пустых запросов

### TC-EXTRA-02: Wrong Content-Type header
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с `Content-Type: text/plain` и JSON body

**Ожидание:**
- HTTP 415 Unsupported Media Type ИЛИ HTTP 400
- Error: expected application/json

**Примечание:** Content-Type validation

### TC-EXTRA-03: Very large request body > 10MB
**Предусловие:** Engine запущен
**Шаги:**
1. POST /api/v1/agents с body > 10MB

**Ожидание:**
- HTTP 413 Payload Too Large ИЛИ connection closed
- Engine НЕ OOM
- Request rejected before full read (limit on body size)

**Примечание:** Body size limit для защиты от DoS

---

## TC-SSE-STREAM: SSE Streaming Reliability (5 TC)

### TC-SSE-STREAM-01: Response headers — no Content-Length
**Предусловие:** Engine запущен, агент и модель настроены

**Шаги:**
1. `curl -sI -X POST http://localhost:8443/api/v1/agents/{name}/chat -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"message":"hi"}'`
2. Проверить response headers

**Ожидание:**
- Content-Type: text/event-stream
- Cache-Control: no-cache
- НЕТ Content-Length header
- Transfer-Encoding: chunked

**PASS:** SSE response headers корректны, нет Content-Length

### TC-SSE-STREAM-02: SSE events delivered — message_delta, message, done
**Предусловие:** Engine запущен, агент настроен с LLM моделью

**Шаги:**
1. `curl -N -X POST http://localhost:8443/api/v1/agents/{name}/chat -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"message":"Say hello"}'`
2. Наблюдать за SSE stream

**Ожидание:**
- Начинается с `: ok` (SSE comment)
- Приходят event: message_delta с content
- Приходит event: message с полным ответом
- Завершается event: done с session_id
- Ответ содержит осмысленный текст (не пустой)

**PASS:** SSE events (message_delta, message, done) доставляются, content не пустой

### TC-SSE-STREAM-03: Long-running chat — stream не обрывается (30+ секунд)
**Предусловие:** Engine запущен, агент с MCP tools

**Шаги:**
1. Отправить сложный запрос требующий tool calls (web_search, knowledge_search)
2. Наблюдать за SSE stream 30+ секунд

**Ожидание:**
- Stream не обрывается по timeout
- tool_call и tool_result events доставляются
- message_delta events продолжают приходить после tool calls
- done event в конце

**PASS:** Long-running SSE stream работает без обрыва

### TC-SSE-STREAM-04: Non-streaming mode (stream: false) — JSON response
**Предусловие:** Engine запущен, агент настроен

**Шаги:**
1. `curl -s -X POST http://localhost:8443/api/v1/agents/{name}/chat -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"message":"Say hello","stream":false}'`

**Ожидание:**
- Content-Type: application/json (не text/event-stream)
- JSON body содержит: agent, message, session_id
- message содержит ответ LLM
- tool_calls array если были вызовы инструментов

**PASS:** Non-streaming response возвращает JSON с message и session_id

### TC-SSE-STREAM-05: Client disconnect — graceful cleanup
**Предусловие:** Engine запущен, агент настроен

**Шаги:**
1. Начать SSE chat: `curl -N -X POST ... -d '{"message":"Write a long essay"}'`
2. Прервать curl (Ctrl+C) через 2 секунды
3. Проверить логи Engine

**Ожидание:**
- Engine НЕ crash, не panic
- Логи: "turn cancelled by user" или context cancelled
- Новый chat request работает после disconnect
- Нет утечки goroutines (проверить /debug/pprof/goroutine если доступен)

**PASS:** Client disconnect обрабатывается gracefully, нет crash/panic

---

## TC-MCP-CRUD: MCP Server API CRUD (5 TC)

### TC-MCP-CRUD-01: Create MCP server with forward_headers
**Предусловие:** Engine запущен, admin JWT получен

**Шаги:**
1. `curl -s -X POST http://localhost:8443/api/v1/mcp-servers -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"test-mcp","type":"sse","url":"http://localhost:9090/sse","forward_headers":["X-Org-Id","X-User-Id"]}'`

**Ожидание:**
- HTTP 201
- Response JSON содержит `forward_headers: ["X-Org-Id", "X-User-Id"]`
- Имя, тип, URL сохранены корректно

**PASS:** MCP server создан с forward_headers в response

### TC-MCP-CRUD-02: Update MCP server forward_headers
**Предусловие:** MCP server "test-mcp" создан (TC-MCP-CRUD-01)

**Шаги:**
1. `curl -s -X PUT http://localhost:8443/api/v1/mcp-servers/test-mcp -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"name":"test-mcp","type":"sse","url":"http://localhost:9090/sse","forward_headers":["Authorization","X-Org-Id","X-User-Id"]}'`

**Ожидание:**
- HTTP 200
- Response JSON содержит `forward_headers: ["Authorization", "X-Org-Id", "X-User-Id"]`
- Три header'а вместо двух

**PASS:** MCP server обновлён с новым forward_headers в response

### TC-MCP-CRUD-03: List MCP servers includes forward_headers
**Предусловие:** MCP server с forward_headers существует

**Шаги:**
1. `curl -s http://localhost:8443/api/v1/mcp-servers -H "Authorization: Bearer $TOKEN"`

**Ожидание:**
- HTTP 200
- Каждый MCP server в массиве содержит поле `forward_headers` (если настроен)
- Значение соответствует сохранённому

**PASS:** GET /mcp-servers включает forward_headers для каждого сервера

### TC-MCP-CRUD-04: Config export includes forward_headers
**Предусловие:** MCP server с forward_headers существует

**Шаги:**
1. `curl -s http://localhost:8443/api/v1/config/export -H "Authorization: Bearer $TOKEN"`
2. Проверить секцию mcp_servers в YAML

**Ожидание:**
- YAML содержит `forward_headers:` для MCP сервера
- Значения — список header names

**PASS:** Config export содержит forward_headers для MCP серверов

### TC-MCP-CRUD-05: Config import with forward_headers
**Предусловие:** Engine запущен

**Шаги:**
1. Подготовить YAML:
```yaml
mcp_servers:
  - name: imported-mcp
    type: sse
    url: http://localhost:9090/sse
    forward_headers:
      - X-Tenant-Id
      - Authorization
```
2. `curl -s -X POST http://localhost:8443/api/v1/config/import -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/x-yaml" --data-binary @mcp.yaml`
3. Проверить: `GET /api/v1/mcp-servers`

**Ожидание:**
- Import success
- MCP server "imported-mcp" создан с forward_headers

**PASS:** Config import персистирует forward_headers

---

## TC-BYOK: Bring Your Own Key (5 TC)

### TC-BYOK-01: Chat with BYOK headers — provider override
**Предусловие:** Engine запущен, агент настроен, BYOK enabled в Settings

**Шаги:**
1. `curl -N -X POST http://localhost:8443/api/v1/agents/{name}/chat -H "Authorization: Bearer $TOKEN" -H "X-Model-Provider: openai" -H "X-Model-API-Key: sk-test-..." -H "X-Model-Name: gpt-4o" -H "Content-Type: application/json" -d '{"message":"hi"}'`

**Ожидание:**
- Request использует предоставленный API key и модель
- SSE response содержит ответ от указанной модели
- API key не сохраняется в DB

**PASS:** BYOK override работает, ключ не персистируется

### TC-BYOK-02: BYOK disabled — headers ignored
**Предусловие:** BYOK disabled в Settings

**Шаги:**
1. Отправить chat с X-Model-Provider, X-Model-API-Key headers

**Ожидание:**
- BYOK headers игнорируются
- Используется default модель агента
- Нет ошибки

**PASS:** При disabled BYOK headers игнорируются

### TC-BYOK-03: BYOK with invalid API key
**Предусловие:** BYOK enabled

**Шаги:**
1. Отправить chat с `X-Model-API-Key: invalid-key-xxx`

**Ожидание:**
- SSE error event с сообщением об ошибке авторизации от провайдера
- НЕ 500 Internal Server Error
- Осмысленное сообщение: "authentication failed" или "invalid API key"

**PASS:** Invalid BYOK key возвращает error event, не crash

### TC-BYOK-04: BYOK missing provider header
**Предусловие:** BYOK enabled

**Шаги:**
1. Отправить chat с `X-Model-API-Key` но БЕЗ `X-Model-Provider`

**Ожидание:**
- Используется default провайдер агента с пользовательским ключом
- ИЛИ ошибка "X-Model-Provider required when using BYOK"

**PASS:** Missing provider handled gracefully

### TC-BYOK-05: BYOK key not persisted
**Предусловие:** BYOK chat выполнен

**Шаги:**
1. GET /api/v1/models → проверить что BYOK ключ не появился

**Ожидание:**
- Список моделей не содержит BYOK ключ
- Session audit не содержит ключ в plaintext

**PASS:** BYOK ключ не сохранён нигде

---

## Итого: 410 TC

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
| TC-KILO | 8 | Integration issues (config import, panic, MCP, Docker) |
| TC-CFG | 11 | Config import/export edge cases (idempotency, UTF-8, malformed YAML) |
| TC-TOOL | 8 | Tool calling & MCP edge cases (timeout, circular spawn, audit) |
| TC-CRASH | 9 | Crash prevention & validation (empty name, special chars, FK protection) |
| TC-AG-EXT | 4 | Agent CRUD edge cases (empty prompt, empty tools, model change, max name) |
| TC-MD-EXT | 4 | Model CRUD edge cases (empty name, invalid URL, delete active, key rotation) |
| TC-MC-EXT | 5 | MCP server edge cases (invalid URL, unreachable, invalid type, timeout) |
| TC-AUTH-EXT | 9 | Auth edge cases (tampered JWT, expired, missing Bearer, concurrent, API key) |
| TC-SESS | 7 | Session edge cases (expired, invalid UUID, SQL injection, concurrent, disconnect) |
| TC-DOCKER | 7 | Docker deployment (missing env, permissions, health check, graceful shutdown) |
| TC-CONC | 2 | Concurrency (same-name race, delete during chat) |
| TC-PROV-EXT | 11 | Provider edge cases (invalid URL, missing key, Azure, Gemini, OpenRouter) |
| TC-KNOW | 7 | Knowledge/RAG edge cases (no path, nonexistent, unsupported format, large file) |
| TC-TRIG-EXT | 7 | Trigger/webhook edge cases (retry backoff, headers, timeout, fan-out) |
| TC-RLIM-EXT | 6 | Rate limiting edge cases (default tier, sliding window, concurrent counting) |
| TC-COMPAT | 5 | Backwards compatibility (old config, old JWT, old SSE, DB migration) |
| TC-SESS-EXT | 3 | Session extra (1000+ messages, null bytes, event flood) |
| TC-SET | 5 | Settings (invalid JSON, circular refs, export/import, secrets, precedence) |
| TC-PERF | 5 | Performance (1000+ agents, 1000+ messages, 10k+ knowledge, 100+ SSE) |
| TC-OBS | 4 | Observability (Prometheus, audit PII, structured logs, tracing IDs) |
| TC-SEC | 5 | Security (XSS, SQL injection, API key masking, CORS, rate limit bypass) |
| TC-CONC-EXT | 4 | Concurrency extra (config reload, multi-stream, crash consistency) |
| TC-EXTRA | 3 | Miscellaneous (empty body, wrong Content-Type, large body) |
| TC-SSE-STREAM | 5 | SSE streaming reliability (headers, events, long-running, non-stream, disconnect) |
| TC-MCP-CRUD | 5 | MCP server API CRUD (forward_headers persistence, config import/export) |
| TC-BYOK | 5 | Bring Your Own Key (provider override, disabled, invalid key, not persisted) |
| **ВСЕГО** | **410** |

### Примечания
- Multi-agent spawn работает через HTTP REST API и gRPC/WS
- TC-MCP-04/05 — quality tests: проверяют не только работоспособность но и качество RAG ответов
- TC-EE-* — требуют Stripe test mode и EE license для полного прогона
- TC-PROV-01..09 (Azure) — требуют Azure OpenAI credentials
- TC-PROV-10..19 (Gemini) — требуют Google API key
- TC-HELM-* — требуют Helm CLI, можно проверить без K8s кластера (helm template/lint)
- TC-METR-* — EE feature, требуют EE лицензию
- TC-KILO-* — integration issues от Kilo IoT, TC-KILO-06/07 требуют Docker
- TC-CFG-* — config edge cases, TC-CFG-10 требует Docker или custom config dir
- TC-TOOL-* — TC-TOOL-07 требует EE лицензию (audit log), TC-TOOL-01/03 требуют controllable MCP server
- TC-CRASH-* — validation и crash prevention, все можно проверить без внешних зависимостей
- TC-AG-EXT/MD-EXT/MC-EXT — CRUD edge cases, все можно проверить без внешних зависимостей
- TC-AUTH-EXT-* — auth edge cases, TC-AUTH-EXT-06 требует scope-based API keys (EE)
- TC-SESS-* — session edge cases, TC-SESS-06 требует мониторинг goroutines
- TC-DOCKER-* — требуют Docker, TC-DOCKER-04 требует load testing tool
- TC-PROV-EXT-* — TC-PROV-EXT-05..07 требуют Azure credentials, TC-PROV-EXT-08..11 требуют Gemini key
- TC-KNOW-* — TC-KNOW-07 требует файл > 100MB для performance теста
- TC-RLIM-EXT-* — EE feature, требуют EE лицензию
- TC-PERF-* — требуют seed data (1000+ agents/messages) и load testing tools
- TC-OBS-* — EE feature (Prometheus), structured logging — CE
- TC-SEC-* — security tests, все можно проверить без внешних зависимостей
- TC-CONC-EXT-03 — требует kill -9 Engine в середине транзакции
