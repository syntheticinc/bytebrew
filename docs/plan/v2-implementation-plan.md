# V2 Implementation Plan

**Дата:** 2026-04-06
**PRD:** `docs/prd/bytebrew-cloud-engine-v2.md` (200+ ACs)
**Test Cases:** `docs/testing/v2-test-cases.md` (245 TCs)
**Discovery:** 100% crystallized, 14 architectural decisions

---

## Обзор

Полная реализация ByteBrew V2 — от текущего состояния (v1 engine + prototype admin) до production-ready open source release. Пять треков (включая тестирование), пять фаз с testing gates.

### Треки

| Трек | Стек | Ответственность |
|------|------|-----------------|
| **A — Engine Backend** | Go | Domain model, usecases, capabilities, flows, resilience |
| **B — Cloud Backend** | Go | Tenant system, Stripe, quota, widget backend, default model |
| **C — Admin Frontend** | React/Vite | Production admin, canvas, drill-in, bottom panel, inspect |
| **D — Public Frontend** | React/Vite | Landing page, docs page, open source prep |
| **T — Testing** | Playwright MCP + Verifier | Ручное тестирование, AC verification, gates |

### Что уже есть (v1)

**Backend:** Agent CRUD, Model CRUD, MCP CRUD, Trigger CRUD, Session management, Chat SSE, Knowledge upload, Auth JWT, Rate limiter, Audit, Health, gRPC streaming, WS bridge
**Frontend:** 23 pages (Canvas, AgentDrillIn, Agents, MCP, Models, Triggers, Inspect, WidgetConfig, etc.), prototype mode with mock data, ReactFlow canvas

### Что нужно для V2

~200+ ACs across: Schemas, Global entities, Capabilities (7 types), Memory, Flows engine, Lifecycle states, Recovery, Resilience, Policies, Tool tiers, MCP catalog+auth, Cloud tenant, Stripe, Widget, AI Assistant, Landing, Docs, OSS

---

## Phase 1: Foundation (Backend Domain + Frontend Production)

**Цель:** Новая domain model в БД, API, admin переведён с prototype на production mode.

**Зависимости:** Нет (стартовая фаза)

### A1: Schema Entity + Global Agents

**PRD:** §6.8, §8.9
**ACs:** AC-ENT-01..05, AC-CANVAS-06..07

**Задачи:**
1. Domain: `schema.go` — Schema entity (name, description, agent_refs[], trigger_refs[], created_at)
2. Domain: Обновить `agent.go` — агент = глобальная сущность (убрать привязку к flow/kit)
3. Domain: `gate.go` — Gate entity (schema_id, condition_type, config JSON, max_iterations, timeout)
4. Domain: `edge.go` — Edge entity (schema_id, source_agent, target_agent, type, config JSON)
5. Infrastructure: `schema_repository.go` — GORM CRUD
6. Infrastructure: обновить `agent_repository.go` — глобальные agents, cross-schema refs
7. Usecase: `schema_create/`, `schema_get/`, `schema_update/`, `schema_delete/`, `schema_list/`
8. Usecase: `agent_create/`, `agent_update/` — обновить (глобальные, без schema binding)
9. Delivery HTTP: `/api/v1/schemas` CRUD, `/api/v1/schemas/{id}/agents` (refs)
10. Delivery HTTP: обновить `/api/v1/agents` — глобальный список, фильтр by schema
11. Migration: SQL migration (schemas, gates, edges tables; agents table cleanup)

**Файлы:** ~15 новых, ~8 изменённых
**Тесты:** Unit tests для каждого usecase, integration test для API

### A2: Capability System (Runtime Injection)

**PRD:** §7.2, §8.8
**ACs:** AC-CAP-01..07, AC-TOOL-01..05

**Задачи:**
1. Domain: `capability.go` — Capability entity (agent_id, type enum, config JSON)
2. Domain: `capability_type.go` — enum: memory, knowledge, guardrail, output_schema, escalation, recovery, policies
3. Infrastructure: `capability_repository.go` — GORM CRUD
4. Service: `capability_injector.go` — при старте агента inject tools по capabilities (Memory → memory_recall/store, Knowledge → knowledge_search, Escalation → escalate)
5. Service: `tool_tier_enforcer.go` — Tier 1 always, Tier 2 auto, Tier 3 CE-only, Tier 4 MCP
6. Usecase: `capability_add/`, `capability_remove/`, `capability_update/`
7. Delivery HTTP: `/api/v1/agents/{name}/capabilities` CRUD
8. Обновить agent runtime: при старте агента читать capabilities → inject tools

