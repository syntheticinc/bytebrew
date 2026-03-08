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

## Автономный workflow (КРИТИЧНО)

### Роль главного агента — ОРКЕСТРАТОР, не исполнитель
**НЕ пишет код сам.** Декомпозирует → делегирует → проверяет → итерирует.

### Цикл: Планирование → Реализация → Ревью → Тестирование → Финал

**Формат задания для суб-агента:**
1. Цель (1-2 предложения)
2. План: путь к файлу, номер этапа
3. Контекст: файлы для изучения
4. Задачи: конкретный список
5. Ограничения: стиль, архитектура
6. Критерий готовности

**Правила:**
- Параллелизация: backend + frontend в одном сообщении
- Не спрашивай пользователя по ходу реализации
- Ревью и тестирование обязательны для КАЖДОГО этапа
- Сверка с планом обязательна
- Не говори "готово" пока build + lint + tests не пройден

## Тестирование

Два уровня = 100% покрытие:
1. **Интеграционные с рендерингом** — MockChatModel → весь стек со стороны клиента (мобильное приложение / cli) → `render(<ChatApp />)` → `lastFrame()`
2. **Prompt Regression** — замороженный контекст → реальная LLM → assertions

**Принцип:** тестируй что ВИДИТ пользователь, не data layer.

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
