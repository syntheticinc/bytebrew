# Bottom Panel — Test Cases

> Rewritten: 2026-04-09. E2E format per `docs/testing/e2e-test-standard.md`.
> **Philosophy:** каждый тест проверяет полный путь — действие в UI → состояние панели → backend verification.
> **Polling strategy:** `mcp__playwright__browser_wait_for` — poll every 2s up to 30s для всех async операций.
> **Self-contained:** каждый тест создаёт и удаляет свои данные. Нет зависимостей между тестами.
> **Source:** `docs/v2-requirements/03-test-cases.md` §42, §43, §37

---

## 42. Bottom Panel Behavior

### TC-PANEL-01: Drag handle resize

**AC:** AC-PANEL-01
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555` (Docker test stack)
- TOKEN obtained:
  ```bash
  TOKEN=$(curl -s -X POST http://localhost:9555/api/v1/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
  ```
- Bottom panel видим

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: bottom panel загружен
3. `mcp__playwright__browser_snapshot` → запомнить исходную высоту панели
4. `mcp__playwright__browser_drag` → drag handle вверх (увеличить панель)
5. `mcp__playwright__browser_snapshot` → проверить: панель стала выше
6. `mcp__playwright__browser_drag` → drag handle вниз (уменьшить панель)
7. `mcp__playwright__browser_snapshot` → проверить: панель стала меньше
8. `mcp__playwright__browser_drag` → drag handle максимально вниз (попытка меньше min height)
9. `mcp__playwright__browser_snapshot` → проверить: min height 150px соблюдена
10. `mcp__playwright__browser_drag` → drag handle максимально вверх (попытка больше max)
11. `mcp__playwright__browser_snapshot` → проверить: max height 70% viewport соблюден

**Backend Verification:**
```bash
# Нет API verification — чисто UI тест
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Панель resizes плавно с drag
- Min height enforced: 150px (нельзя уменьшить)
- Max height enforced: 70% viewport (нельзя увеличить)
- Cursor меняется на `ns-resize` при hover на handle

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Быстрое перетаскивание → панель не прыгает, анимация плавная
- Drag handle на touch-экране → работает корректно

---

### TC-PANEL-02: Collapse/expand toggle

**AC:** AC-PANEL-02
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- Bottom panel открыт

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: bottom panel загружен
3. `mcp__playwright__browser_snapshot` → запомнить текущую высоту
4. `mcp__playwright__browser_click` → collapse toggle (▼)
5. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель свернулась (~40px)
6. `mcp__playwright__browser_snapshot` → проверить: thin bar с tab labels "AI Assistant | Test Flow"
7. `mcp__playwright__browser_click` → expand toggle (▲)
8. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель раскрылась
9. `mcp__playwright__browser_snapshot` → проверить: высота вернулась к предыдущей

**Backend Verification:**
```bash
# Нет API verification — чисто UI тест
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Collapsed: thin bar (~40px) с "AI Assistant | Test Flow" tab labels видимы
- Клик на tab label пока свернуто → раскрывается на этот таб
- Предыдущая высота восстанавливается при expand

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Collapse при min height (150px) → thin bar показывается корректно
- Double-click на toggle → не мигает

---

### TC-PANEL-03: State persistence across navigation

**AC:** AC-PANEL-03
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: bottom panel загружен
3. `mcp__playwright__browser_drag` → resize panel до custom height (~300px)
4. `mcp__playwright__browser_click` → вкладка Test Flow
5. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/agents`
6. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: страница загружена
7. `mcp__playwright__browser_snapshot` → проверить: panel height та же (~300px), активен Test Flow таб
8. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/mcp`
9. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: страница загружена
10. `mcp__playwright__browser_snapshot` → проверить: state сохранён
11. `mcp__playwright__browser_press_key` → `F5` (browser refresh)
12. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: страница перезагружена
13. `mcp__playwright__browser_snapshot` → проверить: panel state восстановлен из localStorage

**Backend Verification:**
```bash
# Нет API verification — state хранится в localStorage
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Panel height, active tab, open/closed state persist across navigation
- State хранится в localStorage
- Browser refresh восстанавливает state из localStorage

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
# Опционально очистить localStorage: mcp__playwright__browser_evaluate → localStorage.clear()
```

**Negative / Edge Cases:**
- localStorage заблокирован (private mode) → panel работает с defaults, нет ошибки
- State из старой версии → graceful fallback к defaults

---

### TC-PANEL-04: Panel on all admin pages

**AC:** AC-PANEL-04
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_snapshot` → bottom panel присутствует (Canvas)
3. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/agents`
4. `mcp__playwright__browser_snapshot` → bottom panel присутствует (Agents)
5. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/mcp`
6. `mcp__playwright__browser_snapshot` → bottom panel присутствует (MCP)
7. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/models`
8. `mcp__playwright__browser_snapshot` → bottom panel присутствует (Models)
9. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/triggers`
10. `mcp__playwright__browser_snapshot` → bottom panel присутствует (Triggers)
11. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/settings`
12. `mcp__playwright__browser_snapshot` → bottom panel присутствует (Settings)
13. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/inspect`
14. `mcp__playwright__browser_snapshot` → bottom panel присутствует (Inspect)

**Backend Verification:**
```bash
# Нет API verification — чисто UI тест
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Panel присутствует на всех страницах (кроме Login)
- На Canvas: panel ниже canvas area
- На других страницах: panel ниже page content
- Panel — один компонент (не ремонтируется при навигации)

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Login page (`/admin/login`) → NO bottom panel
- 404 страница → panel не показывается или показывается gracefully

---

### TC-PANEL-05: Клик по tab label в collapsed bar разворачивает нужный таб

**AC:** AC-PANEL-05
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- Bottom panel свёрнут (collapsed = thin bar)

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: bottom panel загружен
3. `mcp__playwright__browser_click` → collapse toggle (▼) чтобы свернуть панель
4. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель свернулась
5. `mcp__playwright__browser_snapshot` → thin bar виден с labels
6. `mcp__playwright__browser_click` → label "Test Flow" в collapsed bar
7. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель раскрылась
8. `mcp__playwright__browser_snapshot` → проверить: активен таб Test Flow
9. `mcp__playwright__browser_click` → collapse toggle (▼)
10. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель свернулась
11. `mcp__playwright__browser_click` → label "AI Assistant" в collapsed bar
12. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель раскрылась
13. `mcp__playwright__browser_snapshot` → проверить: активен таб AI Assistant

**Backend Verification:**
```bash
# Нет API verification — чисто UI тест
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Панель разворачивается до предыдущей высоты
- Активной становится вкладка соответствующая нажатому label
- `expect(page.locator('[data-tab="testflow"]')).toHaveClass(/active/)` (Test Flow case)

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Клик на уже активный таб в collapsed bar → панель разворачивается, тот же таб активен
- Быстрые клики на разные labels → корректный таб активен в финале

---

### TC-PANEL-06: AI Assistant no-model state показывает actionable link

**AC:** AC-PANEL-06
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- builder-assistant существует но модель не назначена:
  ```bash
  curl -s -X PATCH http://localhost:9555/api/v1/agents/builder-assistant \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"model_id": null}'
  ```

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: bottom panel загружен
3. `mcp__playwright__browser_click` → таб "AI Assistant" (если не активен)
4. `mcp__playwright__browser_snapshot` → проверить содержимое панели
5. Проверить: сообщение об отсутствии модели
6. `mcp__playwright__browser_click` → ссылка/кнопка для перехода к настройке
7. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: URL изменился на drill-in

**Backend Verification:**
```bash
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:9555/api/v1/agents/builder-assistant
# Expected: model_id = null
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)
- Tenant isolation: cross-schema → 403/404 (see TC-CLOUD-02 / SCC-02)

**Expected Result:**
- Отображается сообщение об отсутствии модели
- Присутствует кликабельная ссылка/кнопка для перехода к настройке
- `expect(page.locator('a:has-text("configure")')).toBeVisible()` (или аналогичный текст)
- Клик по ссылке открывает drill-in builder-assistant агента
- Поле ввода сообщений НЕ отображается

**Teardown:**
```bash
# Восстановить модель при необходимости
```

**Negative / Edge Cases:**
- Обновить страницу → no-model state сохраняется
- Назначить модель через API → обновить страницу → chat input появился

---

### TC-PANEL-07: Clear button появляется при наличии сообщений

**AC:** AC-PANEL-07
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- builder-assistant существует с назначенной моделью
- Bottom panel открыт, AI Assistant tab активен

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: AI Assistant chat input виден
3. `mcp__playwright__browser_snapshot` → проверить: Clear button отсутствует (чат пустой)
4. `mcp__playwright__browser_type` → `Hello`
5. `mcp__playwright__browser_press_key` → `Enter`
6. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: ответ ассистента получен
7. `mcp__playwright__browser_snapshot` → проверить: Clear button появился
8. `mcp__playwright__browser_click` → Clear button
9. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: история очищена
10. `mcp__playwright__browser_snapshot` → проверить: история пустая, Clear button исчез

**Backend Verification:**
```bash
# Clear очищает только UI историю — backend verification не требуется
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- При пустом чате Clear button НЕ виден
- После появления сообщений Clear button становится видимым
- Клик по Clear очищает историю сообщений в UI
- После очистки Clear button снова скрывается

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Clear во время получения ответа → ответ отменяется, история очищается
- Только одно сообщение отправлено → Clear button виден, работает

---

## 43. Test Flow

### TC-TESTFLOW-01: HTTP Headers key-value editor

**AC:** AC-TESTFLOW-01
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- Admin UI, canvas page, bottom panel видим

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: bottom panel загружен
3. `mcp__playwright__browser_click` → вкладка "Test Flow"
4. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: Test Flow tab загружен
5. `mcp__playwright__browser_click` → секция "Headers"
6. `mcp__playwright__browser_snapshot` → Headers editor виден
7. `mcp__playwright__browser_click` → Add header row
8. `mcp__playwright__browser_type` → key: `Authorization`
9. `mcp__playwright__browser_type` → value: `Bearer test-token`
10. `mcp__playwright__browser_click` → Add header row
11. `mcp__playwright__browser_type` → key: `X-Custom`
12. `mcp__playwright__browser_type` → value: `value123`
13. `mcp__playwright__browser_snapshot` → два header row видимы
14. `mcp__playwright__browser_click` → Remove первый header row
15. `mcp__playwright__browser_snapshot` → остался только X-Custom

**Backend Verification:**
```bash
# Нет API verification — headers применяются при отправке запроса
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Key-value editor с add/remove rows
- JSON import button → paste JSON объект → заполняет rows
- Headers сохраняются в рамках сессии (не очищаются при переключении табов)

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Пустой key → header не добавляется или показывает ошибку валидации
- Дублирующийся key → предупреждение или перезапись
- JSON import с невалидным JSON → ошибка, editor не ломается

---

### TC-TESTFLOW-02: SSE streaming в Test Flow показывает tool calls inline

**AC:** AC-TESTFLOW-02, AC-TESTFLOW-03
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- Агент настроен с MCP tool (например web search)
- Bottom panel открыт на табе "Test Flow"

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: Test Flow tab загружен
3. `mcp__playwright__browser_click` → поле ввода Test Flow
4. `mcp__playwright__browser_type` → `Search for latest AI news`
5. `mcp__playwright__browser_press_key` → `Enter`
6. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: tool call отображается inline
7. `mcp__playwright__browser_snapshot` → проверить inline tool call display
8. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: финальный ответ получен
9. `mcp__playwright__browser_snapshot` → полный ответ с tool calls и reasoning

**Backend Verification:**
```bash
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:9555/api/v1/sessions?limit=1"
# Expected: сессия создана с tool calls
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)
- Tenant isolation: cross-schema → 403/404 (see TC-CLOUD-02 / SCC-02)