**Файлы:** ~12 новых, ~5 изменённых
**Тесты:** Unit + integration

### A3: Memory System

**PRD:** §8.1
**ACs:** AC-MEM-01..04, AC-MEM-TERM-01..02, AC-MEM-RET-01..03

**Задачи:**
1. Domain: обновить `memory.go` — schema_id (не flow), unlimited retention, max_entries
2. Infrastructure: обновить memory repository — per-schema scope, FIFO eviction при max_entries
3. Infrastructure: `memory_recall_tool.go` — auto-inject в начале сессии (semantic search)
4. Infrastructure: `memory_store_tool.go` — агент решает что сохранить
5. Usecase: `memory_recall/`, `memory_store/`, `memory_list/`, `memory_clear/`
6. Delivery HTTP: `/api/v1/schemas/{id}/memory` — list, clear для пользователя (AC-MEM-03)
7. Migration: обновить memory table (schema_id, убрать flow_id)

**Файлы:** ~8 новых, ~5 изменённых
**Тесты:** Unit + integration (cross-session persistence, schema isolation, FIFO eviction)

### A4: Agent Lifecycle States + SSE Events

**PRD:** §8.3, §8.5
**ACs:** AC-STATE-01..04, AC-EVT-01..03

**Задачи:**
1. Domain: обновить `agent_state.go` — initializing → ready → running → needs_input → blocked → degraded → finished
2. Domain: `sse_event.go` — event types with schema_version field
3. Infrastructure: state machine с transitions + валидация
4. Delivery HTTP: обновить SSE writer — `agent.state_changed` events
5. Delivery HTTP: event schema versioning header
6. Обновить agent runtime: emit state transitions

**Файлы:** ~5 новых, ~8 изменённых
**Тесты:** Unit (state transitions), integration (SSE events)

### C1: Admin Production Mode + Bottom Panel

**PRD:** §5.6, §7.3, §7.5
**ACs:** AC-UX-09..12, AC-PANEL-01..04

**Задачи:**
1. Компонент: `BottomPanel.tsx` — resizable (drag handle, 150px-70%), collapse/expand, 2 tabs
2. Компонент: `SchemaSelector.tsx` — dropdown в panel header
3. Layout: обновить `App.tsx` — BottomPanel на ВСЕХ pages (кроме Login)
4. Layout: panel state persistence (localStorage: height, tab, open/closed)
5. Убрать: "Open Chat" из Sidebar (уже сделано в prototype)
6. Production cleanup: дедупликация, UX polish, убрать legacy redirects
7. API client: переключить на production endpoints (schemas, agents, capabilities)

**Файлы:** ~3 новых, ~10 изменённых
**Тесты:** E2E (Playwright): TC-ASST-01..04, TC-PANEL-01..04

### C2: Canvas Production + Instant Creation

**PRD:** §6.6, §6.7
**ACs:** AC-CANVAS-08..11

**Задачи:**
1. Canvas: убрать agent creation modal → instant node creation (API call + node add)
2. Canvas: убрать trigger creation modal → instant trigger node
3. Компонент: `CronScheduler.tsx` — human-readable presets + Advanced toggle
4. Canvas: edge creation → API call + edge add, edge click → Side Panel config
5. Canvas: gate node creation
6. Обновить API client: schemas, agents, triggers, edges, gates CRUD

**Файлы:** ~3 новых, ~8 изменённых
**Тесты:** E2E: TC-NODE-01..02, TC-CRON-01..03

---

## Phase 2: Core Engine (Flows + Capabilities Runtime)

**Цель:** Flows engine работает (flow/transfer/loop/gate), capabilities активны в runtime.

**Зависимости:** Phase 1 (domain model, schema entity, capability system)

### A5: Flows Engine

**PRD:** §8.2, §6.1-6.5
**ACs:** AC-FLOW-01..06, AC-EDGE-01..04, AC-GATE-01..04

**Задачи:**
1. Service: `flow_executor.go` — orchestrator: read schema edges → execute pipeline
2. Service: `gate_evaluator.go` — evaluate gate conditions (auto/human/LLM/all-completed)
3. Service: `edge_router.go` — route output: full/field_mapping/custom_prompt
4. Domain: flow execution state machine (parallel fork, gate join, loop с max_iterations)
5. Обновить agent runtime: после завершения агента → check outgoing edges → execute next
6. Delivery HTTP: flow execution через existing chat API (entry agent → pipeline)
7. SSE events: `flow.step_started`, `flow.step_completed`, `flow.gate_evaluated`

