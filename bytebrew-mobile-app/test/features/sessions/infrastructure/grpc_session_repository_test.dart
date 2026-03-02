import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart' hide Server;

import 'package:bytebrew_mobile/core/domain/mobile_session.dart';
import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/domain/session.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/mobile_service_client.dart';
import 'package:bytebrew_mobile/features/sessions/infrastructure/grpc_session_repository.dart';

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

/// Fake [MobileServiceClient] that returns a configurable list of sessions.
class FakeMobileServiceClient implements MobileServiceClient {
  bool closeCalled = false;
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
  Future<void> close() async {
    closeCalled = true;
  }

  @override
  Future<PairResult> pair({
    required String token,
    required String deviceName,
    Uint8List? mobilePublicKey,
  }) async {
    throw UnimplementedError();
  }

  @override
  Future<ListSessionsResult> listSessions({
    required String deviceToken,
  }) async {
    listSessionsCalled = true;
    if (listSessionsError != null) {
      throw listSessionsError!;
    }
    return ListSessionsResult(
      sessions: sessionsToReturn,
      serverName: 'Test Server',
      serverId: 'test-server-id',
    );
  }

  @override
  Stream<SessionEvent> subscribeSession({
    required String deviceToken,
    required String sessionId,
    String? lastEventId,
  }) {
    return const Stream.empty();
  }

  @override
  Future<SendCommandResult> sendNewTask({
    required String deviceToken,
    required String sessionId,
    required String task,
  }) async {
    return const SendCommandResult(success: true);
  }

  @override
  Future<SendCommandResult> sendAskUserReply({
    required String deviceToken,
    required String sessionId,
    required String question,
    required String answer,
  }) async {
    return const SendCommandResult(success: true);
  }

  @override
  Future<SendCommandResult> cancelSession({
    required String deviceToken,
    required String sessionId,
  }) async {
    return const SendCommandResult(success: true);
  }
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

/// Creates a test server with a device token for use with ConnectionManager.
Server _testServer({
  String id = 'srv-1',
  String name = 'Dev Workstation',
  String deviceToken = 'token-abc',
}) {
  return Server(
    id: id,
    name: name,
    lanAddress: '192.168.1.100',
    connectionMode: ConnectionMode.lan,
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
  group('GrpcSessionRepository', () {
    late ConnectionManager connectionManager;
    late FakeMobileServiceClient fakeClient;
    late GrpcSessionRepository repo;

    setUp(() {
      fakeClient = FakeMobileServiceClient();
      connectionManager = ConnectionManager(
        clientFactory: (_) => fakeClient,
      );
      repo = GrpcSessionRepository(connectionManager: connectionManager);
    });

    tearDown(() async {
      await connectionManager.disconnectAll();
    });

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

        await connectionManager.connectToServer(_testServer());

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

        await connectionManager.connectToServer(_testServer());

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

        await connectionManager.connectToServer(_testServer());

        final sessions = await repo.listSessions();

        expect(sessions.first.currentTask, 'Refactoring module');
      });

      test('maps hasAskUser correctly', () async {
        fakeClient.sessionsToReturn = [
          _mobileSess(sessionId: 'sess-ask', hasAskUser: true),
        ];

        await connectionManager.connectToServer(_testServer());

        final sessions = await repo.listSessions();

        expect(sessions.first.hasAskUser, isTrue);
      });
    });

    group('status mapping', () {
      Future<SessionStatus> _statusFor(MobileSessionState grpcState) async {
        fakeClient.sessionsToReturn = [
          _mobileSess(sessionId: 'sess-status', status: grpcState),
        ];

        await connectionManager.connectToServer(_testServer());

        final sessions = await repo.listSessions();
        return sessions.first.status;
      }

      test('maps active to active', () async {
        final status = await _statusFor(MobileSessionState.active);
        expect(status, SessionStatus.active);
      });

      test('maps idle to idle', () async {
        final status = await _statusFor(MobileSessionState.idle);
        expect(status, SessionStatus.idle);
      });

      test('maps needsAttention to needsAttention', () async {
        final status = await _statusFor(MobileSessionState.needsAttention);
        expect(status, SessionStatus.needsAttention);
      });

      test('maps completed to idle', () async {
        final status = await _statusFor(MobileSessionState.completed);
        expect(status, SessionStatus.idle);
      });

      test('maps failed to idle', () async {
        final status = await _statusFor(MobileSessionState.failed);
        expect(status, SessionStatus.idle);
      });

      test('maps unspecified to idle', () async {
        final status = await _statusFor(MobileSessionState.unspecified);
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

        await connectionManager.connectToServer(_testServer());

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

        await connectionManager.connectToServer(_testServer());

        final sessions = await repo.listSessions();

        expect(sessions[0].id, 'new-active');
        expect(sessions[1].id, 'old-active');
      });
    });

    group('error handling', () {
      test('silently skips server that throws GrpcError', () async {
        fakeClient.listSessionsError = GrpcError.unavailable('down');

        await connectionManager.connectToServer(_testServer());

        final sessions = await repo.listSessions();

        expect(sessions, isEmpty);
      });

      test('silently skips server that throws generic Exception', () async {
        fakeClient.listSessionsError = Exception('network failure');

        await connectionManager.connectToServer(_testServer());

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

        await connectionManager.connectToServer(_testServer());

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

        await connectionManager.connectToServer(_testServer());

        final sessions = await repo.listSessions();

        expect(sessions.first.projectName, 'fallback-name');
      });
    });
  });
}
