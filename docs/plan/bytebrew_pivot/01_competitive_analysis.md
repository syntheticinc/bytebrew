# ByteBrew — Конкурентный анализ

**Дата:** 14 марта 2026

---

## Позиционирование

### Headline

> **Добавь AI-агента в свой продукт.**

### По аудиториям

**Для инвестора (one-liner):**
> ByteBrew — engine который даёт любому продукту autonomous AI capabilities. Бесплатный для старта, платный при масштабировании. Ниша пуста — ни один продукт не делает этого.

**Для CTO / product owner / vibe coder:**
> Ваш продукт получает AI-агента который сам мониторит, анализирует и предлагает действия. Без AI-команды, без per-token расходов. Конфигурация — не код.
>
> - IoT платформа → агент анализирует телеметрию, находит аномалии, предлагает оператору действия
> - E-commerce → агент интервьюирует покупателя, подбирает товар, оформляет заказ
> - DevOps → агент мониторит метрики, фильтрует noise, находит root cause
> - SaaS → агент проводит пользователя через онбординг, отвечает по продукту
> - Dev tool → агент пишет код, ревьюит, тестирует, создаёт PR

### Аналогия (для pitch deck, не для headline)

Как MySQL для данных — ставится куда угодно, используется под что угодно. Бесплатный community, платный enterprise.

### Ключевое отличие

| Все остальные | ByteBrew |
|--------------|----------|
| Reactive (спросил → ответил) | **Autonomous** (сам мониторит, анализирует, предлагает) |
| Per-token ($600+/мес при 1K диалогов) | **Свои модели, inference = $0** |
| Код нужен (SDK) или не production (Dify) | **Конфигурация, production-grade** |
| Один агент | **Дерево агентов** (supervisor → sub-agents) |
| Вертикальный (только support / только код) | **Универсальный** (любой домен) |
| Human-out-of-the-loop (страшно) | **Human-in-the-loop** по умолчанию (агент предлагает, человек решает) |

---

## Что такое ByteBrew

Autonomous AI agent engine. Инфраструктура — как MySQL для данных, Nginx для HTTP, Redis для кеша. Ставится куда угодно, используется под что угодно.

**Не** SDK (Claude Agent SDK) — не нужно программировать.
**Не** visual builder (Dify) — production-grade, не прототип.
**Не** вертикальный SaaS (Intercom) — универсальный, любой домен.
**Не** per-token cloud (OpenAI AgentKit) — свои модели, фиксированные затраты.

### Кто использует

| Кто | Для чего | Пример |
|-----|----------|--------|
| **Стартап** | Встраивает AI в свой продукт | SaaS с AI-онбордингом, e-commerce с AI sales-агентом |
| **Enterprise** | Автоматизирует внутренние процессы | IoT мониторинг, compliance, HR-бот |
| **Агентство** | Строит AI-решения для клиентов | White-label agent для ресторанов, клиник |
| **Разработчик** | Экспериментирует, учится, строит | Coding agent, research assistant, personal AI |
| **DevOps / IT** | Проактивный мониторинг и автоматизация | Агент-дежурный: проверяет логи, алертит, предлагает fix |

### Примеры конфигураций

| Конфигурация | Что делает агент |
|-------------|-----------------|
| Sales assistant | Интервьюирует покупателя, подбирает товар, проверяет остатки, оформляет заказ |
| IoT assistant | Находит устройства, анализирует телеметрию, формирует отчёты, собирает rules |
| Support agent | Обрабатывает тикеты, ищет в KB, эскалирует к человеку |
| Coding agent | Пишет код, ревьюит, тестирует, создаёт PR |
| Research agent | Ищет в интернете, анализирует, формирует отчёт |
| Scheduling agent | Собирает анамнез, подбирает врача, записывает на приём |
| Monitoring agent | Каждые 30 мин проверяет метрики, алертит при аномалиях |

### Бизнес-модель: free + enterprise (QuestDB / PostHog / MySQL)