**Файлы:** ~8 новых, ~10 изменённых
**Тесты:** Unit (edge routing, gate eval), integration (full pipeline execution)

### A6: Recovery Recipes

**PRD:** §8.4
**ACs:** AC-REC-01..04

**Задачи:**
1. Domain: `recovery_recipe.go` — failure_type enum, recovery_action, retry_count, backoff
2. Service: `recovery_executor.go` — execute recipe per failure type
3. Обновить agent runtime: wrap tool calls + model calls с recovery
4. Degrade scope: per-session flag, reset on new session

**Файлы:** ~4 новых, ~5 изменённых
**Тесты:** Unit (each failure type → recovery action)

### A7: MCP Auth + Catalog

**PRD:** §8.6, §10
**ACs:** AC-AUTH-01..03, AC-MCP-01..07

**Задачи:**
1. Domain: обновить MCP server entity — auth config (type, key_env, client_id, etc.)
2. Infrastructure: `mcp_auth_provider.go` — forward_headers, api_key, oauth2 token refresh
3. Service: `mcp_catalog.go` — load mcp-catalog.yaml, merge with custom servers
4. Data: `mcp-catalog.yaml` — 10-15 curated servers
5. Delivery HTTP: `/api/v1/mcp/catalog` — list catalog + installed
6. Delivery HTTP: обновить `/api/v1/mcp` — auth config support
7. Infrastructure: обновить MCP client — apply auth per-request

**Файлы:** ~6 новых, ~8 изменённых
**Тесты:** Unit (auth provider), integration (catalog API)

### A8: Event Schema Versioning

**PRD:** §8.5
**ACs:** AC-EVT-01..03

**Задачи:**
1. Обновить SSE writer: добавить `schema_version` в каждый event
2. Документировать event contract (types, fields, versions)
3. Client-side: unknown event types → safe ignore

**Файлы:** ~2 изменённых
**Тесты:** Unit (event serialization)

### C3: Edge/Gate Config + Drill-in Polish

**PRD:** §6.3, §6.5, §7.1-7.2
**ACs:** AC-EDGE-01..04, AC-GATE-01..04, AC-UI-01..07

**Задачи:**
1. Компонент: `EdgeConfigPanel.tsx` — Side Panel для edge config (full/field_mapping/custom_prompt)
2. Компонент: `GateConfigPanel.tsx` — Side Panel для gate config (4 condition types)
3. Обновить `AgentDrillInPage.tsx` — production mode (API data, save to backend)
4. Обновить `CapabilityBlock.tsx` — production mode (API CRUD)
5. API client: edge, gate CRUD methods

**Файлы:** ~3 новых, ~8 изменённых
**Тесты:** E2E: edge config, gate config

### C4: Inspect Page Rewrite

**PRD:** §7.4
**ACs:** AC-INSPECT-01..06

**Задачи:**
1. Переписать `InspectPage.tsx` — paginated session table (API: `/api/v1/sessions`)
2. Компонент: `SessionTimeline.tsx` — step timeline с unified icons
3. Search + filter (status multi-select, date range, agent name)
4. Auto-refresh running sessions (SSE subscription)
5. Dead letter task display (⏰ icon)

**Файлы:** ~2 новых, ~2 изменённых
**Тесты:** E2E: TC-INSPECT-01..06

### C5: Test Flow Tab

**PRD:** §7.3, §8.13
**ACs:** AC-TESTFLOW-01..04, AC-TEST-01..03

**Задачи:**
1. Компонент: `TestFlowTab.tsx` — в BottomPanel
2. Компонент: `HeadersEditor.tsx` — key-value pairs, add/remove, JSON import
3. SSE streaming display (inline tool calls, reasoning)
4. "View in Inspect" link → navigate to session detail
5. Headers forwarding: отправлять headers в chat API → forward to MCP

**Файлы:** ~3 новых, ~2 изменённых
**Тесты:** E2E: TC-TESTFLOW-01..03

---

## Phase 3: Advanced Features (Lifecycle + Resilience + Guardrails)

**Цель:** Persistent agents, resilience, output guardrails, knowledge/RAG, policies.

**Зависимости:** Phase 2 (flows engine, capability system runtime)

### A9: Persistent Lifecycle + Task Dispatch

**PRD:** §8.10, §8.11
**ACs:** AC-LIFE-01..04, AC-COMM-01..03

