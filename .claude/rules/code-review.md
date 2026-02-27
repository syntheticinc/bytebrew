# Code Review Guidelines

## Когда делать Code Review

### Самопроверка (ОБЯЗАТЕЛЬНО)
Перед завершением любого этапа работы — проверить свой код по этому гайду:
1. После написания нового кода
2. После рефакторинга
3. Перед коммитом
4. Перед закрытием этапа плана

### Триггеры для немедленного review
- Файл превысил 200 строк
- Метод превысил 30 строк
- Добавлено больше 5 полей в struct
- Добавлен новый mutex
- Код дублируется

---

## Процесс Code Review

### Шаг 1: Проверка размеров (триггеры для анализа)
Не жёсткие лимиты — сигналы что стоит проверить дизайн:
```
□ Файл > 300 строк? → проверить SRP, возможно стоит разбить
□ Метод > 50 строк? → проверить, можно ли выделить подфункции
□ Поля в struct > 7? → проверить, не слишком ли много ответственности
□ Вложенность > 3 уровня? → проверить, можно ли упростить
```

**Если триггер сработал** → проанализировать причину, не обязательно сразу рефакторить

### Шаг 2: SOLID Checklist

#### S — Single Responsibility
```
□ Struct делает ОДНУ вещь?
□ Можно описать ответственность БЕЗ слова "и"?
□ Один файл = один struct?
```

**Тест:** Попробуй описать что делает класс одним предложением без "и":
- ✅ "UserRepository сохраняет пользователей в БД"
- ❌ "AgentCallbackHandler обрабатывает callbacks И управляет планом И накапливает reasoning И считает шаги"

**Как исправить:** Разбить на отдельные компоненты:
```
AgentCallbackHandler (488 строк) →
├── ModelEventHandler.go      (обработка model events)
├── ToolEventHandler.go       (обработка tool events)
├── ReasoningAccumulator.go   (накопление reasoning)
├── StepCounter.go            (управление шагами)
└── PlanProgressEmitter.go    (события плана)
```

#### O — Open/Closed
```
□ Можно добавить новое поведение БЕЗ изменения существующего кода?
□ Используются интерфейсы/стратегии для расширения?
□ Нет switch/case по типам которые будут расширяться?
```

**Пример проблемы:**
```go
// ❌ Каждый новый EventType требует изменения этого switch
func handleEvent(e Event) {
    switch e.Type {
    case "tool_call": ...
    case "answer": ...
    case "reasoning": ...  // Добавление нового типа = изменение кода
    }
}

// ✅ Новый тип = новый handler, существующий код не меняется
type EventHandler interface {
    Handle(ctx context.Context, e Event) error
}

type ToolCallHandler struct{}
type AnswerHandler struct{}
// Добавление ReasoningHandler не требует изменения существующего кода
```

#### I — Interface Segregation
```
□ Интерфейсы маленькие (1-3 метода)?
□ Клиент использует ВСЕ методы интерфейса?
□ Нет "пустых" реализаций методов?
```

**Пример проблемы:**
```go
// ❌ Слишком большой интерфейс
type Repository interface {
    Create(ctx context.Context, u *User) error
    Update(ctx context.Context, u *User) error
    Delete(ctx context.Context, id string) error
    GetByID(ctx context.Context, id string) (*User, error)
    GetByEmail(ctx context.Context, email string) (*User, error)
    List(ctx context.Context) ([]*User, error)
    Search(ctx context.Context, query string) ([]*User, error)
}

// ✅ Маленькие интерфейсы на стороне потребителя
// В usecase/user_create/usecase.go:
type UserCreator interface {
    Create(ctx context.Context, u *User) error
}

// В usecase/user_get/usecase.go:
type UserReader interface {
    GetByID(ctx context.Context, id string) (*User, error)
}
```

