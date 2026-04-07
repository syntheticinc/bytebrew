# AC Sync Matrix — ByteBrew V2

**Дата:** 2026-04-06
**PRD:** `docs/prd/bytebrew-cloud-engine-v2.md`
**Test Cases:** `docs/testing/v2-test-cases.md`
**Codebase:** `engine/` (Go backend + React admin)

---

## Legend

| Status | Meaning |
|--------|---------|
| ✅ DONE | Implemented in code + has test case |
| ⚠️ PARTIAL | Implemented but missing TC or proto coverage |
| ❌ GAP | Not implemented or not wired |
| 🔄 DEFERRED | Marked BACKEND-DEFERRED in PRD |

**Proto** column: ✓ = mock data in `admin/src/mocks/`, − = no proto needed / not applicable
**TC** column: TC-ID if exists, − if none
**Backend Code** column: key files (abbreviated)
**Frontend Code** column: key files (abbreviated)

---

## 1. Cloud (AC-CLOUD)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-CLOUD-01 | Register → tenant → empty workspace | − | TC-CLOUD-01 | `domain/tenant.go`, `service/billing/quota.go` | − | ✅ |
| AC-CLOUD-02 | Tenant A не видит данных Tenant B | − | TC-CLOUD-02 | `delivery/http/tenant_middleware.go` | − | ✅ |
| AC-CLOUD-03 | API calls per-tenant, 429 при превышении | − | TC-CLOUD-03 | `delivery/http/configurable_rate_limiter.go`, `delivery/http/rate_limit_usage_handler.go` | − | ✅ |
| AC-CLOUD-04 | Storage per-tenant (memory+knowledge+sessions) | − | TC-CLOUD-04 | `service/billing/quota.go`, `delivery/http/usage_handler.go` | − | ✅ |
| AC-CLOUD-05 | Cloud агент не может вызвать file/shell tools | − | TC-CLOUD-05, TC-TOOL-03 | `service/cloud/sandbox.go`, `service/cloud/sandbox_test.go` | − | ✅ |
| AC-CLOUD-06 | Rate limit per-tenant burst protection | − | TC-CLOUD-06 | `delivery/http/configurable_rate_limiter.go` | − | ✅ |

---

## 2. Pricing (AC-PRICE)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-PRICE-01 | Free: 1 schema, 10 agents, 1000 API calls | − | TC-PRICE-01 | `service/billing/quota.go`, `domain/billing.go` | − | ✅ |
| AC-PRICE-02 | Превышение лимита → сообщение + CTA upgrade | − | TC-PRICE-02 | `service/billing/quota.go` | − | ⚠️ |
| AC-PRICE-03 | Stripe checkout для Pro и Business | − | TC-PRICE-03 | `service/billing/stripe.go` | − | ✅ |
| AC-PRICE-04 | 14-day Pro trial без карты | − | TC-PRICE-04 | `service/billing/stripe.go`, `domain/billing.go` | − | ✅ |
| AC-PRICE-05 | После trial → возврат на Free (не блокировка) | − | TC-PRICE-05 | `service/billing/stripe.go` | − | ✅ |
| AC-PRICE-06 | Default model (GLM 4.7) без API ключа, лимит 100 req | − | TC-PRICE-06 | `service/cloud/default_model.go` | − | ✅ |
| AC-PRICE-07 | BYOK: вставил ключ → модели доступны | − | TC-PRICE-07 | `service/cloud/default_model.go` | − | ✅ |
| AC-PRICE-08 | Usage dashboard с bar charts per-metric | ✓ | TC-QUOTA-01 | `delivery/http/usage_handler.go` | `admin/src/components/UsageDashboard.tsx` | ✅ |
| AC-PRICE-09 | Warning banner при 80% | ✓ | TC-QUOTA-02 | `service/billing/quota.go` | `admin/src/components/UsageDashboard.tsx` | ⚠️ |
| AC-PRICE-10 | Hard block modal при 100% с upgrade CTA | ✓ | TC-QUOTA-03 | `service/billing/quota.go` | `admin/src/components/UsageDashboard.tsx` | ⚠️ |
| AC-PRICE-11 | Stripe upgrade → лимиты мгновенно | − | TC-QUOTA-04 | `service/billing/stripe.go` | − | ✅ |

---