**Expected Result:**
- Streaming text появляется инкрементально
- Tool calls показываются inline: `[🔧 search_web("latest AI news")]` с expand/collapse
- `expect(page.locator('.tool-call-display')).toBeVisible()`
- Reasoning steps показываются inline: `[💭 Thinking...]`
- SSE токены появляются постепенно

**Teardown:**
```bash
# Удалить тестовую сессию при необходимости
```

**Negative / Edge Cases:**
- Агент не вызывает tools → только текстовый ответ стримится
- Очень длинный tool result → результат truncated с "Show more" или скроллируется

---

### TC-TESTFLOW-03: View in Inspect переходит к текущей сессии

**AC:** AC-TESTFLOW-04
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- В Test Flow отправлено хотя бы одно сообщение (есть активная сессия)

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: Test Flow tab виден
3. `mcp__playwright__browser_click` → вкладка "Test Flow"
4. `mcp__playwright__browser_type` → `Hello, test session`
5. `mcp__playwright__browser_press_key` → `Enter`
6. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: ответ получен
7. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: ссылка "View in Inspect" появилась
8. `mcp__playwright__browser_snapshot` → ссылка видима в области ответа
9. `mcp__playwright__browser_click` → ссылка "View in Inspect"
10. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: URL изменился на `/admin/inspect/{session_id}`
11. `mcp__playwright__browser_snapshot` → InspectPage открыта с трейсом сессии