#### D — Dependency Inversion
```
□ Зависимости через интерфейсы?
□ Интерфейсы определены на стороне ПОТРЕБИТЕЛЯ (в usecase)?
□ Конструктор принимает интерфейсы, не конкретные типы?
```

### Шаг 3: Go Code Style
```
□ Early returns? (ошибки сверху, happy path внизу)
□ Нет else после return?
□ Нет goto?
□ Ошибки обёрнуты с контекстом? (fmt.Errorf("context: %w", err))
□ Используется slog.InfoContext/ErrorContext?
□ context.Context первый параметр везде?
```

### Шаг 4: Тестируемость
```
□ Можно протестировать в изоляции?
□ Все зависимости через интерфейсы (легко мокать)?
□ Нет глобальных переменных/синглтонов?
□ Нет скрытых зависимостей?
```

### Шаг 5: Качество тестирования (КРИТИЧНО)

**Два уровня тестирования, вместе покрывают 100%:**
1. **Интеграционные с рендерингом** — mock LLM → весь стек → `render(<ChatApp />)` → `lastFrame()` → проверяем что ВИДИТ пользователь
2. **Prompt regression** — замороженный контекст → реальная LLM → assertions на ответ

**Ключевой принцип: тестируй rendered output, а не data layer.**

```
□ Код скомпилирован без ошибок? (bun run build, go build)
□ Интеграционный тест с render(<ChatApp />) написан?
□ Assertions на lastFrame() — что ВИДИТ пользователь?
□ bun test ChatApp.e2e.test.tsx — все тесты проходят?
□ Если баг — тест воспроизводит баг без фикса?
□ Prompt regression тест если менялся промпт?
```

**Антипаттерн — тестирование data layer вместо rendered output:**
```
❌ container.messageRepository.findComplete() → проверяет data, не UI
   Баг: данные "правильные", но рендерер показывает "no results"
   Тест прошёл, пользователь видит баг

✅ instance.lastFrame() → проверяет что видит пользователь
   Тест падает когда рендерер показывает "no results"
   Баг пойман
```

**Типичные ошибки:**
- ❌ Assertions на `messageRepository` вместо `lastFrame()` → не ловит баги рендеринга
- ❌ "Посмотрел код — выглядит правильно" → нужно ЗАПУСТИТЬ тест
- ❌ Фикс бага без теста на воспроизведение
- ❌ `expect(msg).toBeDefined()` и всё → проверяет наличие, не содержимое
- ❌ Assertion падает → убрал assertion вместо фикса бага
- ❌ "36 pass / 0 fail" = багов нет → тесты могут не покрывать конкретный баг

### Шаг 6: Состояние
```
□ Минимум mutable state?
□ Один mutex максимум? (больше = разбить struct)
□ Состояние инкапсулировано?
□ Нет публичных полей которые меняются?
```

---

## Red Flags

### Критические (требуют обоснования или исправления)
- [ ] goto — всегда исправлять
- [ ] Игнорирование ошибок (`_ = err`) — всегда исправлять
- [ ] God object (struct делает много разных вещей) — разбить
- [ ] Больше одного mutex в struct — проанализировать, скорее всего нарушен SRP

### Серьёзные (обсудить/исправить)
- [ ] else после return — убрать
- [ ] switch по типам без стратегии расширения — продумать OCP
- [ ] Отсутствие unit тестов для новой логики
- [ ] Дублирование кода
- [ ] Решение выглядит как "обход ограничения" или хак

### Подход к решению (спросить себя)
- Это фикс симптома или решение проблемы?
- Понимаю ли я архитектуру того что меняю?
- Если "как заставить X делать Y" — возможно неправильный путь
- Если не уверен — спросить пользователя прежде чем фиксить

### Триггеры для анализа (не автоматическое исправление)
- [ ] Файл > 300 строк — проверить SRP
- [ ] Метод > 50 строк — проверить возможность разбиения
- [ ] Глубокая вложенность (> 3 уровня) — проверить инверсию условий
- [ ] Интерфейс > 5 методов — проверить ISP
- [ ] Сложная логика без комментария "почему"