```
Community Edition ($0)             Enterprise Edition (Contact us)
├── Полный engine, без лимитов     ├── Всё из CE +
├── Agents, tools, knowledge       ├── Horizontal scaling (multi-node)
├── API (REST, WS)                 ├── AI Observability Dashboard
├── Dashboard (базовый)            │   (prompt analytics, quality metrics,
├── Cron / webhooks / triggers     │    cost tracking, session explorer)
├── MCP, Kits                      └── Priority support + SLA
└── Single node, self-hosted
```

**Аналоги:**

| Продукт | Free | Paid |
|---------|------|------|
| **QuestDB** | Полный engine, single node | HA, replication, RBAC, compression |
| **Dify** | Self-hosted CE | SSO, RBAC, multi-tenant |
| **n8n** | Self-hosted (fair-code) | Cloud executions + SSO/audit |
| **ByteBrew** | Полный engine, single node | Horizontal scaling + AI observability |

---

## Проблема

Компания хочет AI-агента в своём продукте. Что она делает сегодня:

| Вариант | Что происходит | Проблема |
|---------|---------------|----------|
| **OpenAI AgentKit** | Embed за 1 день, per-token billing | **Lock-in + непредсказуемые расходы.** 1000 диалогов/день = $600+/мес только inference. При масштабе съедает маржу |
| **Claude Agent SDK** | Нанимает команду, пишет код | $150K+ на команду, Claude-only, поддержка своими силами |
| **Dify / n8n** | Visual builder, прототип | Не production: Python 10 QPS, нет multi-agent, нет autonomous daemon |
| **Intercom / Zendesk AI** | Вертикальное решение | Только support. $0.99-2/conversation. Нельзя сделать sales или IoT |
| **CrewAI / LangGraph** | Программирует multi-agent | Framework, нужна Python-команда. Нет hosting, нет UI |
| **Ничего** | "Слишком сложно / дорого" | <25% вышли из пилота в production (Gartner) |

### Почему per-token billing — не опция для серьёзного продукта

```
Sales-бот: 1000 диалогов/день × 2K tokens = $600/мес (GPT-5)
           10K диалогов/день               = $6000/мес

Coding agent: 100 задач/день × 100K tokens = $1000+/день

vs ByteBrew + Llama на своих GPU: фиксированная стоимость железа, inference = $0
```

Ни одна продуктовая компания не построит core AI feature на чужом per-token billing. Это как строить финтех на Stripe с 2.9% навсегда — при масштабе съедает всю маржу.

---

## Почему AgentKit — НЕ конкурент

На первый взгляд AgentKit решает ту же задачу: "embed AI agent в свой продукт". Но архитектурно — это разные продукты.

| | OpenAI AgentKit | ByteBrew |
|--|----------------|----------|
| **Бизнес-модель** | Per-token (OpenAI зарабатывает на inference) | Лицензия (компания использует свои модели) |
| **Модели** | OpenAI only (de facto) | Любые (Llama, Qwen, Claude, GPT) |
| **Agent mode** | **Reactive** (user пишет → агент отвечает) | **Autonomous** (daemon, cron, webhooks, proactive) |
| **Triggers** | Только chat | Chat + cron + webhooks + events |
| **Multi-agent** | Handoffs (линейный) | Дерево (Supervisor → sub-agents → spawn) |
| **Kits** | Нет | Engine-level extensions (LSP, recommender, anomaly detector) |
| **Self-hosted** | Нет (inference = OpenAI API) | Полный (свои GPU, air-gap) |
| **Data residency** | OpenAI Cloud | Полный контроль |
| **Стоимость при масштабе** | Растёт линейно (per-token) | Фиксированная (свои GPU) |

**Структурный конфликт интересов:** OpenAI зарабатывает на per-token inference. Self-hosted с Llama на чужих GPU = ноль выручки для OpenAI. Они **не будут** каннибализировать свою модель добавлением full self-hosted.