**Backend Verification:**
```bash
# Получить session_id из URL
SESSION_ID=$(mcp__playwright__browser_evaluate "window.location.pathname.split('/').pop()")
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:9555/api/v1/sessions/$SESSION_ID
# Expected: 200, session exists with messages
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Ссылка "View in Inspect" присутствует после получения ответа
- После клика URL меняется на `/admin/inspect/{session_id}`
- InspectPage открывается с трейсом именно этой сессии
- `expect(page).toHaveURL(/\/inspect\//)`
- Session ID в URL совпадает с ID сессии из Test Flow

**Teardown:**
```bash
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:9555/api/v1/sessions/$SESSION_ID
```

**Negative / Edge Cases:**
- Ссылка недоступна пока ответ ещё стримится → появляется только после завершения
- Inspect page открыта повторно для той же сессии → тот же трейс

---

### TC-TEST-01: Test Flow tab имеет HTTP Headers editor

**AC:** AC-TEST-01 (AC-TESTFLOW-01)
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- Admin UI, canvas page, bottom panel видим

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: bottom panel загружен
3. `mcp__playwright__browser_click` → вкладка "Test Flow"
4. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: Test Flow загружен
5. `mcp__playwright__browser_snapshot` → проверить HTTP Headers секцию
6. `mcp__playwright__browser_click` → Add header → key: `X-Zitadel-Token`, value: `test-token-123`
7. `mcp__playwright__browser_click` → Add header → key: `X-Request-ID`, value: `req-456`
8. `mcp__playwright__browser_snapshot` → оба header добавлены
9. `mcp__playwright__browser_click` → вкладка "AI Assistant" и обратно "Test Flow"
10. `mcp__playwright__browser_snapshot` → headers сохранились после смены табов

**Backend Verification:**
```bash
# Нет API verification — headers проверяются при отправке запроса
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Key-value editor (как GraphQL Playground headers)
- Add/remove header rows работает
- Headers отправляются с test flow запросами
- Headers persist в рамках сессии (не очищаются при смене табов)

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Refresh страницы → headers очищаются (session-only persistence) или сохраняются (localStorage)

---

### TC-TEST-02: Test Flow headers forwarded to MCP tool calls

**AC:** AC-TEST-02 (AC-TESTFLOW-02)
**Layer:** Full-stack
**Test Type:** Integration (API)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- MCP server с forward_headers auth настроен
- Test Flow headers сконфигурированы: `X-Zitadel-Token: test-token-123`
- Агент с MCP tool настроен

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: Test Flow загружен
3. `mcp__playwright__browser_click` → вкладка "Test Flow"
4. Добавить header: `X-Zitadel-Token` = `test-token-123`
5. `mcp__playwright__browser_type` → запрос который вызовет MCP tool call
6. `mcp__playwright__browser_press_key` → `Enter`
7. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: tool call выполнен
8. `mcp__playwright__browser_snapshot` → проверить что tool call завершён

**Backend Verification:**
```bash
# Проверить логи mock MCP server
# Expected: X-Zitadel-Token: test-token-123 получен MCP сервером

# Проверить через session trace
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:9555/api/v1/sessions?limit=1"
# Expected: сессия с tool call steps
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)
- Tenant isolation: cross-schema → 403/404 (see TC-CLOUD-02 / SCC-02)

