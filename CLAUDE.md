# Project: ByteBrew (AI Agent)

## Language
Общайся на русском языке.

## Stack
- Backend: Go (bytebrew-srv)
- CLI: TypeScript/Bun (bytebrew-cli)
- Communication: gRPC

## Server
- Port: **60401** (localhost:60401)
- Start: `cd bytebrew-srv && go run ./cmd/server`
- Logs: `bytebrew-srv/logs/`

## Project Structure
```
bytebrew-srv/
├── cmd/                    # Entry points
├── internal/
│   ├── domain/            # Pure business entities (NO external deps)
│   ├── usecase/           # Business logic + interfaces (consumer-side!)
│   ├── service/           # Reusable helpers
│   ├── delivery/          # gRPC/HTTP handlers (thin!)
│   └── infrastructure/    # DB, external APIs, tools
```

Dependencies: `Delivery → Usecase → Domain ← Infrastructure`

## Architecture (CRITICAL)

### Clean Architecture Layers
- **Domain** — чистые сущности, БЕЗ внешних зависимостей, БЕЗ тегов
- **Usecase** — бизнес-логика + определение интерфейсов (consumer-side)
- **Infrastructure** — реализация интерфейсов (DB, API, tools)
- **Delivery** — тонкие handlers, только трансформация request→usecase→response

### SOLID (обязательно)
- **S** — один struct = одна ответственность. Описание БЕЗ слова "и"
- **O** — расширение через новый код, не изменение существующего
- **L** — подтипы заменяемы
- **I** — маленькие интерфейсы, определённые на стороне потребителя
- **D** — зависимости через интерфейсы

### Consumer-Side Interfaces (ВАЖНО!)
Интерфейсы определяются **В ФАЙЛЕ USECASE**, не в отдельном contract.go:
```go
// usecase/user_create/usecase.go
package user_create

type UserRepository interface {  // ← интерфейс тут
    Create(ctx context.Context, user *domain.User) error
}

type Usecase struct {
    userRepo UserRepository
}
```

### Размеры (триггеры для анализа)
Не жёсткие лимиты, а сигналы что стоит проверить дизайн:
- Файл > 200-300 строк → проверить SRP, возможно стоит разбить
- Метод > 30-50 строк → проверить, можно ли выделить подфункции
- Поля в struct > 5-7 → проверить, не слишком ли много ответственности
- Вложенность > 2-3 уровня → проверить, можно ли инвертировать условия

**Полный гайд:** @.agents/instructions/llm_coding_promt.md

## Go Code Style

### Early Returns (обязательно)
```go
// ✅ ПРАВИЛЬНО — flat, ошибки сверху
func Process(ctx context.Context, id string) (*Order, error) {
    if id == "" {
        return nil, fmt.Errorf("id required")
    }
    order, err := repo.Get(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("get order: %w", err)
    }
    if order == nil {
        return nil, fmt.Errorf("not found")
    }
    return order, nil
}

// ❌ НЕПРАВИЛЬНО — вложенность, else
func Process(ctx context.Context, id string) (*Order, error) {
    if id != "" {
        order, err := repo.Get(ctx, id)
        if err == nil {
            if order != nil {
                return order, nil
            } else {
                return nil, fmt.Errorf("not found")
            }
        } else {
            return nil, err
        }
    } else {
        return nil, fmt.Errorf("id required")
    }
}
```

### Запрещено
- ❌ **goto** — НИКОГДА
- ❌ **else после return** — убирать
- ❌ **Глубокая вложенность** — инвертировать условия
- ❌ **Игнорировать ошибки** — `_ = err` запрещено

### Error Handling
```go
// Всегда оборачивать с контекстом
if err != nil {
    return fmt.Errorf("create user: %w", err)
}
```

### Logging
```go
// Использовать slog с контекстом
slog.InfoContext(ctx, "processing request", "user_id", userID)
slog.ErrorContext(ctx, "failed to save", "error", err)
```

## Принципы разработки

