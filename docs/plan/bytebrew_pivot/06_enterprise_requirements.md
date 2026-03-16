# ByteBrew Enterprise Edition — Requirements

**Дата:** 16 марта 2026
**Статус:** Roadmap. Показываем на сайте и в pitch. Реально не продаём — при обращении честно говорим что фичи в разработке, онбордим как early adopters / потенциальных клиентов.

---

## Позиционирование

```
Community Edition (CE):  бесплатный, полный engine, single node
Enterprise Edition (EE): CE + horizontal scaling + AI observability
```

Два бинарника, один исходный код:
```
bytebrew-srv/
├── internal/          # core engine (общий код)
├── enterprise/        # закрытые enterprise фичи
│   ├── scaling/       # distributed jobs, cross-node events, session routing
│   └── observability/ # metrics aggregator, quality scoring, dashboard
├── cmd/
│   ├── ce/main.go     # Community build
│   └── ee/main.go     # Enterprise build (imports enterprise/)
```

---

## Тарификация

| | Community | Enterprise |
|--|----------|-----------|
| **Engine** | Полный, без лимитов | Полный, без лимитов |
| **Deployment** | Single node | Horizontal scaling (multi-node) |
| **AI Observability** | Базовые логи (JSON, EventStore) | Prompt Analytics Dashboard |
| **Support** | GitHub issues | Priority + SLA |
| **Бинарник** | `bytebrew-ce` | `bytebrew-ee` |
| **Цена** | $0 | Contact us |

---

## Enterprise Feature 1: Horizontal Scaling

### Проблема
Single node CE: один engine процесс. При росте нагрузки (50+ concurrent sessions) — CPU 100%, latency растёт. Клиент не может просто поставить nginx round-robin — сломается cross-node agent spawn (supervisor на Node 1, code-agent попал на Node 2, результат не вернётся).

### Решение
Engine-level distributed orchestration:

```
Load Balancer
├── Engine Node 1 ─┐
├── Engine Node 2 ─┤── PostgreSQL (shared state)
└── Engine Node 3 ─┘── NATS (inter-node messaging)
```

### Компоненты

**Distributed Job Queue:**
- PostgreSQL `SELECT ... FOR UPDATE SKIP LOCKED` для job distribution
- Каждая нода берёт свободные jobs
- Failover: если нода упала, job не потерян (PostgreSQL lock released)

**Cross-node Event Broadcasting:**
- NATS pub/sub между нодами
- Event от agent'а на Node 1 → клиент подключён к Node 2 → NATS доставляет
- SessionRegistry: in-memory per-node + NATS sync

**Session Routing:**
- Agents с kit (developer: LSP, indexing) → sticky sessions (session → node mapping в PostgreSQL)
- Agents без kit → round-robin
- Cross-node agent spawn: supervisor на Node 1 спаунит agent → NATS → agent запускается на Node 2 → результат через NATS обратно

**Zero-downtime Deploy:**
- Graceful drain: нода перестаёт брать новые jobs, дожидается текущих
- Rolling update: ноды обновляются по одной

### Что уже заложено в CE архитектуре
- GORM → PostgreSQL (shared state ready)
- Event broadcasting через interfaces (можно подменить на NATS)
- Job Queue через interfaces (можно сделать distributed)
- Session state через GORM (не in-memory maps)

### Оценка реализации
1-2 недели (NATS интеграция + distributed job queue + session routing)

---

## Enterprise Feature 2: AI Observability (Prompt Analytics)

### Проблема
Агент обслуживает сотни/тысячи пользователей в production. Как понять:
- На каких вопросах агент отвечает плохо?
- Сколько стоит inference per agent / per session?
- Какие tools тормозят?
- Промпт обновили — стало лучше или хуже?

Данные есть в логах (EventStore, Context Logger), но копаться в JSON файлах 500 сессий/день — нереально.

### Решение
Встроенный AI Observability Dashboard — данные из reasoning chain, агрегация, визуализация.

### Архитектура

```
Уже есть в CE:
├── EventStore (все events: tool_call, tool_result, answer, reasoning, agent_spawned/completed)
├── Context Logger (full LLM context per step: промпт, messages, tokens)
└── GORM + PostgreSQL

Enterprise добавляет:
├── Metrics Aggregator (Go) — events → metrics tables
├── Quality Scorer (rule-based + optional LLM-as-judge)
├── Cost Calculator (tokens × model price)
└── Dashboard UI (templ + htmx, embedded в Go binary)
```