Аналогия: AWS никогда не сделает лучший self-hosted PostgreSQL — им нужно чтобы ты платил за RDS.

### Reactive vs Autonomous

```
AgentKit:   User → Chat → Agent → Response → ждёт следующего сообщения

ByteBrew:   [Agent daemon] → мониторит → решает → действует → отчитывается
                 ↑ cron, webhooks, events, user chat — всё запускает агента
```

Пример (IoT платформа):
- **AgentKit:** оператор открыл чат → спросил "какие аномалии?" → агент ответил
- **ByteBrew:** агент **сам** каждые 30 минут проверяет телеметрию → нашёл аномалию → создал алерт → уведомил оператора → предложил rule

---

## Конкуренты (глубокий анализ)

### Реальная картина

```
Прямых конкурентов в нише "embeddable autonomous agent engine": 0

Частичные пересечения:
├── Dify (130K stars, $30M) — reactive workflow builder, не daemon
├── Letta (38K stars, $10M) — stateful agents, не autonomous, Python
├── OpenAI AgentKit — embeddable, но per-token + cloud-only + reactive
└── CopilotKit (28K stars) — embeddable, но UI copilot, не autonomous

Другие рынки (не конкуренты):
├── NemoClaw (NVIDIA) — enterprise internal platform
├── n8n ($2.5B) — workflow automation
├── OpenClaw — personal agent (B2C)
└── Sierra/Decagon ($10B+) — managed CX service
```

### Dify — самый "видимый", но архитектурно другой продукт

130K stars, $30M Pre-A ($180M valuation), 280 enterprise клиентов (Maersk, Novartis).

**Что Dify умеет:** Agent Node (ReAct/FC) внутри workflow, triggers (cron/webhook с v1.5), 200+ LLM, embed API + JS iframe, plugin system.

**Чего НЕ умеет и НЕ сможет без переписывания:**

| Что нужно | Почему архитектурное ограничение |
|-----------|--------------------------------|
| Daemon (всегда живой агент) | Dify = request-response. Trigger → execute → done. Нет continuous reasoning loop |
| Proactive actions | Всегда reactive: trigger fires → workflow runs → done |
| Embeddable in-process | 5+ Docker containers (API + Worker + PG + Redis + Plugin Daemon). Нельзя import в свой процесс |
| Multi-agent tree | Agent = один node в workflow. Нет Supervisor → Sub-agents |
| Long-term memory | Hardcap 2000 tokens / 500 messages |

**Daemon mode для Dify — не feature request, а другой продукт.** Как попросить MySQL стать Redis: оба базы данных, но архитектурно разные.

**Зона риска:** product team скажет "нам Dify API + cron trigger хватит" для простых сценариев (FAQ-бот, support chat). Для reactive chat — Dify достаточно. ByteBrew побеждает там, где нужен **autonomous agent** (IoT мониторинг, sales outreach, coding, proactive support).

### NVIDIA NemoClaw — статус неизвестен (ждём GTC 16 марта)

**Источник:** статья Wired от 9 марта 2026, "люди знакомые с планами NVIDIA". Подхвачено CNBC, TechRadar, Tom's Hardware, Engadget. **Официального подтверждения от NVIDIA нет.**

**Что известно (из утечек):**
- Open-source (Apache 2.0 — заявлено, не подтверждено)
- Enterprise-focused: email processing, calendar, data analysis, cross-system orchestration
- Построен на NeMo + NIM + Nemotron 3 Nano (30B params, 1M context, MoE)
- Multi-agent: supervisor + workers
- Hardware-agnostic (заявлено)
- Партнёры (питчили): Salesforce, Cisco, Google, Adobe, CrowdStrike