### Качество кода
- **Никаких фоллбеков и костылей** — делать правильно или не делать
- Код должен быть расширяемым и легко поддерживаемым
- **Кросс-платформенность** — решения для всех ОС. OS-специфичный код только если нет альтернативы

### Изменения кода (OCP)
- Сначала понять существующий код, потом менять
- Проектировать так, чтобы будущие доработки не требовали переписывания
- Лишний или плохой код — удалять/переписывать, не пытаться расширять

### Подход к решению проблем
- **Не бросаться фиксить симптом** — сначала понять архитектуру и правильный подход
- Если решение выглядит как "обход ограничения" или "хак" — скорее всего неправильный путь
- **Спросить пользователя** если не уверен в правильном подходе

### Архитектура UI клиента (Interactive)
```
┌─────────────────────────────┐
│  Static (история чата)      │ ← Основная часть
│  - завершённые сообщения    │ ← НЕ перерендеривается
│  - завершённые tool results │ ← Добавляется только финальное состояние
├─────────────────────────────┤
│  Dynamic (UI элементы)      │ ← Перерендеривается
│  - поле ввода               │
│  - кол-во токенов           │
│  - прогресс, статус         │
└─────────────────────────────┘
```
- Tool сначала полностью выполняется (данные накапливаются в state)
- Когда tool завершён (есть результат) → добавляется в Static историю
- **НЕ пытаться перерендерить Static** — это by design
- **НЕ добавлять в Static пока нет финального состояния**

### Планирование
- Разбивать на гранулярные этапы
- Каждый этап должен быть тестируемым — выбирать подходящий способ: unit, integration или headless
- После каждого этапа — тестирование на соответствие требованиям
- Этап закрыт только когда его цель подтверждена тестами
- **Перед завершением работы — проверить что ВСЕ пункты плана выполнены**

### Когда НЕ нужно планирование (Quick Fix)

**Без планирования** — делай сразу, если задача:
- Простое изменение конфига (порт, значение, флаг)
- Правка 1-3 файлов с ясной целью
- Тривиальный баг-фикс (опечатка, неправильное значение, пропущенный импорт)
- Изменение промпта/текста
- Обновление константы, переименование

**Как:** делегируй суб-агенту (backend-developer/frontend-developer) напрямую без EnterPlanMode. Или исправь сам если правка < 50 строк.

**С планированием** — если задача:
- Затрагивает 4+ файлов
- Требует архитектурных решений
- Требует исследования (непонятна причина бага, нужен выбор библиотеки)
- Новая фича с неясными требованиями

## Автономный workflow (КРИТИЧНО)

После согласования плана с пользователем — работай АВТОНОМНО по этому циклу.
Пользователь участвует только в двух точках: согласование плана и финальная проверка.

### Роль главного агента — ОРКЕСТРАТОР, не исполнитель

**Главный агент НЕ пишет код сам.** Его задача — управлять процессом:
- Декомпозировать план на задачи для суб-агентов
- Делегировать реализацию специализированным агентам (backend-developer, frontend-developer)
- Проверять результаты через code-reviewer и tester
- Отправлять на доработку если результат не соответствует плану
- Убедиться что КАЖДЫЙ пункт плана реализован, протестирован и production-ready

**Когда главный агент пишет код сам:**
- Только мелкие правки (50-100 строк) по результатам ревью/тестирования
- Фикс очевидного бага обнаруженного при проверке
- Всё остальное — делегировать суб-агентам

### Цикл работы

