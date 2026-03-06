import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart'
    hide WsConnectionStatus;
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_types.dart';
import 'package:bytebrew_mobile/features/sessions/infrastructure/ws_session_repository.dart';

import '../../../helpers/fakes.dart';

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

/// Fake [WsBridgeClient] that returns a configurable list of sessions.
class _FakeWsBridgeClient implements WsBridgeClient {
  bool listSessionsCalled = false;

  /// Sessions to return from [listSessions].
  List<MobileSession> sessionsToReturn = [];

  /// If non-null, [listSessions] will throw this error.
  Object? listSessionsError;

  @override
  Future<PingResult> ping() async {
    return PingResult(
      timestamp: DateTime.now(),
      serverName: 'Test Server',
      serverId: 'test-server-id',
    );
  }

  @override
  Future<ListSessionsResult> listSessions({required String deviceToken}) async {
    listSessionsCalled = true;
    if (listSessionsError != null) throw listSessionsError!;
    return ListSessionsResult(
      sessions: sessionsToReturn,
      serverName: 'Test Server',
      serverId: 'test-server-id',
    );
  }

  @override
  Future<void> dispose() async {}

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// Fake WsConnection stub.
class _FakeWsConnection implements WsConnection {
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// Helper to create a [MobileSession] for tests.
MobileSession _mobileSess({
  String sessionId = 'sess-1',
  String projectKey = 'proj-key',
  String projectRoot = '/home/user/project',
  MobileSessionState status = MobileSessionState.active,
  String currentTask = 'Analyzing code',
  bool hasAskUser = false,
  DateTime? lastActivityAt,
}) {
  return MobileSession(
    sessionId: sessionId,
    projectKey: projectKey,
    projectRoot: projectRoot,
    status: status,
    currentTask: currentTask,
    startedAt: DateTime(2026, 1, 1),
    lastActivityAt: lastActivityAt ?? DateTime(2026, 3, 1, 12, 0),
    hasAskUser: hasAskUser,
    platform: 'linux',
  );
}

/// Creates a test server with a device token.
Server _testServer({
  String id = 'srv-1',
  String name = 'Dev Workstation',
  String deviceToken = 'token-abc',
}) {
  return Server(
    id: id,
    name: name,
    bridgeUrl: 'ws://bridge:8080',
    isOnline: true,
    latencyMs: 10,
    pairedAt: DateTime(2026, 1, 1),
    deviceToken: deviceToken,
  );
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('WsSessionRepository', () {
    late FakeConnectionManager connectionManager;
    late _FakeWsBridgeClient fakeClient;
    late WsSessionRepository repo;

    setUp(() {
      fakeClient = _FakeWsBridgeClient();
      connectionManager = FakeConnectionManager();
      repo = WsSessionRepository(connectionManager: connectionManager);
    });

    /// Adds a fake connected server to the connection manager.
    void _addConnectedServer({Server? server}) {
      final s = server ?? _testServer();
      final conn = WsServerConnection(
        server: s,
        connection: _FakeWsConnection(),
        client: fakeClient,
      )..status = WsConnectionStatus.connected;
      connectionManager.addFakeConnection(s.id, conn);
    }

    group('listSessions', () {
      test('returns empty list when no servers are connected', () async {
        final sessions = await repo.listSessions();

        expect(sessions, isEmpty);
      });

      test('returns sessions from connected server', () async {
        fakeClient.sessionsToReturn = [
          _mobileSess(
            sessionId: 'sess-1',
            projectRoot: '/home/user/my-app',
            status: MobileSessionState.active,
            currentTask: 'Building feature',
          ),
          _mobileSess(
            sessionId: 'sess-2',
            projectRoot: '/home/user/other-app',
            status: MobileSessionState.idle,
            currentTask: '',
          ),
        ];

        _addConnectedServer();

        final sessions = await repo.listSessions();

        expect(sessions, hasLength(2));
        expect(sessions[0].id, 'sess-1');
        expect(sessions[0].projectName, 'my-app');
        expect(sessions[0].serverId, 'srv-1');
        expect(sessions[0].serverName, 'Test Server');
      });

      test('maps currentTask to null when empty', () async {
        fakeClient.sessionsToReturn = [
          _mobileSess(sessionId: 'sess-empty-task', currentTask: ''),
        ];

        _addConnectedServer();

        final sessions = await repo.listSessions();

        expect(sessions.first.currentTask, isNull);
      });

      test('preserves non-empty currentTask', () async {
        fakeClient.sessionsToReturn = [
          _mobileSess(
            sessionId: 'sess-with-task',
            currentTask: 'Refactoring module',
          ),
        ];

        _addConnectedServer();

        final sessions = await repo.listSessions();

        expect(sessions.first.currentTask, 'Refactoring module');
      });

      test('maps hasAskUser correctly', () async {
        fakeClient.sessionsToReturn = [
          _mobileSess(sessionId: 'sess-ask', hasAskUser: true),
        ];

        _addConnectedServer();

        final sessions = await repo.listSessions();

        expect(sessions.first.hasAskUser, isTrue);
      });
    });

    group('status mapping', () {
      Future<SessionStatus> statusFor(MobileSessionState state) async {
        fakeClient.sessionsToReturn = [
          _mobileSess(sessionId: 'sess-status', status: state),
        ];

        _addConnectedServer();

        final sessions = await repo.listSessions();
        return sessions.first.status;
      }

      test('maps active to active', () async {
        final status = await statusFor(MobileSessionState.active);
        expect(status, SessionStatus.active);
      });

      test('maps idle to idle', () async {
        final status = await statusFor(MobileSessionState.idle);
        expect(status, SessionStatus.idle);
      });

      test('maps needsAttention to needsAttention', () async {
        final status = await statusFor(MobileSessionState.needsAttention);
        expect(status, SessionStatus.needsAttention);
      });

      test('maps completed to idle', () async {
        final status = await statusFor(MobileSessionState.completed);
        expect(status, SessionStatus.idle);
      });

      test('maps failed to idle', () async {
        final status = await statusFor(MobileSessionState.failed);
        expect(status, SessionStatus.idle);
      });

      test('maps unspecified to idle', () async {
        final status = await statusFor(MobileSessionState.unspecified);
        expect(status, SessionStatus.idle);
      });
    });

    group('sorting', () {
      test('sorts needsAttention before active before idle', () async {
        final now = DateTime(2026, 3, 1, 12, 0);
        fakeClient.sessionsToReturn = [
          _mobileSess(
            sessionId: 'idle-1',
            status: MobileSessionState.idle,
            lastActivityAt: now,
          ),
          _mobileSess(
            sessionId: 'attention-1',
            status: MobileSessionState.needsAttention,
            lastActivityAt: now,
          ),
          _mobileSess(
            sessionId: 'active-1',
            status: MobileSessionState.active,
            lastActivityAt: now,
          ),
        ];

        _addConnectedServer();

        final sessions = await repo.listSessions();

        expect(sessions[0].id, 'attention-1');
        expect(sessions[1].id, 'active-1');
        expect(sessions[2].id, 'idle-1');
      });

      test('within same status, sorts by most recent activity first', () async {
        fakeClient.sessionsToReturn = [
          _mobileSess(
            sessionId: 'old-active',
            status: MobileSessionState.active,
            lastActivityAt: DateTime(2026, 3, 1, 10, 0),
          ),
          _mobileSess(
            sessionId: 'new-active',
            status: MobileSessionState.active,
            lastActivityAt: DateTime(2026, 3, 1, 12, 0),
          ),
        ];

        _addConnectedServer();

        final sessions = await repo.listSessions();

        expect(sessions[0].id, 'new-active');
        expect(sessions[1].id, 'old-active');
      });
    });

    group('error handling', () {
      test('silently skips server that throws Exception', () async {
        fakeClient.listSessionsError = Exception('network failure');

        _addConnectedServer();

        final sessions = await repo.listSessions();

        expect(sessions, isEmpty);
      });
    });

    group('refresh', () {
      test('completes without error', () async {
        await expectLater(repo.refresh(), completes);
      });
    });

    group('projectName derivation', () {
      test('uses last path segment from projectRoot', () async {
        fakeClient.sessionsToReturn = [
          _mobileSess(
            sessionId: 'sess-proj',
            projectRoot: '/home/dev/bytebrew-srv',
          ),
        ];

        _addConnectedServer();

        final sessions = await repo.listSessions();

        expect(sessions.first.projectName, 'bytebrew-srv');
      });

      test('uses projectKey when projectRoot is empty', () async {
        fakeClient.sessionsToReturn = [
          _mobileSess(
            sessionId: 'sess-noroot',
            projectRoot: '',
            projectKey: 'fallback-name',
          ),
        ];

        _addConnectedServer();

        final sessions = await repo.listSessions();

        expect(sessions.first.projectName, 'fallback-name');
      });
    });
  });
}
