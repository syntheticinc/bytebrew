import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_providers.dart';
import 'package:bytebrew_mobile/core/widgets/status_indicator.dart';
import 'package:bytebrew_mobile/features/settings/presentation/widgets/server_list_tile.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

import '../../../../helpers/fakes.dart';

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

final _now = DateTime.now();

final _onlineServer = Server(
  id: 'srv-1',
  name: 'MacBook Pro',
  bridgeUrl: 'ws://bridge.bytebrew.ai:8080',
  isOnline: true,
  latencyMs: 5,
  pairedAt: _now.subtract(const Duration(days: 30)),
);

final _offlineServer = Server(
  id: 'srv-2',
  name: 'Desktop PC',
  bridgeUrl: 'ws://bridge.bytebrew.ai:8080',
  isOnline: false,
  latencyMs: 0,
  pairedAt: _now.subtract(const Duration(days: 7)),
);

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

void main() {
  testWidgets('ServerListTile renders server name', (tester) async {
    final manager = FakeConnectionManager();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [connectionManagerProvider.overrideWithValue(manager)],
        child: MaterialApp(
          home: Scaffold(body: ServerListTile(server: _onlineServer)),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('MacBook Pro'), findsOneWidget);
  });

  testWidgets('ServerListTile shows "Offline" subtitle when not connected', (
    tester,
  ) async {
    final manager = FakeConnectionManager();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [connectionManagerProvider.overrideWithValue(manager)],
        child: MaterialApp(
          home: Scaffold(body: ServerListTile(server: _onlineServer)),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Offline'), findsOneWidget);
  });

  testWidgets('ServerListTile shows "Online" subtitle when connected', (
    tester,
  ) async {
    final manager = FakeConnectionManager();

    // Create a fake connection with connected status.
    final conn = WsServerConnection(
      server: _onlineServer,
      connection: WsConnection(
        bridgeUrl: _onlineServer.bridgeUrl,
        serverId: _onlineServer.id,
        deviceId: 'dev-1',
      ),
      client: _FakeWsBridgeClient(),
    );
    conn.status = WsConnectionStatus.connected;
    manager.addFakeConnection(_onlineServer.id, conn);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [connectionManagerProvider.overrideWithValue(manager)],
        child: MaterialApp(
          home: Scaffold(body: ServerListTile(server: _onlineServer)),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Online'), findsOneWidget);
    expect(find.text('5ms'), findsOneWidget);
  });

  testWidgets('ServerListTile shows no latency badge when offline', (
    tester,
  ) async {
    final manager = FakeConnectionManager();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [connectionManagerProvider.overrideWithValue(manager)],
        child: MaterialApp(
          home: Scaffold(body: ServerListTile(server: _offlineServer)),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Desktop PC'), findsOneWidget);
    expect(find.text('Offline'), findsOneWidget);
    // No latency badge for offline servers.
    expect(find.textContaining('ms'), findsNothing);
  });

  testWidgets('ServerListTile renders StatusIndicator', (tester) async {
    final manager = FakeConnectionManager();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [connectionManagerProvider.overrideWithValue(manager)],
        child: MaterialApp(
          home: Scaffold(body: ServerListTile(server: _onlineServer)),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.byType(StatusIndicator), findsOneWidget);
  });

  testWidgets('ServerListTile is dismissible when onDismissed is set', (
    tester,
  ) async {
    final manager = FakeConnectionManager();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [connectionManagerProvider.overrideWithValue(manager)],
        child: MaterialApp(
          home: Scaffold(
            body: ServerListTile(server: _onlineServer, onDismissed: () {}),
          ),
        ),
      ),
    );

    await tester.pumpAndSettle();

    // Verify Dismissible is present when onDismissed is provided.
    expect(find.byType(Dismissible), findsOneWidget);
  });

  testWidgets('ServerListTile is not dismissible when onDismissed is null', (
    tester,
  ) async {
    final manager = FakeConnectionManager();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [connectionManagerProvider.overrideWithValue(manager)],
        child: MaterialApp(
          home: Scaffold(body: ServerListTile(server: _onlineServer)),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.byType(Dismissible), findsNothing);
  });
}

/// Minimal fake for WsBridgeClient to create WsServerConnection in tests.
///
/// WsBridgeClient is not used in the tile rendering, so this is a no-op stub.
class _FakeWsBridgeClient implements WsBridgeClient {
  @override
  dynamic noSuchMethod(Invocation invocation) => null;
}