**Задачи:**
1. Service: `persistent_agent_manager.go` — keep-alive persistent agents, context preservation
2. Service: `task_dispatcher.go` — create task → assign to agent → await result event
3. Domain: `task_packet.go` — task with timeout, status, result
4. Domain: agent lifecycle state machine extension (Spawning → Ready → Running → Blocked → Finished, loop for persistent)
5. Service: auto-compaction при context overflow
6. SSE events: `task.dispatched`, `task.completed`, `task.timeout`

**Файлы:** ~6 новых, ~8 изменённых
**Тесты:** Unit + integration (spawn vs persistent, task dispatch, context preservation)

### A10: Agent Resilience

**PRD:** §8.12
**ACs:** AC-RESIL-01..12

**Задачи:**
1. Service: `heartbeat_monitor.go` — heartbeat events, watchdog timer (2× interval)
2. Service: `circuit_breaker.go` — per-MCP, per-model (open/half-open/closed)
3. Service: `dead_letter_queue.go` — task timeout → dead letter, parent notification
4. Обновить MCP client: `tool_call_timeout` per-call (default 30s)
5. Обновить agent runtime: emit heartbeat, handle stuck detection
6. Delivery HTTP: circuit state в MCP status API, dead letters в Inspect API

**Файлы:** ~5 новых, ~8 изменённых
**Тесты:** Unit (circuit breaker state machine, heartbeat timing), integration

### A11: Output Guardrail Pipeline

**PRD:** §7.2 (Guardrail section)
**ACs:** AC-GRD-JSON-01..04, AC-GRD-LLM-01..04, AC-GRD-WH-01..05

**Задачи:**
1. Service: `guardrail_pipeline.go` — post-generation hook, 3 modes
2. Service: `guardrail_json.go` — JSON Schema validation
3. Service: `guardrail_llm_judge.go` — separate LLM call with judge prompt
4. Service: `guardrail_webhook.go` — POST with contract, timeout 10s, retry 1x
5. Domain: on_failure actions (retry max 3, error, fallback)
6. Integration: guard output before sending to user

**Файлы:** ~5 новых, ~3 изменённых
**Тесты:** Unit (each mode), integration (full pipeline)

### A12: Agent Policies Engine

**PRD:** §8.7
**ACs:** AC-POL-01..04

**Задачи:**
1. Service: `policy_engine.go` — evaluate typed conditions, execute actions
2. Domain: `policy_rule.go` — condition_type, action_type, config
3. Service: `policy_actions.go` — block, log_to_webhook, notify, inject_header, write_audit
4. Обновить agent runtime: evaluate policies before/after tool calls

**Файлы:** ~4 новых, ~3 изменённых
**Тесты:** Unit (each condition + action), integration

### A13: Knowledge/RAG Enhancement

**PRD:** §7.2 (Knowledge section)
**ACs:** AC-KB-FMT-01..05, AC-KB-LIST-01..05, AC-KB-PARAM-01..03

**Задачи:**
1. Обновить knowledge processing: PDF, DOCX, DOC, TXT, MD, CSV parsers
2. Обновить `knowledge_search` tool: read top_k, similarity_threshold from agent capability config
3. Delivery HTTP: file listing API (name, type, size, date, status)
4. Infrastructure: indexing pipeline (uploading → indexing → ready → error statuses)

**Файлы:** ~4 новых, ~6 изменённых
**Тесты:** Unit + integration (each format, search with params)

### C6: Widget Config Page (Production)

**PRD:** §9
**ACs:** AC-WID-01..07

**Задачи:**
1. Обновить `WidgetConfigPage.tsx` — production mode (API data, form fields from PRD)
2. Компонент: `WidgetPreview.tsx` — live preview с текущими настройками
3. Embed code generator (Cloud + self-hosted variants)
4. API client: widget CRUD

**Файлы:** ~2 новых, ~2 изменённых
**Тесты:** E2E: TC-WIDGET-01..03

### C7: MCP Page Enhancement

**PRD:** §10
**ACs:** AC-MCP-01..07

**Задачи:**
1. Обновить `MCPPage.tsx` — catalog view (Add from Catalog, Add Custom), cross-refs "Used by agents"
2. Auth config form (4 types)
3. Transport selector (stdio/streamable-http/sse/ws/docker)
4. Env vars editor per MCP server

**Файлы:** ~1 новый, ~2 изменённых
**Тесты:** E2E: catalog browse, add from catalog

---

## Phase 4: Cloud + Widget (Deployment)

**Цель:** Multi-tenant cloud, Stripe billing, widget embed, quota enforcement.

**Зависимости:** Phase 1-3 (engine features complete)

### B1: Tenant System