**Что НЕ существует (на 14 марта):**
- ❌ GitHub репозиторий — нет ни строчки кода
- ❌ Документация — нет на developer.nvidia.com
- ❌ Официальный пресс-релиз — NVIDIA не подтвердила
- ❌ Официальный GTC-блог NVIDIA упоминает OpenClaw (playbook для DGX Spark), но **НЕ NemoClaw**
- ⚠️ SEO-спам сайты (nemoclaw.bot, nemoclaw.so) появились с "техническими деталями" без подтверждения
- ⚠️ "Релиз 6 марта" — ни одного артефакта

**Существует:** `github.com/NVIDIA/NeMo-Agent-Toolkit` (создан март 2025) — Python-библиотека для оркестрации агентов через LangChain/CrewAI. Возможно то, что перебрендируют в NemoClaw.

**Сценарии после GTC 16 марта:**

| Сценарий | Вероятность | Для ByteBrew |
|----------|:-----------:|-------------|
| Rebranding NeMo Agent Toolkit (Python library) | Высокая | Не конкурент — framework, не engine |
| Enterprise platform (ServiceNow-конкурент) | Средняя | Не конкурент — другой рынок (internal agents) |
| Embeddable agent engine (наша ниша) | Очень низкая | Нужно пересмотреть позиционирование |

| | NemoClaw (ожидаемый) | ByteBrew |
|--|---------|----------|
| **Тип** | Enterprise platform для организации | Engine внутри чужого продукта |
| **Агенты для** | Сотрудников компании (Cisco, Salesforce) | Клиентов продукта (покупатели, операторы) |
| **Конкурирует с** | ServiceNow, Salesforce Agentforce | OpenAI AgentKit, Dify API |

**Вердикт:** По имеющимся данным — другой рынок. Но нужно обновить после GTC 16 марта.

### OpenAI AgentKit — per-token embeddable (см. секцию выше)

Лучший embed UX (ChatKit, 50+ виджетов). Но: reactive, per-token ($600+/мес при 1K диалогов), OpenAI lock-in, нет self-hosted, нет daemon.

### Letta (ex-MemGPT) — ближайший по архитектуре

38K stars, $10M seed, Apache 2.0. Stateful agents с памятью, REST API, embeddable.

| | Letta | ByteBrew |
|--|------|----------|
| Stateful agents | **Да** (self-editing memory) | **Да** |
| Autonomous daemon | Нет | **Да** |
| Multi-agent tree | Нет | **Да** |
| YAML config | Нет (Python API) | **Да** |
| Language | Python | **Go** |
| Embeddable | Да (REST API) | **Да** (API + widget + SDK) |

**Вердикт:** близко по идее (server-based agents, embeddable API), но нет daemon, нет multi-agent tree, Python. ByteBrew = Letta + daemon + multi-agent + Go performance + YAML config.

### CopilotKit — embeddable, но UI copilot

28K stars, MIT. React/Angular компоненты для in-app AI copilots.

**Фокус:** UI copilot (подсказки, автокомплит, действия в интерфейсе). Не autonomous daemon. Frontend-specific (React).

### n8n — доказательство бизнес-модели

$2.5B valuation, $40M+ ARR. **15% выручки от embedded/OEM.** Sustainable Use License — NOT open source (запрещает коммерческий embedding без лицензии).

**Не конкурент** (workflow engine, не agent engine), но **benchmark бизнес-модели:** embedded/OEM = реальный revenue stream.

### Вертикальные гиганты (другой рынок)

| Компания | Funding | Valuation | Что делают |
|----------|---------|-----------|-----------|
| Sierra.ai | $635M | $10B | Managed CX service |
| Decagon | $481M | $4.5B | Managed CX service |
| Parloa | $560M | $3B | Contact center AI |
| Kore.ai | $223M | — | Enterprise omnichannel |
| PolyAI | $200M | $750M | Voice AI |

**Все — managed services для support.** "Мы построим тебе support-агента." Не engine. Не universal. Не embeddable. Не self-hosted.

### SDKs / Frameworks (альтернативный подход)