```
Пользователь: задача
        ↓
[1. ПЛАНИРОВАНИЕ] — исследуй кодовую базу И внешние решения, составь план
    ├─ Делегируй researcher agent для исследования кода и интернета
    ├─ Составь план с чёткими этапами
    └─ Каждый этап = конкретная задача для суб-агента
        ↓
Пользователь: согласует план
        ↓
[2. РЕАЛИЗАЦИЯ] — для каждого этапа плана:
        ↓
    ┌─ Сформулируй ДЕТАЛЬНОЕ задание для суб-агента:
    │   • Что именно реализовать (со ссылками на файлы и строки)
    │   • Какие интерфейсы/контракты соблюсти
    │   • Какие файлы создать/изменить
    │   • Путь к файлу плана и номер текущего этапа
    │   • Ссылки на существующий код как образец стиля
    ├─ Делегируй backend-developer (Go) или frontend-developer (TS)
    ├─ Получи результат от суб-агента
    └─ Переходи к ревью
        ↓
[3. РЕВЬЮ] — делегируй code-reviewer agent:
    ├─ Передай: путь к плану, номер этапа, список изменённых файлов
    ├─ Агент запускает golangci-lint, tsc --noEmit
    ├─ Агент проверяет SOLID, архитектуру, качество кода
    ├─ **Сверка с планом:** агент ЧИТАЕТ план и проверяет что ВСЕ требования
    │   текущего этапа реализованы — не только код качественный, но и полный
    ├─ Если нашёл блокеры или пропущенные требования:
    │   → Сформулируй задание на исправление → делегируй суб-агенту → повтори ревью
    └─ Если PASS → переходи к тестированию
        ↓
[4. ТЕСТИРОВАНИЕ] — делегируй tester agent:
    ├─ Передай: путь к плану, номер этапа, что именно тестировать
    ├─ Build (go build, bun run build)
    ├─ Unit тесты (go test, bun test)
    ├─ E2E happy path (headless)
    ├─ **Сверка с планом:** тестировать КАЖДОЕ требование этапа, не только happy path
    ├─ Если FAIL или требование не покрыто:
    │   → Сформулируй задание на фикс → делегируй суб-агенту → повтори тестирование
    └─ Если PASS по ВСЕМ требованиям → этап закрыт, следующий этап плана
        ↓
[5. ФИНАЛ] — когда ВСЕ этапы плана выполнены:
    ├─ **Финальная сверка с планом:** прочитать план ЦЕЛИКОМ, пройти по КАЖДОМУ
    │   пункту и убедиться что он реализован и протестирован
    ├─ Если найден пропущенный пункт → делегировать реализацию → ревью → тест
    ├─ Финальный прогон: build + lint + tests (делегируй tester)
    ├─ Если всё чисто → представь результат пользователю с чеклистом выполненных пунктов
    └─ Пользователь проверяет и коммитит
```

### Правила делегирования

**Формат задания для суб-агента (ОБЯЗАТЕЛЬНО):**
```
1. Цель: что реализовать (1-2 предложения)
2. План: путь к файлу плана, номер этапа
3. Контекст: какие файлы прочитать для понимания существующего кода
4. Задачи: конкретный список что сделать (файлы, функции, интерфейсы)
5. Ограничения: стиль кода, архитектурные правила, что НЕ делать
6. Критерий готовности: как понять что задача выполнена
```

**Параллелизация:** если этап затрагивает И бэкенд И фронтенд — запускай backend-developer и frontend-developer ПАРАЛЛЕЛЬНО (в одном сообщении).

**Итерации:** если суб-агент не справился с первого раза — не переделывай за него. Сформулируй конкретные замечания и отправь на доработку тому же агенту (resume).

### Правила автономной работы

1. **Не спрашивай пользователя** по ходу реализации — действуй по плану
2. **Спроси только если:** решение влияет на архитектуру и план не покрывает этот выбор
3. **Не пиши код сам** — делегируй суб-агентам, проверяй результат
4. **Ревью и тестирование обязательны** для каждого этапа, не только в конце
5. **Hooks дают feedback в реальном времени** — если после Edit/Write пришёл lint error, отправь суб-агенту на исправление
6. **Не говори "готово"** пока финальный прогон (build + lint + tests) не пройден
7. **Сверка с планом обязательна** — при ревью и тестировании передавай агенту путь к файлу плана и текущий этап. Агент ДОЛЖЕН прочитать план и проверить что каждое требование этапа выполнено. Пропущенные пункты = блокер
8. **Управляй процессом** — твоя ценность в том, что ты видишь общую картину и гарантируешь полноту выполнения плана, а не в том что ты пишешь код