## 3. Brewery UX — AI Assistant (AC-UX)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-UX-01 | Новый пользователь: пустой canvas + AI Assistant | ✓ | TC-UX-01, TC-UJ-01 | `service/assistant/router.go` | `admin/src/components/BottomPanel.tsx` | ✅ |
| AC-UX-02 | Vague запрос → уточняющие вопросы (interview) | − | TC-UX-02 | `service/assistant/router.go` | `admin/src/components/BottomPanel.tsx` | ✅ |
| AC-UX-03 | Clear запрос → прямое выполнение | − | TC-UX-03 | `service/assistant/router.go` | − | ✅ |
| AC-UX-04 | Создание agent/edge → нода/edge с анимацией | ✓ | TC-UX-04 | `service/assistant/events.go` | `admin/src/pages/AgentBuilderPage.tsx` | ⚠️ |
| AC-UX-05 | Обновление агента → нода pulse-анимация | ✓ | TC-UX-05 | `service/assistant/events.go` | `admin/src/pages/AgentBuilderPage.tsx` | ⚠️ |
| AC-UX-06 | Self-test → ноды подсвечиваются | ✓ | TC-UX-06 | `service/assistant/builder.go` | `admin/src/pages/AgentBuilderPage.tsx` | ⚠️ |
| AC-UX-07 | Builder assistant не виден в /admin/agents | − | TC-UX-07 | `service/assistant/builder.go` | − | ✅ |
| AC-UX-08 | Builder assistant нельзя удалить/редактировать | − | TC-UX-08 | `service/assistant/builder.go` | − | ✅ |
| AC-UX-09 | Bottom panel на ВСЕХ admin pages | ✓ | TC-ASST-01, TC-ASST-02, TC-PANEL-04 | − | `admin/src/components/Layout.tsx`, `admin/src/components/BottomPanel.tsx` | ✅ |
| AC-UX-10 | Schema selector dropdown в chat header | ✓ | TC-ASST-03 | − | `admin/src/components/SchemaSelector.tsx` | ✅ |
| AC-UX-11 | 1 entry agent per schema — chat → entry agent | − | − | `service/assistant/router.go` | − | ⚠️ |
| AC-UX-12 | "Open Chat" ссылка удалена из admin sidebar | ✓ | TC-ASST-04 | − | `admin/src/components/Layout.tsx` | ✅ |
| AC-UX-13 | Создание агента → нода fade-in на canvas (DEFERRED) | 🔄 | TC-ANIM-01 | `service/assistant/events.go` | − | 🔄 |
| AC-UX-14 | Изменение system prompt → текст стримится (DEFERRED) | 🔄 | TC-ANIM-02 | − | − | 🔄 |
| AC-UX-15 | Добавление capability → slide-down анимация (DEFERRED) | 🔄 | TC-ANIM-03 | − | − | 🔄 |
| AC-UX-16 | Assistant actions → SSE events → UI с анимацией (DEFERRED) | 🔄 | TC-ANIM-04 | `service/assistant/events.go` | − | 🔄 |
| AC-UX-01 (config) | Capability SVG иконки (Lucide, не emoji) | ✓ | TC-UX-CONFIG-01 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-UX-02 (config) | Секции drill-in page collapsible | ✓ | TC-UX-CONFIG-02 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-UX-03 (config) | Model & Lifecycle в 2-колоночном layout | ✓ | TC-UX-CONFIG-03 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |

---

