# Frontend Developer Memory

## Bun mock fetch pattern

```typescript
import { mock } from 'bun:test';
const fetchMock = mock(() => Promise.resolve(new Response('{}', { status: 200 })));
globalThis.fetch = fetchMock as unknown as typeof fetch;
// Access calls: fetchMock.mock.calls[0] as [string, RequestInit]
// mockReturnValueOnce works for sequential mock responses
```

## AuthTokens interface location

`AuthTokens` определён в `src/infrastructure/auth/AuthStorage.ts` (Stage 1 onboarding).
`CloudApiClient.ts` переопределяет свой локальный `AuthTokens` — они совпадают по shape, будут объединены позже.

## Infrastructure paths

- `src/infrastructure/auth/AuthStorage.ts` — хранение токенов (~/.vector/auth.json)
- `src/infrastructure/config/VectorHome.ts` — cross-platform пути к ~/.vector/
- `src/infrastructure/api/CloudApiClient.ts` — HTTP client для Cloud API (Stage 2)

## Cloud API

- Base URL: `VECTOR_CLOUD_URL` env или `http://localhost:60402`
- Все успешные ответы обёрнуты в `{ "data": { ... } }`
- Ошибки: `{ "error": { "code": "...", "message": "..." } }`
- `/auth/refresh` endpoint не реализован на сервере — `doRefreshToken()` бросает `TOKEN_EXPIRED`
