# ByteBrew — Use Cases и примеры

**Дата:** 14 марта 2026

---

## Тезис

AI — не отдельная сфера. Это новый способ решать задачи. Любой продукт решает задачи. С AI — решает лучше, быстрее, дешевле. Кто не добавит AI capabilities — отстанет.

Проблема: добавить AI в продукт сегодня = нанять AI-команду или строить на SDK месяцами. ByteBrew = готовый engine, который даёт продукту "мозги" через конфигурацию.

---

## Пример 1: IoT платформа (ThingsBoard)

### Продукт

ThingsBoard — open-source IoT платформа. Device management, телеметрия, dashboards, Rule Engine. 16K+ GitHub stars.

### Текущие AI-фичи (3/10)

| Фича | Что делает | Ограничение |
|------|-----------|-------------|
| AI Request Node | Одиночный запрос к LLM в Rule Engine | Не agent — один request-response. Нет reasoning, нет tool use, нет памяти |
| Trendz AI Assistant | Чат-виджет для аналитики | Платный. Reactive. Не автономный |
| MCP Server | 120+ tools для внешнего AI-клиента | Требует Claude Desktop / Cursor. Не встроен. Нет agent loop |
| Predictive Analytics | ML для временных рядов | Классический ML, не генеративный AI |

**Суть:** AI = "curl к OpenAI обёрнутый в UI". Нет reasoning loop, нет автономности, нет multi-step.

### Почему ThingsBoard не делает полноценного AI-агента сам

1. **Архитектура.** Rule Engine = data pipeline (DAG: если X то Y). Agent = цикл (observe → think → act). Разные модели исполнения. Встроить agent loop = переписать ядро
2. **Не их фокус.** Компетенция: device management, telemetry, dashboards. AI agent engine — другой продукт
3. **Ресурсы.** Строить agent engine с нуля = 4-6 месяцев + AI-команда

### Что даёт ByteBrew

```yaml
# bytebrew.yaml — IoT agent для ThingsBoard

agents:
  - name: "iot-operator"
    system_prompt: |
      Ты — AI-оператор IoT платформы.
      Мониторишь устройства, анализируешь аномалии,
      предлагаешь правила для Rules Engine.

    tools:
      mcp_servers:
        thingsboard:
          type: "http"
          url: "http://thingsboard:8080/mcp"  # 120+ tools уже есть
      builtin: [ask_user, manage_tasks]

    knowledge: "./docs/devices/"

    triggers:
      - type: "cron"
        schedule: "*/30 * * * *"
        job:
          title: "Проверка телеметрии"
          description: "Проверь телеметрию за последние 30 минут, найди аномалии"
          agent: "iot-operator"
      - type: "webhook"
        path: "/api/alarm"
        job:
          title: "Alarm"
          agent: "iot-operator"

    can_spawn: ["report-generator"]

  - name: "report-generator"
    lifecycle: "spawn"
    system_prompt: "Сформируй структурированный отчёт по данным."
    tools:
      mcp_servers:
        thingsboard:
          type: "http"
          url: "http://thingsboard:8080/mcp"
```

### До и после

| Сценарий | Без ByteBrew | С ByteBrew |
|----------|-------------|------------|
| Аномалия | Alarm → оператор открывает дашборд → ищет вручную | Alarm → **агент анализирует корреляции** → гипотеза → предлагает действие |
| 50 алармов | Оператор сам ищет root cause | Агент **группирует**, находит каскад, показывает root cause |
| Правила Rule Engine | Статичные (написал → забыл) | Агент **наблюдает** за эффективностью → предлагает новые правила |
| "Что с цехом 3?" | Нужно знать UI, строить запрос | **Естественный язык** → агент сам строит запрос через MCP |
| Деградация оборудования | Prediction → alarm → ??? | Prediction → **проверка запчастей** → **планирование** → **задача в CMMS** |
| Ежеутренний отчёт | Оператор собирает вручную | **Cron каждые 8:00** → агент собирает → отправляет |