## 4. Canvas (AC-CANVAS, AC-TRIGGER, AC-EDGE, AC-GATE)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-CANVAS-01 | Gate нода: отображается, настраивается condition | ✓ | TC-CANVAS-01, TC-UJ-05 | `domain/edge.go` | `admin/src/components/builder/GateNode.tsx`, `admin/src/components/builder/GateConfigPanel.tsx` | ✅ |
| AC-CANVAS-02 | Flow/transfer/loop edges drag-and-drop | ✓ | TC-CANVAS-02, TC-UJ-08 | `domain/edge.go` | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-CANVAS-03 | Edge config (full output/field mapping/custom prompt) в Side Panel | ✓ | TC-CANVAS-03, TC-UJ-07 | `domain/edge.go` | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-CANVAS-04 | Parallel: несколько flow edges → fork | − | TC-CANVAS-04 | `service/flow/executor.go` | − | ✅ |
| AC-CANVAS-05 | Gate join: ждёт всех входящих | − | TC-CANVAS-05 | `service/flow/executor.go` | − | ✅ |
| AC-CANVAS-06 | Schemas: create/switch/delete/rename | ✓ | TC-CANVAS-06, TC-UJ-02, TC-UJ-03 | `app/http_adapters.go` | `admin/src/components/SchemaSelector.tsx`, `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-CANVAS-07 | Export/Import per-schema (YAML) | − | TC-CANVAS-07 | − | − | ❌ |
| AC-CANVAS-08 | "+ Add Agent" → нода мгновенно (без модалки) | ✓ | TC-NODE-01 | − | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-CANVAS-09 | "+ Add Trigger" → trigger нода мгновенно | ✓ | TC-NODE-02 | − | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-CANVAS-10 | Trigger schedule — human-readable UI | ✓ | TC-CRON-01, TC-CRON-03 | − | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-CANVAS-11 | Trigger schedule — "Advanced" raw cron toggle | ✓ | TC-CRON-02 | − | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-TRIGGER-01 | Trigger → entry agent → успех | − | TC-TRIGGER-01 | `domain/edge.go`, `delivery/http/agent_handler.go` | − | ✅ |
| AC-TRIGGER-02 | Trigger → non-entry agent → ошибка валидации | − | TC-TRIGGER-02 | `domain/edge.go` | − | ✅ |
| AC-TRIGGER-03 | Несколько triggers на один entry agent | − | TC-TRIGGER-03 | `domain/edge.go` | − | ✅ |
| AC-EDGE-01 | Клик на edge → Side Panel с конфигурацией | ✓ | − | − | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-EDGE-02 | Field Mapping: key-value pairs с add/remove | ✓ | TC-FLOW-06 | `domain/edge.go` | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-EDGE-03 | Custom Prompt: template textarea с `{{}}` | ✓ | − | `domain/edge.go` | `admin/src/pages/AgentBuilderPage.tsx` | ⚠️ |
| AC-EDGE-04 | Delete Edge удаляет edge и закрывает panel | ✓ | − | `delivery/http/agent_handler.go` | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-GATE-01 | Gate нода — Side Panel конфигурация | ✓ | TC-CANVAS-01 | `domain/edge.go` | `admin/src/components/builder/GateConfigPanel.tsx` | ✅ |
| AC-GATE-02 | Каждый condition type — своя форма | ✓ | − | `domain/edge.go` | `admin/src/components/builder/GateConfigPanel.tsx` | ✅ |
| AC-GATE-03 | Max iterations предотвращает бесконечные циклы | − | − | `service/flow/executor.go` | − | ⚠️ |
| AC-GATE-04 | On timeout/failure action выполняется | − | − | `service/flow/executor.go` | − | ⚠️ |

---

## 5. Capabilities (AC-CAP, AC-GRD-JSON, AC-GRD-LLM, AC-GRD-WH, AC-SCH, AC-ESC, AC-NOTIFY)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-CAP-01 | Memory config: cross-session, per-user, retention, max entries | ✓ | TC-CAP-01, TC-UJ-16 | `app/http_adapters_memory.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-CAP-02 | Knowledge: upload, top-k, threshold влияют на retrieval | ✓ | TC-CAP-02, TC-UJ-17 | `delivery/http/knowledge_handler.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-CAP-03 | Guardrail: output проверяется, on-failure выполняется | − | TC-CAP-03, TC-UJ-18 | `service/guardrail/pipeline.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-CAP-04 | Output Schema: enforce блокирует несоответствие | − | TC-CAP-04 | `service/guardrail/pipeline.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-CAP-05 | Escalation: condition → action выполняется | − | TC-CAP-05 | `service/recovery/executor.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-CAP-06 | Recovery: failure type → recovery action | ✓ | TC-CAP-06, TC-UJ-19 | `service/recovery/executor.go`, `domain/recovery_recipe.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-CAP-07 | Policy rule "tool_matches(delete_*) → block" | ✓ | TC-CAP-07, TC-UJ-20 | `service/policy/engine.go`, `domain/policy_rule.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-GRD-JSON-01 | Ответ LLM валидируется против JSON Schema | − | TC-GRD-JSON-01 | `service/guardrail/json_validator.go` | − | ✅ |
| AC-GRD-JSON-02 | Невалидный ответ → retry до 3 раз | − | TC-GRD-JSON-02 | `service/guardrail/pipeline.go` | − | ✅ |
| AC-GRD-JSON-03 | После 3 retry → fallback или error | − | TC-GRD-JSON-03 | `service/guardrail/pipeline.go` | − | ✅ |
| AC-GRD-JSON-04 | JSON Schema editor в UI | ✓ | TC-GRD-JSON-04 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-GRD-LLM-01 | После генерации → вызывается judge LLM | − | TC-GRD-LLM-01 | `service/guardrail/llm_judge.go` | − | ✅ |
| AC-GRD-LLM-02 | Judge промпт настраивается в UI | ✓ | TC-GRD-LLM-02 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-GRD-LLM-03 | Judge возвращает yes/no → on_failure при no | − | TC-GRD-LLM-03 | `service/guardrail/llm_judge.go` | − | ✅ |
| AC-GRD-LLM-04 | UI чётко показывает: промпт для judge, не для агента | ✓ | TC-GRD-LLM-04 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-GRD-WH-01 | POST с response payload на webhook URL | − | TC-GRD-WH-01 | `service/guardrail/webhook.go` | − | ✅ |
| AC-GRD-WH-02 | Webhook возвращает `{"pass": true/false, "reason": "..."}` | − | TC-GRD-WH-02 | `service/guardrail/webhook.go` | − | ✅ |
| AC-GRD-WH-03 | pass=false → on_failure action | − | TC-GRD-WH-03 | `service/guardrail/webhook.go` | − | ✅ |
| AC-GRD-WH-04 | timeout (10s) → on_failure | − | TC-GRD-WH-04 | `service/guardrail/webhook.go` | − | ✅ |
| AC-GRD-WH-05 | Auth: Bearer token в UI | ✓ | TC-GRD-WH-05 | `service/guardrail/webhook.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-SCH-01 | Output Schema → response_format в LLM API | − | TC-SCH-01 | `service/guardrail/pipeline.go` | − | ✅ |
| AC-SCH-02 | Output Schema и Guardrail включены одновременно | − | TC-SCH-02 | `service/guardrail/pipeline.go` | − | ✅ |
| AC-SCH-03 | Guardrail после генерации, Schema до | − | TC-SCH-03 | `service/guardrail/pipeline.go` | − | ✅ |
| AC-ESC-01 | Терминология "transfer_to_user" (не "human") | ✓ | TC-ESC-01 | `service/recovery/executor.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-ESC-02 | Условия — typed dropdown (не CEL) | ✓ | TC-ESC-02 | `domain/recovery_recipe.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-ESC-03 | confidence_below(0.7) триггерит escalation | − | TC-ESC-03 | `service/recovery/executor.go` | − | ✅ |
| AC-ESC-04 | transfer_to_user прерывает агента | − | TC-ESC-04 | `service/flow/executor.go` | − | ✅ |
| AC-NOTIFY-01 | Notify webhook → JSON payload | − | TC-NOTIFY-01 | `service/guardrail/webhook.go` | − | ✅ |
| AC-NOTIFY-02 | Auth: none, api_key, forward_headers, oauth2 | ✓ | TC-NOTIFY-02 | `service/guardrail/webhook.go`, `domain/mcp_auth.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-NOTIFY-03 | Timeout → логирование, не блокирование агента | − | TC-NOTIFY-03 | `service/guardrail/webhook.go` | − | ✅ |

---