### Тестирование (ОБЯЗАТЕЛЬНО)

#### Два уровня тестирования — вместе покрывают 100%

Вся система покрывается ДВУМЯ видами тестов. Других видов нет.

```
┌─────────────────────────────────────────────────────────────────────┐
│  Уровень 1: ИНТЕГРАЦИОННЫЕ ТЕСТЫ С РЕНДЕРИНГОМ                     │
│                                                                     │
│  MockChatModel (Go)              ← единственный мок                │
│       ↓                                                             │
│  REACT Agent → Tools → Proxy → gRPC → Client → render(<ChatApp />) │
│       ↓                                              ↓              │
│  Engine + FlowHandler                          lastFrame() →        │
│                                                проверяем что ВИДИТ  │
│                                                пользователь         │
│                                                                     │
│  Покрывает: сервер, gRPC, клиент, pipeline, UI рендеринг           │
│  НЕ покрывает: качество ответов LLM (мокаем)                       │
├─────────────────────────────────────────────────────────────────────┤
│  Уровень 2: PROMPT REGRESSION ТЕСТЫ                                │
│                                                                     │
│  Замороженный контекст → текущий system prompt → реальная LLM       │
│       ↓                                                             │
│  Assertions: tool calls, описания, структура ответа                 │
│                                                                     │
│  Покрывает: качество ответов LLM, промпты, tool schemas            │
│  НЕ покрывает: транспорт, клиент, UI                               │
└─────────────────────────────────────────────────────────────────────┘

Уровень 1 + Уровень 2 = 100% покрытие
```

**Принцип: тестируй что видит пользователь, а не что прошло через pipeline.**

#### Уровень 1: Интеграционные тесты с рендерингом UI

Мокается ТОЛЬКО LLM. Весь остальной стек — production код, **включая рендеринг UI.**

Тест рендерит `<ChatApp />` через `ink-testing-library` и проверяет `lastFrame()` — то, что пользователь реально видит на экране. Не `messageRepository.findComplete()`, а rendered output.

**Инфраструктура:**
- `bytebrew-srv/cmd/testserver/` — Go сервер с `MockChatModel`, сценарии через `--scenario`
- `bytebrew-cli/src/test-utils/TestServerHelper.ts` — build + start/stop Go binary
- `bytebrew-cli/src/presentation/app/__tests__/ChatApp.e2e.test.tsx` — тесты

**Паттерн теста:**
```typescript
import { render } from 'ink-testing-library';

it('smart_search показывает результаты', async () => {
  await server.start('smart-search-scenario');
  const container = createTestContainer(server.port, testDir);
  try {
    const instance = render(<ChatApp container={container} />);
    await connectAndSend(container, 'user message');
    await waitForProcessingStopped(container);

    // Проверяем что ВИДИТ пользователь
    const frame = instance.lastFrame();
    expect(frame).toContain('SmartSearch');
    expect(frame).not.toContain('no results');  // ← ловит баг!
    expect(frame).toContain('[grep]');           // ← ловит баг!
  } finally {
    instance.unmount();
    await container.dispose();
  }
});
```

**Почему render(), а не messageRepository:**
- `messageRepository.findComplete()` проверяет что данные прошли через pipeline
- `render() + lastFrame()` проверяет что пользователь ВИДИТ правильный результат
- Пример бага: `messageRepository` содержит `result: "grep: 5, vector: 3"` (тест прошёл), но рендерер показывает "no results" (пользователь видит баг). Тест с `render()` это ловит, тест с `messageRepository` — нет.

**Как добавлять тест:**
1. Добавить сценарий в `mock_chat_model.go` (switch case по scenario name)
2. Если нужны новые tools — добавить в `helpers.go` → `testFlowConfig()`
3. Написать тест в `ChatApp.e2e.test.tsx` с `render(<ChatApp />)` и assertions на `lastFrame()`
4. **Assertions на rendered output, не на data layer**

