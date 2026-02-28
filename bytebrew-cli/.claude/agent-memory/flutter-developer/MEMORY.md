# Flutter Developer Agent Memory

## Key Bug Patterns

### CLI-to-Mobile type mismatch
- CLI `Message.toSnapshot()` returns `timestamp: Date` which `JSON.stringify()` serializes as ISO 8601 string
- Flutter `MessageMapper` must handle BOTH `int` (ms epoch) AND `String` (ISO 8601) for timestamp
- Fixed in `_parseTimestamp()` method -- check `is int` then `is String` then fallback to `DateTime.now()`
- Same pattern: always use `dynamic` + type checks when parsing JSON from CLI, never assume concrete type with `as T`

## Test Patterns

### WsChatRepository tests
- Use `_FakeWebSocketChannel` with `StreamController<dynamic>.broadcast()` for mock WS
- `channel.receive(jsonEncode({...}))` to simulate incoming data
- `repo.watchMessages().first` to get next emission (use `await` with timeout awareness)
- `await Future<void>.delayed(const Duration(milliseconds: 50))` between sequential events
- Record pattern: `final (:repo, :channel) = await _createConnectedRepo();`

## File Locations
- MessageMapper: `bytebrew-mobile-app/lib/features/chat/infrastructure/message_mapper.dart`
- WsChatRepository: `bytebrew-mobile-app/lib/features/chat/infrastructure/ws_chat_repository.dart`
- CLI MobileProxyServer: `bytebrew-cli/src/infrastructure/mobile/MobileProxyServer.ts`
- CLI Message entity: `bytebrew-cli/src/domain/entities/Message.ts`

## Pre-existing Issues (not from our changes)
- `test/helpers/test_app.dart:37` -- `List<Object>` vs `List<Override>` type error
- `lib/features/auth/infrastructure/cloud_auth_repository.dart:52` -- unused catch clause warning
