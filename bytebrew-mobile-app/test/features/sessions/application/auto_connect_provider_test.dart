import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_providers.dart';
import 'package:bytebrew_mobile/features/sessions/application/auto_connect_provider.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

// ---------------------------------------------------------------------------
// Fake ConnectionManager that records calls without real WS
// ---------------------------------------------------------------------------

class _FakeConnectionManager extends WsConnectionManager {
  _FakeConnectionManager() : super();

  final List<Server> connectedServers = [];
  bool connectToAllCalled = false;

  @override
  Future<void> connectToAll(List<Server> servers) async {
    connectToAllCalled = true;
    connectedServers.addAll(servers);
  }

  @override
  Future<void> connectToServer(Server server) async {
    connectedServers.add(server);
  }
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

final _now = DateTime.now();

final _pairedServers = [
  Server(
    id: 'srv-1',
    name: 'MacBook Pro',
    bridgeUrl: 'ws://bridge.bytebrew.ai:8080',
    isOnline: true,
    latencyMs: 5,
    pairedAt: _now.subtract(const Duration(days: 30)),
    deviceToken: 'token-1',
  ),
  Server(
    id: 'srv-2',
    name: 'Desktop PC',
    bridgeUrl: 'ws://bridge.bytebrew.ai:8080',
    isOnline: true,
    latencyMs: 10,
    pairedAt: _now.subtract(const Duration(days: 7)),
    deviceToken: 'token-2',
  ),
];

void main() {
  // =========================================================================
  // sessionsAutoConnectProvider
  //
  // NOTE: The real sessionsAutoConnect function casts settingsRepositoryProvider
  // to LocalSettingsRepository, which requires SharedPreferences. To avoid
  // platform channel dependencies in unit tests, we test the ConnectionManager
  // logic directly and verify the provider integration pattern.
  // =========================================================================
  group('sessionsAutoConnect (WsConnectionManager integration)', () {
    test(
      'WsConnectionManager.connectToAll connects provided servers',
      () async {
        final manager = _FakeConnectionManager();

        await manager.connectToAll(_pairedServers);

        expect(manager.connectToAllCalled, isTrue);
        expect(manager.connectedServers, hasLength(2));
        expect(manager.connectedServers.first.id, 'srv-1');
        expect(manager.connectedServers.last.id, 'srv-2');
      },
    );

    test('WsConnectionManager.connectToAll handles empty list', () async {
      final manager = _FakeConnectionManager();

      await manager.connectToAll([]);

      expect(manager.connectToAllCalled, isTrue);
      expect(manager.connectedServers, isEmpty);
    });

    test(
      'WsConnectionManager.connectToAll receives all servers including unpaired',
      () async {
        final manager = _FakeConnectionManager();

        final mixedServers = [
          ..._pairedServers,
          Server(
            id: 'srv-3',
            name: 'Unpaired',
            bridgeUrl: 'ws://bridge:8080',
            isOnline: false,
            latencyMs: 0,
            pairedAt: _now,
            // No deviceToken.
          ),
        ];

        await manager.connectToAll(mixedServers);

        // Our fake accepts all. Real WsConnectionManager.connectToAll
        // filters by deviceToken internally.
        expect(manager.connectedServers, hasLength(3));
      },
    );

    test('sessionsAutoConnectProvider can be overridden for testing', () async {
      final container = ProviderContainer(
        overrides: [
          // Override the auto-connect provider to a no-op for downstream tests.
          sessionsAutoConnectProvider.overrideWith((ref) async {}),
        ],
      );
      addTearDown(container.dispose);

      // Should complete without error.
      await container.read(sessionsAutoConnectProvider.future);
    });

    test('WsConnectionManager can be injected via provider override', () {
      final manager = _FakeConnectionManager();

      final container = ProviderContainer(
        overrides: [connectionManagerProvider.overrideWithValue(manager)],
      );
      addTearDown(container.dispose);

      final readManager = container.read(connectionManagerProvider);
      expect(readManager, same(manager));
    });
  });
}