| | Claude Agent SDK | CrewAI (44K stars) | Google ADK (18K stars) |
|--|-----------------|-------|------------|
| Код нужен | Да (Python/TS) | Да (Python) | Да (Python/Go) |
| Daemon | Строить самому | Нет | Нет |
| Embeddable | Строить самому | Строить самому | Строить самому |
| Multi-agent | Subagents | Crews + YAML | Hierarchical tree |
| Model lock-in | Claude only | Нет | Gemini-optimized |

**ByteBrew vs SDKs:** разные уровни абстракции. SDK = кирпичи. ByteBrew = готовый дом.

**Google ADK** — единственный с Go SDK и hierarchical agent tree. Но это library, не server/daemon. Если Google добавит server mode — пересечение.

### Chatbot Platforms

Voiceflow, Botpress, Rasa — **chatbot ≠ agent.** Chatbot отвечает на вопросы. Agent автономно действует.

### Infrastructure (не конкуренты, интеграции)

| | Что | Для ByteBrew |
|--|-----|-------------|
| Composio (35K stars, $29M) | 1000+ tool integrations | MCP tools source |
| E2B ($32M) | Sandboxed execution | Sandbox для code agents |
| Braintrust ($80M) | Observability/evals | Monitoring layer |
| Letta (38K stars, $10M) | Stateful agent memory | Архитектурный конкурент (см. выше) |

---

## Матрица: кто что закрывает

| | Free | Autonomous Daemon | Embeddable | Model-Agnostic | Multi-Agent Tree | YAML Config | Go |
|---|:---:|:---:|:---:|:---:|:---:|:---:|:---:|
| **ByteBrew** | **Да** | **Да** | **Да** | **Да** | **Да** | **Да** | **Да** |
| Dify | Да (огр.) | Нет | Частично | Да | Нет | Нет | Нет |
| Letta | Да | Нет | Да | Да | Нет | Нет | Нет |
| OpenAI AgentKit | Нет | Нет | Да | Нет | Нет | Нет | Нет |
| CopilotKit | Да | Нет | Да | Да | Нет | Нет | Нет |
| CrewAI | Да | Нет | Частично | Да | Да (crews) | Да | Нет |
| Google ADK | Да | Нет | Да | Частично | Да | Нет | Частично |
| OpenClaw | Да | Да | Нет (B2C) | Да | Нет | Да | Нет |

**Ни один продукт не закрывает все 7 свойств.** ByteBrew — единственный.

### Позиционирование

```
                     No code (configure)
                              ↑
                              |
     Dify, n8n               |              ★ ByteBrew ★
     (visual, reactive       |              (autonomous agents,
      workflows)             |               embeddable, daemon)
                              |
  Reactive ←──────────────────┼──────────────→ Autonomous
  (trigger → execute → done)  |               (daemon, proactive)
                              |
     Voiceflow, Botpress     |              Claude SDK, CrewAI
     (chatbot builders)      |              (autonomous possible,
                              |               но нужна команда)
                              ↓
                     Code required (SDK/framework)
```

---

## Honest Assessment: где сильны, где слабы

### Где ByteBrew — killer

| Use case | Почему killer | Конкуренты |
|----------|--------------|-----------|
| **IoT / Industrial** | Ни одна IoT платформа не имеет autonomous agent с reasoning. ThingsBoard AI = "curl к LLM". ByteBrew + MCP = трансформация продукта | **Никого.** PagerDuty = DevOps, не IoT. Вертикальных AI agent для IoT нет |
| **DevOps proactive monitoring** | Agent daemon мониторит метрики, находит root cause, предлагает fix ДО того как позвонит клиент | PagerDuty AI (частично), но cloud-only и per-user pricing |
| **Complex SaaS automation** | Multi-agent tree для сложных workflows (coding, research, multi-step tasks) | SDK-based решения (Claude Agent SDK), но нужна команда |

### Где ByteBrew конкурентоспособен, но не killer

