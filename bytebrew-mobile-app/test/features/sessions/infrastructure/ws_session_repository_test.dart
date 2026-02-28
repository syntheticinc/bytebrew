import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';
import 'package:bytebrew_mobile/features/sessions/infrastructure/ws_session_repository.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('WsSessionRepository', () {
    test('returns one session when meta is present', () async {
      final wsRepo = _FakeWsChat(
        meta: {
          'sessionId': 'sess-123',
          'projectName': 'my-project',
          'projectPath': '/home/user/my-project',
        },
      );

      final repo = WsSessionRepository(wsRepo);
      final sessions = await repo.listSessions();

      expect(sessions, hasLength(1));

      final session = sessions.first;
      expect(session.id, 'sess-123');
      expect(session.projectName, 'my-project');
      expect(session.serverName, 'CLI');
      expect(session.serverId, 'ws-connected');
      expect(session.status, SessionStatus.active);
      expect(session.hasAskUser, isFalse);
      expect(session.currentTask, isNull);
    });

    test('returns empty list when meta is null', () async {
      final wsRepo = _FakeWsChat(meta: null);

      final repo = WsSessionRepository(wsRepo);
      final sessions = await repo.listSessions();

      expect(sessions, isEmpty);
    });

    test('returns empty list when wsRepo is null', () async {
      final repo = WsSessionRepository(null);
      final sessions = await repo.listSessions();

      expect(sessions, isEmpty);
    });

    test('uses fallback values for missing meta fields', () async {
      final wsRepo = _FakeWsChat(meta: <String, dynamic>{});

      final repo = WsSessionRepository(wsRepo);
      final sessions = await repo.listSessions();

      expect(sessions, hasLength(1));
      expect(sessions.first.id, 'live');
      expect(sessions.first.projectName, 'Unknown');
    });

    test('refresh completes without error', () async {
      final repo = WsSessionRepository(null);

      // Should not throw.
      await repo.refresh();
    });
  });
}

/// Minimal fake [WsChatRepository] that bypasses WebSocket entirely and
/// exposes a controllable [meta] value.
///
/// Only the [meta] getter is used by [WsSessionRepository]; all other members
/// delegate to [noSuchMethod] which will throw if invoked unexpectedly.
class _FakeWsChat implements WsChatRepository {
  _FakeWsChat({this.meta});

  @override
  final Map<String, dynamic>? meta;

  @override
  String get wsUrl => 'ws://fake';

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}