### Dashboard функционал

**Session Explorer:**
- Browse любую сессию
- Видишь полный reasoning trace: user input → thinking → tool calls → tool results → response
- Как LangSmith trace view, но встроенный

**Quality Metrics:**
- Rule-based scoring:
  - Completion rate (% сессий завершённых без ошибки)
  - Escalation rate (% сессий с эскалацией к человеку)
  - Avg response time (от вопроса до ответа)
  - Avg steps count (меньше = эффективнее)
  - Tool error rate
- Trend по дням/неделям: quality растёт или падает?
- Optional: LLM-as-judge sampling (каждая N-я сессия оценивается LLM)

**Cost Tracking:**
- Tokens in/out per session, per agent, per day
- Estimated cost (tokens × model price из конфига)
- Cost trend: растёт с ростом usage? какой agent самый дорогой?

**Escalation Analytics:**
- Top вопросы на которых agent эскалирует
- Pattern analysis: "30% эскалаций про возврат → добавить tool для возврата"

**Tool Performance:**
- Latency per tool (avg, p95, p99)
- Error rate per tool
- Call frequency: какие tools используются чаще

**Prompt Comparison:**
- Сравнить две версии конфига agent'а
- Метрики version A vs version B (quality, cost, speed)

### Data Pipeline

```
EventStore events
    ↓ (Go aggregator, периодический или streaming)
PostgreSQL metrics tables:
    session_metrics:  session_id, agent, duration, tokens_in, tokens_out, steps, escalated, error, quality_score
    agent_metrics:    agent_name, date, sessions_count, avg_duration, avg_tokens, escalation_rate, error_rate
    tool_metrics:     tool_name, agent, date, call_count, avg_latency, error_rate
    cost_metrics:     agent_name, date, total_tokens, estimated_cost
```

### Dashboard UI

**templ + htmx** — server-side rendered, embedded в Go binary.
- Нет Node.js, нет npm build
- Компилируется в тот же binary (EE)
- Lightweight, быстрый

### Что уже заложено в CE архитектуре
- EventStore хранит все events (tool_call, tool_result, answer, reasoning, agent lifecycle)
- Context Logger пишет полный LLM контекст per step (tokens, messages)
- GORM + PostgreSQL для metrics tables

### Оценка реализации
- Metrics aggregator: 3-5 дней
- Quality scorer (rule-based): 2-3 дня
- Cost calculator: 1-2 дня
- Dashboard UI (templ + htmx): 1-2 недели
- **Итого: 2-3 недели**

---

## Revenue Projection (для seed pitch)

```
Year 1:
  - Community adoption: 500+ installs (engine + coding daemon)
  - Enterprise pilots: 5-10 companies testing EE
  - Enterprise revenue: 10 × EE license = первые контракты

Year 2:
  - Community: 2000+ installs
  - Enterprise conversion: 5-10% of community
  - Enterprise revenue: 100 × EE license

Year 3:
  - Enterprise: 300+ EE clients
```

**Note:** конкретные цены EE = "Contact us". Revenue projection уточняется после первых enterprise контрактов.

---

## Competitor Pricing Reference

| Product | Enterprise Price | What They Sell |
|---------|-----------------|---------------|
| QuestDB Enterprise | Custom ($50K+/год) | HA, replication, RBAC, TLS, compression |
| LangSmith Plus | $39/seat/мес | Tracing, eval, prompt management |
| Dify Enterprise | Custom | SSO, RBAC, multi-tenant |
| n8n Enterprise | Custom (~€800+/мес) | SSO, audit, execution limits |
| CrewAI Enterprise | ~$120K/год | Self-hosted, SOC2, SSO, dedicated engineers |

ByteBrew EE pricing = contact us. Ориентир: $500-1000/мес — значительно ниже CrewAI ($120K/год) и QuestDB (custom), в диапазоне n8n/Dify.

---

## Implementation Priority

| # | Что | Когда |
|---|-----|-------|
| 1 | CE MVP (engine pivot) | Сейчас (seed milestone) |
| 2 | AI Observability (metrics + dashboard) | После CE checkpoint 2 |
| 3 | Horizontal Scaling (NATS + distributed jobs) | После observability |
| 4 | EE binary build (ce/ vs ee/ entry points) | После обе фичи готовы |