**PRD:** §3
**ACs:** AC-CLOUD-01..06

**Задачи:**
1. Migration: `tenant_id` на все таблицы (agents, schemas, triggers, gates, edges, sessions, memory, mcp_servers, capabilities)
2. Middleware: `tenant_middleware.go` — auto-scope все queries по tenant_id from JWT
3. Обновить все repositories: tenant-scoped queries
4. Registration → create tenant → empty workspace

**Файлы:** ~3 новых, ~20 изменённых (all repositories)
**Тесты:** Integration: TC-CLOUD-01..06 (tenant isolation, no cross-tenant leakage)

### B2: Stripe + Quota

**PRD:** §4
**ACs:** AC-PRICE-01..11

**Задачи:**
1. Stripe: новые products (free, pro, business)
2. Service: `quota_enforcer.go` — check limits per-tenant (schemas, agents, API calls, storage)
3. Delivery HTTP: `/api/v1/usage` — usage dashboard data
4. Delivery HTTP: quota middleware — 429 при превышении
5. Webhooks: Stripe payment → plan update → limits increase
6. 14-day trial logic

**Файлы:** ~5 новых, ~8 изменённых
**Тесты:** Integration: TC-PRICE-01..07, TC-QUOTA-01..04

### B3: Default Model Proxy

**PRD:** §4.2
**ACs:** AC-PRICE-06

**Задачи:**
1. Service: `default_model_proxy.go` — proxy to GLM 4.7 API
2. Rate limit: 100 req/month per tenant
3. Auto-provision: every new tenant gets default model

**Файлы:** ~2 новых, ~2 изменённых

### B4: Widget Backend

**PRD:** §9
**ACs:** AC-WID-01..07

**Задачи:**
1. Delivery HTTP: `/widget/{id}.js` — serve widget script
2. CORS: domain whitelist enforcement
3. Widget session → chat API → entry agent of bound schema
4. Widget CRUD API

**Файлы:** ~4 новых, ~2 изменённых
**Тесты:** Integration: TC-WIDGET-02..03

### B5: Cloud Sandbox

**PRD:** §3.3
**ACs:** AC-CLOUD-05

**Задачи:**
1. Service: `cloud_sandbox.go` — block Tier 3 tools in cloud mode
2. Error messages: structured (not silent fail)

**Файлы:** ~1 новый, ~2 изменённых
**Тесты:** Integration: TC-CLOUD-05

### C8: Pricing/Quota UX

**PRD:** §4.6
**ACs:** AC-PRICE-08..11

**Задачи:**
1. Компонент: `UsageDashboard.tsx` — bar charts, plan badge, billing dates
2. Компонент: `QuotaBanner.tsx` — 80%/95%/100% warning levels
3. Stripe Checkout integration (redirect + webhook callback)
4. Обновить `SettingsPage.tsx` — Usage tab

**Файлы:** ~3 новых, ~2 изменённых
**Тесты:** E2E: TC-QUOTA-01..04

---

## Phase 5: AI Assistant + Launch

**Цель:** Builder assistant, landing page, docs, open source preparation.

**Зависимости:** Phase 1-4 (everything works)

### A14: Builder Assistant Agent

**PRD:** §5.1-5.5, §5.8
**ACs:** AC-UX-01..08

**Задачи:**
1. Domain: `builder_assistant.go` — system agent (not user-visible)
2. Service: `assistant_router.go` — classify request (interview/assembly/direct/answer)
3. Service: `assistant_interview.go` — entropy-reduction questioning
4. Service: `assistant_assembler.go` — create schema + agents + edges + triggers
5. Infrastructure: admin tools (CRUD schema, agent, trigger, edge, MCP)
6. Service: `assistant_self_test.go` — mock run after assembly
7. Delivery HTTP: `/api/v1/admin/assistant/chat` — separate endpoint
8. SSE events: `admin.node_create`, `admin.field_update`, `admin.page_navigate`

**Файлы:** ~8 новых, ~5 изменённых
**Тесты:** Integration (routing, interview flow, assembly), E2E (Playwright)

### A15: Live Animation Backend

**PRD:** §5.7
**ACs:** AC-UX-13..16

**Задачи:**
1. SSE events: admin action events с animation hints
2. Обновить assistant assembler: emit SSE per action
3. Event types: node_create, node_update, edge_create, page_navigate, field_update

**Файлы:** ~2 новых, ~3 изменённых
**Тесты:** Integration: SSE events received

### C9: Live Animation Frontend

**PRD:** §5.5, §5.7
**ACs:** AC-UX-13..16

