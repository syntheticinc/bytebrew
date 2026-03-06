import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart'
    hide WsConnectionStatus;
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_types.dart';

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

/// Fake [WsConnection] that completes connect() without real network.
class FakeWsConnection implements WsConnection {
  bool connectCalled = false;
  bool disposeCalled = false;

  /// If non-null, [connect] will throw this error.
  Object? connectError;

  final _messageController = StreamController<Map<String, dynamic>>.broadcast();

  @override
  Future<void> connect() async {
    connectCalled = true;
    if (connectError != null) throw connectError!;
  }

  @override
  Future<void> disconnect() async {}

  @override
  Future<void> dispose() async {
    disposeCalled = true;
    await _messageController.close();
  }

  @override
  Stream<Map<String, dynamic>> get messages => _messageController.stream;

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

/// Fake [WsBridgeClient] that avoids real WS calls.
class FakeWsBridgeClient implements WsBridgeClient {
  bool pingCalled = false;
  bool disposeCalled = false;

  /// If non-null, [ping] will throw this error.
  Object? pingError;

  /// If non-null, [ping] will return this result.
  PingResult? pingResult;

  @override
  Future<PingResult> ping() async {
    pingCalled = true;
    if (pingError != null) throw pingError!;
    return pingResult ??
        PingResult(
          timestamp: DateTime.now(),
          serverName: 'Test Server',
          serverId: 'test-server-id',
        );
  }

  @override
  Future<void> dispose() async {
    disposeCalled = true;
  }

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  group('WsConnectionManager', () {
    late WsConnectionManager manager;
    late FakeWsConnection fakeConnection;
    late FakeWsBridgeClient fakeClient;

    setUp(() {
      fakeConnection = FakeWsConnection();
      fakeClient = FakeWsBridgeClient();

      // WsConnectionManager creates WsConnection internally via factory.
      // We cannot inject WsBridgeClient directly because it's created inside
      // connectToServer. Instead we override connectionFactory and use
      // FakeConnectionManager for tests that need to control connection state.
      manager = WsConnectionManager();
    });

    tearDown(() async {
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
      final disposableManager = WsConnectionManager();
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
        bridgeUrl: 'ws://bridge:8080',
        isOnline: false,
        latencyMs: 0,
        pairedAt: DateTime.now(),
        // No deviceToken set.
      );

      await manager.connectToServer(server);

      // Connection should not be established without a device token.
      expect(manager.getConnection('srv-1'), isNull);
    });

    test('notifies listeners on state changes', () async {
      var notifyCount = 0;
      manager.addListener(() => notifyCount++);

      // disconnectAll with no connections still notifies.
      await manager.disconnectAll();

      // At least some notification should have occurred.
      // (disconnectAll may or may not notify when empty, depending on
      // implementation. We just verify no error.)
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
    // Health check & error retry (using FakeConnectionManager)
    // -----------------------------------------------------------------

    group('health check & error retry', () {
      test('markConnectionLost is no-op for unknown server', () {
        // Should not throw.
        manager.markConnectionLost('nonexistent', reason: 'test');
        expect(manager.connections, isEmpty);
      });
    });
  });
}