| Use case | Почему не killer | Конкуренты |
|----------|-----------------|-----------|
| **Sales assistant (chat)** | Reactive chat — достаточно. Daemon не нужен для "покупатель пишет → агент отвечает" | Intercom Fin ($0.99/resolution), Relevance AI, OpenAI AgentKit |
| **Support bot** | Тот же reactive паттерн | Intercom, Zendesk AI, Sierra ($10B) |
| **Simple FAQ/onboarding** | Dify сделает это быстрее через visual builder | Dify, Botpress, Voiceflow |

### Где не стоит конкурировать

| Use case | Почему | Кто лучше |
|----------|--------|-----------|
| **Простой chatbot** | Overkill. Autonomous engine для FAQ = пушкой по воробьям | Dify, Botpress, Voiceflow |
| **Enterprise с бюджетом** | Наймут команду, построят на Claude SDK, получат ровно то что хотят | Claude Agent SDK, OpenAI Agents API |

### Ключевой инсайт: autonomous ≠ overkill

"40% agent projects canceled" (Gartner) и "85% accuracy → 10-step = 20% success" — это про **full autonomy без human-in-the-loop**.

ByteBrew по умолчанию = **human-in-the-loop:** `confirmation_required: true`, `escalation`, `rules`. Агент **предлагает**, человек **подтверждает**. Это 2-3 шага, не 10. Success rate совсем другой.

Для IoT оператора: агент проанализировал → показал корреляцию → предложил действие → оператор нажал "Подтвердить". Это не "AI управляет заводом". Это "AI помогает оператору управлять заводом".

---

## Угрозы

| Угроза | Вероятность | Timeline | Митигация |
|--------|:-----------:|:--------:|-----------|
| **NVIDIA NemoClaw** окажется embeddable engine | Очень низкая | Неизвестно | По утечкам — enterprise internal platform. Кода нет. **Обновить после GTC 16 марта** |
| **Dify** запустит отдельный agent-продукт | Средняя | 6-12 мес | $30M + 130K stars = ресурсы есть. Но IoT/industrial — не их domain. Архитектура (Python) ограничивает performance |
| **Google ADK** добавит server/daemon mode | Средняя | 6-12 мес | Go SDK + hierarchical agents уже есть. Но Google фокус на Vertex Cloud |
| **OpenAI AgentKit** добавит self-hosted | Очень низкая | — | Структурный конфликт (per-token revenue) |
| **Letta** добавит daemon + multi-agent | Средняя | 3-6 мес | Python, $10M. Closest архитектурно, но нет domain expertise |
| **Новый стартап** в той же нише | Средняя | 3-6 мес | 4.5 мес head start. Moat = speed to PMF + первые клиенты |
| **Agent skepticism** (40% projects canceled) | Высокая | Сейчас | Human-in-the-loop по умолчанию. Free tier снижает риск. Фокус на простых доказуемых use cases |

### Главная угроза — не конкуренты

Главная угроза — **не доказать PMF до того как кто-то войдёт в нишу.** Код повторяют за 2-3 месяца. Клиентскую базу и domain expertise — нет.

---

## Competitive Moat

### Что уже построено (ByteBrew 1.x)

| Компонент | Наши сроки | Статус |
|-----------|-----------|:------:|
| Go agent engine (ReAct, multi-agent, 26+ tools, streaming) | 3.5 месяца | **Есть** |
| Mobile app (Flutter: чат, pairing, reconnect, plans, ask-user) | 8 дней | **Есть** |
| E2E encryption + Bridge (NAT traversal) | ~2 недели | **Есть** |
| Event Store + guaranteed delivery | ~1 неделя | **Есть** |
| WS transport, session management | ~2 недели | **Есть** |
| **Суммарно** | **~4.5 месяца** | Solo developer + AI |

Для конкурента с командой 3-5 + AI — реально повторить за **2-3 месяца**. Код — не долгосрочный moat.

### Реальный moat

- **Speed to market** — 4.5 месяца head start + working product
- **Product-market fit** — кто первый найдёт платящих клиентов, тот выиграл
- **Domain expertise** — знание проблем клиентов из реальных внедрений
- **Интеграции** — каждый MCP server / plugin = switching cost
- **Community** — free tier привлекает разработчиков, они приносят интеграции и фидбек

