# Тест отображения вопроса в AskUserPrompt

## Цель
Проверить что вопрос `ask_user` отображается над input полем в interactive режиме.

## Изменения
- `AskUserPrompt.tsx` теперь рендерит `question` prop над input полем
- Первая строка вопроса с префиксом `> ` (cyan bold)
- Остальные строки — plain text
- Длинные вопросы (>10 строк) усекаются: первые 8 строк + "..." + последняя строка

## Визуальная проверка

### 1. Короткий вопрос (простой)
```
> Do you approve this task?

┌──────────────────────────────────┐
│ Your answer (Enter = approved) ▌ │
└──────────────────────────────────┘
Enter to send (empty = approved)
```

### 2. Многострочный вопрос (Task approval)
```
> ## Task: Add Health Check Endpoint
- Create /health endpoint
- Add timeout parameter
- Write unit tests

Approve this task?

┌──────────────────────────────────┐
│ Your answer (Enter = approved) ▌ │
└──────────────────────────────────┘
Enter to send (empty = approved)
```

### 3. Длинный вопрос (>10 строк, усечённый)
```
> ## Summary
Line 2
Line 3
...
Line 15 (last line)

┌──────────────────────────────────┐
│ Your answer (Enter = approved) ▌ │
└──────────────────────────────────┘
Enter to send (empty = approved)
```

## Тестовая команда
Запустить interactive CLI и дождаться `ask_user` промпта:
```bash
cd vector-cli-node
bun dist/index.js
```

Ожидаемый результат:
- Вопрос виден НАД input полем
- Первая строка с префиксом `> ` (cyan bold)
- Input field сохраняет своё поведение
- Hint text под input полем

## Unit тесты
```bash
cd vector-cli-node
bun test src/presentation/components/__tests__/AskUserPrompt.test.tsx
```

Тесты проверяют:
- Отображение вопроса
- Префикс `> ` на первой строке
- Усечение длинных вопросов (>10 строк)
- Поведение input field (ввод текста, Enter, default answer)

## Статус
✅ Код изменён
✅ Unit тесты обновлены и проходят (21 pass)
✅ Код собирается без ошибок
⏳ Требуется визуальная проверка в interactive режиме