**Expected Result:**
- MCP tool получает заголовок `X-Zitadel-Token: test-token-123`
- forward_headers mechanism передаёт custom headers в MCP calls
- Собственные headers агента не утекают (только user-set Test Flow headers)

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Header не задан → MCP вызывается без extra headers (backward compatible)
- Неверный token в header → MCP возвращает 401, агент обрабатывает ошибку gracefully

---

### TC-TEST-03: Trigger config имеет Custom Headers field

**AC:** AC-TEST-03
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- Trigger configuration panel открыт (есть trigger нода на canvas)

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: canvas загружен
3. `mcp__playwright__browser_click` → trigger нода на canvas
4. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: trigger config panel открылся
5. `mcp__playwright__browser_snapshot` → найти секцию "Custom Headers"
6. `mcp__playwright__browser_click` → Add header
7. `mcp__playwright__browser_type` → key: `Authorization`, value: `Bearer ext-token-789`
8. `mcp__playwright__browser_snapshot` → header добавлен

**Backend Verification:**
```bash
# После сохранения trigger config
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:9555/api/v1/triggers/{trigger_id}
# Expected: custom_headers содержит Authorization header
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Key-value editor для custom headers в trigger config
- Headers включаются когда trigger срабатывает (webhook payload или cron context)
- Headers передаются через agent chain в MCP servers

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Trigger без MCP tools → custom headers игнорируются без ошибки
- Сохранение trigger без headers → backward compatible

---

### TC-TEST-04: Chat API accepts optional headers field *(BACKEND-DEFERRED)*

**AC:** AC-TEST-04
**Layer:** Backend (Go)
**Test Type:** Integration (API)

> **Status: BACKEND-DEFERRED** — Requires Chat API modification.

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- MCP server с forward_headers настроен

**Steps:**
1. POST webhook с headers:
   ```bash
   curl -s -X POST http://localhost:9555/api/v1/webhooks/{path} \
     -H "Authorization: Bearer $TOKEN" \
     -H "Content-Type: application/json" \
     -d '{
       "message": "Search for latest news",
       "headers": {"X-Zitadel-Token": "user-token-abc"}
     }'
   # Expected: 200, запрос обработан
   ```
2. Агент вызывает MCP tool
3. Проверить что MCP server получил custom header

**Backend Verification:**
```bash
# Проверить без поля headers (backward compatible)
curl -s -X POST http://localhost:9555/api/v1/webhooks/{path} \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}'
# Expected: 200, обрабатывается нормально (нет extra headers)
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Webhook trigger принимает optional поле `headers` в request body
- Headers передаются в MCP tool calls во время сессии
- Отсутствие поля `headers` → нет extra headers (backward compatible)

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- `headers` не объект (строка/массив) → 400 Bad Request
- Пустой объект `headers: {}` → нет extra headers, нет ошибки

