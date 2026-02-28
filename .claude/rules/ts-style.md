---
paths:
  - "bytebrew-cli/**/*.ts"
  - "bytebrew-cli/**/*.tsx"
  - "bytebrew-cloud-web/**/*.ts"
  - "bytebrew-cloud-web/**/*.tsx"
---

# TypeScript Code Style

## Runtime
- bytebrew-cli: **Bun** (NOT Node.js — requires bun:sqlite)
- bytebrew-cloud-web: Node.js + Vite

## Strict TypeScript
- `strict: true` — no `any` без обоснования
- Explicit return types для public API
- Prefer `interface` over `type` для объектов

## Patterns
- **Immutable entities** с factory methods
- **EventBus** для inter-component communication (НЕ polling)
- **Container (DI)** — зависимости через конструктор

## Testing (ink-testing-library)
```typescript
const tick = () => new Promise(r => setTimeout(r, 10));
// ALWAYS await tick() after stdin.write() before checking state
```

## Assertions: rendered output, НЕ data layer
```typescript
// ✅ ПРАВИЛЬНО
const frame = instance.lastFrame();
expect(frame).toContain('expected');

// ❌ НЕПРАВИЛЬНО
const messages = container.messageRepository.findComplete();
```