**Задачи:**
1. SSE subscriber: listen to admin events
2. Canvas animations: fade-in, pulse, fade-out
3. Drill-in animations: text streaming, slide-down
4. Page navigation: auto-navigate when assistant switches context

**Файлы:** ~3 новых, ~5 изменённых
**Тесты:** E2E: TC-ANIM-01..04

### C10: AI Assistant Chat UI

**Задачи:**
1. Обновить BottomPanel AI Assistant tab: connect to builder assistant API
2. Interview mode UI (questions + answers)
3. Assembly progress UI (step indicators)
4. Self-test results display

**Файлы:** ~2 новых, ~3 изменённых

### D1: Landing Page

**PRD:** §11
**ACs:** AC-LAND-01..04

**Задачи:**
1. 9 секций по структуре из PRD
2. Responsive design
3. "Try free" → registration, "Self-host" → docs
4. Pricing section с актуальными тарифами

**Файлы:** ~5 новых (in cloud-web)
**Тесты:** E2E: TC-LAND-01..04

### D2: Documentation Page

**PRD:** §14.5
**ACs:** AC-DOCS-01..03

**Задачи:**
1. Docs page с sidebar navigation
2. Getting Started, Concepts, Configuration, API Reference, Self-Hosting, Widget, Examples
3. Content writing (parallel with development)

**Файлы:** ~8 новых (in cloud-web)
**Тесты:** E2E: page loads, navigation works

### D3: Open Source Preparation

**PRD:** §14
**ACs:** AC-OSS-01..06

**Задачи:**
1. Repo restructure: engine + admin → public repo
2. `git filter-repo` — clean secrets from history
3. LICENSE (BSL 1.1), README.md, CONTRIBUTING.md, CODE_OF_CONDUCT.md
4. `.github/` — issue templates, PR templates, CI workflows
5. Release workflow: auto-update Change Date
6. Verify: `git clone` + `docker build` + `docker run` works

**Файлы:** ~10 новых
**Тесты:** CI: TC-OSS-01..06

### D4: Integration Testing + Security Audit

**Задачи:**
1. Full end-to-end scenarios (registration → schema → agents → chat → inspect)
2. Tenant isolation audit
3. Cloud sandbox verification
4. Rate limit / quota stress test
5. MCP auth verification

---

## Зависимости между фазами

```
Phase 1 ──────┬──────── Phase 2 ──────┬──────── Phase 3 ────┐
(Foundation)  │        (Core Engine)  │        (Advanced)    │
              │                       │                      │
              │                       │                      ▼
              │                       │              Phase 4 (Cloud)
              │                       │                      │
              │                       │                      ▼
              └───────────────────────┴───────────── Phase 5 (Launch)
```

**Критический путь:** A1 → A5 → A9 → A14 (domain → flows → lifecycle → assistant)

**Параллелизация внутри фаз:**
- Phase 1: A1-A4 параллельно с C1-C2 (backend и frontend независимы)
- Phase 2: A5-A8 параллельно с C3-C5
- Phase 3: A9-A13 параллельно с C6-C7
- Phase 4: B1-B5 параллельно с C8
- Phase 5: A14-A15 параллельно с C9-C10 параллельно с D1-D4

---

## Work Packages для Team Execution