**Запуск:** `bun test src/presentation/app/__tests__/ChatApp.e2e.test.tsx`

#### Уровень 2: Prompt Regression Tests

Тесты для проверки что изменения в промпте не ухудшают ответы LLM.

**Принцип:** замороженный контекст из реальной сессии → текущий system prompt → реальная LLM → assertions на структуру ответа.

```
ContextSnapshot JSON (fixture из логов bytebrew-srv/logs/)
    ↓
Harness: config.Load() → ChatModel → BindTools(tool schemas) → Generate(messages)
    ↓
Assertions: tool calls, description quality, response structure
```

**Файлы:**
- `bytebrew-srv/tests/prompt_regression/` — тесты, harness, assertions
- `bytebrew-srv/tests/prompt_regression/fixtures/` — JSON fixtures (замороженные контексты)
- Build tag: `//go:build prompt` — НЕ запускаются в `go test ./...`

**Как создать fixture:**
1. Запустить сервер и выполнить headless запрос
2. Найти нужный `supervisor_step_N_context.json` в `bytebrew-srv/logs/`
3. Обернуть в `{ "name": "...", "description": "...", "snapshot": <содержимое> }`
4. При необходимости добавить system prompt как первое сообщение (REACT agent добавляет его отдельно от message array)
5. Можно добавить синтетические сообщения для однозначного контекста (например, manage_tasks start + response)

**Как добавить тест:**
1. Создать fixture в `fixtures/` (из реальной сессии или синтетический)
2. Написать тест с `//go:build prompt`:
```go
fixture, _ := LoadFixture("my_fixture")
messages := harness.ReconstructMessages(&fixture.Snapshot, "")
result, _ := harness.Generate(ctx, messages)
AssertHasToolCall(t, result, "manage_subtasks")
AssertSubtaskDescriptionQuality(t, result)
```

**Assertions (assertions.go):**
- `AssertHasToolCall(t, msg, toolName)` — ответ содержит tool call
- `AssertToolCallArg(t, msg, toolName, argName)` — аргумент существует и непустой
- `AssertSubtaskDescriptionQuality(t, msg)` — description > title, > 100 символов, не дублирует title

**Запуск:**
```bash
cd bytebrew-srv
go test -tags prompt -v -timeout 300s ./tests/prompt_regression/...
```

**Когда использовать:**
- После изменения system prompt (prompts.yaml) — проверить что ответы не деградировали
- При добавлении новых tool schemas — проверить что LLM корректно вызывает tools
- При тюнинге качества (описания задач, подзадач) — зафиксировать baseline и сравнить

**Context logger** (`internal/infrastructure/agents/context_logger.go`):
- Логирует полный контекст LLM на каждом шаге в `bytebrew-srv/logs/<session>/`
- `ContextSnapshot` — тип для fixtures, содержит Messages с ToolCalls, Arguments, ToolCallID
- Snapshot НЕ включает system prompt (REACT agent добавляет его отдельно)

#### Вспомогательные методы (НЕ замена интеграционных тестов)

| Метод | Когда использовать | Ограничение |
|---|---|---|
| **Unit тесты** | Изолированная логика (парсинг, форматирование) | Не покрывает интеграцию |
| **Headless** | Ad-hoc проверка с реальным сервером | Ручной, не автоматизирован, не рендерит UI |

Unit тесты и headless — дополнение, а не замена. Если фича не покрыта интеграционным тестом с рендерингом — она не протестирована.

### Качество тестирования (КРИТИЧНО)

**Правило:** новая фича или баг-фикс = интеграционный тест с `render(<ChatApp />)`.

