# ByteBrew CLI (TypeScript/Bun)

## Stack
- Bun runtime (NOT Node.js — requires bun:sqlite)
- Ink (React for terminal UI), TypeScript strict
- gRPC client, Zustand (viewStore)

## Commands
```bash
bun run build                    # Build
bun test                         # All tests
bun test src/presentation/app/__tests__/ChatApp.e2e.test.tsx  # E2E tests
```

## Architecture: Static vs Dynamic UI
```
┌─────────────────────────────┐
│  Static (история чата)      │ ← НЕ перерендеривается
│  - завершённые сообщения    │
│  - завершённые tool results │ ← Добавляется только финальное состояние
├─────────────────────────────┤
│  Dynamic (UI элементы)      │ ← Перерендеривается
│  - поле ввода, статус       │
└─────────────────────────────┘
```
- Tool завершён → добавляется в Static
- НЕ перерендерить Static — by design
- НЕ добавлять в Static пока нет финального состояния

## Key Patterns
- **EventBus** для inter-component communication (НЕ polling, НЕ setInterval)
- **Immutable Message entity** с factory methods
- **Container (DI)** pattern
- **ink-testing-library** с `await tick()` pattern

## Testing (КРИТИЧНО)

### Integration с рендерингом (Level 1)
```typescript
const instance = render(<ChatApp container={container} />);
await connectAndSend(container, 'user message');
await waitForProcessingStopped(container);
const frame = instance.lastFrame();
expect(frame).toContain('expected text');  // ← rendered output!
```

**Антипаттерн:** assertions на `messageRepository` вместо `lastFrame()`.

### Ink Component Tests
- `const tick = () => new Promise(r => setTimeout(r, 10));`
- ALWAYS `await tick()` after `stdin.write()` before checking state
- Key codes: `\r` (Enter), `\u007f` (Backspace), `\x1b[B` (Down), `\x1b[A` (Up)

## Headless Testing
**НИКОГДА без `-C` на тестовый проект!**
```bash
bun dist/index.js -C ../test-project ask --headless "prompt"
bun dist/index.js -C ../test-project ask --headless "prompt" --output test-output/tc-X.txt
```