---

## 37. Assistant on All Pages

### TC-ASST-01: Bottom panel видим на странице Agents

**AC:** AC-UX-09
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/agents`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: страница загружена
3. `mcp__playwright__browser_snapshot` → bottom panel виден внизу страницы
4. `mcp__playwright__browser_click` → вкладка AI Assistant
5. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: AI Assistant tab активен
6. `mcp__playwright__browser_type` → `Привет`
7. `mcp__playwright__browser_press_key` → `Enter`
8. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: ответ получен
9. `mcp__playwright__browser_snapshot` → ответ ассистента виден

**Backend Verification:**
```bash
curl -s -H "Authorization: Bearer $TOKEN" "http://localhost:9555/api/v1/sessions?limit=1"
# Expected: сессия создана из Agents page
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Bottom panel виден внизу страницы (как на Canvas)
- AI Assistant tab функционален
- Panel state (open/closed, height) совпадает со state на Canvas page

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Agents page без агентов → AI Assistant всё равно работает

---

### TC-ASST-02: Bottom panel видим на всех admin pages

**AC:** AC-UX-09 (AC-PANEL-04)
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/agents`
2. `mcp__playwright__browser_snapshot` → bottom panel присутствует
3. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/mcp`
4. `mcp__playwright__browser_snapshot` → bottom panel присутствует
5. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/models`
6. `mcp__playwright__browser_snapshot` → bottom panel присутствует
7. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/triggers`
8. `mcp__playwright__browser_snapshot` → bottom panel присутствует
9. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/settings`
10. `mcp__playwright__browser_snapshot` → bottom panel присутствует

**Backend Verification:**
```bash
# Нет API verification — чисто UI тест
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Bottom panel присутствует на каждой admin странице
- Panel state (height, active tab, open/closed) сохраняется при навигации

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Login page — NO bottom panel (не аутентифицирован)
- Panel свёрнут → navigate → всё равно свёрнут

---

### TC-ASST-03: Schema selector в заголовке чата

**AC:** AC-UX-10
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- Существует минимум 2 схемы (schema A и schema B)
- Builder-assistant с моделью назначен

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: AI Assistant tab загружен
3. `mcp__playwright__browser_click` → schema selector dropdown в заголовке чата
4. `mcp__playwright__browser_snapshot` → список схем виден
5. `mcp__playwright__browser_click` → выбрать schema B
6. `mcp__playwright__browser_snapshot` → выбрана schema B
7. `mcp__playwright__browser_type` → `Test message for schema B`
8. `mcp__playwright__browser_press_key` → `Enter`
9. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: ответ получен

**Backend Verification:**
```bash
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:9555/api/v1/schemas
# Expected: обе схемы существуют
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)
- Tenant isolation: cross-schema → 403/404 (see TC-CLOUD-02 / SCC-02)

**Expected Result:**
- Schema selector показывает все доступные схемы
- Переключение схемы меняет контекст Assistant
- Сообщение отправлено entry agent выбранной схемы
- Test Flow tab также переключается на выбранную схему

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Только одна схема → selector disabled или скрыт
- Смена схемы при активном чате → предупреждение или автоматическая очистка истории

---

### TC-ASST-04: "Open Chat" link удалён из sidebar

**AC:** AC-UX-12
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: страница загружена
3. `mcp__playwright__browser_snapshot` → проверить sidebar navigation
4. Проверить все nav items на наличие/отсутствие "Chat" / "Open Chat"

**Backend Verification:**
```bash
# Нет API verification — чисто UI тест
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- "Open Chat" / "Chat" link НЕ присутствует в sidebar
- Все ожидаемые nav items присутствуют: Canvas, MCP Servers, Models, Triggers, Settings и т.д.

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Поиск по тексту "chat" в sidebar → нет результатов

---

### TC-PANEL-NOMODEL-01: AI Assistant "No model configured" state

**AC:** AC-BA-01 (AC-PANEL-06)
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- Агент `builder-assistant` существует, но без назначенной модели:
  ```bash
  curl -s -X PATCH http://localhost:9555/api/v1/agents/builder-assistant \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"model_id": null}'
  ```

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: bottom panel загружен
3. `mcp__playwright__browser_click` → вкладка "AI Assistant"
4. `mcp__playwright__browser_snapshot` → проверить содержимое панели