### Для ThingsBoard: интеграция за часы, не месяцы

ThingsBoard уже имеет MCP Server (120+ tools). ByteBrew подключается к нему. Всё что нужно — написать YAML конфиг. Не нужно менять код ThingsBoard, не нужна AI-команда.

---

## Пример 2: Интернет-магазин

### Продукт

E-commerce платформа. Каталог товаров, корзина, оплата, доставка. API для всего.

### Текущие AI-фичи (типично)

- Рекомендации товаров (ML, collaborative filtering) — стандарт
- Чат-бот для FAQ (scripted, не AI) — слабый
- Поиск (keyword, может semantic) — базовый

### Чего не хватает

Никто не помогает покупателю **выбрать**. Покупатель приходит: "Мне нужен ноутбук для видеомонтажа, бюджет $1500". Сейчас: сам ищет, сам сравнивает, сам решает. 70% бросают корзину.

### Что даёт ByteBrew

```yaml
agents:
  - name: "sales-consultant"
    system_prompt: |
      Ты — консультант магазина. Помоги покупателю найти идеальный товар.
      Выясни потребности, предложи 2-3 варианта с объяснением, помоги оформить.

    tools:
      custom:
        - name: "search_products"
          endpoint: "GET https://api.shop.com/products"
          params: { query: "string", category: "string", max_price: "number" }
        - name: "check_stock"
          endpoint: "GET https://api.shop.com/products/{id}/stock"
        - name: "create_order"
          endpoint: "POST https://api.shop.com/orders"
          confirmation_required: true
      builtin: [ask_user, knowledge_search]

    knowledge: "./docs/sales/"

    rules:
      - "Всегда проверяй наличие перед предложением"
      - "Скидка не больше 15%"

    can_spawn: ["product-researcher"]

  - name: "product-researcher"
    lifecycle: "spawn"
    system_prompt: "Исследуй товар: характеристики, отзывы, сравнение с аналогами."
    tools:
      builtin: [knowledge_search, web_search]
```

### До и после

| Сценарий | Без ByteBrew | С ByteBrew |
|----------|-------------|------------|
| "Ноутбук для видеомонтажа" | Покупатель ищет по фильтрам, 50 результатов | Агент **спрашивает**: бюджет? какой софт? портативность? → **3 варианта с объяснением** |
| Товар out of stock | Покупатель узнаёт при оформлении | Агент **проверяет наличие** до предложения, предлагает альтернативу |
| Сравнение товаров | Покупатель открывает 5 вкладок | Агент **сам сравнивает**, объясняет разницу |
| Оформление заказа | 5 шагов checkout | Агент **оформляет** одной командой (с подтверждением) |
| Повторный визит | Всё с нуля | Агент **помнит** предпочтения (память сессии) |

---

## Пример 3: SaaS продукт (CRM/Project Management/etc.)

### Проблема

Сложный SaaS = высокий churn на онбординге. Пользователь не понимает как настроить → бросает.

### Что даёт ByteBrew

```yaml
agents:
  - name: "onboarding-guide"
    system_prompt: |
      Ты — ассистент продукта. Помоги новому пользователю настроить всё.
      Проведи через ключевые шаги, ответь на вопросы, покажи фичи.

    tools:
      mcp_servers:
        product-api:
          type: "http"
          url: "http://api:3000/mcp"
      builtin: [ask_user, knowledge_search]

    knowledge: "./docs/product/"

    triggers:
      - type: "webhook"
        path: "/api/user-signup"
        job:
          title: "Онбординг нового пользователя"
          agent: "onboarding-guide"
```

### До и после

| | Без | С ByteBrew |
|--|-----|-----------|
| Онбординг | Документация + 5-step wizard | Агент **проводит** через настройку, отвечает на вопросы по ходу |
| "Как сделать X?" | Поиск в docs | Агент **сам делает** X или показывает как |
| Новая фича | Email "мы добавили X" | Агент **предлагает** попробовать X в контексте работы пользователя |
| Застрял | Support ticket → ждать | Агент **помогает** сразу, эскалирует к человеку если не может |