| # | Package | Track | Phase | Agents | Effort |
|---|---------|-------|-------|--------|--------|
| WP-01 | Schema Entity + Global Agents | A | 1 | backend-developer | L |
| WP-02 | Capability System | A | 1 | backend-developer | L |
| WP-03 | Memory System | A | 1 | backend-developer | M |
| WP-04 | Lifecycle States + SSE | A | 1 | backend-developer | M |
| WP-05 | Admin Bottom Panel | C | 1 | admin-developer | M |
| WP-06 | Canvas Instant Creation + Cron | C | 1 | admin-developer | M |
| WP-07 | Flows Engine | A | 2 | backend-developer (opus) | XL |
| WP-08 | Recovery Recipes | A | 2 | backend-developer | M |
| WP-09 | MCP Auth + Catalog | A | 2 | backend-developer | L |
| WP-10 | Event Versioning | A | 2 | backend-developer | S |
| WP-11 | Edge/Gate Config UI | C | 2 | admin-developer | M |
| WP-12 | Inspect Page Rewrite | C | 2 | admin-developer | M |
| WP-13 | Test Flow Tab | C | 2 | admin-developer | M |
| WP-14 | Persistent Lifecycle + Task Dispatch | A | 3 | backend-developer (opus) | XL |
| WP-15 | Agent Resilience | A | 3 | backend-developer (opus) | XL |
| WP-16 | Guardrail Pipeline | A | 3 | backend-developer | L |
| WP-17 | Policies Engine | A | 3 | backend-developer | M |
| WP-18 | Knowledge/RAG Enhancement | A | 3 | backend-developer | M |
| WP-19 | Widget Config UI | C | 3 | admin-developer | M |
| WP-20 | MCP Page Enhancement | C | 3 | admin-developer | M |
| WP-21 | Tenant System | B | 4 | backend-developer (opus) | XL |
| WP-22 | Stripe + Quota | B | 4 | backend-developer | L |
| WP-23 | Default Model Proxy | B | 4 | backend-developer | S |
| WP-24 | Widget Backend | B | 4 | backend-developer | M |
| WP-25 | Cloud Sandbox | B | 4 | backend-developer | S |
| WP-26 | Pricing/Quota UX | C | 4 | admin-developer | M |
| WP-27 | Builder Assistant | A | 5 | backend-developer (opus) | XL |
| WP-28 | Live Animation (BE+FE) | A+C | 5 | backend + admin | L |
| WP-29 | Landing Page | D | 5 | frontend-developer | L |
| WP-30 | Documentation Page | D | 5 | frontend-developer | L |
| WP-31 | Open Source Prep | D | 5 | general | M |
| WP-32 | Integration Testing + Security | D | 5 | tester + security | L |
| **Testing Gates** | | | | | |
| TG-1 | Phase 1 Testing Gate | T | 1 | qa-tester + verifier | L |
| TG-2 | Phase 2 Testing Gate | T | 2 | qa-tester + verifier | L |
| TG-3 | Phase 3 Testing Gate | T | 3 | qa-tester + verifier | XL |
| TG-4 | Phase 4 Testing Gate | T | 4 | qa-tester + verifier | L |
| TG-5 | Phase 5 Final Acceptance | T | 5 | qa-tester + verifier + security | XL |

**Effort scale:** S (< 5 файлов), M (5-12), L (12-20), XL (20+)

---

## Testing Strategy (три уровня)

Тестирование — **не завершающий этап**, а **gate на переход к следующей фазе**. Каждая фаза проходит три уровня проверки.

### T1: Unit + Integration (в рамках каждого WP)

**Кто:** Разработчик (backend-developer / admin-developer)
**Когда:** Одновременно с написанием кода
**Что:**
- Go: `go test ./...` — unit тесты для каждого нового struct/interface/function
- React: `npx vitest` — unit тесты для новых компонентов
- API: integration тесты через `httptest` (реальный PostgreSQL, не моки)
- Compilation: `go build ./...`, `npm run build` — zero errors

**Критерий:** Все тесты проходят, zero compilation errors. Без этого WP не считается завершённым.

### T2: Ручное тестирование через Playwright MCP (после завершения фазы)

**Кто:** qa-tester agent через `mcp__playwright__*` tools
**Когда:** После завершения всех WP фазы, на Docker test stack (`localhost:9555`)
**Цель:** **НАЙТИ БАГ**, не подтвердить что работает

**Процесс:**
1. Собрать Docker: `docker compose -f docker-compose-sse-test.yml -p sse-test up -d --build`
2. Playwright MCP: `browser_navigate` → `browser_snapshot` → `browser_click` → `browser_fill_form` → `browser_take_screenshot`
3. Пройти по UI сценариям из TC файла (соответствующие секции фазы)
4. **Exploratory testing:**
   - Нестандартные входные данные (unicode, пустые строки, SQL injection patterns)
   - Быстрые повторные клики, concurrent requests
   - Refresh страницы в середине операции
   - Граничные значения (max_entries = 0, max_iterations = 0, empty schema)
   - Навигация "назад" в браузере
5. Скриншоты каждого экрана → сравнение с PRD спецификацией

**Артефакт:** Отчёт с найденными багами, скриншоты, шаги воспроизведения.

**Какие TC по фазам:**

| Phase | TC Groups |
|-------|-----------|
| 1 | TC-PANEL-*, TC-ASST-*, TC-NODE-*, TC-CRON-*, TC-ENT-* |
| 2 | TC-FLOW-*, TC-REC-*, TC-MCP-*, TC-INSPECT-*, TC-TESTFLOW-*, TC-CANVAS-* |
| 3 | TC-LIFE-*, TC-COMM-*, TC-RESIL-*, TC-GRD-*, TC-POL-*, TC-KB-*, TC-WIDGET-* |
| 4 | TC-CLOUD-*, TC-PRICE-*, TC-QUOTA-* |
| 5 | TC-UX-*, TC-ANIM-*, TC-LAND-*, TC-OSS-*, TC-UJ-* (full user journeys) |

