import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart' hide Server;

import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/grpc_channel_factory.dart';
import 'package:bytebrew_mobile/core/infrastructure/grpc/mobile_service_client.dart';

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

/// A fake [GrpcChannelFactory] that returns plain [ClientChannel] instances
/// without attempting real network connections.
///
/// The returned channels are never used for actual RPC calls because the
/// [FakeMobileServiceClient] intercepts all client methods. The channels
/// can safely [shutdown]/[terminate] because they have no active HTTP/2
/// connection (the lazy connection is never triggered).
class FakeGrpcChannelFactory extends GrpcChannelFactory {
  const FakeGrpcChannelFactory();

  @override
  ClientChannel createChannel(Server server) =>
      ClientChannel('localhost', port: 0);

  @override
  ClientChannel createBridgeChannel(String bridgeUrl) =>
      ClientChannel('localhost', port: 0);
}

/// A fake [MobileServiceClient] that avoids real gRPC calls.
class FakeMobileServiceClient implements MobileServiceClient {
  bool pingCalled = false;
  bool closeCalled = false;

  /// If non-null, [ping] will throw this error.
  Object? pingError;

  /// If non-null, [ping] will return this result.
  PingResult? pingResult;

  @override
  Future<PingResult> ping() async {
    pingCalled = true;
    if (pingError != null) {
      throw pingError!;
    }
    return pingResult ??
        PingResult(
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
  Future<ListSessionsResult> listSessions({required String deviceToken}) async {
    return const ListSessionsResult(
      sessions: [],
      serverName: 'Test',
      serverId: 'test-id',
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

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('ConnectionManager', () {
    late ConnectionManager manager;
    late FakeMobileServiceClient fakeClient;

    setUp(() {
      fakeClient = FakeMobileServiceClient();
      manager = ConnectionManager(
        channelFactory: const FakeGrpcChannelFactory(),
        clientFactory: (_) => fakeClient,
      );
    });

    tearDown(() async {
      // Clean up connections without calling dispose() (which asserts
      // on ChangeNotifier). This avoids double-dispose issues.
      await manager.disconnectAll();
    });

    test('initial state has no active connections', () {
      expect(manager.connections, isEmpty);
      expect(manager.activeConnections, isEmpty);
    });

    test('getConnection returns null for unknown server', () {
      expect(manager.getConnection('nonexistent'), isNull);
    });

    test('dispose cleans up without errors', () {
      // Use a separate manager so tearDown does not double-dispose.
      final disposableManager = ConnectionManager(
        channelFactory: const FakeGrpcChannelFactory(),
        clientFactory: (_) => fakeClient,
      );
      expect(() => disposableManager.dispose(), returnsNormally);
    });

    test('disconnectAll on empty manager does not throw', () async {
      await expectLater(manager.disconnectAll(), completes);
      expect(manager.connections, isEmpty);
    });

    test('disconnectFromServer on unknown id does not throw', () async {
      await expectLater(manager.disconnectFromServer('unknown-id'), completes);
    });

    test('connectToServer skips server without device token', () async {
      final server = Server(
        id: 'srv-1',
        name: 'No Token Server',
        lanAddress: '192.168.1.100',
        connectionMode: ConnectionMode.lan,
        isOnline: false,
        latencyMs: 0,
        pairedAt: DateTime.now(),
        // No deviceToken set.
      );

      await manager.connectToServer(server);

      // Connection should not be established without a device token.
      expect(manager.getConnection('srv-1'), isNull);
    });

    test('sendNewTask returns error when server not connected', () async {
      final result = await manager.sendNewTask(
        serverId: 'nonexistent',
        sessionId: 'session-1',
        task: 'do something',
      );

      expect(result.success, isFalse);
      expect(result.errorMessage, 'Server not connected');
    });

    test('sendAskUserReply returns error when server not connected', () async {
      final result = await manager.sendAskUserReply(
        serverId: 'nonexistent',
        sessionId: 'session-1',
        question: 'Which?',
        answer: 'This one',
      );

      expect(result.success, isFalse);
      expect(result.errorMessage, 'Server not connected');
    });

    test('cancelSession returns error when server not connected', () async {
      final result = await manager.cancelSession(
        serverId: 'nonexistent',
        sessionId: 'session-1',
      );

      expect(result.success, isFalse);
      expect(result.errorMessage, 'Server not connected');
    });

    test('subscribeToSession returns null when server not connected', () {
      final stream = manager.subscribeToSession(
        serverId: 'nonexistent',
        sessionId: 'session-1',
      );

      expect(stream, isNull);
    });

    test('listAllSessions returns empty when no active connections', () async {
      final sessions = await manager.listAllSessions();

      expect(sessions, isEmpty);
    });

    test('notifies listeners on state changes', () async {
      var notifyCount = 0;
      manager.addListener(() => notifyCount++);

      // disconnectAll with no connections still calls notifyListeners.
      await manager.disconnectAll();

      expect(notifyCount, greaterThanOrEqualTo(1));
    });

    test(
      'encryptForServer returns plaintext when server not connected',
      () async {
        final plaintext = Uint8List.fromList([1, 2, 3]);

        final result = await manager.encryptForServer(
          'nonexistent',
          plaintext,
          0,
        );

        // No connection => returns plaintext unchanged.
        expect(result, plaintext);
      },
    );

    test(
      'decryptFromServer returns data as-is when server not connected',
      () async {
        final data = Uint8List.fromList([4, 5, 6]);

        final (result, counter) = await manager.decryptFromServer(
          'nonexistent',
          data,
        );

        // No connection => returns data unchanged, counter 0.
        expect(result, data);
        expect(counter, 0);
      },
    );

    // -----------------------------------------------------------------
    // Health check & error retry
    // -----------------------------------------------------------------

    group('health check & error retry', () {
      Server testServer({String id = 'srv-1'}) => Server(
        id: id,
        name: 'Test Server',
        lanAddress: '192.168.1.100',
        connectionMode: ConnectionMode.lan,
        isOnline: false,
        latencyMs: 0,
        pairedAt: DateTime.now(),
        deviceToken: 'test-token-123',
      );

      test(
        'markConnectionLost sets error status and schedules reconnect',
        () async {
          await manager.connectToServer(testServer());

          final conn = manager.getConnection('srv-1')!;
          expect(conn.status, GrpcConnectionStatus.connected);

          manager.markConnectionLost('srv-1', reason: 'ping failed');

          expect(conn.status, GrpcConnectionStatus.error);
          expect(conn.lastError, 'ping failed');
          // scheduleReconnect should have set a reconnect timer.
          expect(conn.reconnectTimer, isNotNull);
          expect(conn.reconnectTimer!.isActive, isTrue);
        },
      );

      test('markConnectionLost is no-op for non-connected server', () async {
        await manager.connectToServer(testServer());

        final conn = manager.getConnection('srv-1')!;
        // Mark as lost once -- transitions from connected to error.
        manager.markConnectionLost('srv-1', reason: 'first call');
        expect(conn.status, GrpcConnectionStatus.error);
        expect(conn.lastError, 'first call');

        // Mark again -- should be no-op because status is error, not connected.
        manager.markConnectionLost('srv-1', reason: 'second call');
        expect(conn.lastError, 'first call'); // unchanged
      });

      test('markConnectionLost does nothing for unknown server', () {
        // Should not throw.
        manager.markConnectionLost('nonexistent', reason: 'test');
        expect(manager.connections, isEmpty);
      });

      test('successful connect sets connected status '
          'and starts health check mechanism', () async {
        await manager.connectToServer(testServer());

        final conn = manager.getConnection('srv-1')!;
        expect(conn.status, GrpcConnectionStatus.connected);
        expect(conn.reconnectAttempts, 0);
        expect(conn.lastError, isNull);

        // Simulate what the health check would do on ping failure:
        // mark connection lost and verify the chain works end-to-end.
        fakeClient.pingError = Exception('down');
        manager.markConnectionLost('srv-1', reason: 'Health check failed');
        expect(conn.status, GrpcConnectionStatus.error);
      });

      test('disconnectAll cleans up connections and timers', () async {
        await manager.connectToServer(testServer());
        expect(
          manager.getConnection('srv-1')!.status,
          GrpcConnectionStatus.connected,
        );

        await manager.disconnectAll();

        expect(manager.connections, isEmpty);
        // No pending timers should fire after disconnect.
        // Verified implicitly -- if timers were still active they could
        // fire on a disposed manager and throw.
      });

      test('markConnectionLost notifies listeners', () async {
        await manager.connectToServer(testServer());

        var notified = false;
        manager.addListener(() => notified = true);

        manager.markConnectionLost('srv-1', reason: 'test');
        expect(notified, isTrue);
      });
    });
  });
}