---

## Пример 4: ByteBrew Code (серверный coding daemon)

### Продукт

ByteBrew Code — серверный daemon для AI-разработки. Ставится на сервер компании рядом с git. Engine + Developer Kit (LSP, indexing, git). Разработчики подключаются через CLI, CTO через Mobile.

```
Сервер компании:
  [Engine + Developer Kit + PostgreSQL]
       ├── git clone → worktree per task
       ├── LSP servers → ошибки видны сразу
       ├── Code indexing → семантический поиск
       └── Agent: код → тесты → review → PR

Разработчик: CLI → "Добавь endpoint /api/health" → PR через 5 мин
CTO: Mobile → видит прогресс, одобряет PR
```

### Конфигурация

```yaml
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

triggers:
  - type: "webhook"
    path: "/api/github/issues"
    job:
      title: "GitHub Issue"
      agent: "supervisor"
  - type: "cron"
    schedule: "0 8 * * 1-5"
    job:
      title: "Утренний прогон тестов"
      description: "Запусти все тесты, если упало — создай PR с фиксом"
      agent: "supervisor"
```

### До и после

| | Без ByteBrew Code | С ByteBrew Code |
|--|-------------------|----------------|
| "Добавь endpoint /api/health" | Разработчик пишет 30-60 мин | CLI → агент: git clone → код → тесты → review → PR за 5 мин |
| Code review | Ждать коллегу | Review Agent (свежий контекст) → замечания → fix |
| GitHub Issue assigned | Разработчик читает → думает → пишет | Webhook → агент берёт issue → PR автоматически |
| "Тесты упали после merge" | Разработчик дебажит | **Cron**: агент проверяет → находит проблему → PR с фиксом |
| Новый разработчик: "Где auth?" | Ищет в коде часами | CLI → агент через indexing → "auth в internal/middleware/auth.go, строка 45" |

---

## Пример 5: Медицина / Клиника

### Проблема

Регистратура перегружена. Пациент звонит / пишет — долгое ожидание. Запись на приём = 5-10 минут диалога.

### Что даёт ByteBrew

```yaml
agents:
  - name: "clinic-receptionist"
    system_prompt: |
      Ты — регистратор клиники. Помоги пациенту записаться на приём.
      Собери жалобы, подбери специалиста, найди свободное время.

    tools:
      mcp_servers:
        clinic-api:
          type: "http"
          url: "http://clinic-api:3000/mcp"
          # get_doctors, get_schedule, create_appointment, get_patient

    rules:
      - "Никогда не ставь диагноз"
      - "При экстренных симптомах — немедленно направить в скорую"
      - "Персональные данные не запрашивай кроме ФИО и даты рождения"

    escalation:
      triggers: ["экстренный", "жалоба", "врач"]
      webhook: "https://clinic-api:3000/escalation"
```

---

## Пример 6: DevOps / Monitoring

### Проблема

Дежурный инженер получает 100+ алертов за ночь. 90% — noise. 10% — реальные проблемы. Разбирается вручную.

### Что даёт ByteBrew

```yaml
agents:
  - name: "oncall-assistant"
    system_prompt: |
      Ты — AI-дежурный. Мониторишь алерты, фильтруешь шум,
      анализируешь реальные проблемы, предлагаешь решения.

    tools:
      mcp_servers:
        grafana:
          type: "http"
          url: "http://grafana:3000/mcp"
        pagerduty:
          type: "http"
          url: "http://pagerduty-mcp:3001/mcp"
      builtin: [shell_exec, web_fetch]

    triggers:
      - type: "webhook"
        path: "/api/alerts"  # PagerDuty/Grafana шлёт alert
      - type: "cron"
        schedule: "0 * * * *"
        prompt: "Проверь метрики за последний час, найди тренды"

    escalation:
      triggers: ["P1", "data loss", "security"]
      webhook: "https://pagerduty.com/api/escalation"
```

