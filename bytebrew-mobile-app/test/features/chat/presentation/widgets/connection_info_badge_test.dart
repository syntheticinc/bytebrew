import 'package:bytebrew_mobile/core/domain/server.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_bridge_client.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection.dart';
import 'package:bytebrew_mobile/core/infrastructure/ws/ws_connection_manager.dart';
import 'package:bytebrew_mobile/features/chat/presentation/widgets/connection_info_badge.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

// ---------------------------------------------------------------------------
// Fake stubs for WsConnection and WsBridgeClient
// ---------------------------------------------------------------------------

class _FakeWsConnection implements WsConnection {
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

class _FakeWsBridgeClient implements WsBridgeClient {
  @override
  dynamic noSuchMethod(Invocation invocation) => super.noSuchMethod(invocation);
}

// ---------------------------------------------------------------------------
// Test data
// ---------------------------------------------------------------------------

final _now = DateTime.now();

final _testServer = Server(
  id: 'srv-1',
  name: 'Test Server',
  bridgeUrl: 'ws://bridge.bytebrew.ai:8080',
  isOnline: true,
  latencyMs: 5,
  pairedAt: _now,
);

WsServerConnection _makeConnection({required WsConnectionStatus status}) {
  final connection = WsServerConnection(
    server: _testServer,
    connection: _FakeWsConnection(),
    client: _FakeWsBridgeClient(),
  );
  connection.status = status;
  return connection;
}

Widget _buildBadge(WsServerConnection connection) {
  return MaterialApp(
    home: Scaffold(body: ConnectionInfoBadge(connection: connection)),
  );
}

void main() {
  testWidgets('ConnectionInfoBadge shows "Bridge" for connected connection', (
    tester,
  ) async {
    final connection = _makeConnection(status: WsConnectionStatus.connected);

    await tester.pumpWidget(_buildBadge(connection));
    await tester.pumpAndSettle();

    expect(find.text('Bridge'), findsOneWidget);
  });

  testWidgets('ConnectionInfoBadge shows cloud icon', (tester) async {
    final connection = _makeConnection(status: WsConnectionStatus.connected);

    await tester.pumpWidget(_buildBadge(connection));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.cloud_outlined), findsOneWidget);
  });

  testWidgets('ConnectionInfoBadge hides lock icon when no encryption', (
    tester,
  ) async {
    final connection = _makeConnection(status: WsConnectionStatus.connected);
    // No cipher or sharedSecret set -> hasEncryption = false.

    await tester.pumpWidget(_buildBadge(connection));
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.lock), findsNothing);
  });

  testWidgets(
    'ConnectionInfoBadge shows "Bridge" for disconnected connection',
    (tester) async {
      final connection = _makeConnection(
        status: WsConnectionStatus.disconnected,
      );

      await tester.pumpWidget(_buildBadge(connection));
      await tester.pumpAndSettle();

      // Badge always shows "Bridge" regardless of status.
      expect(find.text('Bridge'), findsOneWidget);
    },
  );
}
