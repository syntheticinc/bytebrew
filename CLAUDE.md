# Project: ByteBrew (AI Agent)

## Language
Общайся на русском языке.

## Stack
- Backend: Go (bytebrew-srv) — gRPC server, port **60401**
- CLI: TypeScript/Bun (bytebrew-cli) — Ink TUI
- Cloud API: Go (bytebrew-cloud-api) — REST API
- Cloud Web: React/Vite (bytebrew-cloud-web)
- Mobile: Flutter (bytebrew-mobile-app)
- Communication: gRPC (srv↔cli), REST (cloud)

## Architecture (CRITICAL)

### Clean Architecture
```
Delivery → Usecase → Domain ← Infrastructure
```
- **Domain** — чистые сущности, БЕЗ внешних зависимостей, БЕЗ тегов
- **Usecase** — бизнес-логика + определение интерфейсов (consumer-side!)
- **Infrastructure** — реализация интерфейсов (DB, API, tools)
- **Delivery** — тонкие handlers, только трансформация request→usecase→response

### SOLID (обязательно)
- **S** — один struct = одна ответственность. Описание БЕЗ слова "и"
- **O** — расширение через новый код, не изменение существующего
- **L** — подстановка подклассов без изменения кода
- **I** — маленькие интерфейсы, определённые на стороне потребителя
- **D** — зависимости через интерфейсы

### Consumer-Side Interfaces
Интерфейсы определяются **В ФАЙЛЕ USECASE**, не в отдельном contract.go.

### Размеры (триггеры для анализа)
- Файл > 200-300 строк → проверить SRP
- Метод > 30-50 строк → выделить подфункции
- Поля в struct > 5-7 → слишком много ответственности
- Вложенность > 2-3 уровня → инвертировать условия

**Полный гайд:** @.agents/instructions/llm_coding_promt.md

## Принципы разработки
- **Никаких фоллбеков и костылей** — делать правильно или не делать
- **Кросс-платформенность** — решения для всех ОС
- **Не бросаться фиксить симптом** — сначала понять архитектуру
- **Спросить пользователя** если не уверен в подходе
- Лишний или плохой код — удалять/переписывать

## Cloud Infrastructure

- **Cloud server:** `49.12.226.216` — SSH: `root@49.12.226.216`
- **Bridge:** `bridge.bytebrew.ai` (Caddy TLS → internal port 8443), systemd `bytebrew-bridge.service`
- **Cloud API:** `api.bytebrew.ai`, systemd `bytebrew-cloud-api.service`
- **Config:** `/etc/bytebrew/bridge.env`, Caddyfile: `/etc/caddy/Caddyfile`
- **Deploy bridge:** `scp bin/bytebrew-bridge root@49.12.226.216:/usr/local/bin/ && ssh root@49.12.226.216 systemctl restart bytebrew-bridge`

## Server Port Discovery

Сервер записывает port file при старте. Не хардкодить порт — читать из файла.

| OS | Port file path |
|----|---------------|
| Windows | `%APPDATA%/bytebrew/server.port` |
| macOS | `~/Library/Application Support/bytebrew/server.port` |
| Linux | `${XDG_DATA_HOME:-~/.local/share}/bytebrew/server.port` |

```json
{"pid": 131956, "port": 60466, "host": "127.0.0.1", "startedAt": "2026-03-07T15:29:27Z"}
```

**Проверка:** `cat "$APPDATA/bytebrew/server.port"` (Windows/bash)

## Commands
```bash
# Server
cd bytebrew-srv && go run ./cmd/server

# CLI (runtime: bun, не node — из-за bun:sqlite)
cd bytebrew-cli && bun run build

# Tests
cd bytebrew-srv && go test ./...
cd bytebrew-cli && bun test

# Cloud
cd bytebrew-cloud-api && go run ./cmd/server
cd bytebrew-cloud-web && npm run build

# Kill server (read port from port file)
cat "$APPDATA/bytebrew/server.port"  # get PID
taskkill /F /PID <pid>
```

## Планирование

### Quick Fix (без планирования)
- Простое изменение конфига, правка 1-3 файлов, тривиальный баг-фикс
- Делегируй суб-агенту напрямую или исправь сам если < 50 строк

### С планированием
- 4+ файлов, архитектурные решения, исследование, новая фича

## Верификация результатов (КРИТИЧНО)

**ВСЕГДА перепроверять результат работы.** Не останавливаться пока не убедишься что всё работает.

- После каждого изменения — запустить тесты и убедиться что они проходят
- Если изменения затрагивают runtime (мобильное приложение, CLI) — проверить на реальном устройстве/процессе
- Hot restart/reload после изменений Flutter кода — убедиться что новый код применился
- Если суб-агент "написал фикс" — проверить что фикс реально работает (тесты, логи, устройство)
- Не доверять "тесты прошли" без проверки что тесты покрывают конкретный баг
- Если пользователь говорит "не работает" после фикса — проблема НЕ решена, нужно копать глубже