**Backend Verification:**
```bash
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:9555/api/v1/agents/builder-assistant
# Expected: model_id = null
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Видна иконка предупреждения (треугольник)
- Текст содержит "needs a model assigned" или аналогичный
- Присутствует кликабельная ссылка/кнопка для конфигурации
- Поле ввода сообщений НЕ отображается в этом состоянии

**Teardown:**
```bash
# Восстановить модель builder-assistant при необходимости
```

**Negative / Edge Cases:**
- Перейти на другую страницу и вернуться → warning state сохраняется

---

### TC-PANEL-COLLAPSE-TABS-01: Клик по label в collapsed bar переключает и разворачивает

**AC:** AC-PANEL-02 (AC-PANEL-05)
**Layer:** Frontend (React)
**Test Type:** E2E (Playwright)

**Setup:**
- Engine запущен на `http://localhost:9555`
- TOKEN obtained (см. TC-PANEL-01)
- Bottom panel свёрнут (collapsed = thin bar ~40px)

**Steps:**
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: bottom panel загружен
3. `mcp__playwright__browser_click` → collapse toggle (▼)
4. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель свернулась
5. `mcp__playwright__browser_snapshot` → thin bar виден с labels "AI Assistant | Test Flow"
6. `mcp__playwright__browser_click` → label "Test Flow" в thin bar
7. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель раскрылась
8. `mcp__playwright__browser_snapshot` → активна вкладка Test Flow (не AI Assistant)
9. `mcp__playwright__browser_click` → collapse toggle (▼) снова
10. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель свернулась
11. `mcp__playwright__browser_click` → label "AI Assistant"
12. `mcp__playwright__browser_wait_for` → poll every 2s up to 10s: панель раскрылась
13. `mcp__playwright__browser_snapshot` → активна вкладка AI Assistant

**Backend Verification:**
```bash
# Нет API verification — чисто UI тест
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Панель разворачивается до предыдущей высоты
- Активной становится вкладка соответствующая нажатому label
- Повтор для AI Assistant label → разворачивается на AI Assistant tab

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Панель уже развёрнута на Test Flow → клик на "AI Assistant" label → переключает таб без collapse/expand

---

## 48. Test Flow Session Management

### TC-TESTFLOW-SESSION-01: Session dropdown shows recent sessions and allows switching

**AC:** AC-TESTFLOW-05, AC-TESTFLOW-06
**Layer:** E2E
**Test Type:** E2E (Playwright)
**Prerequisites:**
- Engine на `http://localhost:9555` (Docker test stack)
- Агент `test-agent` создан и имеет модель

**Setup:**
```bash
TOKEN=$(curl -s -X POST http://localhost:9555/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
```

**Steps:**

#### Core
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_snapshot` → bottom panel видим
3. `mcp__playwright__browser_click` → Test Flow tab
4. `mcp__playwright__browser_snapshot` → Test Flow active, agent selector видим
5. `mcp__playwright__browser_snapshot` → Session dropdown видим с "New Session"
6. Send test message → type "hello" + click Run
7. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: assistant response appears
8. `mcp__playwright__browser_snapshot` → session ID показан в dropdown
9. `mcp__playwright__browser_click` → session dropdown → verify session appears in list
10. `mcp__playwright__browser_click` → "New Session" item
11. `mcp__playwright__browser_snapshot` → chat cleared, dropdown shows "New Session"

**Backend Verification:**
```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:9555/api/v1/sessions?agent_name=test-agent&per_page=5"
# Expected: sessions array with at least 1 session
```

**Security Cross-References:**
- ⛔ GATE SCC-01: `curl http://localhost:9555/api/v1/sessions` (no token) → 401

**Expected Result:**
- Session dropdown показывает список сессий с timestamps
- Клик на сессию загружает её историю
- "New Session" очищает чат и начинает новую сессию

**Teardown:**
```bash
# Sessions auto-cleaned — no explicit teardown needed
```

---

### TC-TESTFLOW-SESSION-02: Delete session from dropdown

**AC:** AC-TESTFLOW-08
**Layer:** E2E
**Test Type:** E2E (Playwright)
**Prerequisites:**
- Engine на `http://localhost:9555`
- Минимум 1 существующая сессия для test-agent

**Setup:**
```bash
TOKEN=$(curl -s -X POST http://localhost:9555/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
```

**Steps:**