## 6. Knowledge (AC-KB)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-KB-FMT-01 | Загрузка .pdf → успешная индексация | − | TC-KB-FMT-01 | `delivery/http/knowledge_handler.go`, `domain/knowledge.go` | − | ✅ |
| AC-KB-FMT-02 | Загрузка .docx → успешная индексация | − | TC-KB-FMT-02 | `delivery/http/knowledge_handler.go` | − | ✅ |
| AC-KB-FMT-03 | Загрузка .doc → успешная индексация | − | TC-KB-FMT-03 | `delivery/http/knowledge_handler.go` | − | ✅ |
| AC-KB-FMT-04 | Загрузка .txt, .md, .csv → успешная индексация | − | TC-KB-FMT-04 | `delivery/http/knowledge_handler.go` | − | ✅ |
| AC-KB-FMT-05 | Неподдерживаемый формат → внятная ошибка | − | TC-KB-FMT-05 | `delivery/http/knowledge_handler.go` | − | ✅ |
| AC-KB-LIST-01 | Загруженный файл появляется в списке | ✓ | TC-KB-LIST-01 | `delivery/http/knowledge_handler.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-KB-LIST-02 | Отображаются: имя, тип, размер, дата, статус | ✓ | TC-KB-LIST-02 | `infrastructure/persistence/models/knowledge.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-KB-LIST-03 | Статус: uploading → indexing → ready | ✓ | TC-KB-LIST-03 | `domain/knowledge.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-KB-LIST-04 | Удалить файл из knowledge base | ✓ | TC-KB-LIST-04 | `delivery/http/knowledge_handler.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-KB-LIST-05 | Переиндексировать файл | ✓ | TC-KB-LIST-05 | `delivery/http/knowledge_handler.go` | `admin/src/pages/AgentDrillInPage.tsx` | ⚠️ |
| AC-KB-PARAM-01 | top_k настраивается в capability config (default: 5) | ✓ | TC-KB-PARAM-01 | `domain/knowledge.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-KB-PARAM-02 | similarity_threshold в capability config (default: 0.75) | ✓ | TC-KB-PARAM-02 | `domain/knowledge.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-KB-PARAM-03 | knowledge_search tool использует значения из agent config | − | TC-KB-PARAM-03 | `infrastructure/tools/builtin_tool_store.go` | − | ✅ |

---

## 7. Memory (AC-MEM, AC-MEM-TERM, AC-MEM-RET)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-MEM-01 | Агент помнит информацию из предыдущей сессии | − | TC-MEM-01 | `app/http_adapters_memory.go` | − | ✅ |
| AC-MEM-02 | Memory schema A изолирована от schema B | − | TC-MEM-02 | `app/http_adapters_memory.go` | − | ✅ |
| AC-MEM-03 | Пользователь просматривает и очищает memory через UI | ✓ | TC-MEM-03 | `app/http_adapters_memory.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-MEM-04 | Memory в tenant storage, учитывается в quota | − | TC-MEM-04 | `service/billing/quota.go` | − | ✅ |
| AC-MEM-TERM-01 | В UI Memory capability нет "Flow" — используется "Schema" | ✓ | TC-MEM-TERM-01 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-MEM-TERM-02 | Memory hint: "per-schema, cross-session" | ✓ | TC-MEM-TERM-02 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-MEM-RET-01 | Default retention = Unlimited (не 30 дней) | ✓ | TC-MEM-RET-01 | `app/http_adapters_memory.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-MEM-RET-02 | max_entries ограничивает количество | ✓ | TC-MEM-RET-02 | `app/http_adapters_memory.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-MEM-RET-03 | При max_entries → FIFO eviction | − | TC-MEM-RET-03 | `app/http_adapters_memory.go` | − | ✅ |

---

## 8. Flows (AC-FLOW)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-FLOW-01 | Flow edge: после Agent A → Agent B запускается | − | TC-FLOW-01 | `service/flow/executor.go`, `domain/edge.go` | − | ✅ |
| AC-FLOW-02 | Transfer edge: Agent A → Agent B → завершается | − | TC-FLOW-02 | `service/flow/executor.go` | − | ✅ |
| AC-FLOW-03 | Loop edge: gate fail → возврат к агенту (max_iterations) | − | TC-FLOW-03 | `service/flow/executor.go` | − | ✅ |
| AC-FLOW-04 | Gate: auto-condition проверяет output | − | TC-FLOW-04 | `service/flow/executor.go` | − | ✅ |
| AC-FLOW-05 | Parallel: fork + gate join | − | TC-FLOW-05 | `service/flow/executor.go` | − | ✅ |
| AC-FLOW-06 | Edge config: field mapping | − | TC-FLOW-06 | `domain/edge.go`, `service/flow/executor.go` | − | ✅ |

---

## 9. Agent Lifecycle States (AC-STATE)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-STATE-01 | Каждый агент имеет явное состояние через API | − | TC-STATE-01 | `domain/agent_lifecycle.go`, `service/lifecycle/manager.go` | − | ✅ |
| AC-STATE-02 | SSE event `agent.state_changed` при переходе | − | TC-STATE-02 | `service/lifecycle/dispatcher.go`, `domain/agent_event.go` | − | ✅ |
| AC-STATE-03 | UI показывает состояние на ноде (badge/icon) | ✓ | TC-STATE-03, TC-UJ-12 | − | `admin/src/components/builder/AgentNode.tsx` | ✅ |
| AC-STATE-04 | `blocked` содержит reason для пользователя | − | TC-STATE-04 | `domain/agent_lifecycle.go` | − | ✅ |

---

## 10. Recovery (AC-REC)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-REC-01 | Degrade mode до конца сессии | − | TC-REC-01 | `service/recovery/executor.go`, `domain/recovery_recipe.go` | − | ✅ |
| AC-REC-02 | Новая сессия — полноценный набор компонентов | − | TC-REC-02 | `service/recovery/executor.go` | − | ✅ |
| AC-REC-03 | 1 авто recovery attempt перед escalation | − | TC-REC-01, TC-REC-03 | `service/recovery/executor.go` | − | ✅ |
| AC-REC-04 | Recovery events в Agent Inspection | − | TC-REC-04 | `service/recovery/executor.go` | `admin/src/pages/InspectPage.tsx` | ✅ |

---

## 11. Event Schema (AC-EVT)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-EVT-01 | Все SSE events содержат `schema_version` | − | TC-EVT-01 | `delivery/http/event_converter.go`, `domain/agent_event.go` | − | ✅ |
| AC-EVT-02 | Неизвестные event types безопасно игнорируются | − | TC-EVT-02 | − | `admin/src/pages/AgentBuilderPage.tsx` | ⚠️ |
| AC-EVT-03 | Документирован event contract | − | TC-EVT-03 | − | − | ❌ |

---

## 12. MCP Auth (AC-AUTH)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-AUTH-01 | MCP server config принимает auth section | ✓ | TC-AUTH-01, TC-UJ-39 | `service/mcp/auth_provider.go`, `domain/mcp_auth.go` | − | ✅ |
| AC-AUTH-02 | forward_headers прокидываются в MCP calls | − | TC-AUTH-02 | `service/mcp/auth_provider.go` | − | ✅ |
| AC-AUTH-03 | API key из env variable, не plain text | − | TC-AUTH-03 | `service/mcp/auth_provider.go`, `domain/mcp_auth.go` | − | ✅ |

---

## 13. Agent Policies (AC-POL)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-POL-01 | Conditions — typed dropdown (не free text) | ✓ | TC-POL-01 | `domain/policy_rule.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-POL-02 | inject_header → custom header в MCP tool requests | ✓ | TC-POL-01, TC-UJ-21 | `service/policy/engine.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-POL-03 | Webhook auth — 4 auth types | ✓ | TC-POL-02, TC-UJ-22 | `service/policy/engine.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-POL-04 | block action блокирует tool execution с сообщением | − | TC-POL-02 | `service/policy/engine.go` | − | ✅ |

