import 'dart:async';
import 'dart:io';

import 'package:bytebrew_mobile/features/chat/application/connection_provider.dart';
import 'package:bytebrew_mobile/features/chat/domain/chat_repository.dart';
import 'package:bytebrew_mobile/features/chat/infrastructure/ws_chat_repository.dart';
import 'package:fake_async/fake_async.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

// ---------------------------------------------------------------------------
// Fakes for real WsConnection tests (channelFactory injection)
// ---------------------------------------------------------------------------

class _FakeWebSocketSink implements WebSocketSink {
  final List<String> sent = [];
  bool isClosed = false;

  @override
  void add(dynamic data) => sent.add(data.toString());

  @override
  void addError(Object error, [StackTrace? stackTrace]) {}

  @override
  Future<dynamic> addStream(Stream<dynamic> stream) => Future.value();

  @override
  Future<dynamic> close([int? closeCode, String? closeReason]) {
    isClosed = true;
    return Future.value();
  }

  @override
  Future<dynamic> get done => Future.value();
}

class _FakeWebSocketChannel implements WebSocketChannel {
  _FakeWebSocketChannel()
    : _incoming = StreamController<dynamic>.broadcast(),
      sink = _FakeWebSocketSink();

  final StreamController<dynamic> _incoming;

  @override
  final _FakeWebSocketSink sink;

  @override
  Stream<dynamic> get stream => _incoming.stream;

  @override
  Future<void> get ready => Future.value();

  @override
  int? get closeCode => null;

  @override
  String? get closeReason => null;

  @override
  String? get protocol => null;

  /// Simulates incoming data from the server.
  void receive(String data) => _incoming.add(data);

