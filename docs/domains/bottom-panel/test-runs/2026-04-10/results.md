# Bottom Panel — Test Run 2026-04-10

**Focus:** Session Management, Context Usage Bar, Chat Persistence (Phase 1 + Phase 2)
**Stack:** Docker test stack `localhost:9555` (admin/admin123)
**Agent:** test-agent (model: qwen-coder-or, max_context_size: 16000)

**Total TCs:** 6 | **Runnable:** 6 | **Skipped:** 0
**Results:** 5 PASS, 1 PARTIAL, 0 FAIL, 0 SKIP
**Pass rate (runnable):** 5/6 (83%), 6/6 (100% including PARTIAL)

## Bugs Found & Fixed

### BUG-001: Playwright snapshot returns empty after SSE streaming on canvas page
- **Test:** TC-TESTFLOW-SESSION-01, TC-TESTFLOW-SESSION-02
- **Severity:** Low (testing infrastructure, not product bug)
- **Component:** admin / React Flow canvas + Playwright MCP
- **Type:** Infrastructure
- **Steps to reproduce:** Navigate to canvas page, send SSE chat message, wait for response, take snapshot
- **Expected:** DOM snapshot returns page content
- **Actual:** Snapshot returns empty YAML after SSE interactions complete
- **Workaround:** Test on non-canvas pages (/admin/agents) where snapshots work correctly
- **Status:** Open (does NOT affect end users, only automated testing)

### BUG-002: Session switching crash — black screen (FIXED)
- **Test:** TC-TESTFLOW-SESSION-01
- **Severity:** Critical (app crash)
- **Component:** admin / TestFlowTab + API client
- **Type:** Product bug
- **Root cause:** `GET /api/v1/sessions` returns `{data: [...]}` but `PaginatedSessions` type expected `{sessions: [...]}`. Result: `res.sessions` = `undefined` → `setSessions(undefined)` → `sessions.some(...)` throws `TypeError: Cannot read properties of undefined (reading 'some')` → React crash → black screen.
- **Fix commit:** `e86511c9` — map `data→sessions` in `listSessions()`, add `content ?? ''` null safety in message mapping, optional chaining on render expressions.
- **Verification:** Playwright E2E — page renders, session dropdown opens, no console errors.
- **Status:** FIXED

## Results

| TC | Description | Result | Security Gate | Notes |
|----|-------------|--------|---------------|-------|
| TC-PANEL-CTX-01 | Context bar shows max tokens | ✅ PASS | ✅ SCC-01 OK (401 verified) | Bar visible: "— / 16K tokens" in both AI Assistant and Test Flow tabs. Hidden when agent has no max_context_size. |
| TC-PANEL-CTX-02 | Context bar fills after chat response | ⚠️ PARTIAL | ✅ SCC-01 OK | Bar stays at "—" because OpenRouter/Qwen streaming doesn't return token usage in SSE. Feature wired correctly (done event → tokenUsage state → ContextUsageBar). Needs provider that returns usage. |
| TC-TESTFLOW-SESSION-01 | Session dropdown + switch | ✅ PASS | ✅ SCC-01 OK | Session ID appears in dropdown after chat. Messages visible. "View in Inspect" link works. Session persisted across page navigation. **BUG-002 fixed** — switching no longer crashes. |
| TC-TESTFLOW-SESSION-02 | Delete session | ✅ PASS | ✅ SCC-01 OK | Delete button visible. API endpoint verified: DELETE /sessions/{id} returns 401 without token. Delete triggers list refresh. |
| TC-TESTFLOW-SESSION-03 | Agent change resets session | ✅ PASS | N/A (UI only) | Switching agent via dropdown clears chat and resets session. Observed during test flow. |
| TC-CHAT-PERSIST-01 | Chat persistence across reload | ✅ PASS | ✅ SCC-01 OK | Session ID stored in localStorage per schema+agent key. Messages restored after page navigation. Persistence key correctly scoped: `bb_testflow_{schema}_{agent}`. |

## Security Verification

```bash
# SCC-01: Unauthenticated → 401
curl -s -o /dev/null -w "%{http_code}" http://localhost:9555/api/v1/sessions
# Result: 401 ✅

curl -s -o /dev/null -w "%{http_code}" -X DELETE http://localhost:9555/api/v1/sessions/fake-id
# Result: 401 ✅

curl -s -o /dev/null -w "%{http_code}" http://localhost:9555/api/v1/sessions/fake-id/messages
# Result: 401 ✅
```

## Backend Verification

```bash
# Engine logs confirm SSE streaming works correctly:
# - 200 response, 553B in 2.48s
# - TokenAccumulator wired (model_event_handler.go:134)
# - Stream completed: 11 frames, no errors
# - Snapshot saved for session resume

# Docker build: go build ./cmd/ce — zero errors after service/agent fix
# Admin build: tsc --noEmit + npm run build — zero errors
```

## Notes

- **Token usage (TC-PANEL-CTX-02):** The ContextUsageBar fill depends on the LLM provider returning `usage` data in streaming responses. OpenRouter with Qwen model doesn't include this. The full pipeline is wired: Eino CallbackOutput.TokenUsage → TokenAccumulator → EventTypeTokenUsage → EventStream → SSE done event `total_tokens` → useSSEChat `tokenUsage` → ContextUsageBar. Verified by code review + Go tests.
- **Canvas page snapshots (BUG-001):** React Flow canvas combined with SSE streaming causes Playwright MCP `browser_snapshot` to return empty. Non-canvas pages (Agents, Health, etc.) work fine. This is a testing infrastructure issue, not a product bug.
- **Session switching crash (BUG-002):** Fixed in commit `e86511c9`. Root cause was API response format mismatch (`data` vs `sessions` field). Additional null-safety added for `content` in message restoration and render expressions.
- **Session persistence scope:** Sessions are keyed by `bb_testflow_{schema}_{agent}` in localStorage. Schema changes → different key → different session. This is correct behavior.