---

## Рыночные данные

| Метрика | Значение | Источник |
|---------|----------|----------|
| AI agent market | $7.3B (2025) → $41B (2030) | Gartner |
| CAGR | 41% | Gartner |
| Enterprise apps с AI agents к 2026 | 40% (vs <5% в 2025) | Gartner |
| Agentic AI → 30% enterprise software revenue к 2035 | $450B+ | Gartner |
| "Нет внутренней экспертизы" | 65% компаний | Gartner |
| Вышли из пилота в production | <25% | Gartner |
| Требуют security + auditability | 75% | Deloitte |
| AI agent startup funding H1 2025 | $2.8B | CB Insights |
| AI agent startups tracked | 400+ в 16 категориях | CB Insights |

---

## Ценообразование

### Почему НЕ per-token и НЕ per-conversation

Per-token (AgentKit) и per-conversation (Intercom) = vendor tax который растёт с масштабом. Продуктовая компания не хочет переменные AI-расходы в core продукте.

ByteBrew: **inference бесплатный** (компания использует свои модели). Платят за **платформу**, не за usage.

### ByteBrew pricing

| Tier | Что включено | Цена |
|------|-------------|------|
| **Community Edition** | Полный engine без лимитов. Agents, tools, MCP, kits (включая developer), triggers, API, dashboard. Single node. | **$0** |
| **Enterprise Edition** | CE + horizontal scaling (multi-node) + AI Observability Dashboard (prompt analytics, quality metrics, cost tracking, session explorer). Priority support + SLA. | **Contact us** |

Единая тарификация для всех use cases: sales agent, IoT agent, coding agent — один engine, одна лицензия.

**Воронка:**
```
Стартап ставит CE бесплатно (single node)
    → продукт растёт, нагрузка увеличивается
    → нужен horizontal scaling + production observability
    → покупает Enterprise (contact us)
```

---

## Резюме для pitch

**Одно предложение:** ByteBrew — autonomous AI agent engine. Добавляет "мозги" в существующие продукты через конфигурацию. Агент не просто отвечает — мониторит, анализирует, предлагает действия.

**Проблема:** AI перестал быть отдельной сферой — это способ решать задачи. Каждый продукт нуждается в AI capabilities. Но: SDKs требуют команду ($150K+). Per-token platforms (AgentKit) съедают маржу. Visual builders (Dify) = reactive chatbots, не autonomous agents.

**Решение:** Ready-to-deploy autonomous agent engine. Конфигурация, не код. Daemon — агент работает 24/7, мониторит, анализирует, предлагает действия. Human-in-the-loop по умолчанию.

**Где побеждаем:** задачи требующие proactive reasoning — агент **сам** мониторит, анализирует, предлагает. IoT, DevOps, complex automation, coding — любой продукт где нужны "мозги", а не просто чат.

**Где не конкурируем:** простой reactive chat (Intercom дешевле), FAQ-бот (Dify проще).

**Первые pilots:** продукты с существующим API/MCP, слабыми AI-фичами, и болью "хотим AI но нет команды". IoT (ThingsBoard, 120 MCP tools) — один из примеров, не единственный.

**Traction:** Working engine (Go, 4.5 месяца). Multi-agent orchestration. Open-source clients: CLI (TypeScript) + Mobile (Flutter) + Bridge relay. E2E encryption.

**Market:** $7.3B → $41B (CAGR 41%). $2.8B invested in H1 2025. 75% начинают с free/open-source.

**Бизнес-модель:** Community Edition бесплатный (полный engine, все kits). Enterprise Edition = scaling + AI observability (contact us pricing). Единая тарификация для всех use cases. Как QuestDB: core бесплатный, Enterprise для production at scale.

**Ask:** Seed round для абстрагирования engine + первые pilots.