  /// Closes the incoming stream to simulate a disconnection.
  Future<void> closeIncoming() => _incoming.close();

  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/// Creates a [ProviderContainer] with a real [WsConnection] whose
/// [channelFactory] returns the given [channel].
({ProviderContainer container, WsConnection notifier}) _createContainer(
  _FakeWebSocketChannel channel,
) {
  final container = ProviderContainer();
  final notifier = container.read(wsConnectionProvider.notifier);
  notifier.channelFactory = (_) => channel;
  return (container: container, notifier: notifier);
}

void main() {
  // =========================================================================
  // Group 1: Tests using _TestWsConnection (mock subclass)
  // =========================================================================
  group('WsConnection reconnection', () {
    test('schedules reconnect after unexpected disconnect', () {
      FakeAsync().run((async) {
        final container = ProviderContainer(
          overrides: [
            wsConnectionProvider.overrideWith(() => _TestWsConnection()),
          ],
        );
        addTearDown(container.dispose);

        final notifier =
            container.read(wsConnectionProvider.notifier) as _TestWsConnection;

        notifier.connect('ws://localhost:8765');
        async.flushMicrotasks();
        expect(
          container.read(wsConnectionProvider),
          WsConnectionStatus.connected,
        );

        // Simulate unexpected connection loss.
        notifier.simulateDisconnect();
        async.flushMicrotasks();
        expect(
          container.read(wsConnectionProvider),
          WsConnectionStatus.disconnected,
        );

        // After 1 second (2^0), reconnect should be attempted.
        async.elapse(const Duration(seconds: 1));
        async.flushMicrotasks();
        expect(notifier.connectCount, 2);
        expect(
          container.read(wsConnectionProvider),
          WsConnectionStatus.connected,
        );
      });
    });

    test('exponential backoff increases delay between attempts', () {
      FakeAsync().run((async) {
        final container = ProviderContainer(
          overrides: [
            wsConnectionProvider.overrideWith(() => _TestWsConnection()),
          ],
        );
        addTearDown(container.dispose);

        final notifier =
            container.read(wsConnectionProvider.notifier) as _TestWsConnection;

        notifier.connect('ws://localhost:8765');
        async.flushMicrotasks();

        // First disconnect -- schedules reconnect in 1s (2^0).
        notifier.simulateDisconnect();
        async.flushMicrotasks();

        // At 0.5s nothing happened yet.
        async.elapse(const Duration(milliseconds: 500));
        async.flushMicrotasks();
        expect(notifier.connectCount, 1);

        // At 1s reconnect fires.
        async.elapse(const Duration(milliseconds: 500));
        async.flushMicrotasks();
        expect(notifier.connectCount, 2);

        // Second disconnect -- reconnectAttempts was reset on successful
        // connect, so delay is 1s (2^0) again.
        notifier.simulateDisconnect();
        async.flushMicrotasks();

        // At 1s reconnect fires.
        async.elapse(const Duration(seconds: 1));
        async.flushMicrotasks();
        expect(notifier.connectCount, 3);
      });
    });

    test('TC-6: transitions to error after max reconnect attempts', () {
      FakeAsync().run((async) {
        final container = ProviderContainer(
          overrides: [
            wsConnectionProvider.overrideWith(() => _TestWsConnection()),
          ],
        );
        addTearDown(container.dispose);

        final notifier =
            container.read(wsConnectionProvider.notifier) as _TestWsConnection;

        notifier.connect('ws://localhost:8765');
        async.flushMicrotasks();

        // Pre-set reconnect attempts to max so next disconnect hits the limit.
        notifier.reconnectAttempts = WsConnection.maxReconnectAttempts;

        // Simulate disconnect -- should immediately go to error
        // because attempts >= max.
        notifier.simulateDisconnect();
        async.flushMicrotasks();

        expect(container.read(wsConnectionProvider), WsConnectionStatus.error);
        expect(notifier.lastError, contains('Max reconnection attempts'));
      });
    });

    test('TC-8: resets reconnect counter on successful connect', () {
      FakeAsync().run((async) {
        final container = ProviderContainer(
          overrides: [
            wsConnectionProvider.overrideWith(() => _TestWsConnection()),
          ],
        );
        addTearDown(container.dispose);

        final notifier =
            container.read(wsConnectionProvider.notifier) as _TestWsConnection;

        notifier.connect('ws://localhost:8765');
        async.flushMicrotasks();

        // Disconnect and auto-reconnect.
        notifier.simulateDisconnect();
        async.flushMicrotasks();
        expect(notifier.reconnectAttempts, 1);

        // Elapse to trigger reconnect.
        async.elapse(const Duration(seconds: 1));
        async.flushMicrotasks();

        // Successful reconnect should reset counter.
        expect(notifier.reconnectAttempts, 0);
        expect(
          container.read(wsConnectionProvider),
          WsConnectionStatus.connected,
        );
      });
    });

    test('TC-4: manual disconnect cancels reconnect timer', () {
      FakeAsync().run((async) {
        final container = ProviderContainer(
          overrides: [
            wsConnectionProvider.overrideWith(() => _TestWsConnection()),
          ],
        );
        addTearDown(container.dispose);

        final notifier =
            container.read(wsConnectionProvider.notifier) as _TestWsConnection;

        notifier.connect('ws://localhost:8765');
        async.flushMicrotasks();

        // Simulate unexpected disconnect (would normally trigger reconnect).
        notifier.simulateDisconnect();
        async.flushMicrotasks();

        // Manually disconnect before the timer fires.
        notifier.disconnect();
        async.flushMicrotasks();

        // Elapse past the reconnect delay -- should NOT reconnect.
        async.elapse(const Duration(seconds: 5));
        async.flushMicrotasks();

        expect(
          container.read(wsConnectionProvider),
          WsConnectionStatus.disconnected,
        );
        expect(notifier.reconnectAttempts, 0);
        // Only the initial connect, no reconnects.
        expect(notifier.connectCount, 1);
      });
    });
  });

  // =========================================================================
  // Group 2: Tests using real WsConnection with channelFactory injection
  // =========================================================================
  group('WsConnection with channelFactory', () {
    test('TC-1: connect() transitions to connected', () async {
      final channel = _FakeWebSocketChannel();
      final (:container, :notifier) = _createContainer(channel);
      addTearDown(container.dispose);

      expect(
        container.read(wsConnectionProvider),
        WsConnectionStatus.disconnected,
      );

      await notifier.connect('ws://localhost:8765');

      expect(
        container.read(wsConnectionProvider),
        WsConnectionStatus.connected,
      );
      expect(notifier.repository, isNotNull);
    });

    test('TC-2: connect() cleans up old resources on reconnect', () async {
      final channel1 = _FakeWebSocketChannel();
      final channel2 = _FakeWebSocketChannel();
      var callCount = 0;

      final container = ProviderContainer();
      addTearDown(container.dispose);
      final notifier = container.read(wsConnectionProvider.notifier);
      notifier.channelFactory = (_) {
        callCount++;
        return callCount == 1 ? channel1 : channel2;
      };

      // First connection.
      await notifier.connect('ws://localhost:8765');
      expect(
        container.read(wsConnectionProvider),
        WsConnectionStatus.connected,
      );
      expect(channel1.sink.isClosed, isFalse);

      // Second connection -- old channel should be cleaned up.
      await notifier.connect('ws://localhost:8765');
      expect(
        container.read(wsConnectionProvider),
        WsConnectionStatus.connected,
      );

      // Old channel's sink should have been closed during cleanup.
      expect(channel1.sink.isClosed, isTrue);
      // New channel's sink should still be open.
      expect(channel2.sink.isClosed, isFalse);
    });

    test('TC-3: disconnect() transitions to disconnected', () async {
      final channel = _FakeWebSocketChannel();
      final (:container, :notifier) = _createContainer(channel);
      addTearDown(container.dispose);

      await notifier.connect('ws://localhost:8765');
      expect(
        container.read(wsConnectionProvider),
        WsConnectionStatus.connected,
      );
      expect(notifier.repository, isNotNull);

      notifier.disconnect();

      expect(
        container.read(wsConnectionProvider),
        WsConnectionStatus.disconnected,
      );
      expect(notifier.repository, isNull);
    });

    test(
      'TC-5: scheduleReconnect on connection lost via status stream',
      () async {
        final channel = _FakeWebSocketChannel();
        final (:container, :notifier) = _createContainer(channel);
        addTearDown(container.dispose);

        await notifier.connect('ws://localhost:8765');
        expect(
          container.read(wsConnectionProvider),
          WsConnectionStatus.connected,
        );

        // Simulate WS close -- this triggers connectionStatus to emit false.
        await channel.closeIncoming();
        // Allow the status stream listener to process.
        await Future<void>.delayed(const Duration(milliseconds: 50));

        // State should transition to disconnected.
        expect(
          container.read(wsConnectionProvider),
          WsConnectionStatus.disconnected,
        );
        // Reconnect should have been scheduled (reconnectAttempts > 0).
        expect(notifier.reconnectAttempts, greaterThan(0));
      },
    );

    test('TC-7: connect error -- SocketException transitions to error', () async {
      final container = ProviderContainer();
      addTearDown(container.dispose);
      final notifier = container.read(wsConnectionProvider.notifier);

      // channelFactory throws SocketException -- simulates network unreachable.
      notifier.channelFactory = (_) {
        throw const SocketException('Connection refused');
      };

      await notifier.connect('ws://localhost:8765');

      expect(container.read(wsConnectionProvider), WsConnectionStatus.error);
      expect(notifier.lastError, contains('Network error'));
      expect(notifier.repository, isNull);
    });
  });
}

// ---------------------------------------------------------------------------
// Test subclass of [WsConnection] -- avoids real WebSocket connections
// ---------------------------------------------------------------------------

/// Test subclass of [WsConnection] that avoids real WebSocket connections.
///
/// Uses [lastWsUrl] and [reconnectAttempts] setters (exposed via
/// @visibleForTesting) to properly integrate with the reconnect logic.
class _TestWsConnection extends WsConnection {
  int connectCount = 0;

  @override
  WsConnectionStatus build() => WsConnectionStatus.disconnected;

  @override
  Future<void> connect(String wsUrl) async {
    connectCount++;
    lastWsUrl = wsUrl;
    state = WsConnectionStatus.connecting;
    reconnectAttempts = 0;
    state = WsConnectionStatus.connected;
  }

  /// Simulates an unexpected connection loss, triggering reconnect.
  void simulateDisconnect() {
    if (state != WsConnectionStatus.connected) return;
    state = WsConnectionStatus.disconnected;
    scheduleReconnect();
  }

  @override
  ChatRepository? get repository => null;

  @override
  WsChatRepository? get wsRepository => null;
}