---

## Примеры рефакторинга

### Пример 1: Разбиение большого файла

**До:** `agent_callback_handler.go` (488 строк, 5 ответственностей)

**После:**
```
internal/infrastructure/agents/
├── callback_handler.go           # Координатор (50 строк)
├── model_event_handler.go        # Model callbacks (100 строк)
├── tool_event_handler.go         # Tool callbacks (80 строк)
├── reasoning_accumulator.go      # Reasoning state (60 строк)
├── step_counter.go               # Step management (40 строк)
└── plan_progress_emitter.go      # Plan events (50 строк)
```

### Пример 2: Выделение интерфейса

**До:**
```go
type Usecase struct {
    db *gorm.DB  // Конкретный тип
}

func (u *Usecase) CreateUser(ctx context.Context, email string) error {
    return u.db.Create(&User{Email: email}).Error
}
```

**После:**
```go
// В том же файле usecase.go
type UserRepository interface {
    Create(ctx context.Context, user *User) error
}

type Usecase struct {
    repo UserRepository  // Интерфейс
}

func (u *Usecase) CreateUser(ctx context.Context, email string) error {
    return u.repo.Create(ctx, &User{Email: email})
}
```

### Пример 3: Устранение вложенности

**До:**
```go
func Process(items []Item) error {
    if len(items) > 0 {
        for _, item := range items {
            if item.IsValid() {
                if item.NeedsProcessing() {
                    if err := process(item); err != nil {
                        return err
                    }
                }
            }
        }
    }
    return nil
}
```

**После:**
```go
func Process(items []Item) error {
    if len(items) == 0 {
        return nil
    }

    for _, item := range items {
        if err := processItem(item); err != nil {
            return err
        }
    }
    return nil
}

func processItem(item Item) error {
    if !item.IsValid() {
        return nil
    }
    if !item.NeedsProcessing() {
        return nil
    }
    return process(item)
}
```

---

## Сверка с планом (ОБЯЗАТЕЛЬНО)

При ревью агент ДОЛЖЕН:
1. **Прочитать файл плана** (путь передаётся при делегировании)
2. **Найти текущий этап** и его требования
3. **Пройти по КАЖДОМУ пункту** этапа и проверить:
   - Пункт реализован в коде?
   - Реализация соответствует описанию в плане?
   - Нет ли отклонений от требований?
4. **Перечислить пропущенные пункты** — это блокер, этап не может быть закрыт

**Формат отчёта по сверке:**
```
## Сверка с планом (Этап N: название)

✅ Пункт 1 — реализован (файл:строка)
✅ Пункт 2 — реализован (файл:строка)
❌ Пункт 3 — НЕ РЕАЛИЗОВАН: описание что пропущено
⚠️ Пункт 4 — частично: реализовано X, но не Y

Пропущено: 1 пункт → БЛОКЕР
```

**Типичные причины пропусков:**
- Агент "забыл" пункт при реализации
- Агент решил что пункт неважен и пропустил
- Агент реализовал похожее, но не то что описано в плане
- Агент сделал часть пункта, но не весь

---

## Чеклист перед завершением этапа

```
□ План прочитан, КАЖДЫЙ пункт текущего этапа проверен
□ Нет пропущенных требований из плана
□ Триггеры размеров проверены (если сработали — проанализированы)
□ SOLID принципы соблюдены (или есть обоснование отступления)
□ Go code style соблюдён
□ Тесты написаны и проходят
□ Критические red flags отсутствуют
□ Код можно понять без объяснений автора
```

**Критические red flags (goto, игнорирование ошибок, god object) — блокируют закрытие этапа.**
**Пропущенные пункты плана — блокируют закрытие этапа.**
**Триггеры размеров — требуют анализа, но не обязательно рефакторинга.**