### T3: Приёмочное тестирование на базе AC (gate на следующую фазу)

**Кто:** verifier agent (отдельный от разработчика и QA)
**Когда:** После T2, перед началом следующей фазы
**Цель:** Формальная сверка каждого AC из PRD → PASS/FAIL

**Процесс:**
1. Прочитать PRD — найти все ACs, относящиеся к текущей фазе
2. Для каждого AC:
   - Воспроизвести сценарий (API curl, Playwright, или code inspection)
   - Зафиксировать результат: PASS / FAIL / BLOCKED (с причиной)
3. Составить **матрицу покрытия**:
   ```
   ## Phase N Acceptance Report

   | AC-ID | Description | Method | Result | Evidence |
   |-------|------------|--------|--------|----------|
   | AC-ENT-01 | Agent — global entity | API test | ✅ PASS | curl response |
   | AC-ENT-02 | Canvas click → agent page | Playwright | ✅ PASS | screenshot |
   | AC-FLOW-03 | Loop edge with gate | API test | ❌ FAIL | infinite loop, no max_iterations check |
   ```
4. **Gate rule:** 100% PASS required. FAIL → fix → re-test. BLOCKED → обоснование + план.

**Какие ACs по фазам:**

| Phase | AC Groups | Кол-во |
|-------|-----------|--------|
| 1 | AC-ENT-*, AC-CANVAS-06..11, AC-MEM-*, AC-STATE-*, AC-EVT-*, AC-PANEL-*, AC-UX-09..12 | ~40 |
| 2 | AC-FLOW-*, AC-REC-*, AC-AUTH-*, AC-MCP-01..07, AC-EDGE-*, AC-GATE-*, AC-INSPECT-*, AC-TESTFLOW-*, AC-UI-*, AC-CAP-* | ~55 |
| 3 | AC-LIFE-*, AC-COMM-*, AC-RESIL-*, AC-GRD-*, AC-POL-*, AC-KB-*, AC-SCH-*, AC-ESC-*, AC-NOTIFY-*, AC-TOOL-*, AC-WID-* | ~65 |
| 4 | AC-CLOUD-*, AC-PRICE-*, AC-TEST-04 | ~20 |
| 5 | AC-UX-01..08, AC-UX-13..16, AC-LAND-*, AC-OSS-*, AC-DOCS-* | ~25 |

### Workflow по фазе

```
┌─────────────┐     ┌──────────────┐     ┌───────────────┐     ┌──────────────┐
│ WP Development │──→│ T1: Unit/Int  │──→│ T2: Playwright  │──→│ T3: AC Gate   │
│ (backend+FE)   │   │ (per WP)      │   │ MCP (ручное)    │   │ (приёмочное)  │
└─────────────┘     └──────────────┘     └───────────────┘     └──────────────┘
                                               │                      │
                                          Баги найдены?          100% PASS?
                                               │                      │
                                          Fix → re-T2            Fix → re-T3
                                                                      │
                                                                 ✅ → Phase N+1
```

### Phase 5: Final Acceptance (расширенный)

Phase 5 включает **полное приёмочное тестирование** всего продукта:

1. **Full regression** — все 245 TCs проходят
2. **User Journey testing** — TC-UJ-01..40 (полные сценарии от регистрации до production use)
3. **Security audit** — tenant isolation, cloud sandbox, injection protection
4. **Performance** — 100 concurrent sessions, burst rate limiting
5. **Docker smoke test** — `git clone` → `docker build` → `docker run` → admin login → first chat
6. **Cross-browser** — Chrome, Firefox, Safari (Playwright)
7. **Final AC matrix** — все 200+ ACs → 100% PASS

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Flows engine complexity (fork/join/loop) | Delay Phase 2 | Start with simple linear, add parallel later |
| AI Assistant качество routing | Phase 5 блокер | Fallback: manual-only mode (no AI), assistant = stretch goal |
| Tenant migration объём | Phase 4 длинный | Generate migration script, test on copy of DB |
| Knowledge indexing performance | Large files slow | Background job queue, progress status |
| Circuit breaker false positives | Degraded mode too often | Conservative thresholds, manual override |

---

## Quick Start

```bash
# Phase 1 kick-off: parallel backend + frontend
# Backend: start with WP-01 (Schema Entity)
# Frontend: start with WP-05 (Bottom Panel)
```
