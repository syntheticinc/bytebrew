# Frontend Developer Memory

## Key Files
- `src/application/services/handlers/ToolExecutionHandler.ts` — обработка TOOL_CALL, dispatching
- `src/infrastructure/tools/ToolExecutorAdapter.ts` — permission check + local tool execution
- `src/infrastructure/persistence/InMemoryMessageRepository.ts` — репозиторий, findByToolCallId
- `src/config/container.ts` — DI контейнер, ShellSessionManager подключается здесь
- `src/tools/executeCommand.ts` — ExecuteCommandTool, foreground/background/legacy

## Proxied Tools Pattern (execute_command и др.)
Сервер отправляет ДВА TOOL_CALL для каждого proxied tool:
1. `server-execute_command-5` (callback callId) — для отображения в UI
2. `execute_command-5` (proxy callId) — для фактического выполнения

Клиент должен:
- При получении proxy TOOL_CALL — проверять есть ли уже сообщение `server-{callId}`
- Если есть — переиспользовать его messageId, не создавать новое сообщение
- Так избегается дублирование tool call в UI

Реализовано в `handleToolCall()`:
```typescript
const serverCallId = `server-${toolCall.callId}`;
const serverMessage = ctx.messageRepository.findByToolCallId(serverCallId);
if (serverMessage) {
  // reuse message, start execution only
}
```

## Defensive Error Handling в handleToolCall
Синхронная часть `handleToolCall` (создание Message, сохранение в repo) обёрнута в try/catch.
Если синхронная часть падает для client-side tool — отправить error result серверу через
`ctx.streamGateway.sendToolResult(callId, '[ERROR] ...', err)` чтобы proxy не завис навсегда.

## Logging Pattern
Ключевые точки для дебага зависания execute_command:
- `handleToolCall`: "TOOL_CALL received" (callId, toolName, isServerSide)
- `executeToolAsync`: "start", "completed", "Sending tool result to server"
- `ToolExecutorAdapter.execute`: "start", "server-side tool skip", "Permission check result"
- Все через `getLogger()` из `src/lib/logger.ts`

## Тестирование
- E2E тесты: `bun test src/presentation/app/__tests__/ChatApp.e2e.test.tsx`
- Mobile Chain E2E: `bun test tests/e2e/mobile-chain-e2e.test.ts`
- Build: `bun run build` (в bytebrew-cli/)
- Таймаут E2E: 180s (Go server build + тесты)

## Mobile Chain E2E Infrastructure
- `src/test-utils/TestServerHelper.ts` — Go test server (bytebrew-srv)
- `src/test-utils/BridgeHelper.ts` — Go bridge relay (bytebrew-bridge)
- `tests/e2e/WsMobileSimulator.ts` — WS client simulating mobile app
- Bridge config: env vars `BRIDGE_PORT`, `BRIDGE_AUTH_TOKEN`
- Bridge validates port >= 1 (no port 0), use findFreePort() + health poll
- Container needs `streamGateway.connect()` before sendMessage works (normally done by useStreamConnection hook)

## Bridge DeviceId Bug (Fixed 2026-03-06)
EventBroadcaster.subscribe must use bridge-level deviceId (from URL query param), NOT authenticated deviceId (from PairingService). They are different UUIDs. Without this, bridge drops events because it doesn't find the device. Fix in `MobileRequestHandler.dispatch`: subscribe/unsubscribe use bridge-level `deviceId` parameter.