#### Core
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_click` → Test Flow tab
3. Send test message → create a session
4. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: response received
5. `mcp__playwright__browser_click` → session dropdown open
6. `mcp__playwright__browser_snapshot` → session visible in list with delete button (trash icon)
7. `mcp__playwright__browser_click` → delete button on session
8. `mcp__playwright__browser_handle_dialog` → confirm deletion
9. `mcp__playwright__browser_snapshot` → session removed from list, chat cleared

**Backend Verification:**
```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:9555/api/v1/sessions?agent_name=test-agent&per_page=5"
# Expected: deleted session no longer in list
```

**Security Cross-References:**
- ⛔ GATE SCC-01: `curl -X DELETE http://localhost:9555/api/v1/sessions/{id}` (no token) → 401

**Expected Result:**
- Confirmation dialog appears before delete
- Session removed from dropdown list
- If deleted session was active, chat is cleared

**Teardown:**
```bash
# Session already deleted — no teardown needed
```

---

### TC-TESTFLOW-SESSION-03: Agent change resets session list

**AC:** AC-TESTFLOW-09
**Layer:** E2E
**Test Type:** E2E (Playwright)
**Prerequisites:**
- Engine на `http://localhost:9555`
- Минимум 2 агента созданы

**Steps:**

#### Core
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_click` → Test Flow tab
3. Send message to agent A → creates session
4. `mcp__playwright__browser_wait_for` → response received
5. `mcp__playwright__browser_select_option` → switch to agent B in agent dropdown
6. `mcp__playwright__browser_snapshot` → chat cleared, session dropdown reset
7. `mcp__playwright__browser_click` → session dropdown
8. `mcp__playwright__browser_snapshot` → only agent B's sessions shown (or empty)

**Backend Verification:**
```bash
# Нет API verification — чисто UI тест
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Switching agent clears chat and resets session list
- Session dropdown shows only sessions for the new agent

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

---

## 49. Context Usage Bar

### TC-PANEL-CTX-01: Context bar shows max tokens for agent

**AC:** AC-PANEL-08, AC-PANEL-10
**Layer:** E2E
**Test Type:** E2E (Playwright)
**Prerequisites:**
- Engine на `http://localhost:9555`
- Агент с `max_context_size > 0` существует

**Steps:**

#### Core
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_snapshot` → bottom panel visible
3. `mcp__playwright__browser_click` → AI Assistant tab (if not active)
4. `mcp__playwright__browser_snapshot` → verify context bar visible between messages and input
5. Verify label format: "— / {N}K tokens" (dash = no usage data yet)

#### Extended
> ⚠️ Requires: Test Flow tab with agent that has max_context_size set
6. `mcp__playwright__browser_click` → Test Flow tab
7. `mcp__playwright__browser_snapshot` → context bar visible in Test Flow too

**Backend Verification:**
```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  http://localhost:9555/api/v1/agents/test-agent
# Expected: max_context_size > 0
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Context bar visible in both AI Assistant and Test Flow tabs
- Shows max token count from agent's max_context_size
- Bar is empty (no fill) before any interaction
- Hidden if agent has no max_context_size

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

---

### TC-PANEL-CTX-02: Context bar fills after chat response with token data

**AC:** AC-PANEL-09
**Layer:** E2E
**Test Type:** E2E (Playwright)
**Prerequisites:**
- Engine на `http://localhost:9555`
- Агент с моделью и `max_context_size > 0`
- Модель провайдер возвращает token usage в response

**Steps:**

#### Core
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_click` → Test Flow tab
3. `mcp__playwright__browser_snapshot` → context bar shows "— / {N}K tokens"
4. Send test message → type "hello" + click Run
5. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: response complete (done event)
6. `mcp__playwright__browser_snapshot` → context bar now shows "{used} / {max} tokens" with fill bar

**Backend Verification:**
```bash
# Token usage is returned in SSE done event — verify by checking the streaming response includes total_tokens
```

**Security Cross-References:**
- Requires valid JWT → 401 without token (see TC-AUTH-BEARER-01 / SCC-01)

**Expected Result:**
- Before interaction: bar is empty, label shows "— / {max}K tokens"
- After response: bar fills proportionally, label shows "{used} / {max}K tokens"
- Color: green (0-60%), yellow (60-85%), red (85-100%)

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

**Negative / Edge Cases:**
- Model provider doesn't return token usage → bar stays empty, label stays "— / {max}K tokens"
- max_context_size = 0 → bar hidden

---

### TC-CHAT-PERSIST-01: Chat session survives page reload

**AC:** AC-CHAT-01, AC-CHAT-02
**Layer:** E2E
**Test Type:** E2E (Playwright)
**Prerequisites:**
- Engine на `http://localhost:9555`
- Агент с моделью

**Steps:**