**Цикл:** Изменение → Тест → Верификация на устройстве → Подтверждение ИЛИ повторная итерация

## Сверка с планом перед завершением (КРИТИЧНО)

**НИКОГДА не объявлять работу завершённой при наличии активного плана без 100% сверки.**

Перед тем как сказать "готово" или остановиться:
1. **Открыть файл плана** и пройти по КАЖДОМУ пункту/тесткейсу
2. **Составить матрицу покрытия**: план vs реальность (что сделано, что нет)
3. **Если покрытие < 100%** — продолжать работу, не останавливаться
4. **Если пункт невозможен** (требует внешние зависимости, реальное устройство) — явно указать и обосновать, спросить пользователя
5. **Не округлять "почти готово" до "готово"** — 90% ≠ 100%

**Типичные ошибки:**
- Реализовал код → написал пару тестов → объявил "готово" → в плане 100+ тесткейсов, покрыто 30%
- Суб-агент вернул "всё готово" → не проверил что именно он сделал vs что было в задании
- Этап "завершён" → но в плане есть конкретные тесткейсы (TC-XX-YY) которые не написаны
- "Тесты проходят" → но это тесты которые УЖЕ были, а новые для нового кода не написаны

**Правило:** Если в плане написано "100% покрытие тестами" или перечислены конкретные TC-ID — каждый из них должен существовать как реальный тест в коде.

## Автономный workflow (КРИТИЧНО)

### Принцип: полная автономия по плану

Если есть план (phase plan, implementation plan) — агент работает **полностью автономно**. Не спрашивает пользователя, не останавливается "для подтверждения". Пользователь участвует только на checkpoint'ах (приёмка).

### Роль главного агента — ОРКЕСТРАТОР

Декомпозирует → делегирует суб-агентам → проверяет результат → итерирует.

**Формат задания для суб-агента:**
1. Цель (1-2 предложения)
2. План: путь к файлу, номер этапа
3. Контекст: файлы для изучения
4. Задачи: конкретный список
5. Ограничения: стиль, архитектура
6. Критерий готовности

### Цикл выполнения Phase (повторять для каждой Phase)

```
1. РЕАЛИЗАЦИЯ
   - Прочитать задачи из phase plan
   - Исследовать текущий код (grep, read файлы) перед изменениями
   - Написать код (через суб-агентов или напрямую)
   - git commit после каждой завершённой задачи

2. SELF-REVIEW
   - Прочитать ВСЕ изменённые файлы (`git diff`)
   - Проверить: SOLID, Clean Architecture, правила из CLAUDE.md
   - Проверить размеры: файл > 300 строк? метод > 50? struct > 7 полей?
   - Проверить: нет hardcoded values, нет TODO без issue
   - Если проблемы → исправить → commit

3. UNIT ТЕСТЫ
   - Написать unit тесты на КАЖДЫЙ новый struct/interface/function
   - go build → go test → ВСЕ проходят (не только новые)
   - Если тесты падают → исправить код или тесты → commit

4. РУЧНОЕ ТЕСТИРОВАНИЕ (MCP)
   - Запустить сервер: go build + запуск
   - MCP tui-test: выполнить сценарии из AC
   - Если сценарий fail → исправить → вернуться к п.1
   - Остановить сервер после тестирования

5. СВЕРКА С AC (100% покрытие)
   - Открыть AC для этой phase
   - Пройти КАЖДЫЙ AC-N.x
   - Составить матрицу: AC-ID → PASS/FAIL
   - Если покрытие < 100% → вернуться к п.1
   - НЕЛЬЗЯ пропускать AC без явного обоснования

6. BACKWARD COMPAT ПРОВЕРКА
   - Legacy конфиг (без agents:) → сервер стартует, работает как раньше
   - Существующие тесты не сломаны

7. ФИНАЛИЗАЦИЯ
   - Все AC пройдены
   - Все тесты проходят
   - go build OK
   - git commit с пометкой "Phase N: [название] — complete"
   - → Перейти к Phase N+1
```

### Правила автономной работы

- **Не спрашивай пользователя** по ходу реализации
- **Параллелизация:** backend + frontend суб-агенты в одном сообщении
- **Ревью и тестирование** обязательны для КАЖДОГО этапа — не пропускать
- **Не говори "готово"** пока build + tests + AC не пройдены
- **При блокере** (не можешь решить, архитектурный вопрос) — запиши в phase plan и продолжай с остальными задачами. Блокеры = вопросы для checkpoint'а
- **Коммиты** — атомарные, после каждой завершённой задачи (не один большой коммит)

### Phase Plans