### До и после

| | Без | С ByteBrew |
|--|-----|-----------|
| 100 алертов за ночь | Дежурный разбирает все | Агент **фильтрует** noise → показывает 10 реальных |
| "Почему latency выросла?" | Дежурный лезет в Grafana, логи, traces | Агент **сам проверяет** метрики → находит root cause |
| Новый деплой → ошибки | Дежурный замечает через 30 мин | **Cron каждый час** → агент замечает тренд → алертит за 5 мин |

---

## Где ByteBrew — killer, а где нет

### Killer use cases (proactive reasoning, нет конкурентов)

| Use case | Почему killer |
|----------|--------------|
| **IoT / Industrial** | Ни одна IoT платформа не имеет autonomous agent с reasoning. ThingsBoard AI = одиночный запрос к LLM. ByteBrew = трансформация продукта |
| **DevOps proactive monitoring** | Daemon мониторит метрики, находит root cause, предлагает fix ДО жалобы клиента |
| **Coding daemon (ByteBrew Code)** | Серверный agent: git clone → LSP → code → tests → review → PR. Webhook от GitHub → автоматическая работа. Как Devin но self-hosted |
| **Complex SaaS automation** | Multi-step workflows с reasoning, не предопределённый pipeline |

### Конкурентоспособные use cases (есть альтернативы, но ByteBrew выигрывает по модели)

| Use case | ByteBrew value | Альтернатива | Когда ByteBrew лучше |
|----------|---------------|-------------|---------------------|
| Sales assistant | Agent интервьюирует, подбирает, оформляет | Intercom ($0.99/resolution) | Self-hosted нужен, или volume большой (per-resolution дорого) |
| Support | Agent решает тикеты | Zendesk AI, Intercom | Нужен universal (support + sales + onboarding в одном engine) |
| Onboarding | Agent проводит через настройку | Dify widget | Нужен autonomous (agent сам инициирует, не ждёт) |

### Где НЕ стоит конкурировать

| Use case | Почему | Что лучше |
|----------|--------|-----------|
| Простой FAQ-бот | Overkill — autonomous engine для FAQ = пушкой по воробьям | Dify, Botpress, Voiceflow |
| Scripted chatbot | Нет reasoning needed | Любой chatbot builder |

---

## Паттерн: что объединяет killer use cases

```
Продукт имеет данные + API
     ↓
Сегодня: данные → dashboard → человек разбирается
     ↓
С ByteBrew: данные → agent reasoning → предложение → человек подтверждает
```

Ключевое: agent **добавляет reasoning layer** между данными и действием. Не заменяет человека — помогает ему принять решение быстрее.

**Время интеграции:** часы-дни (если у продукта есть API / MCP). Не месяцы.

---

## Почему компании сегодня не делают это сами

| Барьер | Данные | Как ByteBrew решает |
|--------|--------|---------------------|
| **Нет AI-экспертизы** | 41% компаний (LangChain) | Не нужна — конфигурация, не программирование |
| **Интеграция сложная** | 46% называют главным барьером | MCP + declarative tools. Подключение к API без кода |
| **Не масштабируется из пилота** | <25% вышли в production (Gartner) | Production-grade engine (Go, not Python prototype) |
| **Дорого** | $150K+ на AI-команду | Бесплатный community, платный enterprise |
| **Не знают что есть** | 75% начинают с free tools | Free tier = distribution. Разработчик находит → пробует → внедряет |
| **Страх автономности** | 60% не доверяют автономным решениям | Human-in-the-loop по умолчанию: `confirmation_required`, `rules`, `escalation`. Агент **предлагает**, человек **подтверждает** |
| **"AI это не наш фокус"** | Компании думают что AI = отдельная сфера | AI = способ решать задачи лучше. Каждый продукт решает задачи. ByteBrew даёт AI capabilities без смены фокуса |