#### Core
1. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/`
2. `mcp__playwright__browser_click` → AI Assistant tab
3. Send message "hello" → wait for response
4. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: assistant response appears
5. `mcp__playwright__browser_snapshot` → messages visible in chat
6. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/` (reload)
7. `mcp__playwright__browser_wait_for` → poll every 2s up to 30s: "Restoring session..." spinner OR messages restored
8. `mcp__playwright__browser_snapshot` → previous messages restored from backend

#### Extended
9. `mcp__playwright__browser_click` → New Session button
10. `mcp__playwright__browser_snapshot` → chat cleared
11. `mcp__playwright__browser_navigate` → `http://localhost:9555/admin/` (reload)
12. `mcp__playwright__browser_snapshot` → chat empty (localStorage cleared)

**Backend Verification:**
```bash
# Session ID is stored in localStorage — verify via browser_evaluate
```

**Security Cross-References:**
- ⛔ GATE SCC-01: `curl http://localhost:9555/api/v1/sessions/{id}/messages` (no token) → 401

**Expected Result:**
- After reload: session restored, messages visible
- After New Session + reload: chat is empty (no stored session)

**Teardown:**
```bash
# Нет созданных данных — teardown не требуется
```

---

## AC→TC Coverage Matrix

| AC-ID | Description | TC-ID | Status |
|-------|-------------|-------|--------|
| AC-PANEL-01 | Drag handle resize (min 150px, max 70% viewport) | TC-PANEL-01 | Covered |
| AC-PANEL-02 | Collapse/expand toggle, collapsed = thin bar с tab labels | TC-PANEL-02, TC-PANEL-COLLAPSE-TABS-01 | Covered |
| AC-PANEL-03 | Panel state persists across navigation (localStorage) | TC-PANEL-03 | Covered |
| AC-PANEL-04 | Bottom panel на всех admin pages | TC-PANEL-04, TC-ASST-02 | Covered |
| AC-PANEL-05 | Collapsed bar — клик по tab label разворачивает и активирует таб | TC-PANEL-05, TC-PANEL-COLLAPSE-TABS-01 | Covered |
| AC-PANEL-06 | AI Assistant no-model state с actionable ссылкой | TC-PANEL-06, TC-PANEL-NOMODEL-01 | Covered |
| AC-PANEL-07 | AI Assistant Clear button при наличии сообщений | TC-PANEL-07 | Covered |
| AC-TESTFLOW-01 | Test Flow HTTP Headers key-value editor | TC-TESTFLOW-01, TC-TEST-01 | Covered |
| AC-TESTFLOW-02 | Headers forwarded to MCP tool calls | TC-TEST-02 | Covered |
| AC-TESTFLOW-03 | SSE streaming показывает tool calls и reasoning inline | TC-TESTFLOW-02 | Covered |
| AC-TESTFLOW-04 | "View in Inspect" link переходит на InspectPage | TC-TESTFLOW-03 | Covered |
| AC-TEST-03 | Trigger config Custom Headers field | TC-TEST-03 | Covered |
| AC-TEST-04 | Chat API accepts optional headers field | TC-TEST-04 | Covered (BACKEND-DEFERRED) |
| AC-UX-09 | AI Assistant inline bottom panel на всех pages | TC-ASST-01, TC-ASST-02 | Covered |
| AC-UX-10 | Schema selector в chat header | TC-ASST-03 | Covered |
| AC-UX-12 | "Open Chat" удалён из sidebar | TC-ASST-04 | Covered |
| AC-BA-01 | No model → warning + Configure link (panel view) | TC-PANEL-NOMODEL-01 | Covered |
| AC-PANEL-08 | Context Usage Bar shows max_context_size | TC-PANEL-CTX-01 | ✅ Covered |
| AC-PANEL-09 | Context Usage Bar fills with token data + color gradient | TC-PANEL-CTX-02 | ✅ Covered |
| AC-PANEL-10 | Context Usage Bar hidden when max_context_size null | TC-PANEL-CTX-01 | ✅ Covered |
| AC-TESTFLOW-05 | Session dropdown shows recent sessions | TC-TESTFLOW-SESSION-01 | ✅ Covered |
| AC-TESTFLOW-06 | Switch session loads history | TC-TESTFLOW-SESSION-01 | ✅ Covered |
| AC-TESTFLOW-07 | New Session creates fresh session | TC-TESTFLOW-SESSION-01 | ✅ Covered |
| AC-TESTFLOW-08 | Delete session with confirmation | TC-TESTFLOW-SESSION-02 | ✅ Covered |
| AC-TESTFLOW-09 | Agent change resets session list | TC-TESTFLOW-SESSION-03 | ✅ Covered |
| AC-CHAT-01 | Session persistence across reload | TC-CHAT-PERSIST-01 | ✅ Covered |
| AC-CHAT-02 | Reset session clears localStorage | TC-CHAT-PERSIST-01 | ✅ Covered |