Если работа организована через phase plans (`docs/plan/.../implementation/phase-N.md`):
1. Прочитай `master.md` — порядок, зависимости, ключевые решения
2. Прочитай `phase-N.md` — самодостаточный план текущей phase
3. Выполни цикл выше
4. После завершения — перейди к `phase-N+1.md`
5. На checkpoint'ах — остановись, сообщи пользователю результат

## Тестирование

### Три уровня

1. **Unit тесты** — для каждого нового struct/interface/function. В пакете рядом с кодом
2. **Интеграционные с рендерингом** — MockChatModel → весь стек → `render(<ChatApp />)` → `lastFrame()`
3. **Prompt Regression** — замороженный контекст → реальная LLM → assertions

**Принцип:** тестируй что ВИДИТ пользователь, не data layer.

### Acceptance Criteria (ОБЯЗАТЕЛЬНО для каждой новой фичи)

**Каждая новая фича описывается как Acceptance Criteria** — пошаговый сценарий, который можно выполнить руками или автоматизировать.

**Формат AC:**
```markdown
### AC-X.Y: Название сценария

**Предусловие:** что должно быть настроено

1. Действие (конкретное: запустить, нажать, вызвать, curl)
2. Действие
3. Проверить: ожидаемый результат
4. Проверить: ещё один результат
```

**Хранение:**
- AC документ: `docs/plan/bytebrew_pivot/05_acceptance_criteria.md` — все AC по phases
- Acceptance тесты: `bytebrew-srv/tests/acceptance/` — shell scripts / Go tests, каждый AC-ID = файл
- Unit тесты: в пакете рядом с кодом (`*_test.go`)

**Правило: каждая новая фича ОБЯЗАНА иметь:**
1. AC в формате пошагового сценария (в 05_acceptance_criteria.md)
2. Unit test (в пакете рядом с кодом)
3. Ручной тест (в tests/acceptance/ или через MCP tui-test / marionette)

**Это нужно для:** regression testing, документации, onboarding новых разработчиков.

Подробности в CLAUDE.md каждого пакета и @docs/testing/headless-testing.md

## Headless тестирование

**НИКОГДА без `-C` на тестовый проект!**
```bash
cd bytebrew-cli
bun dist/index.js -C ../test-project ask --headless "prompt"
```

## MCP TUI Testing (mcp-tui-test)

Для ручного и автоматизированного тестирования CLI через MCP.

### Запуск CLI
```
# ВАЖНО: bun напрямую не работает — нужен cmd /c wrapper
# ВАЖНО: используй stream mode (buffer mode не работает на Windows)
mcp__tui-test__launch_tui(
  command: 'cmd /c "cd /d C:\\path\\to\\bytebrew-cli && bun dist\\index.js -C ..\\test-project session -s localhost:60466"',
  session_id: "cli",
  timeout: 300,
  mode: "stream"
)
```

### Отправка сообщений и проверка
```
# Отправить текст + Enter
mcp__tui-test__send_keys(keys: "hello\n", session_id: "cli")

# Ждать ответ (regex pattern)
mcp__tui-test__expect_text(pattern: "Hello|hello", session_id: "cli", timeout: 30)

# Захватить экран
mcp__tui-test__capture_screen(session_id: "cli")
```

### Ограничения
- **Ink TUI** (interactive mode) — Ink перерисовывает экран через ANSI escapes, stream mode не обновляется. Используй `session` command (headless multi-turn) вместо `chat`.
- **Buffer mode** — не работает на Windows (`SpawnPipe.read_nonblocking` error)
- **Bun без cmd /c** — `cd` не распознаётся, нужен `cmd /c` wrapper

## MCP Marionette (Flutter)

Для тестирования Flutter app на реальном устройстве через Debug VM Service.

### Подключение
```
# URI из вывода flutter run: "A Dart VM Service on ... is available at: http://127.0.0.1:PORT/TOKEN=/ws"
mcp__marionette__connect(uri: "ws://127.0.0.1:PORT/TOKEN=/ws")
```

### Взаимодействие
```
mcp__marionette__take_screenshots()           # Скриншот текущего состояния
mcp__marionette__get_interactive_elements()    # Список интерактивных элементов
mcp__marionette__tap(key: "element_key")       # Тап по key
mcp__marionette__tap(text: "Button Text")      # Тап по тексту
mcp__marionette__enter_text(key: "input_key", input: "text")  # Ввод текста
mcp__marionette__hot_reload()                  # Hot reload после изменений кода
```

### Важно
- Элементы идентифицируются по `ValueKey<String>` — если key нет, добавь в код
- `enter_text` НЕ вызывает `onChanged` — используй `controller.addListener` вместо `onChanged`
- После изменений кода — обязательно `hot_reload`

## Compact Instructions
При сжатии контекста ОБЯЗАТЕЛЬНО сохранить:
- Ссылку на текущий план (путь к файлу)
- Список выполненных и невыполненных этапов
- Текущий этап работы и его цель