---

## 14. Tool Architecture (AC-TOOL)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-TOOL-01 | Tier 1 tools доступны без настройки | − | TC-TOOL-01 | `infrastructure/tools/builtin_tool_store.go` | − | ✅ |
| AC-TOOL-02 | Tier 2 tools авто-добавляются с capability | − | TC-TOOL-02 | `service/capability/injector.go` | − | ✅ |
| AC-TOOL-03 | Tier 3 заблокированы в Cloud (ошибка) | − | TC-TOOL-03, TC-CLOUD-05 | `service/cloud/sandbox.go`, `service/capability/tier_enforcer.go` | − | ✅ |
| AC-TOOL-04 | Tier 4 tools через MCP configuration | − | TC-TOOL-04 | `infrastructure/tools/builtin_tool_store.go` | − | ✅ |
| AC-TOOL-05 | web_search ТОЛЬКО через MCP (не нативный) | − | TC-TOOL-05 | `infrastructure/websearch/tavily_provider.go` | − | ✅ |

---

## 15. Entity Relationships (AC-ENT)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-ENT-01 | Агент — глобальная сущность | ✓ | TC-ENT-01 | `app/http_adapters.go` | `admin/src/pages/AgentBuilderPage.tsx` | ✅ |
| AC-ENT-02 | Клик на агента в canvas → навигация на страницу | ✓ | TC-ENT-02 | − | `admin/src/pages/AgentBuilderPage.tsx`, `admin/src/App.tsx` | ✅ |
| AC-ENT-03 | Страница агента — "Используется в: [список схем]" | ✓ | TC-ENT-03 | `app/http_adapters.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-ENT-04 | Кнопка "← Вернуться на Canvas" | ✓ | TC-ENT-04 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-ENT-05 | MCP серверы — глобальные с кросс-ссылками | ✓ | TC-ENT-05 | `app/http_adapters.go` | `admin/src/App.tsx` | ✅ |

---

## 16. Persistent Agent Lifecycle (AC-LIFE) — BACKEND-DEFERRED

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-LIFE-01 | Spawn sub-agent: task → уничтожается | 🔄 | TC-LIFE-01 | `domain/agent_lifecycle.go`, `service/lifecycle/manager.go` | − | 🔄 |
| AC-LIFE-02 | Persistent sub-agent: задача → контекст сохраняется | 🔄 | TC-LIFE-02 | `domain/agent_lifecycle.go`, `service/lifecycle/manager.go` | − | 🔄 |
| AC-LIFE-03 | Parent reset не влияет на persistent child | 🔄 | TC-LIFE-03 | `service/lifecycle/manager.go` | − | 🔄 |
| AC-LIFE-04 | Persistent agent: auto-compact при переполнении | 🔄 | TC-LIFE-04 | `service/lifecycle/manager.go` | − | 🔄 |

---

## 17. Inter-Agent Communication (AC-COMM) — BACKEND-DEFERRED

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-COMM-01 | Parent отправляет task persistent child | 🔄 | TC-COMM-01 | `service/lifecycle/dispatcher.go`, `domain/task_packet.go` | − | 🔄 |
| AC-COMM-02 | Persistent child возвращает результат через event | 🔄 | TC-COMM-02 | `service/lifecycle/dispatcher.go` | − | 🔄 |
| AC-COMM-03 | Parent получает результат, продолжает работу | 🔄 | TC-COMM-03 | `service/lifecycle/dispatcher.go` | − | 🔄 |

---

## 18. Agent Resilience (AC-RESIL) — BACKEND-DEFERRED

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-RESIL-01 | Sub-agent heartbeat каждые N секунд | 🔄 | TC-RESIL-01 | `service/resilience/heartbeat.go` | − | 🔄 |
| AC-RESIL-02 | Parent получает event при stuck sub-agent | 🔄 | TC-RESIL-01 | `service/resilience/heartbeat.go` | − | 🔄 |
| AC-RESIL-03 | Spawn agent stuck → kill + re-spawn | 🔄 | TC-RESIL-02 | `service/resilience/heartbeat.go` | − | 🔄 |
| AC-RESIL-04 | Persistent agent stuck → interrupt → kill → escalate | 🔄 | TC-RESIL-03 | `service/resilience/heartbeat.go` | − | 🔄 |
| AC-RESIL-05 | MCP tool call timeout → structured error | 🔄 | TC-RESIL-04 | `infrastructure/mcp/client.go` | − | 🔄 |
| AC-RESIL-06 | tool_call_timeout настраивается per-agent/per-MCP | 🔄 | TC-RESIL-04 | `infrastructure/mcp/client.go` | − | 🔄 |
| AC-RESIL-07 | Task dispatch timeout → status `timeout`, parent event | 🔄 | TC-RESIL-05 | `service/resilience/dead_letter.go` | − | 🔄 |
| AC-RESIL-08 | Dead letter tasks в Inspect view | 🔄 | TC-RESIL-05, TC-INSPECT-05 | `service/resilience/dead_letter.go` | `admin/src/pages/InspectPage.tsx` | 🔄 |
| AC-RESIL-09 | Circuit breaker открывается после 3 failures | 🔄 | TC-RESIL-06 | `service/resilience/circuit_breaker.go` | − | 🔄 |
| AC-RESIL-10 | Circuit open → MCP tools → tool_unavailable | 🔄 | TC-RESIL-06 | `infrastructure/tools/circuit_breaker_tool_wrapper.go` | − | 🔄 |
| AC-RESIL-11 | Circuit half-open через reset interval | 🔄 | TC-RESIL-07 | `service/resilience/circuit_breaker.go` | − | 🔄 |
| AC-RESIL-12 | Circuit state в Admin UI | 🔄 | TC-RESIL-08 | `service/resilience/circuit_breaker.go` | − | 🔄 |

---

## 19. Testing Infrastructure (AC-TEST)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-TEST-01 | Test Flow tab — HTTP Headers editor | ✓ | TC-TEST-01, TC-TESTFLOW-01 | − | `admin/src/components/TestFlowTab.tsx` | ✅ |
| AC-TEST-02 | Headers из Test Flow → MCP tool calls | − | TC-TEST-02, TC-TESTFLOW-02 | `app/http_adapters.go` | `admin/src/components/TestFlowTab.tsx` | ✅ |
| AC-TEST-03 | Trigger config — Custom Headers | ✓ | TC-TEST-03 | `delivery/http/agent_handler.go` | `admin/src/components/builder/TriggerNode.tsx` | ✅ |
| AC-TEST-04 | Chat API — optional headers field (DEFERRED) | 🔄 | TC-TEST-04 | − | − | 🔄 |

---

## 20. Widget (AC-WID)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-WID-01 | Widget script загружается → chat bubble | ✓ | TC-WID-01, TC-UJ-30, TC-WIDGET-02 | `delivery/http/widget_handler.go` | `admin/src/pages/WidgetConfigPage.tsx` | ✅ |
| AC-WID-02 | Пользователь пишет → агент отвечает через SSE | − | TC-WID-02 | `delivery/http/widget_handler.go` | − | ✅ |
| AC-WID-03 | Widget: primary color, position, welcome message | ✓ | TC-WID-03, TC-UJ-31, TC-WIDGET-01 | `domain/widget.go` | `admin/src/pages/WidgetConfigPage.tsx` | ✅ |
| AC-WID-04 | Widget ID привязан к tenant + schema | − | TC-WID-04 | `domain/widget.go`, `domain/billing.go` | − | ✅ |
| AC-WID-05 | Domain whitelist (CORS) | − | TC-WIDGET-03 | `delivery/http/widget_handler.go` | `admin/src/pages/WidgetConfigPage.tsx` | ✅ |
| AC-WID-06 | Self-hosted widget script раздаётся Engine'ом | − | − | `delivery/http/widget_handler.go` | − | ⚠️ |
| AC-WID-07 | Live preview в Admin | ✓ | TC-WIDGET-01 | − | `admin/src/pages/WidgetConfigPage.tsx` | ✅ |

---

## 21. MCP Catalog (AC-MCP)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-MCP-01 | Built-in catalog: 10-15 проверенных серверов | ✓ | TC-MCP-01, TC-MCP-04 | `service/mcp/catalog.go` | − | ✅ |
| AC-MCP-02 | "Add from Catalog" с поиском и category filter | ✓ | TC-MCP-02, TC-MCP-05..07 | `delivery/http/catalog_handler.go` | − | ✅ |
| AC-MCP-03 | "Add Custom Server" (transport, URL/command, env vars, auth) | ✓ | TC-MCP-03, TC-MCP-08 | `delivery/http/catalog_handler.go` | − | ✅ |
| AC-MCP-04 | MCP серверы глобальные (не per-agent, не per-schema) | ✓ | TC-MCP-09 | `app/http_adapters.go` | − | ✅ |
| AC-MCP-05 | Агент подключает MCP серверы из глобального списка | − | TC-MCP-10 | `app/http_adapters.go` | − | ✅ |
| AC-MCP-06 | MCP server detail — "Used by agents: [список]" | ✓ | − | `delivery/http/catalog_handler.go` | − | ⚠️ |
| AC-MCP-07 | Клик на agent в "Used by" → навигация на agent page | ✓ | − | − | `admin/src/App.tsx` | ⚠️ |

---

## 22. Admin UI (AC-UI)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-UI-01 | Drill-in: клик на ноду → полноэкранная конфигурация с breadcrumb | ✓ | TC-UI-01, TC-UJ-04, TC-UJ-13..15 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-UI-02 | Capability blocks: [+ Add] → dropdown → inline config | ✓ | TC-UI-02 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-UI-03 | Bottom panel: resizable drag handle, два таба | ✓ | TC-UI-03, TC-UJ-09, TC-UJ-10 | − | `admin/src/components/BottomPanel.tsx` | ✅ |
| AC-UI-04 | Inspect: session history с reasoning, tool calls, timing | ✓ | TC-UI-04, TC-UJ-25..29 | − | `admin/src/pages/InspectPage.tsx` | ✅ |
| AC-UI-05 | Нет отдельного /admin/agents (drill-in из canvas) | ✓ | TC-UI-05, TC-UJ-06 | − | `admin/src/App.tsx` | ✅ |
| AC-UI-06 | Model Parameters per-agent (Temperature, Top P, Max Tokens, Stop Sequences) | ✓ | TC-PARAM-01..04 | `delivery/http/agent_handler.go` | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |
| AC-UI-07 | Tools section — по тирам с реальными именами | ✓ | TC-PARAM-05 | − | `admin/src/pages/AgentDrillInPage.tsx` | ✅ |

---

## 23. Bottom Panel & Test Flow (AC-PANEL, AC-TESTFLOW)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-PANEL-01 | Drag handle resize (min 150px, max 70% viewport) | ✓ | TC-PANEL-01 | − | `admin/src/components/BottomPanel.tsx` | ✅ |
| AC-PANEL-02 | Collapse/expand toggle, collapsed = thin bar | ✓ | TC-PANEL-02 | − | `admin/src/components/BottomPanel.tsx` | ✅ |
| AC-PANEL-03 | State persists (localStorage) across navigation | ✓ | TC-PANEL-03 | − | `admin/src/hooks/useBottomPanel.tsx` | ✅ |
| AC-PANEL-04 | Panel на ВСЕХ admin pages | ✓ | TC-PANEL-04 | − | `admin/src/components/Layout.tsx` | ✅ |
| AC-TESTFLOW-01 | Test Flow — HTTP Headers key-value editor | ✓ | TC-TESTFLOW-01 | − | `admin/src/components/TestFlowTab.tsx` | ✅ |
| AC-TESTFLOW-02 | Headers → MCP tool calls через forward_headers | − | TC-TESTFLOW-02 | `app/http_adapters.go` | `admin/src/components/TestFlowTab.tsx` | ✅ |
| AC-TESTFLOW-03 | SSE streaming с tool calls и reasoning inline | ✓ | TC-TESTFLOW-03 | `delivery/http/chat_handler.go` | `admin/src/components/TestFlowTab.tsx` | ✅ |
| AC-TESTFLOW-04 | "View in Inspect" → InspectPage | ✓ | − | − | `admin/src/components/TestFlowTab.tsx` | ⚠️ |

---

## 24. Inspect Page (AC-INSPECT)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-INSPECT-01 | Session list — paginated table (20/page) | ✓ | TC-INSPECT-01 | `delivery/http/agent_handler.go` | `admin/src/pages/InspectPage.tsx` | ✅ |
| AC-INSPECT-02 | Search по session ID и agent name | ✓ | TC-INSPECT-02 | `delivery/http/agent_handler.go` | `admin/src/pages/InspectPage.tsx` | ✅ |
| AC-INSPECT-03 | Filter по status (multi-select) | ✓ | TC-INSPECT-03 | `delivery/http/agent_handler.go` | `admin/src/pages/InspectPage.tsx` | ✅ |
| AC-INSPECT-04 | Session detail — timeline с unified icons и timing | ✓ | TC-INSPECT-04 | `delivery/http/event_converter.go` | `admin/src/pages/InspectPage.tsx` | ✅ |
| AC-INSPECT-05 | Dead letter tasks с ⏰ icon и причиной | 🔄 | TC-INSPECT-05 | `service/resilience/dead_letter.go` | `admin/src/pages/InspectPage.tsx` | 🔄 |
| AC-INSPECT-06 | Running sessions auto-refresh через SSE | ✓ | TC-INSPECT-06 | `delivery/http/chat_handler.go` | `admin/src/pages/InspectPage.tsx` | ✅ |

---

## 25. Landing Page (AC-LAND)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-LAND-01 | Landing загружается, hero section виден | − | TC-LAND-01 | − | `cloud-web/src/pages/Landing.tsx` | ⚠️ |
| AC-LAND-02 | "Try free" → registration | − | TC-LAND-02 | − | `cloud-web/src/pages/Landing.tsx` | ⚠️ |
| AC-LAND-03 | Pricing section — актуальные тарифы | − | TC-LAND-03 | − | `cloud-web/src/pages/Landing.tsx` | ⚠️ |
| AC-LAND-04 | "Self-host" → docs с docker run | − | TC-LAND-04 | − | `cloud-web/src/pages/Landing.tsx` | ⚠️ |

---

## 26. Open Source (AC-OSS)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-OSS-01 | LICENSE файл в каждом repo | − | TC-OSS-01 | `engine/LICENSE` | − | ⚠️ |
| AC-OSS-02 | README с описанием, quick start, license badge | − | TC-OSS-02 | − | − | ⚠️ |
| AC-OSS-03 | Git history чистая (нет secrets) | − | TC-OSS-03 | − | − | ⚠️ |
| AC-OSS-04 | git clone + docker build + docker run → работает | − | TC-OSS-04 | `engine/Dockerfile` | − | ⚠️ |
| AC-OSS-05 | Лицензия отправлена партнёру для верификации | − | TC-OSS-05 | − | − | ❌ |
| AC-OSS-06 | CI/CD auto-обновляет Change Date в LICENSE | − | TC-OSS-06 | − | − | ❌ |

---

## 27. Documentation (AC-DOCS)

| AC-ID | Description | Proto | TC | Backend Code | Frontend Code | Status |
|-------|-------------|:-----:|:--:|--------------|---------------|:------:|
| AC-DOCS-01 | Docs page — sidebar с навигацией | − | − | − | `cloud-web/src/pages/Docs.tsx` | ⚠️ |
| AC-DOCS-02 | Getting Started: docker run → admin → first agent → chat | − | − | − | `cloud-web/src/pages/Docs.tsx` | ⚠️ |
| AC-DOCS-03 | API Reference — все REST endpoints с примерами | − | − | − | `cloud-web/src/pages/Docs.tsx` | ⚠️ |

---

## Summary

| Category | Total ACs | ✅ DONE | ⚠️ PARTIAL | ❌ GAP | 🔄 DEFERRED |
|----------|:---------:|:-------:|:----------:|:------:|:-----------:|
| Cloud (AC-CLOUD) | 6 | 6 | 0 | 0 | 0 |
| Pricing (AC-PRICE) | 11 | 8 | 3 | 0 | 0 |
| Brewery UX (AC-UX) | 19 | 13 | 2 | 0 | 4 |
| Canvas (AC-CANVAS, TRIGGER, EDGE, GATE) | 21 | 15 | 4 | 1 | 0 |
| Capabilities (AC-CAP, GRD, SCH, ESC, NOTIFY) | 29 | 29 | 0 | 0 | 0 |
| Knowledge (AC-KB) | 13 | 12 | 1 | 0 | 0 |
| Memory (AC-MEM, TERM, RET) | 9 | 9 | 0 | 0 | 0 |
| Flows (AC-FLOW) | 6 | 6 | 0 | 0 | 0 |
| Lifecycle States (AC-STATE) | 4 | 4 | 0 | 0 | 0 |
| Recovery (AC-REC) | 4 | 4 | 0 | 0 | 0 |
| Event Schema (AC-EVT) | 3 | 1 | 1 | 1 | 0 |
| MCP Auth (AC-AUTH) | 3 | 3 | 0 | 0 | 0 |
| Agent Policies (AC-POL) | 4 | 4 | 0 | 0 | 0 |
| Tool Architecture (AC-TOOL) | 5 | 5 | 0 | 0 | 0 |
| Entity Relationships (AC-ENT) | 5 | 5 | 0 | 0 | 0 |
| Persistent Lifecycle (AC-LIFE) | 4 | 0 | 0 | 0 | 4 |
| Inter-Agent Comm (AC-COMM) | 3 | 0 | 0 | 0 | 3 |
| Resilience (AC-RESIL) | 12 | 0 | 0 | 0 | 12 |
| Testing Infra (AC-TEST) | 4 | 3 | 0 | 0 | 1 |
| Widget (AC-WID) | 7 | 5 | 1 | 0 | 0 |
| MCP Catalog (AC-MCP) | 7 | 5 | 2 | 0 | 0 |
| Admin UI (AC-UI) | 7 | 7 | 0 | 0 | 0 |
| Bottom Panel / Test Flow (AC-PANEL, TESTFLOW) | 8 | 7 | 1 | 0 | 0 |
| Inspect Page (AC-INSPECT) | 6 | 5 | 0 | 0 | 1 |
| Landing (AC-LAND) | 4 | 0 | 4 | 0 | 0 |
| Open Source (AC-OSS) | 6 | 0 | 4 | 2 | 0 |
| Documentation (AC-DOCS) | 3 | 0 | 3 | 0 | 0 |
| **TOTAL** | **203** | **155** | **26** | **4** | **25** |

### Итог

- **✅ DONE:** 155 / 178 non-deferred (87%)
- **⚠️ PARTIAL:** 26 / 178 non-deferred (15%) — реализованы, но нет TC или frontend/backend wire-up не подтверждён
- **❌ GAP:** 4 — требуют реализации: AC-CANVAS-07 (schema export/import), AC-EVT-03 (event contract docs), AC-OSS-05 (лицензия партнёру), AC-OSS-06 (CI/CD Change Date)
- **🔄 DEFERRED:** 25 — BACKEND-DEFERRED согласно PRD: AC-LIFE-01..04, AC-COMM-01..03, AC-RESIL-01..12, AC-TEST-04, AC-INSPECT-05, AC-UX-13..16

### Критические GAP (требуют работы)

1. **AC-CANVAS-07** — Export/Import schema (YAML): нет ни backend, ни frontend кода
2. **AC-EVT-03** — Документирован event contract: нет документации SSE events
3. **AC-OSS-05** — Лицензия не отправлена партнёру (внешнее действие)
4. **AC-OSS-06** — CI/CD workflow для BSL Change Date не реализован

### PARTIAL — требуют проверки

- **AC-PRICE-09, AC-PRICE-10** — Warning/block UX при 80%/100% quota: `UsageDashboard.tsx` есть, но wire-up с backend quota signals не подтверждён
- **AC-LAND-01..04** — Landing page: файл существует, но V2 messaging (hero text, pricing) требует верификации
- **AC-OSS-01..04** — Open source readiness: структурно готово, но полный checklist (clean history, README badges) не проверен
- **AC-DOCS-01..03** — Docs page: файл существует, но V2 content (API reference, self-hosting guide) требует верификации
- **AC-UX-04..06** — Canvas анимации при assistant actions: SSE events есть, frontend анимации — PARTIAL (prototype only)
- **AC-MCP-06..07** — "Used by agents" cross-reference в MCP page: catalog handler есть, UI cross-navigation не подтверждена
