# Bottom Panel — Acceptance Criteria

> Source: ACs embedded in `docs/domains/bottom-panel/requirements.md` (from `docs/v2-requirements/01-prd.md` §7.3)

---

See `requirements.md` in this domain for the full acceptance criteria list.

## Summary of ACs

**Bottom Panel:**
- AC-PANEL-01: Drag handle resize (min 150px, max 70% viewport)
- AC-PANEL-02: Collapse/expand toggle, collapsed = thin bar с tab labels
- AC-PANEL-03: Panel state (height, tab, open/closed) persists across page navigation (localStorage)
- AC-PANEL-04: Bottom panel доступен на ВСЕХ admin pages, не только Canvas

**Test Flow:**
- AC-TESTFLOW-01: Test Flow tab имеет HTTP Headers key-value editor (add/remove/JSON import)
- AC-TESTFLOW-02: Headers прокидываются в agent → MCP tool calls через forward_headers
- AC-TESTFLOW-03: SSE response streaming показывает tool calls и reasoning inline
- AC-TESTFLOW-04: "View in Inspect" link переходит на InspectPage с текущей сессией

**AI Assistant — Panel States:**
- AC-PANEL-05: Collapsed bar — клик по tab label разворачивает панель и активирует соответствующий таб
- AC-PANEL-06: AI Assistant показывает "no model configured" state с actionable ссылкой когда модель не назначена
- AC-PANEL-07: AI Assistant показывает Clear button когда в чате есть сообщения; Clear очищает историю чата

**Context Usage Bar:**
- AC-PANEL-08: Context Usage Bar отображает max_context_size агента между messages и input (формат: "— / 16K tokens")
- AC-PANEL-09: Context Usage Bar заполняется после SSE ответа (total_tokens из done event), цвет: green 0-60%, yellow 60-85%, red 85-100%
- AC-PANEL-10: Context Usage Bar скрыт когда max_context_size не определён (null)

**Test Flow — Session Management:**
- AC-TESTFLOW-05: Session dropdown показывает список recent sessions для текущего агента (production mode only)
- AC-TESTFLOW-06: Переключение сессии загружает историю сообщений через loadSession
- AC-TESTFLOW-07: "New Session" создаёт чистую сессию, очищает чат
- AC-TESTFLOW-08: Delete session удаляет сессию с confirmation, обновляет список
- AC-TESTFLOW-09: Смена агента сбрасывает и перезагружает список сессий

**Chat Persistence:**
- AC-CHAT-01: Session ID сохраняется в localStorage, история восстанавливается после reload страницы
- AC-CHAT-02: Reset session очищает localStorage и историю чата
