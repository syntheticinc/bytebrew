import 'package:bytebrew_mobile/core/domain/server.dart';
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

final _bridgeServer = Server(
  id: 'srv-2',
  name: 'Desktop PC',
  bridgeUrl: 'ws://bridge.bytebrew.ai:8080',
  isOnline: true,
  latencyMs: 45,
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

  testWidgets('ServerListTile shows bridge URL in subtitle', (tester) async {
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

    expect(find.text('Bridge: ws://bridge.bytebrew.ai:8080'), findsOneWidget);
  });

  testWidgets('ServerListTile shows latency', (tester) async {
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

    expect(find.text('5ms'), findsOneWidget);
  });

  testWidgets('ServerListTile shows "Offline" route when not connected', (
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

  testWidgets('ServerListTile shows latency for bridge server', (tester) async {
    final manager = FakeConnectionManager();

    await tester.pumpWidget(
      ProviderScope(
        overrides: [connectionManagerProvider.overrideWithValue(manager)],
        child: MaterialApp(
          home: Scaffold(body: ServerListTile(server: _bridgeServer)),
        ),
      ),
    );

    await tester.pumpAndSettle();

    expect(find.text('Desktop PC'), findsOneWidget);
    expect(find.text('45ms'), findsOneWidget);
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