**Антипаттерн — тестировать data layer вместо rendered output:**
```typescript
// ❌ НЕПРАВИЛЬНО — проверяет pipeline, не то что видит пользователь
const messages = container.messageRepository.findComplete();
expect(messages.find(m => m.toolCall?.toolName === 'smart_search')).toBeDefined();
// Тест прошёл, но пользователь видит "no results" — баг не пойман

// ✅ ПРАВИЛЬНО — проверяет rendered output
const frame = instance.lastFrame();
expect(frame).toContain('SmartSearch');
expect(frame).not.toContain('no results');
// Тест падает если рендерер показывает "no results" — баг пойман
```

#### Чеклист перед "готово"
```
□ Интеграционный тест написан (testserver + render(<ChatApp />))
□ Тест проверяет RENDERED OUTPUT через lastFrame()
□ Assertions на то что ВИДИТ пользователь, не на data layer
□ bun test ChatApp.e2e.test.tsx — все тесты проходят
□ Код компилируется (bun run build)
□ Если баг — тест воспроизводит баг без фикса, проходит с фиксом
```

После любых изменений, всегда запускай интеграционыый тест который покрывает выполненное изменение!

#### Типичные ошибки

| Ошибка | Почему плохо | Как правильно |
|---|---|---|
| Assertions на `messageRepository` вместо `lastFrame()` | Data может быть "правильной", но рендерер показывает баг | Всегда проверять rendered output |
| `expect(toolMsg).toBeDefined()` и всё | Проверяет наличие, не содержимое | Проверять что конкретно видит пользователь |
| "36 pass / 0 fail" = багов нет | Тесты могут не проверять то что сломано | Assertions должны покрывать конкретный баг |
| Убрал failing assertion вместо фикса | Баг остаётся, тест молчит | Понять ПОЧЕМУ assertion падает, пофиксить причину |

**Полный гайд:** @docs/testing/headless-testing.md

## Commands
```bash
# Server
cd bytebrew-srv && go run ./cmd/server

# CLI (runtime: bun, не node — из-за bun:sqlite)
cd bytebrew-cli && bun run build

# Tests
cd bytebrew-srv && go test ./...

# Kill server (Windows)
netstat -ano | findstr :60401
taskkill /F /PID <pid>
```

## Headless тестирование (КРИТИЧНО — читай внимательно!)

### Тестовый проект (ОБЯЗАТЕЛЬНО)

**НИКОГДА не запускай агент на рабочей директории проекта!** Агент может модифицировать файлы.

Тестовый проект: `C:\Users\busul\GolandProjects\usm-epicsmasher\test-project`

Все headless/interactive тесты запускать ТОЛЬКО с `-C` на тестовый проект:
```bash
cd bytebrew-cli
bun dist/index.js -C C:\Users\busul\GolandProjects\usm-epicsmasher\test-project ask --headless "prompt"
```

**Запрещено:**
- `bun dist/index.js ask --headless "prompt"` — БЕЗ `-C`, агент работает в `bytebrew-cli/`!
- `cd .. && bun bytebrew-cli/dist/index.js ask --headless "prompt"` — агент в корне проекта!
- Любой запуск без `-C test-project` — потенциально опасен

**Правильно:**
```bash
# Из bytebrew-cli (бинарник тут):
bun dist/index.js -C ../test-project ask --headless "prompt"
bun dist/index.js -C ../test-project ask --headless "prompt" --output test-output/tc-X.txt
bun dist/index.js -C ../test-project ask --headless --unknown-cmd allow-once "prompt"

# Multi-turn сессия:
(echo "Вопрос 1"; sleep 30; echo "Вопрос 2") | bun dist/index.js -C ../test-project session
```

### Результаты и тест-кейсы

**Результаты тестов:** `test-output/tc-*.txt`

**Тест-кейсы:** `docs/testing/test-cases.md`

**Важно:** `--output` работает только в headless режиме. В interactive режиме Ink пишет напрямую в stdout, минуя перехват.

## Compact Instructions
При сжатии контекста ОБЯЗАТЕЛЬНО сохранить:
- Ссылку на текущий план (путь к файлу)
- Список выполненных и невыполненных этапов
- Текущий этап работы и его цель
- Не начинать действовать "по наитию" — сначала прочитать план
